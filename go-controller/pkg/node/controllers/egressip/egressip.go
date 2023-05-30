package egressip

import (
	"fmt"
	"net"
	"sort"
	"sync"

	egressipapi "github.com/ovn-org/ovn-kubernetes/go-controller/pkg/crd/egressip/v1"
	egressipinformer "github.com/ovn-org/ovn-kubernetes/go-controller/pkg/crd/egressip/v1/apis/informers/externalversions/egressip/v1"
	egressiplisters "github.com/ovn-org/ovn-kubernetes/go-controller/pkg/crd/egressip/v1/apis/listers/egressip/v1"
	"github.com/ovn-org/ovn-kubernetes/go-controller/pkg/factory"
	"github.com/ovn-org/ovn-kubernetes/go-controller/pkg/node/iptables"
	"github.com/ovn-org/ovn-kubernetes/go-controller/pkg/node/linkmanager"
	"github.com/ovn-org/ovn-kubernetes/go-controller/pkg/node/routemanager"
	"github.com/ovn-org/ovn-kubernetes/go-controller/pkg/node/rulemanager"
	"github.com/ovn-org/ovn-kubernetes/go-controller/pkg/util"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	v12 "k8s.io/client-go/informers/core/v1"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
	utiliptables "k8s.io/kubernetes/pkg/util/iptables"
	net2 "k8s.io/utils/net"

	"github.com/vishvananda/netlink"
)

const (
	ipRulePriority  = 6000 // the priority of the ip rules created by the controller. Egress Service priority is 5000.
	routeTableStart = 1000
	multiNICChain   = utiliptables.Chain("OVN-KUBE-EGRESS-IP-Multi-NIC")
	// Sample IPtables save output which the following regex expression with match upon to get interface, pod ip, and egress IP.
	// -A POSTROUTING -s 10.244.2.4/32 -o dummy0 -j SNAT --to-source 1.1.1.1
	snatRegExMatch = "-A %s -s ([^ ]*) -o ([^ ]*) .* --to ([^ ]*)"
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

	_, err := eIPInformer.Informer().AddEventHandler(factory.WithUpdateHandlingForObjReplace(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.onAdd,
		UpdateFunc: c.onUpdate,
		DeleteFunc: c.onDelete,
	}))
	if err != nil {
		return nil, fmt.Errorf("failed to create egress IP controller because unable to add EgressIP event handlers: %v", err)
	}

	_, err = nodeInformer.AddEventHandler(factory.WithUpdateHandlingForObjReplace(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.onAdd,
		UpdateFunc: c.onUpdate,
		DeleteFunc: c.onDelete,
	}))
	if err != nil {
		return nil, fmt.Errorf("failed to create egress IP controller because unable to add Node event handlers: %v", err)
	}

	_, err = namespaceInformer.Informer().AddEventHandler(factory.WithUpdateHandlingForObjReplace(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.onAdd,
		UpdateFunc: c.onUpdate,
		DeleteFunc: c.onDelete,
	}))
	if err != nil {
		return nil, fmt.Errorf("failed to create egress IP controller because unable to add namespace event handlers: %v", err)
	}

	_, err = podInformer.Informer().AddEventHandler(factory.WithUpdateHandlingForObjReplace(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.onAdd,
		UpdateFunc: c.onUpdate,
		DeleteFunc: c.onDelete,
	}))
	if err != nil {
		return nil, fmt.Errorf("failed to create egress IP controller because unable to add pod event handlers: %v", err)
	}
	return c, nil
}

func (c *Controller) Run(stopCh <-chan struct{}, wg *sync.WaitGroup) error {
	klog.Info("Starting Egress IP Controller")
	if !cache.WaitForCacheSync(stopCh, c.nodeSynced) {
		return fmt.Errorf("timed out waiting for informer caches to sync")
	}

	go func() {
		var err error
		defer wg.Done()
		for {
			select {
			case <-stopCh:
				return
			case <-c.triggerReconcileCh:
				if err = c.reconcile(); err != nil {
					klog.Errorf("Failed to reconcile egress IP: %v", err)
				}
			}
		}
	}()

	wg.Add(3)
	c.linkManager.Run(stopCh, wg)
	c.iptablesManager.Run(stopCh, wg)
	c.ruleManager.Run(stopCh, wg)
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
	linkRoutes := make(map[string][]netlink.Route)
	for _, link := range links {
		addresses, err := c.getValidLinkAddresses(link)
		if err != nil {
			klog.Errorf("Egress IP skipping link %s because unable to get link addresses: %v", link.Attrs().Name, err)
			continue
		}
		linkAddresses[link.Attrs().Name] = addresses

		filter, mask := filterRouteByLinkTable(link, getRouteTableID(link.Attrs().Index))
		routes, err := netlink.RouteListFiltered(netlink.FAMILY_ALL, filter, mask)
		if err != nil {
			klog.Errorf("Failed to list routes for link %s in-order to cleanup stale egress IPs: %v", link.Attrs().Name, err)
			continue
		}
		linkRoutes[link.Attrs().Name] = routes
	}

	filter, mask := filterRuleByPriority(ipRulePriority)
	foundRules, err := netlink.RuleListFiltered(netlink.FAMILY_ALL, filter, mask)
	if err != nil {
		klog.Errorf("Failed to list rules: %v", err)
	}

	validEIPs := make([]*egressipapi.EgressIP, 0)
	validRules := make([]netlink.Rule, 0)
	validEgressIPAddressess := make([]netlink.Addr, 0)
	validRoutesPerLinks := make([]routemanager.RoutesPerLink, 0)
	validV4IPTablesRules := make([]iptables.RuleArgs, 0)
	validV6IPTablesRules := make([]iptables.RuleArgs, 0)

	for _, eIP := range egressIPs {
		if len(eIP.Status.Items) == 0 {
			continue
		}
		for _, status := range eIP.Status.Items {
			if !status.RouteViaHost {
				continue
			}
			if status.Node != c.nodeName {
				continue
			}
			if status.EgressIP == "" {
				continue
			}
			eIPIP, eIPIPNet, err := net.ParseCIDR(fmt.Sprintf("%s/32", status.EgressIP))
			if err != nil {
				klog.Errorf("Failed to parse egress IP %q: %v", status.EgressIP, err)
				continue
			}
			found, link, err := c.GetInterfaceNetworkContainingIP(eIPIP)
			if err != nil {
				klog.Errorf("Failed to find network for egress IP %s and therefore failed to configure this egress IP: %v", status.EgressIP, err)
				continue
			}
			if !found {
				klog.Errorf("Expected to find network interface to assign egress IP %s. Failed to assign", status.EgressIP)
				continue
			}
			validEIPs = append(validEIPs, eIP.DeepCopy())
			validEgressIPAddressess = append(validEgressIPAddressess, netlink.Addr{
				IPNet:     eIPIPNet,
				Scope:     int(netlink.SCOPE_UNIVERSE),
				LinkIndex: link.Attrs().Index,
			})

			_, defaultCIDRIPNet, err := net.ParseCIDR("0.0.0.0/0")
			if err != nil {
				klog.Errorf("failed to parse CIDR '0.0.0.0/0': %v", err)
				continue
			}
			validRoutesPerLinks = append(validRoutesPerLinks, routemanager.RoutesPerLink{
				Link:  link,
				Table: getRouteTableID(link.Attrs().Index),
				Routes: []routemanager.Route{
					{
						Subnet: defaultCIDRIPNet,
					},
				},
			})
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
					args := []string{"-s", podIP.String(), "-o", link.Attrs().Name, "-j", "SNAT", "--to", status.EgressIP}
					ruleArgs := iptables.RuleArgs{Args: args}
					if isIPv6 {
						validV6IPTablesRules = append(validV6IPTablesRules, ruleArgs)
					} else {
						validV4IPTablesRules = append(validV4IPTablesRules, ruleArgs)
					}
				}
			}
		}
	}

	if c.v4 {
		c.iptablesManager.EnsureRulesOwnChain(utiliptables.TableNAT, multiNICChain, utiliptables.ProtocolIPv4, validV4IPTablesRules, snatRegExMatch)
	}
	if c.v6 {
		c.iptablesManager.EnsureRulesOwnChain(utiliptables.TableNAT, multiNICChain, utiliptables.ProtocolIPv6, validV4IPTablesRules, snatRegExMatch)
	}
	// remove any invalid addresses
	var isValidEIP bool
	for _, addresses := range linkAddresses {
		for _, linkAddress := range addresses {
			if linkAddress.Label != c.nodeName {
				continue
			}
			isValidEIP = false
			for _, eIP := range validEgressIPAddressess {
				if eIP.IP.Equal(linkAddress.IP) {
					isValidEIP = true
					break
				}
			}

			if !isValidEIP {
				if err = c.linkManager.DelAddress(linkAddress); err != nil {
					klog.Errorf("Failed to del egress IP %s to link manager: %v", linkAddress.IP.String(), err)
				}
			}
		}
	}
	// ensure valid addresses are assigned. Fine to repeat requests.
	for _, eIP := range validEgressIPAddressess {
		if err = c.linkManager.AddAddress(eIP); err != nil {
			klog.Errorf("Failed to add egress IP %s to link manager: %v", eIP.IP.String(), err)
		}
	}
	// cleanup stale routes in the table IDs we manage
	var found bool
	for infName, linkRoute := range linkRoutes {
		link, err := netlink.LinkByName(infName)
		if err != nil {
			klog.Errorf("Failed to get link %s by name. Stale routes maybe present: %v", infName, err)
			continue
		}
		for _, route := range linkRoute {
			found = false
			for _, validRoutesPerlink := range validRoutesPerLinks {
				if validRoutesPerlink.Link.Attrs().Index != route.LinkIndex {
					continue
				}
				if validRoutesPerlink.Table != route.Table {
					continue
				}

				for _, validRoute := range validRoutesPerlink.Routes {
					if validRoute.Equal(routemanager.ConvertNetlinkRouteToRoute(route)) {
						found = true
						break
					}
				}
			}
			if !found {
				c.routeManager.Del(routemanager.RoutesPerLink{Link: link, Table: getRouteTableID(link.Attrs().Index),
					Routes: []routemanager.Route{routemanager.ConvertNetlinkRouteToRoute(route)}})
			}
		}
	}
	// ensure what we expect is present
	for _, validRoutesPerLink := range validRoutesPerLinks {
		c.routeManager.Add(validRoutesPerLink)
	}

	for _, foundRule := range foundRules {
		found = false
		for _, expectedRule := range validRules {
			if areNetlinkRulesEqual(expectedRule, foundRule) {
				found = true
				break
			}
		}
		if !found {
			if err = c.ruleManager.DeleteRule(&foundRule); err != nil {
				klog.Errorf("Failed to delete stale rule (%+v): %v", foundRule, err)
			}
		}
	}
	for _, validRule := range validRules {
		if err = c.ruleManager.AddRule(&validRule); err != nil {
			klog.Errorf("Failed to add rule %v to rule manager: %v", validRule, err)
		}
	}

	return nil
}

// getValidLinkAddresses returns a list of valid link addresses. Networks will be extracted from the Address and therefore
// the address must be valid for egress IP. Filter
func (c *Controller) getValidLinkAddresses(link netlink.Link) ([]netlink.Addr, error) {
	validAddresses := make([]netlink.Addr, 0)
	if link.Attrs().Flags != net.FlagUp {
		return validAddresses, nil
	}
	if link.Attrs().Flags == net.FlagLoopback {
		return validAddresses, nil
	}
	// skip bridges
	if link.Attrs().MasterIndex != 0 {
		return validAddresses, nil
	}
	// skip slave devices
	if link.Attrs().ParentIndex != 0 {
		return validAddresses, nil
	}
	linkAddresses, err := c.linkManager.GetLinkAddressesByIPFamily(link)
	if err != nil {
		return validAddresses, fmt.Errorf("failed to get all valid link addresses: %v", err)
	}
	for _, address := range linkAddresses {
		// consider only GLOBAL scope addresses
		if address.Scope != int(netlink.SCOPE_UNIVERSE) {
			continue
		}
		validAddresses = append(validAddresses, address)
	}
	return validAddresses, nil
}

func getRouteTableID(linkIndex int) int {
	return linkIndex + routeTableStart
}

type InfIndex struct {
	addr netlink.Addr
	link netlink.Link
}

func (c *Controller) GetInterfaceNetworkContainingIP(ip net.IP) (bool, netlink.Link, error) {
	links, err := netlink.LinkList()
	if err != nil {
		return false, nil, fmt.Errorf("failed to list links: %v", err)
	}
	infAddresses := make([]InfIndex, 0)
	for _, link := range links {
		addressesFound, err := c.getValidLinkAddresses(link)
		if err != nil {
			klog.Errorf("failed to get address from link %q", link.Attrs().Name)
			continue
		}
		for _, addressFound := range addressesFound {
			infAddresses = append(infAddresses, InfIndex{addressFound, link})
		}
	}
	possibleNetworks := make([]InfIndex, 0)
	// TODO implement LPM
	for _, infAddress := range infAddresses {
		if infAddress.addr.IPNet.Contains(ip) {
			possibleNetworks = append(possibleNetworks, infAddress)
		}
	}
	if len(possibleNetworks) == 0 {
		return false, nil, nil
	}
	sort.Slice(possibleNetworks, func(i, j int) bool {
		ia1 := possibleNetworks[i]
		ia2 := possibleNetworks[j]
		_, ia1AddrLen := ia1.addr.Mask.Size()
		_, ia2AddrLen := ia2.addr.Mask.Size()
		return ia1AddrLen > ia2AddrLen
	})
	return true, possibleNetworks[0].link, nil
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

func filterRouteByLinkTable(link netlink.Link, table int) (*netlink.Route, uint64) {
	return &netlink.Route{
			LinkIndex: link.Attrs().Index,
			Table:     table,
		},
		netlink.RT_FILTER_OIF | netlink.RT_FILTER_TABLE
}

func filterRuleByPriority(priority int) (*netlink.Rule, uint64) {
	return &netlink.Rule{
			Priority: priority,
		},
		netlink.RT_FILTER_TABLE
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
