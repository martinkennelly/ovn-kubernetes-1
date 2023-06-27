//go:build linux
// +build linux

package node

import (
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/ovn-org/ovn-kubernetes/go-controller/pkg/config"
	"github.com/ovn-org/ovn-kubernetes/go-controller/pkg/factory"
	"github.com/ovn-org/ovn-kubernetes/go-controller/pkg/kube"
	"github.com/ovn-org/ovn-kubernetes/go-controller/pkg/node/linkmanager"
	"github.com/ovn-org/ovn-kubernetes/go-controller/pkg/types"
	"github.com/ovn-org/ovn-kubernetes/go-controller/pkg/util"

	"github.com/vishvananda/netlink"
	kapi "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
	utilnet "k8s.io/utils/net"
)

type addressManager struct {
	nodeName       string
	watchFactory   factory.NodeWatchFactory
	ovnAddresses   sets.Set[string] // key is IP (no mask) e.g. 1.1.1.1. Only ovn managed networks are allows
	hostAddresses  sets.Set[string] // key is IP + mask e.g. 1.1.1.1/24. OVN managed and non OVN managed networks
	nodeAnnotator  kube.Annotator
	mgmtPortConfig *managementPortConfig
	// useNetlink indicates the addressManager should use machine
	// information from netlink. Set to false for testcases.
	useNetlink bool

	// compare node primary IP change
	nodePrimaryAddr net.IP
	gatewayBridge   *bridgeConfiguration

	OnChanged func()
	sync.Mutex
}

// initializes a new address manager which will hold all the IPs on a node
func newAddressManager(nodeName string, k kube.Interface, config *managementPortConfig, watchFactory factory.NodeWatchFactory, gwBridge *bridgeConfiguration) *addressManager {
	return newAddressManagerInternal(nodeName, k, config, watchFactory, gwBridge, true)
}

// newAddressManagerInternal creates a new address manager; this function is
// only expose for testcases to disable netlink subscription to ensure
// reproducibility of unit tests.
func newAddressManagerInternal(nodeName string, k kube.Interface, config *managementPortConfig, watchFactory factory.NodeWatchFactory, gwBridge *bridgeConfiguration, useNetlink bool) *addressManager {
	mgr := &addressManager{
		nodeName:       nodeName,
		watchFactory:   watchFactory,
		ovnAddresses:   sets.New[string](),
		hostAddresses:  sets.New[string](),
		mgmtPortConfig: config,
		gatewayBridge:  gwBridge,
		OnChanged:      func() {},
		useNetlink:     useNetlink,
	}
	mgr.nodeAnnotator = kube.NewNodeAnnotator(k, nodeName)
	mgr.sync()
	return mgr
}

// updates the address manager with a new IP
// returns true if there was an update
func (c *addressManager) addAddr(ip net.IP) bool {
	c.Lock()
	defer c.Unlock()
	if !c.ovnAddresses.Has(ip.String()) && c.isValidNodeIP(ip) {
		klog.Infof("Adding IP: %s, to node IP manager", ip)
		c.ovnAddresses.Insert(ip.String())
		return true
	}

	return false
}

// removes IP from address manager
// returns true if there was an update
func (c *addressManager) delAddr(ip net.IP) bool {
	c.Lock()
	defer c.Unlock()
	if c.ovnAddresses.Has(ip.String()) && c.isValidNodeIP(ip) {
		klog.Infof("Removing IP: %s, from node IP manager", ip)
		c.ovnAddresses.Delete(ip.String())
		return true
	}

	return false
}

// updates the address manager with a new IP from an address seen on the node,
// returns true if there was an update
func (c *addressManager) addHostAddr(ipnet *net.IPNet) bool {
	c.Lock()
	defer c.Unlock()
	if !c.hostAddresses.Has(ipnet.String()) {
		klog.Infof("Adding IP: %s, to node IP manager", ipnet)
		c.hostAddresses.Insert(ipnet.String())
		return true
	}

	return false
}

// removes IP from address manager
// returns true if there was an update
func (c *addressManager) delHostAddr(ipnet *net.IPNet) bool {
	c.Lock()
	defer c.Unlock()
	if c.hostAddresses.Has(ipnet.String()) {
		klog.Infof("Removing IP: %s, from node IP manager", ipnet)
		c.hostAddresses.Delete(ipnet.String())
		return true
	}

	return false
}

// ListAddresses returns all the addresses we know about
func (c *addressManager) ListAddresses() []net.IP {
	c.Lock()
	defer c.Unlock()
	addrs := sets.List(c.ovnAddresses)
	out := make([]net.IP, 0, len(addrs))
	for _, addr := range addrs {
		ip := net.ParseIP(addr)
		if ip == nil {
			continue
		}
		out = append(out, ip)
	}
	return out
}

type subscribeFn func() (bool, chan netlink.AddrUpdate, error)

func (c *addressManager) Run(stopChan <-chan struct{}, doneWg *sync.WaitGroup) {
	addrSubscribeOptions := netlink.AddrSubscribeOptions{
		ErrorCallback: func(err error) {
			klog.Errorf("Failed during AddrSubscribe callback: %v", err)
			// Note: Not calling sync() from here: it is redudant and unsafe when stopChan is closed.
		},
	}

	subscribe := func() (bool, chan netlink.AddrUpdate, error) {
		addrChan := make(chan netlink.AddrUpdate)
		if err := netlink.AddrSubscribeWithOptions(addrChan, stopChan, addrSubscribeOptions); err != nil {
			return false, nil, err
		}
		// sync the manager with current addresses on the node
		c.sync()
		return true, addrChan, nil
	}

	c.runInternal(stopChan, doneWg, subscribe)
}

// runInternal can be used by testcases to provide a fake subscription function
// rather than using netlink
func (c *addressManager) runInternal(stopChan <-chan struct{}, doneWg *sync.WaitGroup, subscribe subscribeFn) {
	// Add an event handler to the node informer. This is needed for cases where users first update the node's IP
	// address but only later update kubelet configuration and restart kubelet (which in turn will update the reported
	// IP address inside the node's status field).
	nodeInformer := c.watchFactory.NodeInformer()
	_, err := nodeInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: func(old, new interface{}) {
			c.handleNodePrimaryAddrChange()
		},
	})
	if err != nil {
		klog.Fatalf("Could not add node event handler while starting address manager %v", err)
	}

	doneWg.Add(1)
	go func() {
		defer doneWg.Done()

		addressSyncTimer := time.NewTicker(30 * time.Second)
		defer addressSyncTimer.Stop()

		subscribed, addrChan, err := subscribe()
		if err != nil {
			klog.Error("Error during netlink subscribe for IP Manager: %v", err)
		}

		for {
			select {
			case _, ok := <-addrChan:
				addressSyncTimer.Reset(30 * time.Second)
				if !ok {
					if subscribed, addrChan, err = subscribe(); err != nil {
						klog.Error("Error during netlink re-subscribe due to channel closing for IP Manager: %v", err)
					}
					continue
				}
				addrChanged := c.refreshAddresses()
				c.handleNodePrimaryAddrChange()
				if addrChanged || !c.doesNodeHostAddressesMatch() {
					klog.Infof("Address set change. Updating node address annotation.", c.ovnAddresses)
					err := c.updateNodeAddressAnnotations()
					if err != nil {
						klog.Errorf("Address Manager failed to update node address annotations: %v", err)
					}
					c.OnChanged()
				}
			case <-addressSyncTimer.C:
				if subscribed {
					klog.V(5).Info("Node IP manager calling sync() explicitly")
					c.sync()
				} else {
					if subscribed, addrChan, err = subscribe(); err != nil {
						klog.Error("Error during netlink re-subscribe for IP Manager: %v", err)
					}
				}
			case <-stopChan:
				return
			}
		}
	}()

	klog.Info("Node IP manager is running")
}

// updates OVN's EncapIP if the node IP changed
func (c *addressManager) handleNodePrimaryAddrChange() {
	nodePrimaryAddrChanged, err := c.nodePrimaryAddrChanged()
	if err != nil {
		klog.Errorf("Address Manager failed to check node primary address change: %v", err)
		return
	}
	if nodePrimaryAddrChanged {
		klog.Infof("Node primary address changed to %v. Updating OVN encap IP.", c.nodePrimaryAddr)
		c.updateOVNEncapIPAndReconnect()
	}
}

// updateNodeAddressAnnotations updates all relevant annotations for the node including
// k8s.ovn.org/host-addresses, k8s.ovn.org/node-primary-ifaddr, k8s.ovn.org/l3-gateway-config.
func (c *addressManager) updateNodeAddressAnnotations() error {
	var err error
	var ifAddrs []*net.IPNet

	// Get node information
	node, err := c.watchFactory.GetNode(c.nodeName)
	if err != nil {
		return err
	}

	if c.useNetlink {
		// get updated interface IP addresses for the gateway bridge
		ifAddrs, err = c.gatewayBridge.updateInterfaceIPAddresses(node)
		if err != nil {
			return err
		}
	}

	// update k8s.ovn.org/host-addresses
	if err = c.updateHostAddresses(node); err != nil {
		return err
	}

	// sets both IPv4 and IPv6 primary IP addr in annotation k8s.ovn.org/node-primary-ifaddr
	// Note: this is not the API node's internal interface, but the primary IP on the gateway
	// bridge (cf. gateway_init.go)
	if err = util.SetNodePrimaryIfAddrs(c.nodeAnnotator, ifAddrs); err != nil {
		return err
	}

	// update k8s.ovn.org/l3-gateway-config
	gatewayCfg, err := util.ParseNodeL3GatewayAnnotation(node)
	if err != nil {
		return err
	}
	gatewayCfg.IPAddresses = ifAddrs
	err = util.SetL3GatewayConfig(c.nodeAnnotator, gatewayCfg)
	if err != nil {
		return err
	}

	// push all updates to the node
	err = c.nodeAnnotator.Run()
	if err != nil {
		return err
	}
	return nil
}

func (c *addressManager) updateHostAddresses(node *kapi.Node) error {
	if config.OvnKubeNode.Mode == types.NodeModeDPU {
		// For DPU mode, here we need to use the DPU host's IP address which is the tenant cluster's
		// host internal IP address instead.
		nodeAddrStr, err := util.GetNodePrimaryIP(node)
		if err != nil {
			return err
		}
		nodeAddrSet := sets.New[string](nodeAddrStr)
		return util.SetNodeHostAddresses(c.nodeAnnotator, nodeAddrSet)
	}

	return util.SetNodeHostAddresses(c.nodeAnnotator, c.hostAddresses)
}

func (c *addressManager) doesNodeHostAddressesMatch() bool {
	c.Lock()
	defer c.Unlock()

	node, err := c.watchFactory.GetNode(c.nodeName)
	if err != nil {
		klog.Errorf("Unable to get node from informer")
		return false
	}
	// check to see if ips on the node differ from what we stored
	// in host-address annotation
	nodeHostAddresses, err := util.ParseNodeHostAddresses(node)
	if err != nil {
		klog.Errorf("Unable to parse addresses from node host %s: %s", node.Name, err.Error())
		return false
	}

	return nodeHostAddresses.Equal(c.hostAddresses)
}

func isSupportedLinkType(linkType string) bool {
	// vlan | veth | vcan | dummy | ifb | macvlan | macvtap |
	// bridge | bond | ipoib | ip6tnl | ipip | sit | vxlan |
	// gre | gretap | ip6gre | ip6gretap | vti | vti6 | nlmon |
	// bond_slave | ipvlan | xfrm | device
	switch linkType {
	case "vlan", "dummy", "bond", "device":
		return true
	default:
		return false
	}
}

func (c *addressManager) refreshAddresses() (changed bool) {
	// list all address on the node excluding:
	// 1. Links that are down
	// 2. Loopback, slave or bridge type interfaces
	// 3. Scope of address must be global / universe
	// 4. Any ovn kube managed addresses we wish to filter automatically - currently only egress IP, and it uses node name as label.
	// Get node information
	node, err := c.watchFactory.GetNode(c.nodeName)
	if err != nil {
		klog.Errorf("Node IP handler: failed to get node from watcher: %v", err)
		return
	}
	eIPConfig, err := util.ParseNodePrimaryIfAddr(node)
	if err != nil {
		klog.Errorf("Node IP hand;er: failed to get node primary interface addresses: %v", err)
		return
	}

	if eIPConfig == nil || (len(eIPConfig.V4) == 0 && len(eIPConfig.V6) == 0) {
		klog.Errorf("Node IP: failed to refresh addresses because failed to find node primary network information")
		return
	}

	addressesAdded := sets.New[string]()
	if len(eIPConfig.V4) > 0 {
		if c.addAddr(eIPConfig.V4[0].IP) {
			changed = true
		}
		addressesAdded.Insert(eIPConfig.V4[0].IP.String())
	}
	if len(eIPConfig.V6) > 0 {
		if c.addAddr(eIPConfig.V6[0].IP) {
			changed = true
		}
		addressesAdded.Insert(eIPConfig.V6[0].IP.String())
	}

	// remove any addresses that aren't seen anymore
	staleAddresses := c.ovnAddresses.Difference(addressesAdded)
	for _, staleAddress := range staleAddresses.UnsortedList() {
		ip := net.ParseIP(staleAddress)
		if ip == nil {
			klog.Errorf("Node IP: failed to parse address %s", staleAddress)
			continue
		}
		changed = true
		c.delAddr(ip)
	}

	addressesAdded = sets.New[string]()
	if len(eIPConfig.V4) > 0 {
		eIPConfig.V4[0].Net.IP = eIPConfig.V4[0].IP
		if c.addHostAddr(eIPConfig.V4[0].Net) {
			changed = true
		}
		addressesAdded.Insert(eIPConfig.V4[0].Net.String())
	}
	if len(eIPConfig.V6) > 0 {
		eIPConfig.V6[0].Net.IP = eIPConfig.V6[0].IP
		if c.addHostAddr(eIPConfig.V4[0].Net) {
			changed = true
		}
		addressesAdded.Insert(eIPConfig.V6[0].Net.String())
	}

	links, err := netlink.LinkList()
	if err != nil {
		klog.Errorf("Node IP Handler: failed to list address on node: %v", err)
	}
	for _, link := range links {
		if supported := isSupportedLinkType(link.Type()); !supported {
			continue
		}
		excludeBridges := true
		addresses, err := linkmanager.GetExternallyAvailableNetlinkLinkAddresses(link, config.IPv4Mode, config.IPv6Mode, excludeBridges)
		if err != nil {
			klog.Errorf("Node IP handler: failed to get IP addresses from link %s: %v", link.Attrs().Name, err)
			continue
		}
		for _, address := range addresses {
			// skip custom IPs added by link manager. They are labeled with a known label
			if address.Label == linkmanager.GetAssignedAddressLabel(link.Attrs().Name) {
				continue
			}
			address.IPNet.IP = address.IP
			c.addHostAddr(address.IPNet)
			addressesAdded.Insert(address.IPNet.String())
		}
	}

	// remove any addresses that aren't seen anymore
	staleAddresses = c.hostAddresses.Difference(addressesAdded)
	for _, staleAddress := range staleAddresses.UnsortedList() {
		ip, ipnet, err := net.ParseCIDR(staleAddress)
		if err != nil {
			klog.Errorf("Node IP: failed to parse address %s", staleAddress)
			continue
		}
		ipnet.IP = ip
		c.delHostAddr(ipnet)
	}
	c.hostAddresses = addressesAdded
	return
}

// nodePrimaryAddrChanged returns false if there is an error or if the IP does
// match, otherwise it returns true and updates the current primary IP address.
func (c *addressManager) nodePrimaryAddrChanged() (bool, error) {
	node, err := c.watchFactory.GetNode(c.nodeName)
	if err != nil {
		return false, err
	}
	// check to see if ips on the node differ from what we stored
	// in addressManager and it's an address that is known locally
	nodePrimaryAddrStr, err := util.GetNodePrimaryIP(node)
	if err != nil {
		return false, err
	}
	nodePrimaryAddr := net.ParseIP(nodePrimaryAddrStr)

	if nodePrimaryAddr == nil {
		return false, fmt.Errorf("failed to parse the primary IP address string from kubernetes node status")
	}
	c.Lock()
	exists := c.ovnAddresses.Has(nodePrimaryAddrStr)
	c.Unlock()

	if !exists || c.nodePrimaryAddr.Equal(nodePrimaryAddr) {
		return false, nil
	}
	c.nodePrimaryAddr = nodePrimaryAddr

	return true, nil
}

// updateOVNEncapIP updates encap IP to OVS when the node primary IP changed.
func (c *addressManager) updateOVNEncapIPAndReconnect() {
	checkCmd := []string{
		"get",
		"Open_vSwitch",
		".",
		"external_ids:ovn-encap-ip",
	}
	encapIP, stderr, err := util.RunOVSVsctl(checkCmd...)
	if err != nil {
		klog.Warningf("Unable to retrieve configured ovn-encap-ip from OVS: %v, %q", err, stderr)
	} else {
		encapIP = strings.TrimSuffix(encapIP, "\n")
		if len(encapIP) > 0 && c.nodePrimaryAddr.String() == encapIP {
			klog.V(4).Infof("Will not update encap IP, value: %s is the already configured", c.nodePrimaryAddr)
			return
		}
	}

	confCmd := []string{
		"set",
		"Open_vSwitch",
		".",
		fmt.Sprintf("external_ids:ovn-encap-ip=%s", c.nodePrimaryAddr),
	}

	_, stderr, err = util.RunOVSVsctl(confCmd...)
	if err != nil {
		klog.Errorf("Error setting OVS encap IP: %v  %q", err, stderr)
		return
	}

	// force ovn-controller to reconnect SB with new encap IP immediately.
	// otherwise there will be a max delay of 200s due to the 100s
	// ovn-controller inactivity probe.
	_, stderr, err = util.RunOVNAppctlWithTimeout(5, "-t", "ovn-controller", "exit", "--restart")
	if err != nil {
		klog.Errorf("Failed to exit ovn-controller %v %q", err, stderr)
		return
	}
}

// detects if the IP is valid for a node
// excludes things like local IPs, mgmt port ip, special masquerade IP
func (c *addressManager) isValidNodeIP(addr net.IP) bool {
	if addr == nil {
		return false
	}
	if addr.IsLinkLocalUnicast() {
		return false
	}
	if addr.IsLoopback() {
		return false
	}

	if utilnet.IsIPv4(addr) {
		if c.mgmtPortConfig.ipv4 != nil && c.mgmtPortConfig.ipv4.ifAddr.IP.Equal(addr) {
			return false
		}
	} else if utilnet.IsIPv6(addr) {
		if c.mgmtPortConfig.ipv6 != nil && c.mgmtPortConfig.ipv6.ifAddr.IP.Equal(addr) {
			return false
		}
	}

	if util.IsAddressReservedForInternalUse(addr) {
		return false
	}

	return true
}

func (c *addressManager) sync() {
	var addrChanged bool
	if c.useNetlink {
		if c.refreshAddresses() {
			addrChanged = true
		}
	}

	c.handleNodePrimaryAddrChange()
	if addrChanged || !c.doesNodeHostAddressesMatch() {
		klog.Info("Node address changed. Updating annotations.")
		err := c.updateNodeAddressAnnotations()
		if err != nil {
			klog.Errorf("Address Manager failed to update node address annotations: %v", err)
		}
	}
}
