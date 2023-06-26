package egressip

import (
	"fmt"
	"github.com/ovn-org/ovn-kubernetes/go-controller/pkg/util"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"net"
	"net/netip"
	"sync"
	"time"

	egressipinformer "github.com/ovn-org/ovn-kubernetes/go-controller/pkg/crd/egressip/v1/apis/informers/externalversions/egressip/v1"
	egressiplisters "github.com/ovn-org/ovn-kubernetes/go-controller/pkg/crd/egressip/v1/apis/listers/egressip/v1"
	"github.com/ovn-org/ovn-kubernetes/go-controller/pkg/node/iptables"
	"github.com/ovn-org/ovn-kubernetes/go-controller/pkg/node/linkmanager"
	"github.com/ovn-org/ovn-kubernetes/go-controller/pkg/node/routemanager"
	"github.com/ovn-org/ovn-kubernetes/go-controller/pkg/node/rulemanager"
	"k8s.io/apimachinery/pkg/labels"
	v12 "k8s.io/client-go/informers/core/v1"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
	utiliptables "k8s.io/kubernetes/pkg/util/iptables"
	net2 "k8s.io/utils/net"

	"github.com/gaissmai/cidrtree"
	"github.com/vishvananda/netlink"
)

const (
	ipRulePriority    = 6000 // the priority of the ip rules created by the controller. Egress Service priority is 5000.
	routeTableStart   = 1000
	multiNICChainName = "OVN-KUBE-EGRESS-IP-Multi-NIC"
	multiNICChain     = utiliptables.Chain(multiNICChainName)
	skipBridges       = true
)

var (
	_, defaultCIDR, _ = net.ParseCIDR("0.0.0.0/0")
	jumpRule          = []iptables.RuleArgs{{Args: []string{"-j", multiNICChainName}}}
)

type Controller struct {
	egressIPLister egressiplisters.EgressIPLister
	egressIPSynced cache.InformerSynced

	nodeLister corelisters.NodeLister
	nodeSynced cache.InformerSynced

	namespaceLister corelisters.NamespaceLister
	namespaceSynced cache.InformerSynced

	podLister corelisters.PodLister
	podSynced cache.InformerSynced

	routeManager    *routemanager.Controller
	linkManager     *linkmanager.Controller
	ruleManager     *rulemanager.Controller
	iptablesManager *iptables.Controller

	triggerReconcileCh chan struct{}
	nodeName           string
	v4                 bool
	v6                 bool
}

func NewController(eIPInformer egressipinformer.EgressIPInformer, nodeInformer cache.SharedIndexInformer,
	namespaceInformer v12.NamespaceInformer, podInformer v12.PodInformer, routeManager *routemanager.Controller,
	v4, v6 bool, nodeName string) (*Controller, error) {
	var err error
	c := &Controller{
		egressIPLister:     eIPInformer.Lister(),
		egressIPSynced:     eIPInformer.Informer().HasSynced,
		nodeLister:         corelisters.NewNodeLister(nodeInformer.GetIndexer()),
		nodeSynced:         nodeInformer.HasSynced,
		namespaceLister:    namespaceInformer.Lister(),
		namespaceSynced:    namespaceInformer.Informer().HasSynced,
		podLister:          podInformer.Lister(),
		podSynced:          podInformer.Informer().HasSynced,
		routeManager:       routeManager,
		linkManager:        linkmanager.NewController(nodeName, v4, v6),
		ruleManager:        rulemanager.NewController(v4, v6),
		iptablesManager:    iptables.NewController(v4, v6),
		triggerReconcileCh: make(chan struct{}, 1),
		nodeName:           nodeName,
		v4:                 v4,
		v6:                 v6,
	}

	for _, resourceInformer := range []struct {
		resourceName string
		cache.SharedIndexInformer
	}{
		{
			"egressip",
			eIPInformer.Informer(),
		},
		{
			"node",
			nodeInformer,
		},
		{
			"namespace",
			namespaceInformer.Informer(),
		},
		{
			"pod",
			podInformer.Informer(),
		},
	} {
		_, err = resourceInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
			AddFunc:    c.onAdd,
			UpdateFunc: c.onUpdate,
			DeleteFunc: c.onDelete,
		})
		if err != nil {
			return c, fmt.Errorf("failed to add event hander for resource %q: %v", resourceInformer.resourceName, err)
		}
	}
	return c, nil
}

func (c *Controller) Run(stopCh <-chan struct{}, wg *sync.WaitGroup) error {
	klog.Info("Starting Egress IP Controller")

	for _, resource := range []struct {
		name string
		cache.InformerSynced
	}{
		{
			"egressip",
			c.egressIPSynced,
		},
		{
			"node",
			c.nodeSynced,
		},
		{
			"namespace",
			c.namespaceSynced,
		},
		{
			"pod",
			c.podSynced,
		},
	} {
		if !cache.WaitForNamedCacheSync(resource.name, stopCh, resource.InformerSynced) {
			return fmt.Errorf("timed out waiting for %q caches to sync", resource.name)
		}
	}
	// tell rule manager that we want to fully own all rules at a particular priority. Any rules created with this priority
	// ,and we do not recognize it, will be removed.
	if err := c.ruleManager.OwnPriority(ipRulePriority); err != nil {
		klog.Errorf("Egress IP controller: failed to own priority %d - stale entries maybe present: %v", ipRulePriority, err)
	}

	go func() {
		var err error
		for {
			select {
			case <-stopCh:
				wg.Done()
				return
			case <-c.triggerReconcileCh:
				if err = c.reconcile(); err != nil {
					klog.Errorf("Failed to reconcile egress IP: %v", err)
				}
			}
		}
	}()

	wg.Add(3)
	go func() {
		c.linkManager.Run(stopCh, 1*time.Minute)
		wg.Done()
	}()

	go func() {
		c.iptablesManager.Run(stopCh, 2*time.Minute)
		wg.Done()
	}()

	go func() {
		c.ruleManager.Run(stopCh, 3*time.Minute)
		wg.Done()
	}()

	return nil
}

func (c *Controller) triggerReconcile() {
	select {
	case c.triggerReconcileCh <- struct{}{}:
	default:
	}
}

func (c *Controller) onAdd(_ interface{}) {
	c.triggerReconcile()
}

func (c *Controller) onUpdate(_, _ interface{}) {
	c.triggerReconcile()
}

func (c *Controller) onDelete(_ interface{}) {
	c.triggerReconcile()
}

func (c *Controller) reconcile() error {
	var err error
	egressIPs, err := c.egressIPLister.List(labels.Everything())
	if err != nil {
		return fmt.Errorf("failed to ensure egress IP is correctly configured because we could not list egress IPs: %v", err)
	}
	links, err := netlink.LinkList()
	if err != nil {
		return fmt.Errorf("failed to ensure IP is correctly configured becase we could not list links: %v", err)
	}

	// get address map for each interface -> addresses/mask
	// also map address/mask -> interface name
	linkAddresses := make(map[string][]netlink.Addr)
	existingLinkRoutes := make(map[string][]netlink.Route)
	for _, link := range links {
		addresses, err := linkmanager.GetExternallyAvailableNetlinkLinkAddresses(link, c.v4, c.v6, skipBridges)
		if err != nil {
			klog.Errorf("Egress IP skipping link %s because unable to get link addresses: %v", link.Attrs().Name, err)
			continue
		}
		linkAddresses[link.Attrs().Name] = addresses

		filter, mask := filterRouteByLinkTable(link.Attrs().Index, getRouteTableID(link.Attrs().Index))
		existingRoutes, err := netlink.RouteListFiltered(netlink.FAMILY_ALL, filter, mask)
		if err != nil {
			klog.Errorf("Failed to list routes for link %s in-order to cleanup stale egress IPs: %v", link.Attrs().Name, err)
			continue
		}
		// netlink package returns a nil value for route dst field when the dst is address 0.0.0.0/0
		// lets manually detect this and fill in dst
		// remove after: https://github.com/vishvananda/netlink/pull/852
		for i, existingRoute := range existingRoutes {
			if existingRoute.Dst == nil || existingRoute.Dst.IP == nil {
				if existingRoute.Src == nil && existingRoute.Gw == nil && existingRoute.MPLSDst == nil {
					// broadcast is 0.0.0.0/0 constant
					existingRoutes[i].Dst = defaultCIDR
				}
			}
			//TODO: remove invalid default routes

		}
		existingLinkRoutes[link.Attrs().Name] = existingRoutes
	}

	filter, mask := filterRuleByPriority(ipRulePriority)
	existingRules, err := netlink.RuleListFiltered(netlink.FAMILY_ALL, filter, mask)
	if err != nil {
		return fmt.Errorf("failed to list rules: %v", err)
	}

	validRules := make([]netlink.Rule, 0)
	validEIPAddressess := make([]netlink.Addr, 0)
	validRoutesPerLinks := make([]routemanager.RoutesPerLink, 0)
	validV4IPTablesRules := make([]iptables.RuleArgs, 0)
	validV6IPTablesRules := make([]iptables.RuleArgs, 0)

	var switcha bool
	for _, eIP := range egressIPs {
		if len(eIP.Status.Items) == 0 {
			continue
		}
		for _, status := range eIP.Status.Items {
			if status.Node != c.nodeName {
				switcha = true
				continue
			}
			if status.EgressIP == "" {
				continue
			}

			var isOVNManaged bool
			node, err := c.nodeLister.Get(status.Node)
			if err == nil {
				isOVNManaged, err = util.IsOVNManagedNetwork(node, net.ParseIP(status.EgressIP))
				if err != nil {
					klog.Errorf("Egress IP: failed to determine if egress IP %s is OVN managed: %v", status.EgressIP, err)
					continue
				}
			}
			if isOVNManaged {
				continue
			}
			eIPIP, eIPIPNet, err := net.ParseCIDR(fmt.Sprintf("%s/32", status.EgressIP))
			if err != nil {
				klog.Errorf("Failed to parse egress IP %q: %v", status.EgressIP, err)
				continue
			}
			// TODO: Validate there isnt already an IP on the system on any interface and addresses with different characteristics
			// that we dont expose to known node annotation host-addresses i.e. openvswitch type interface addresses or
			// even addresses on loopback interfaces.
			found, link, err := c.GetLinkContainingIPAndValidate(eIPIP, status.Network)
			if err != nil {
				klog.Fatalf("Failed to find network for egress IP %s and therefore failed to configure this egress IP: %v", status.EgressIP, err)
				//continue
			}
			if !found {
				klog.Errorf("Expected to find network interface to assign egress IP %s. Failed to assign", status.EgressIP)
				//TODO: figure out if we should now clear the status and force CM to recompute
				continue
			}
			if link == nil {
				klog.Errorf("Failed to get a valid link for egress IP %s and network %s", status.EgressIP, status.Network)
				continue
			}
			namespaceSelector, err := metav1.LabelSelectorAsSelector(&eIP.Spec.NamespaceSelector)
			if err != nil {
				klog.Errorf("Invalid namespaceSelector for egress IP %s: %v", eIP.Name, err)
				continue
			}
			podSelector, err := metav1.LabelSelectorAsSelector(&eIP.Spec.PodSelector)
			if err != nil {
				klog.Errorf("Invalid podSelector for egress IP %s: %v", eIP.Name, err)
				continue
			}
			namespaces, err := c.namespaceLister.List(namespaceSelector)
			if err != nil {
				klog.Errorf("Failed to list namespaces using selector %s to configure egress IP %s: %v",
					namespaceSelector.String(), eIP.Name, err)
				continue
			}

			selectedPods := make([]*v1.Pod, 0)
			for _, namespace := range namespaces {
				namespaceLabels := labels.Set(namespace.Labels)
				if namespaceSelector.Matches(namespaceLabels) {
					pods, err := c.podLister.Pods(namespace.Name).List(podSelector)
					if err != nil {
						klog.Errorf("Failed to list pods using selector %s to configure egress IP %s: %v",
							podSelector.String(), eIP.Name, err)
						continue
					}
					for _, pod := range pods {
						if util.PodCompleted(pod) {
							continue
						}
						if len(pod.Status.PodIPs) == 0 {
							continue
						}
						selectedPods = append(selectedPods, pod)
					}
				}
			}
			// need to determine which pods are selected for this.. and for each pod create rule + iptables
			// TODO (martinkennelly): extract logic from cluster manager to "match" pods and import it here and within cluster manager
			for _, pod := range selectedPods {
				r := netlink.NewRule()
				r.Table = link.Attrs().Index + routeTableStart
				r.Priority = ipRulePriority
				if pod.Status.PodIP == "" || len(pod.Status.PodIPs) == 0 {
					continue
				}
				for _, podIP := range pod.Status.PodIPs {
					_, podIPNet, isIPv6, err := getNetworkInfoFromIP(podIP.IP)
					if err != nil {
						klog.Errorf("Failed to configure egress IP %s for pod %s because unable to get pod IP: %v",
							eIP.Name, pod.Name, err)
						continue
					}
					r.Src = podIPNet
					if isIPv6 {
						r.Family = netlink.FAMILY_V6
					} else {
						r.Family = netlink.FAMILY_V4
					}
					validRules = append(validRules, *r)
					args := []string{"-s", podIPNet.String(), "-o", link.Attrs().Name, "-j", "SNAT", "--to-source", status.EgressIP}
					ruleArgs := iptables.RuleArgs{Args: args}
					if isIPv6 {
						validV6IPTablesRules = append(validV6IPTablesRules, ruleArgs)
					} else {
						validV4IPTablesRules = append(validV4IPTablesRules, ruleArgs)
					}
				}
			}
			if len(validV4IPTablesRules) > 0 || len(validV6IPTablesRules) > 0 {
				// EIP is to be set when node is selected, pods have been selected
				validEIPAddressess = append(validEIPAddressess, netlink.Addr{
					IPNet:     eIPIPNet,
					Scope:     int(netlink.SCOPE_UNIVERSE),
					LinkIndex: link.Attrs().Index,
				})

				validRoutesPerLinks = append(validRoutesPerLinks, routemanager.RoutesPerLink{
					Link: link,
					Routes: []routemanager.Route{
						{
							Table:  getRouteTableID(link.Attrs().Index),
							Subnet: defaultCIDR,
						},
					},
				})
			}
		}
	}
	klog.Infof("#### valid ipv4 rules %+v", validV4IPTablesRules)

	if c.v4 {
		if err = c.iptablesManager.OwnChain(utiliptables.TableNAT, multiNICChain, utiliptables.ProtocolIPv4); err != nil {
			return fmt.Errorf("unable to own chain %s: %v", multiNICChain, err)
		}
		ruleArgs, err := c.iptablesManager.GetIPv4ChainRuleArgs(utiliptables.TableNAT, multiNICChain)
		if err != nil {
			return fmt.Errorf("unable to get chain %s rules: %v", multiNICChain, err)
		}

		c.removeStaleIPTableRules(ruleArgs, validV4IPTablesRules, utiliptables.ProtocolIPv4)

		if len(validV4IPTablesRules) > 0 {
			if err = c.iptablesManager.EnsureRules(utiliptables.TableNAT, multiNICChain, utiliptables.ProtocolIPv4, validV4IPTablesRules); err != nil {
				return fmt.Errorf("failed to ensure rules (%+v) in chain %s: %v", validV4IPTablesRules, multiNICChain, err)
			}
			if err = c.iptablesManager.EnsureRules(utiliptables.TableNAT, utiliptables.ChainPostrouting, utiliptables.ProtocolIPv4, jumpRule); err != nil {
				return fmt.Errorf("failed to create rule in chain %s to jump to chain %s: %v", utiliptables.ChainPostrouting, multiNICChain, err)
			}
		}
	}
	if c.v6 {
		if err = c.iptablesManager.OwnChain(utiliptables.TableNAT, multiNICChain, utiliptables.ProtocolIPv6); err != nil {
			return fmt.Errorf("unable to own chain %s: %v", multiNICChain, err)
		}
		ruleArgs, err := c.iptablesManager.GetIPv6ChainRuleArgs(utiliptables.TableNAT, multiNICChain)
		if err != nil {
			return fmt.Errorf("unable to get chain %s rules: %v", multiNICChain, err)
		}

		c.removeStaleIPTableRules(ruleArgs, validV6IPTablesRules, utiliptables.ProtocolIPv6)

		if len(validV6IPTablesRules) > 0 {
			if err = c.iptablesManager.EnsureRules(utiliptables.TableNAT, multiNICChain, utiliptables.ProtocolIPv6, validV4IPTablesRules); err != nil {
				return fmt.Errorf("unable to ensure iptables rules: %v", err)
			}
			if err = c.iptablesManager.EnsureRules(utiliptables.TableNAT, utiliptables.ChainPostrouting, utiliptables.ProtocolIPv4, jumpRule); err != nil {
				return fmt.Errorf("unable to ensure iptables rules for jump rule: %v", err)
			}
		}
	}
	c.removeStaleAddresses(linkAddresses, validEIPAddressess)

	// ensure valid addresses are assigned. Fine to repeat requests.
	for _, eIP := range validEIPAddressess {
		if err = c.linkManager.AddAddress(eIP); err != nil {
			klog.Errorf("Failed to add egress IP %s to link manager: %v", eIP.IP.String(), err)
		}
	}
	// cleanup stale routes in the table IDs we manage
	c.removeStaleRoutes(existingLinkRoutes, validRoutesPerLinks)
	// ensure what we expect is present. It's safe to send multiple equal adds
	for _, validRoutesPerLink := range validRoutesPerLinks {
		c.routeManager.Add(validRoutesPerLink)
	}

	c.removeStaleRules(existingRules, validRules)
	// ensure what we expect is present. It's safe to send multiple adds
	for _, validRule := range validRules {
		if err = c.ruleManager.AddRule(validRule); err != nil {
			klog.Errorf("Failed to add rule %v to rule manager: %v", validRule, err)
		}
	}

	if switcha {
		klog.Infof("!!!! should be no valid rules.... below")
	}

	klog.Infof("Egress IP count %d, valid IP address %v, valid rules %v valid IPtable rules %v, valid routes %v", len(egressIPs), validEIPAddressess, validRules, validV4IPTablesRules, validRoutesPerLinks)

	return nil
}

func (c *Controller) removeStaleRules(existingRules, validRules []netlink.Rule) {
	var found bool
	var err error
	for _, foundRule := range existingRules {
		found = false
		for _, expectedRule := range validRules {
			if areNetlinkRulesEqual(expectedRule, foundRule) {
				found = true
				break
			}
		}
		if !found {
			if err = c.ruleManager.DeleteRule(foundRule); err != nil {
				klog.Errorf("Failed to delete stale rule (%+v): %v", foundRule, err)
			}
		}
	}
}

func (c *Controller) removeStaleRoutes(existingLinkRoutes map[string][]netlink.Route, validRoutesPerLinks []routemanager.RoutesPerLink) {
	// cleanup stale routes in the table IDs we manage
	var found bool
	for linkName, existingRoutes := range existingLinkRoutes {
		link, err := netlink.LinkByName(linkName)
		if err != nil {
			klog.Errorf("Failed to get link %s by name. Stale routes maybe present: %v", linkName, err)
			continue
		}
		for _, existingRoute := range existingRoutes {
			if existingRoute.LinkIndex != link.Attrs().Index {
				continue
			}
			found = false
			for _, validRoutesPerLink := range validRoutesPerLinks {
				if validRoutesPerLink.Link.Attrs().Index != existingRoute.LinkIndex {
					continue
				}
				for _, validRoute := range validRoutesPerLink.Routes {
					if validRoute.Equal(routemanager.ConvertNetlinkRouteToRoute(existingRoute)) {
						found = true
						break
					}
				}
			}

			if !found {
				existingRoute.Table = getRouteTableID(link.Attrs().Index)
				c.routeManager.Del(routemanager.RoutesPerLink{Link: link,
					Routes: []routemanager.Route{routemanager.ConvertNetlinkRouteToRoute(existingRoute)}})
			}
		}
	}
}

func (c *Controller) removeStaleIPTableRules(existingRules, wantedRules []iptables.RuleArgs, proto utiliptables.Protocol) {
	for _, existingRule := range existingRules {
		var isValid bool
		for _, validRule := range wantedRules {
			if areEqual(validRule.Args, existingRule.Args) {
				isValid = true
				break
			}
		}
		if !isValid {
			if err := c.iptablesManager.DeleteRule(utiliptables.TableNAT, multiNICChain, proto,
				existingRule); err != nil {
				klog.Errorf("Egress IP reconcile unable to delete rule %v in table NAT and chain %s: %v",
					existingRule, multiNICChain, err)
			}
		}
	}
}

func (c *Controller) removeStaleAddresses(foundLinkAddresses map[string][]netlink.Addr, validAddresses []netlink.Addr) {
	// remove any invalid addresses or stale egress IPs
	var isValidEIP bool
	for linkName, addresses := range foundLinkAddresses {
		for _, linkAddress := range addresses {
			if linkAddress.Label != linkmanager.GetAssignedAddressLabel(linkName) {
				continue
			}
			isValidEIP = false
			for _, validAddress := range validAddresses {
				if validAddress.IP.Equal(linkAddress.IP) {
					isValidEIP = true
					break
				}
			}

			if !isValidEIP {
				if err := c.linkManager.DelAddress(linkAddress); err != nil {
					klog.Errorf("Failed to delete egress IP %s to link manager: %v", linkAddress.IP.String(), err)
				}
			}
		}
	}
}

func getRouteTableID(linkIndex int) int {
	return linkIndex + routeTableStart
}

func (c *Controller) GetLinkContainingIPAndValidate(ip net.IP, expectedNetwork string) (bool, netlink.Link, error) {
	prefixLinks := map[string]netlink.Link{} // key is network CIDR
	prefixes := make([]netip.Prefix, 0)
	links, err := netlink.LinkList()
	if err != nil {
		return false, nil, fmt.Errorf("failed to list links: %v", err)
	}
	for _, link := range links {
		link := link
		linkPrefixes, err := linkmanager.GetExternallyAvailableNetipAddresses(link, c.v4, c.v6, skipBridges)
		if err != nil {
			klog.Errorf("Failed to get address from link %s: %v", link.Attrs().Name, err)
			continue
		}
		prefixes = append(prefixes, linkPrefixes...)
		// create lookup table for later retrieval
		for _, prefixFound := range linkPrefixes {
			_, ipNet, err := net.ParseCIDR(prefixFound.String())
			if err != nil {
				klog.Errorf("Egress IP: skipping prefix %q due to parsing CIDR error: %v", prefixFound.String(), err)
				continue
			}
			prefixLinks[ipNet.String()] = link
		}
	}
	lpmTree := cidrtree.New(prefixes...)
	addr, err := netip.ParseAddr(ip.String())
	if err != nil {
		return false, nil, fmt.Errorf("failed to convert IP %s to netip addr: %v", ip.String(), err)
	}
	klog.Errorf("GetLinkContainingIPAndValidate about to look up with address %s", addr.String())
	network, found := lpmTree.Lookup(addr)
	if !found {
		klog.Errorf(" ## LPM tres entry not found - LPM tress is %v", lpmTree.String())
		return false, nil, nil
	}
	if network.String() != expectedNetwork {
		return false, nil, fmt.Errorf("EgressIP %s is was assigned to network %s but we got network %s. Mismatch therefore not assigning",
			ip.String(), expectedNetwork, network.String())
	}
	klog.Errorf("## GetLinkContainingIPAndValidate AT THE END gonna look up with key %q", network.String())
	return true, prefixLinks[network.String()], nil
}

func getNetworkInfoFromIP(ipStr string) (net.IP, *net.IPNet, bool, error) {
	var err error
	var ip net.IP
	var ipNet *net.IPNet
	var v6 bool
	ip = net.ParseIP(ipStr)
	if net2.IsIPv6(ip) {
		v6 = true
		ip, ipNet, err = net.ParseCIDR(fmt.Sprintf("%s/128", ipStr))
	} else {
		ip, ipNet, err = net.ParseCIDR(fmt.Sprintf("%s/32", ipStr))
	}
	return ip, ipNet, v6, err
}

func filterRouteByLinkTable(linkIndex, tableID int) (*netlink.Route, uint64) {
	return &netlink.Route{
			LinkIndex: linkIndex,
			Table:     tableID,
		},
		netlink.RT_FILTER_OIF | netlink.RT_FILTER_TABLE
}

func filterRuleByPriority(priority int) (*netlink.Rule, uint64) {
	return &netlink.Rule{
			Priority: priority,
		},
		netlink.RT_FILTER_PRIORITY
}

func areNetlinkRulesEqual(r1, r2 netlink.Rule) bool {
	if r1.Table != r2.Table {
		return false
	}
	if r1.Priority != r2.Priority {
		return false
	}
	if r1.Src != nil && r2.Src == nil {
		return false
	}
	if r2.Src != nil && r1.Src == nil {
		return false
	}
	if r1.Src != nil && r2.Src != nil && !r1.Src.IP.Equal(r2.Src.IP) {
		return false
	}
	if r1.Src != nil && r2.Src != nil && r1.Src.Mask.String() != r2.Src.Mask.String() {
		return false
	}
	if r1.Dst != nil && r2.Dst == nil {
		return false
	}
	if r2.Dst != nil && r1.Dst == nil {
		return false
	}
	if r1.Dst != nil && r2.Dst != nil && !r1.Dst.IP.Equal(r2.Dst.IP) {
		return false
	}
	if r1.Dst != nil && r2.Dst != nil && r1.Dst.Mask.String() != r2.Dst.Mask.String() {
		return false
	}
	if r1.Family != r2.Family {
		return false
	}
	if r1.IifName != r2.IifName {
		return false
	}
	if r1.OifName != r2.OifName {
		return false
	}
	if r1.Mask != r2.Mask {
		return false
	}
	if r1.Mark != r2.Mark {
		return false
	}
	return true
}

func areEqual(s1, s2 []string) bool {
	if len(s1) != len(s2) {
		return false
	}
	for _, s11 := range s1 {
		var found bool
		for _, s22 := range s2 {
			if s22 == s11 {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	for _, s22 := range s2 {
		var found bool
		for _, s11 := range s1 {
			if s11 == s22 {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}
