package egressip

import (
	"fmt"
	"net"
	"net/netip"
	"reflect"
	"sync"
	"time"

	eipv1 "github.com/ovn-org/ovn-kubernetes/go-controller/pkg/crd/egressip/v1"
	egressipinformer "github.com/ovn-org/ovn-kubernetes/go-controller/pkg/crd/egressip/v1/apis/informers/externalversions/egressip/v1"
	egressiplisters "github.com/ovn-org/ovn-kubernetes/go-controller/pkg/crd/egressip/v1/apis/listers/egressip/v1"
	"github.com/ovn-org/ovn-kubernetes/go-controller/pkg/node/iprulemanager"
	"github.com/ovn-org/ovn-kubernetes/go-controller/pkg/node/iptables"
	"github.com/ovn-org/ovn-kubernetes/go-controller/pkg/node/linkmanager"
	"github.com/ovn-org/ovn-kubernetes/go-controller/pkg/node/routemanager"
	"github.com/ovn-org/ovn-kubernetes/go-controller/pkg/util"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	coreinformers "k8s.io/client-go/informers/core/v1"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
	utiliptables "k8s.io/kubernetes/pkg/util/iptables"

	"github.com/gaissmai/cidrtree"
	"github.com/vishvananda/netlink"
)

const (
	routeRulePriority = 6000 // the priority of the ip routing rules created by the controller. Egress Service priority is 5000.
	routeTableStart   = 1000
	multiNICChainName = "OVN-KUBE-EGRESS-IP-MULTI-NIC"
	multiNICChain     = utiliptables.Chain(multiNICChainName)
)

var (
	_, defaultCIDR, _ = net.ParseCIDR("0.0.0.0/0")
	jumpRule          = []iptables.RuleArg{{Args: []string{"-j", multiNICChainName}}}
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
	ruleManager     *iprulemanager.Controller
	iptablesManager *iptables.Controller

	triggerReconcileCh chan struct{}
	nodeName           string
	v4                 bool
	v6                 bool
	clean              bool // used to short-circuit reconciliation
}

func NewController(eIPInformer egressipinformer.EgressIPInformer, nodeInformer cache.SharedIndexInformer,
	namespaceInformer coreinformers.NamespaceInformer, podInformer coreinformers.PodInformer, routeManager *routemanager.Controller,
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
		ruleManager:        iprulemanager.NewController(v4, v6),
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
	if err := c.ruleManager.OwnPriority(routeRulePriority); err != nil {
		return fmt.Errorf("failed to own priority %d for IP rules: %v", routeRulePriority, err)
	}
	if c.v4 {
		if err := c.iptablesManager.OwnChain(utiliptables.TableNAT, multiNICChain, utiliptables.ProtocolIPv4); err != nil {
			return fmt.Errorf("unable to own chain %s: %v", multiNICChain, err)
		}
	}
	if c.v6 {
		if err := c.iptablesManager.OwnChain(utiliptables.TableNAT, multiNICChain, utiliptables.ProtocolIPv6); err != nil {
			return fmt.Errorf("unable to own chain %s: %v", multiNICChain, err)
		}
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
		c.linkManager.Run(stopCh, 2*time.Minute)
		wg.Done()
	}()

	go func() {
		c.iptablesManager.Run(stopCh, 6*time.Minute)
		wg.Done()
	}()

	go func() {
		c.ruleManager.Run(stopCh, 5*time.Minute)
		wg.Done()
	}()
	return nil
}

func (c *Controller) isEgressLabelApplied() bool {
	node, err := c.nodeLister.Get(c.nodeName)
	if err != nil {
		klog.Errorf("Failed to determine if node %q has egress IP label applied: %w", c.nodeName, err)
		return false
	}
	nodeEgressLabel := util.GetNodeEgressLabel()
	nodeLabels := node.GetLabels()
	_, hasEgressLabel := nodeLabels[nodeEgressLabel]
	return hasEgressLabel
}

func (c *Controller) triggerReconcile() {
	select {
	case c.triggerReconcileCh <- struct{}{}:
		klog.V(5).Infof("Egress IP: sync requested for non-OVN managed networks")
	default:
		klog.V(5).Infof("Egress IP: sync already requested for non-OVN managed networks")
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

func (c *Controller) cleanNode() {
	if linkAddresses, linkRoutes, err := c.getLinkAddressRoutes(); err != nil {
		klog.Errorf("Failed to reconcile and ensure no stale egress IP addresses and routes: %v", err)
	} else {
		c.removeStaleAddresses(linkAddresses, nil)
		c.removeStaleRoutes(linkRoutes, nil)
	}
	if routingRules, err := getRoutingRules(); err != nil {
		klog.Errorf("Failed to reconcile and ensure no stale routing rules: %v", err)
	} else {
		c.removeStaleRoutingRules(routingRules, nil)
	}
	if c.v4 {
		if err := c.iptablesManager.FlushChain(utiliptables.TableNAT, multiNICChain, utiliptables.ProtocolIPv4); err != nil {
			klog.Errorf("Failed to flush IPv4 rules in chain %q: %v", multiNICChain, err)
		}
	}
	if c.v6 {
		if err := c.iptablesManager.FlushChain(utiliptables.TableNAT, multiNICChain, utiliptables.ProtocolIPv6); err != nil {
			klog.Errorf("Failed to flush IPv6 rules in chain %q: %v", multiNICChain, err)
		}
	}
}

func (c *Controller) reconcile() error {
	// short-circuit if no egress IP label is present making no assumptions about previous config. Clean at least
	// once to remove any previous config. When this component comes up, we clean at least once.
	if !c.isEgressLabelApplied() {
		if !c.clean {
			c.cleanNode()
			c.clean = true
		}
		return nil
	}
	c.clean = false
	eIPs, err := c.egressIPLister.List(labels.Everything())
	if err != nil {
		return fmt.Errorf("failed to ensure egress IP is correctly configured because we could not list egress IPs: %v", err)
	}
	// TODO(martinkennelly): short-circuit reconciliation here if no non-OVN managed EgressIP exists
	existingLinkAddresses, existingLinkRoutes, err := c.getLinkAddressRoutes()
	if err != nil {
		return fmt.Errorf("failed to reconcile: %v", err)
	}
	existingRules, err := getRoutingRules()
	if err != nil {
		return fmt.Errorf("failed to reconcile and list routing rules: %v", err)
	}
	// ip rules used to steer the packet into the right routing table. One rule for every pod IP address.
	routingRules := make([]netlink.Rule, 0)
	// egress IP addresses added to link to facilitate ARP/ND requests
	ipAddresses := make([]netlink.Addr, 0)
	// routes to steer a packet out the correct interface. The routes will reside within custom routing tables. One per interface.
	routesPerLinks := make([]routemanager.RoutesPerLink, 0)
	// iptables rules used to SNAT the egress traffic to egress IP
	v4IPTablesSNATRules := make([]iptables.RuleArg, 0)
	v6IPTablesSNATRules := make([]iptables.RuleArg, 0)

	for _, eIP := range eIPs {
		if len(eIP.Status.Items) == 0 {
			continue
		}
		for _, status := range eIP.Status.Items {
			if status.Node != c.nodeName {
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
					klog.Errorf("Failed to determine if egress IP %s is OVN managed: %v", status.EgressIP, err)
					continue
				}
			} else {
				klog.Errorf("Failed to find node %s and therefore will not reconcile egress IP %s: %v", status.Node,
					status.EgressIP, err)
			}
			if isOVNManaged {
				continue
			}
			eIPIP, eIPIPNet, err := net.ParseCIDR(fmt.Sprintf("%s/32", status.EgressIP))
			if err != nil {
				klog.Errorf("Failed to parse egress IP %q: %v", status.EgressIP, err)
				continue
			}
			found, link, err := c.GetLinkContainingIPAndValidate(eIPIP, status.Network)
			if err != nil {
				klog.Errorf("Failed to find network for egress IP %s and therefore failed to configure this egress IP: %v", status.EgressIP, err)
				continue
			}
			if !found {
				klog.Errorf("Expected to find network interface to assign egress IP %s. Failed to assign", status.EgressIP)
				continue
			}
			if link == nil {
				klog.Errorf("Failed to get a valid link for egress IP %s and network %s", status.EgressIP, status.Network)
				continue
			}
			selectedPods, err := c.GetSelectedPodsForEIP(eIP)
			if err != nil {
				klog.Errorf("Failed to generate a list of selected pods for Egress IP %s: %v", eIP.Name, err)
				continue
			}
			generatedRoutingRules, err := generateIPRules(selectedPods, link.Attrs().Index)
			if err != nil {
				klog.Errorf("Failed to generate a list of routing rules for Egress IP %s: %v", eIP.Name, err)
				continue
			}
			routingRules = append(routingRules, generatedRoutingRules...)
			v4IPTablesRules, v6IPTablesRules, err := generateIPTablesSNATRules(selectedPods, status.EgressIP, link.Attrs().Name)
			if err != nil {
				klog.Errorf("Failed to generate a list of IPTable SNAT rules for Egress IP %s: %v", eIP.Name, err)
				continue
			}
			v4IPTablesSNATRules = append(v4IPTablesSNATRules, v4IPTablesRules...)
			v6IPTablesSNATRules = append(v6IPTablesSNATRules, v6IPTablesRules...)

			if len(v4IPTablesSNATRules) > 0 || len(v6IPTablesSNATRules) > 0 {
				// EIP is to be set when node is selected, pods have been selected
				ipAddresses = append(ipAddresses, netlink.Addr{
					IPNet:     eIPIPNet,
					Scope:     int(netlink.SCOPE_UNIVERSE),
					LinkIndex: link.Attrs().Index,
				})

				routesPerLinks = append(routesPerLinks, routemanager.RoutesPerLink{
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

	if c.v4 {
		ruleArgs, err := c.iptablesManager.GetIPv4ChainRuleArgs(utiliptables.TableNAT, multiNICChain)
		if err != nil {
			return fmt.Errorf("unable to get chain %s rules: %v", multiNICChain, err)
		}
		c.removeStaleIPTableRules(ruleArgs, v4IPTablesSNATRules, utiliptables.ProtocolIPv4)

		if len(v4IPTablesSNATRules) > 0 {
			if err = c.iptablesManager.EnsureRules(utiliptables.TableNAT, multiNICChain, utiliptables.ProtocolIPv4, v4IPTablesSNATRules); err != nil {
				return fmt.Errorf("failed to ensure rules (%+v) in chain %s: %v", v4IPTablesSNATRules, multiNICChain, err)
			}
			if err = c.iptablesManager.EnsureRules(utiliptables.TableNAT, utiliptables.ChainPostrouting, utiliptables.ProtocolIPv4, jumpRule); err != nil {
				return fmt.Errorf("failed to create rule in chain %s to jump to chain %s: %v", utiliptables.ChainPostrouting, multiNICChain, err)
			}
		}
	}
	if c.v6 {
		ruleArgs, err := c.iptablesManager.GetIPv6ChainRuleArgs(utiliptables.TableNAT, multiNICChain)
		if err != nil {
			return fmt.Errorf("unable to get chain %s rules: %v", multiNICChain, err)
		}
		c.removeStaleIPTableRules(ruleArgs, v6IPTablesSNATRules, utiliptables.ProtocolIPv6)

		if len(v6IPTablesSNATRules) > 0 {
			if err = c.iptablesManager.EnsureRules(utiliptables.TableNAT, multiNICChain, utiliptables.ProtocolIPv6, v6IPTablesSNATRules); err != nil {
				return fmt.Errorf("unable to ensure iptables rules: %v", err)
			}
			if err = c.iptablesManager.EnsureRules(utiliptables.TableNAT, utiliptables.ChainPostrouting, utiliptables.ProtocolIPv6, jumpRule); err != nil {
				return fmt.Errorf("unable to ensure iptables rules for jump rule: %v", err)
			}
		}
	}
	c.removeStaleAddresses(existingLinkAddresses, ipAddresses)

	// ensure valid addresses are assigned. Fine to repeat requests.
	for _, eIP := range ipAddresses {
		if err = c.linkManager.AddAddress(eIP); err != nil {
			klog.Errorf("Failed to add egress IP %s to link manager: %v", eIP.IP.String(), err)
		}
	}
	// cleanup stale routes in the table IDs we manage
	c.removeStaleRoutes(existingLinkRoutes, routesPerLinks)
	// ensure what we expect is present. It's safe to send multiple equal adds
	for _, validRoutesPerLink := range routesPerLinks {
		c.routeManager.Add(validRoutesPerLink)
	}
	c.removeStaleRoutingRules(existingRules, routingRules)
	// ensure what we expect is present. It's safe to send multiple adds
	for _, validRule := range routingRules {
		if err = c.ruleManager.AddIPRule(validRule); err != nil {
			klog.Errorf("Failed to add rule %v to rule manager: %v", validRule, err)
		}
	}
	return nil
}

func (c *Controller) GetSelectedPodsForEIP(eIP *eipv1.EgressIP) ([]*corev1.Pod, error) {
	podSelector, err := metav1.LabelSelectorAsSelector(&eIP.Spec.PodSelector)
	if err != nil {
		return nil, fmt.Errorf("invalid podSelector for egress IP %s: %v", eIP.Name, err)
	}
	namespaceSelector, err := metav1.LabelSelectorAsSelector(&eIP.Spec.NamespaceSelector)
	if err != nil {
		return nil, fmt.Errorf("invalid namespaceSelector for egress IP %s: %v", eIP.Name, err)
	}
	namespaces, err := c.namespaceLister.List(namespaceSelector)
	if err != nil {
		return nil, fmt.Errorf("failed to list namespaces using selector %s to configure egress IP %s: %v",
			namespaceSelector.String(), eIP.Name, err)
	}
	selectedPods := make([]*corev1.Pod, 0)
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
				if util.PodWantsHostNetwork(pod) {
					continue
				}
				if len(pod.Status.PodIPs) == 0 {
					continue
				}
				selectedPods = append(selectedPods, pod)
			}
		}
	}
	return selectedPods, nil
}

// getLinkAddressRoutes iterates through all links and attempts to build a map of all associated link addresses
// that are externally available for that link.
// Also, builds a map of links and their associated IP routes.
func (c *Controller) getLinkAddressRoutes() (map[string][]netlink.Addr, map[string][]netlink.Route, error) {
	links, err := netlink.LinkList()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list links: %v", err)
	}
	// get address map for each interface -> addresses/mask
	// also map address/mask -> interface name
	linkAddresses := make(map[string][]netlink.Addr)
	linkRoutes := make(map[string][]netlink.Route)
	for _, link := range links {
		// build map of link name -> addresses
		// get all link addresses including any assigned addresses used for EIP
		addresses, err := linkmanager.GetExternallyAvailableAddresses(link, c.v4, c.v6)
		if err != nil {
			klog.Warningf("Skipping link %q because unable to get link addresses: %v", link.Attrs().Name, err)
			continue
		}
		validAddresses := make([]netlink.Addr, 0, len(addresses))
		// filter addresses that do not give routing information about the network it is attached to and therefore not suitable
		// for EIP assignment. For example, if V4 address and /32 subnet mask, we cannot determine if an EIP assigned to
		// this link will be routable and therefore is not suitable for EIP assignment. Also, ensure we don't filter assigned EIP addresses.
		for _, address := range addresses {
			netMaskSize, _ := address.Mask.Size()
			if address.IP.To4() != nil && netMaskSize == 32 && address.Label != linkmanager.GetAssignedAddressLabel(link.Attrs().Name) {
				continue
			}
			if address.IP.To4() == nil && netMaskSize == 128 && address.Label != linkmanager.GetAssignedAddressLabel(link.Attrs().Name) {
				continue
			}
			validAddresses = append(validAddresses, address)
		}
		linkAddresses[link.Attrs().Name] = validAddresses
		// build map of links name -> routes
		filter, mask := filterRouteByLinkTable(link.Attrs().Index, getRouteTableID(link.Attrs().Index))
		var ipFamily int
		if c.v4 && c.v6 {
			ipFamily = netlink.FAMILY_ALL
		} else if c.v4 {
			ipFamily = netlink.FAMILY_V4
		} else {
			ipFamily = netlink.FAMILY_V6
		}
		existingRoutes, err := netlink.RouteListFiltered(ipFamily, filter, mask)
		if err != nil {
			klog.Errorf("Failed to list routes for link %q: %v", link.Attrs().Name, err)
			continue
		}
		// netlink package returns a nil value for route dst field when the dst is address 0.0.0.0/0
		// lets manually detect this and fill in dst
		for i, existingRoute := range existingRoutes {
			if existingRoute.Dst == nil || existingRoute.Dst.IP == nil {
				if existingRoute.Src == nil && existingRoute.Gw == nil && existingRoute.MPLSDst == nil {
					// if the dst field was not set we treat it as "any" destination
					existingRoutes[i].Dst = defaultCIDR
				}
			}
		}
		linkRoutes[link.Attrs().Name] = existingRoutes
	}
	return linkAddresses, linkRoutes, nil
}

func (c *Controller) removeStaleRoutingRules(existingRules, validRules []netlink.Rule) {
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
			if err = c.ruleManager.DeleteIPRule(foundRule); err != nil {
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

func (c *Controller) removeStaleIPTableRules(existingRules, wantedRules []iptables.RuleArg, proto utiliptables.Protocol) {
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

func getRouteTableID(ifIndex int) int {
	return ifIndex + routeTableStart
}

// GetLinkContainingIPAndValidate iterates through all links found locally building a map of addresses associated with
// each link and attempts to find a network that will host the func parameter IP address using longest-prefix-match.
// Expected network is validated against calculated network to ensure they match, otherwise, we return an error.
// If no network is found to host the func parameter IP, no error is returned. For success, true, no error
// and a non-nil link is returned otherwise, error is returned.
func (c *Controller) GetLinkContainingIPAndValidate(ip net.IP, expectedNetwork string) (bool, netlink.Link, error) {
	prefixLinks := map[string]netlink.Link{} // key is network CIDR
	prefixes := make([]netip.Prefix, 0)
	links, err := netlink.LinkList()
	if err != nil {
		return false, nil, fmt.Errorf("failed to list links: %v", err)
	}
	for _, link := range links {
		link := link
		linkPrefixes, err := linkmanager.GetExternallyAvailablePrefixesExcludeAssigned(link, c.v4, c.v6)
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
	network, found := lpmTree.Lookup(addr)
	if !found {
		return false, nil, nil
	}
	if network.String() != expectedNetwork {
		return false, nil, fmt.Errorf("EgressIP %s is was assigned to network %s but we got network %s. Mismatch therefore not assigning",
			ip.String(), expectedNetwork, network.String())
	}
	return true, prefixLinks[network.String()], nil
}

func generateIPTablesSNATRules(pods []*corev1.Pod, eIPIP, infName string) ([]iptables.RuleArg, []iptables.RuleArg, error) {
	v4IPTablesSNATRules := make([]iptables.RuleArg, 0)
	v6IPTablesSNATRules := make([]iptables.RuleArg, 0)
	for _, pod := range pods {
		ips, err := util.DefaultNetworkPodIPs(pod)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to generate iptables SNAT rules for pod %s/%s: %v", pod.Namespace, pod.Name, err)
		}
		for _, ip := range ips {
			if len(ip) == 0 {
				return nil, nil, fmt.Errorf("failed to generate iptables SNAT rules because invalid IP %q for pod %s/%s",
					ip.String(), pod.Namespace, pod.Name)
			}
			if ip.To4() != nil { // v4
				ipFullMask := fmt.Sprintf("%s/32", ip.String())
				v4IPTablesSNATRules = append(v4IPTablesSNATRules, iptables.RuleArg{Args: []string{"-s", ipFullMask,
					"-o", infName, "-j", "SNAT", "--to-source", eIPIP}})
			} else { // v6
				ipFullMask := fmt.Sprintf("%s/128", ip.String())
				v6IPTablesSNATRules = append(v6IPTablesSNATRules, iptables.RuleArg{Args: []string{"-s", ipFullMask,
					"-o", infName, "-j", "SNAT", "--to-source", eIPIP}})
			}
		}
	}
	return v4IPTablesSNATRules, v6IPTablesSNATRules, nil
}

// generateIPRules generates IP rules at a predefined priority for each pod IP with a custom routing table based
// from the links 'ifindex'
func generateIPRules(selectedPods []*corev1.Pod, ifIndex int) ([]netlink.Rule, error) {
	ipRules := make([]netlink.Rule, 0)
	for _, pod := range selectedPods {
		r := netlink.NewRule()
		r.Table = getRouteTableID(ifIndex)
		r.Priority = routeRulePriority
		ips, err := util.DefaultNetworkPodIPs(pod)
		if err != nil {
			return nil, fmt.Errorf("unable to get pod IP(s) %s/%s: %v", pod.Namespace, pod.Name, err)
		}
		for _, ip := range ips {
			if len(ip) == 0 {
				return nil, fmt.Errorf("invalid IP %q for pod %s/%s", ip.String(), pod.Namespace, pod.Name)
			}
			var ipFullMask string
			if ip.To4() != nil { // v4
				ipFullMask = fmt.Sprintf("%s/32", ip.String())
				r.Family = netlink.FAMILY_V4
			} else { // v6
				ipFullMask = fmt.Sprintf("%s/128", ip.String())
				r.Family = netlink.FAMILY_V6
			}
			_, ipNet, err := net.ParseCIDR(ipFullMask)
			if err != nil {
				return nil, fmt.Errorf("failed to parse CIDR %q for pod %s/%s: %v", ipFullMask, pod.Namespace, pod.Name, err)
			}
			r.Src = ipNet
			ipRules = append(ipRules, *r)
		}
	}
	return ipRules, nil
}

func getRoutingRules() ([]netlink.Rule, error) {
	filter, mask := filterRuleByPriority(routeRulePriority)
	rules, err := netlink.RuleListFiltered(netlink.FAMILY_ALL, filter, mask)
	if err != nil {
		return nil, fmt.Errorf("failed to list rules: %v", err)
	}
	return rules, nil
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
	return reflect.DeepEqual(r1, r2)
}

func areEqual(s1, s2 []string) bool {
	return reflect.DeepEqual(s1, s2)
}
