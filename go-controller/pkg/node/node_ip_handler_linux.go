//go:build linux
// +build linux

package node

import (
	"fmt"
	"github.com/ovn-org/ovn-kubernetes/go-controller/pkg/node/linkmanager"
	"net"
	"sync"
	"time"

	"github.com/ovn-org/ovn-kubernetes/go-controller/pkg/factory"
	"github.com/ovn-org/ovn-kubernetes/go-controller/pkg/kube"
	"github.com/ovn-org/ovn-kubernetes/go-controller/pkg/util"

	"github.com/vishvananda/netlink"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
	utilnet "k8s.io/utils/net"
)

type addressManager struct {
	nodeName       string
	watchFactory   factory.NodeWatchFactory
	addresses      sets.Set[string]
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
	// flags to detect which IP family(s) we support
	v4 bool
	v6 bool
}

const skipBridges = false

// initializes a new address manager which will hold all the IPs on a node
func newAddressManager(nodeName string, k kube.Interface, config *managementPortConfig, watchFactory factory.NodeWatchFactory, gwBridge *bridgeConfiguration, v4, v6 bool) *addressManager {
	return newAddressManagerInternal(nodeName, k, config, watchFactory, gwBridge, true, v4, v6)
}

// newAddressManagerInternal creates a new address manager; this function is
// only expose for testcases to disable netlink subscription to ensure
// reproducibility of unit tests.
func newAddressManagerInternal(nodeName string, k kube.Interface, config *managementPortConfig, watchFactory factory.NodeWatchFactory, gwBridge *bridgeConfiguration, useNetlink, v4, v6 bool) *addressManager {
	mgr := &addressManager{
		nodeName:       nodeName,
		watchFactory:   watchFactory,
		addresses:      sets.New[string](),
		mgmtPortConfig: config,
		gatewayBridge:  gwBridge,
		OnChanged:      func() {},
		useNetlink:     useNetlink,
		v4:             v4,
		v6:             v6,
	}
	mgr.nodeAnnotator = kube.NewNodeAnnotator(k, nodeName)
	mgr.sync()
	return mgr
}

// updates the address manager with a new IP
// returns true if there was an update
func (c *addressManager) addAddr(ipnet net.IPNet) bool {
	c.Lock()
	defer c.Unlock()
	if !c.addresses.Has(ipnet.String()) && c.isValidNodeIP(ipnet.IP) {
		klog.Infof("Adding IP: %s, to node IP manager", ipnet.String())
		c.addresses.Insert(ipnet.String())
		return true
	}

	return false
}

// removes IP from address manager
// returns true if there was an update
func (c *addressManager) delAddr(ipnet net.IPNet) bool {
	c.Lock()
	defer c.Unlock()
	klog.Errorf("## node ip: about to remove %s and does address contain it %v and is it valid %v", ipnet.String(), c.addresses.Has(ipnet.String()), c.isValidNodeIP(ipnet.IP))
	if c.addresses.Has(ipnet.String()) && c.isValidNodeIP(ipnet.IP) {
		klog.Errorf("Removing IP: %s, from node IP manager", ipnet.String())
		c.addresses.Delete(ipnet.String())
		return true
	}

	return false
}

// ListAddresses returns all the addresses we know about
func (c *addressManager) ListAddresses() []net.IP {
	c.Lock()
	defer c.Unlock()
	addrs := sets.List(c.addresses)
	out := make([]net.IP, 0, len(addrs))
	for _, addr := range addrs {
		ip, _, err := net.ParseCIDR(addr)
		if err != nil {
			klog.Errorf("Failed to parse %s: %v", addr, err)
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

	c.refreshAddresses()
	if err = c.updateNodeAddressAnnotations(); err != nil {
		klog.Errorf("Address manager failed to set node annotation. It will be retried: %v", err)
	}
	c.OnChanged()

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
				klog.Errorf("### NODE IP HANDER ADDR CN FIRED WITH OK CHAN %v", ok)
				addressSyncTimer.Reset(30 * time.Second)
				if !ok {
					klog.Errorf("### !! NOEIP HANDER: ADDR CHAN CLOSED EWW")
					if subscribed, addrChan, err = subscribe(); err != nil {
						klog.Error("Error during netlink re-subscribe due to channel closing for IP Manager: %v", err)
					}
					klog.Errorf("### NOEIP HANDER: SHOULD BE RESUBBED")
					continue
				}
				c.sync()
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

func (c *addressManager) refreshAddresses() {
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

	addressesAdded := sets.New[string]()
	if c.v4 {
		if eIPConfig != nil && eIPConfig.V4.Net != nil {
			ipNet := *eIPConfig.V4.Net
			ipNet.IP = eIPConfig.V4.IP
			c.addAddr(ipNet)
			addressesAdded.Insert(ipNet.String())
		}

	}
	if c.v6 {
		if eIPConfig != nil && eIPConfig.V6.Net != nil {
			ipNet := *eIPConfig.V6.Net
			ipNet.IP = eIPConfig.V6.IP
			c.addAddr(ipNet)
			addressesAdded.Insert(ipNet.String())
		}
	}
	links, err := netlink.LinkList()
	if err != nil {
		klog.Errorf("Node IP Handler: failed to list address on node: %v", err)
	}
	for _, link := range links {
		klog.Infof("## considering %s and type %s", link.Attrs().Name, link.Type())
		if supported := isSupportedLinkType(link.Type()); !supported {
			klog.Infof("##### skipping because link %s isnt supported of type %s", link.Attrs().Name, link.Type())
			continue
		}
		addresses, err := linkmanager.GetExternallyAvailableNetlinkLinkAddresses(link, c.v4, c.v6, skipBridges)
		if err != nil {
			klog.Errorf("Node IP handler: failed to get IP addresses from link %s: %v", link.Attrs().Name, err)
			continue
		}
		for _, address := range addresses {
			// skip custom IPs added by link manager. They are labeled with a known label
			if address.Label == linkmanager.GetAssignedAddressLabel(link.Attrs().Name) {
				continue
			}
			klog.Infof("## NODE IP adding link %s addr %s", link.Attrs().Name, address.String())
			c.addAddr(*address.IPNet)
			addressesAdded.Insert(address.IPNet.String())
		}
	}
	// remove any addresses that aren't seen anymore
	staleAddresses := c.addresses.Difference(addressesAdded)
	klog.Errorf("NODE IP Stale addresses: %v", staleAddresses)
	for _, staleAddress := range staleAddresses.UnsortedList() {
		ip, ipNet, _ := net.ParseCIDR(staleAddress)
		ipNet.IP = ip
		klog.Errorf("## NODE IP deleting stale %s", staleAddress)
		c.delAddr(*ipNet)
	}
	klog.Errorf("## node IP after sync addresses is %v", c.addresses.UnsortedList())
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
	if err = util.SetNodeHostAddresses(c.nodeAnnotator, c.addresses); err != nil {
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

	return nodeHostAddresses.Equal(c.addresses)
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
	exists := c.addresses.Has(nodePrimaryAddrStr)
	c.Unlock()

	if !exists || c.nodePrimaryAddr.Equal(nodePrimaryAddr) {
		return false, nil
	}
	c.nodePrimaryAddr = nodePrimaryAddr

	return true, nil
}

// updateOVNEncapIP updates encap IP to OVS when the node primary IP changed.
func (c *addressManager) updateOVNEncapIPAndReconnect() {
	cmd := []string{
		"set",
		"Open_vSwitch",
		".",
		fmt.Sprintf("external_ids:ovn-encap-ip=%s", c.nodePrimaryAddr),
	}
	_, stderr, err := util.RunOVSVsctl(cmd...)
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
// excludes things like local IPs, mgmt port ip, special masquerade IP and /32 or /128 length addresses
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
	} else {
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
	if c.useNetlink {
		c.refreshAddresses()
	}
	c.handleNodePrimaryAddrChange()
	if !c.doesNodeHostAddressesMatch() {
		klog.Infof("Node address changed. Updating annotations.")
		if err := c.updateNodeAddressAnnotations(); err != nil {
			klog.Errorf("Address Manager failed to update node address annotations: %v", err)
		}
		c.OnChanged()
	}
}
