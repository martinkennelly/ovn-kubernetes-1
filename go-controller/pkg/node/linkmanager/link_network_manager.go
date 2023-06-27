package linkmanager

import (
	"fmt"
	"net/netip"
	"os/exec"
	"strings"
	"sync"
	"time"

	"k8s.io/klog/v2"

	"github.com/vishvananda/netlink"
)

// Gather all suitable interface address + network mask and offer this as a service.
// Also offer address assignment to interfaces and ensure the state we want is maintained through a sync func

type LinkAddress struct {
	Link      netlink.Link
	Addresses []netlink.Addr
}

type Controller struct {
	mu          *sync.Mutex
	name        string
	ipv4Enabled bool
	ipv6Enabled bool
	store       map[string][]netlink.Addr
}

// netlinkOps is used to fake out any netlink ops
var netlinkOps NetLinkOps = defaultNetLinkOps{}

func SetNetlinkOps(no NetLinkOps) {
	netlinkOps = no
}

// NewController creates a controller to manage linux network interfaces
func NewController(name string, v4, v6 bool) *Controller {
	return &Controller{
		mu:          &sync.Mutex{},
		name:        name,
		ipv4Enabled: v4,
		ipv6Enabled: v6,
		store:       make(map[string][]netlink.Addr, 0),
	}
}

// Run starts the controller and syncs at least every syncPeriod
func (c *Controller) Run(stopCh <-chan struct{}, syncPeriod time.Duration) {
	ticker := time.NewTicker(syncPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-stopCh:
			return
		case <-ticker.C:
			c.mu.Lock()
			c.reconcile()
			c.mu.Unlock()
		}
	}

}

// AddAddress stores the address in a store and ensures its applied
func (c *Controller) AddAddress(address netlink.Addr) error {
	if address.LinkIndex == 0 {
		return fmt.Errorf("link index must be non-zero")
	}
	if address.IPNet == nil {
		return fmt.Errorf("IP must be non-nil")
	}
	if address.IPNet.IP.IsUnspecified() {
		return fmt.Errorf("IP must be specified")
	}
	link, err := netlinkOps.LinkByIndex(address.LinkIndex)
	if err != nil {
		return fmt.Errorf("no valid link associated with addresses %s: %v", address.String(), err)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	// overwrite label to the name of this component in-order to aid address ownership. Label must start with link name.
	address.Label = GetAssignedAddressLabel(link.Attrs().Name)
	c.addAddressToStore(link.Attrs().Name, address)
	c.reconcile()
	return nil
}

// DelAddress removes the address from the store and ensure its removed from a link
func (c *Controller) DelAddress(address netlink.Addr) error {
	if address.LinkIndex == 0 {
		return fmt.Errorf("link index must be non-zero")
	}
	if address.IPNet == nil {
		return fmt.Errorf("IP must be non-nil")
	}
	if address.IPNet.IP.IsUnspecified() {
		return fmt.Errorf("IP must be specified")
	}
	link, err := netlinkOps.LinkByIndex(address.LinkIndex)
	if err != nil {
		return fmt.Errorf("no valid link associated with addresses %s: %v", address.String(), err)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.delAddressFromStore(link.Attrs().Name, address)
	c.reconcile()
	return nil
}

func (c *Controller) reconcile() {
	// 1. get all the links on the node
	// 2. iterate over the links and get the addresses associated with it
	// 3. cleanup any stale addresses from link that we no longer managed
	// 4. remove any stale addresses from links that we do manage
	// 5. add addresses that are missing from a link that we managed
	links, err := netlinkOps.LinkList()
	if err != nil {
		klog.Errorf("Link Network Manager: failed to list links: %v", err)
		return
	}
	for _, link := range links {
		// get all addresses associated with the link depending on which IP families we support
		addressesFound, err := getAllLinkAddressesByIPFamily(link, c.ipv4Enabled, c.ipv6Enabled)
		if err != nil {
			klog.Errorf("Link Network Manager: failed to get address from link %q", link.Attrs().Name)
			continue
		}
		addressesWanted, found := c.store[link.Attrs().Name]
		if !found {
			// we dont managed this link, check we  used to manage it and if a stale address is present, delete it
			for _, address := range addressesFound {
				// we label any address we create, so if we aren't managed a link, we must remove any stale addresses
				if address.Label == GetAssignedAddressLabel(link.Attrs().Name) {
					if err := netlinkOps.AddrDel(link, &address); err != nil {
						klog.Errorf("Link Network Manager: failed to delete address %q from link %q",
							address.String(), link.Attrs().Name)
					} else {
						klog.Infof("Link Network Manager: successfully removed stale address %q from link %q",
							address.String(), link.Attrs().Name)
					}
				}
			}
			continue
		}
		// ensure any addresses found that we previously managed are deleted
		for _, addrFound := range addressesFound {
			if !containsAddress(addressesWanted, addrFound) {
				// every address we added is labeled with a well-known label to ensure we clean any addresses we previously managed.
				if addrFound.Label == GetAssignedAddressLabel(link.Attrs().Name) {
					// delete an unmanaged address that we used to managed
					if err = netlinkOps.AddrDel(link, &addrFound); err != nil {
						klog.Errorf("Link manager: failed to delete stale address %q from link %q: %v",
							addrFound, link.Attrs().Name, err)
					} else {
						klog.Infof("Link manager: deleted stale assigned address %s on link %s", addrFound.String(), link.Attrs().Name)
					}

				}
			}
		}
		// add the addresses we want that are not found on the link
		for _, addressWanted := range addressesWanted {
			var exists bool
			for _, addressFound := range addressesFound {
				if addressFound.Equal(addressWanted) {
					exists = true
					break
				}
			}
			if !exists {
				// Use arping to try to update other hosts ARP caches, in case this IP was
				// previously active on another node. (Based on code from "ifup".)
				go func() {
					out, err := exec.Command("/sbin/arping", "-q", "-A", "-c", "1", "-I", link.Attrs().Name, addressWanted.IP.String()).CombinedOutput()
					if err != nil {
						klog.Warningf("Failed to send ARP claim for IP %s: %v (exec command output: %q)", addressWanted.IP.String(), err, string(out))
						return
					}
					time.Sleep(2 * time.Second)
					_ = exec.Command("/sbin/arping", "-q", "-U", "-c", "1", "-I", link.Attrs().Name, addressWanted.IP.String()).Run()
				}()

				if err = netlinkOps.AddrAdd(link, &addressWanted); err != nil {
					klog.Errorf("Link manager: failed to add address %q to link %q: %v",
						addressWanted.String(), link.Attrs().Name, err)
				}
				// Use arping to try to update other hosts ARP caches, in case this IP was
				// previously active on another node. (Based on code from "ifup".)
				go func() {
					out, err := exec.Command("/sbin/arping", "-q", "-A", "-c", "1", "-I", link.Attrs().Name, addressWanted.IP.String()).CombinedOutput()
					if err != nil {
						klog.Warningf("Failed to send ARP claim for IP %s: %v (exec command output: %q)", addressWanted.IP.String(), err, string(out))
						return
					}
					time.Sleep(2 * time.Second)
					_ = exec.Command("/sbin/arping", "-q", "-U", "-c", "1", "-I", link.Attrs().Name, addressWanted.IP.String()).Run()
				}()
				klog.Infof("Link manager completed adding address %s to link %s", addressWanted, link.Attrs().Name)
			}
		}
	}
}

func (c *Controller) addAddressToStore(linkName string, newAddress netlink.Addr) {
	addressesSaved, found := c.store[linkName]
	if !found {
		c.store[linkName] = []netlink.Addr{newAddress}
		return
	}
	// check if the address already exists
	for _, addressSaved := range addressesSaved {
		if addressSaved.Equal(newAddress) {
			return
		}
	}
	// add it to store if not found
	c.store[linkName] = append(addressesSaved, newAddress)
}

func (c *Controller) delAddressFromStore(linkName string, address netlink.Addr) {
	addressesSaved, found := c.store[linkName]
	if !found {
		return
	}
	temp := addressesSaved[:0]
	for _, addressSaved := range addressesSaved {
		if !addressSaved.Equal(address) {
			temp = append(temp, address)
		}
	}
	c.store[linkName] = temp
}

// GetAssignedAddressLabel returns the label that must be assigned to each egress IP address bound to an interface
func GetAssignedAddressLabel(linkName string) string {
	//TODO: undeterstand why label char limt is 15 and how can a label work with a long link name otherwise it may block eip assignment
	return fmt.Sprintf("%sovn", linkName)
}

// GetExternallyAvailableNetlinkLinkAddresses gets all addresses assigned on an interface with the following characteristics:
// Must be up
// Must not be loopback
// Must not have a parent
// Address must have scope universe
func GetExternallyAvailableNetlinkLinkAddresses(link netlink.Link, v4, v6, skipBridges bool) ([]netlink.Addr, error) {
	validAddresses := make([]netlink.Addr, 0)
	flags := link.Attrs().Flags.String()
	// exclude interfaces that aren't up
	if !strings.Contains(flags, "up") {
		return validAddresses, nil
	}
	// skip any devices which have an owner or parent
	if skipBridges {
		if link.Attrs().MasterIndex != 0 {
			return validAddresses, nil
		}
	}
	// ditto
	if link.Attrs().ParentIndex != 0 {
		return validAddresses, nil
	}
	linkAddresses, err := getAllLinkAddressesByIPFamily(link, v4, v6)
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

// GetExternallyAvailableNetipAddresses returns address Prefixes from interfaces with the following characteristics:
// Must be up
// Must not be loopback
// Must not have a parent
// Address must have scope universe
func GetExternallyAvailableNetipAddresses(link netlink.Link, v4, v6, skipBridges bool) ([]netip.Prefix, error) {
	validAddresses := make([]netip.Prefix, 0)
	flags := link.Attrs().Flags.String()
	// exclude loopback interfaces
	if strings.Contains(flags, "loopback") {
		return validAddresses, nil
	}
	// exclude interfaces that aren't up
	if !strings.Contains(flags, "up") {
		return validAddresses, nil
	}
	// skip any devices which have an owner or parent
	if skipBridges {
		if link.Attrs().MasterIndex != 0 {
			return validAddresses, nil
		}
	}
	if link.Attrs().ParentIndex != 0 {
		return validAddresses, nil
	}
	linkAddresses, err := getAllLinkAddressesByIPFamily(link, v4, v6)
	if err != nil {
		return validAddresses, fmt.Errorf("failed to get all valid link addresses: %v", err)
	}
	for _, address := range linkAddresses {
		// consider only GLOBAL scope addresses
		if address.Scope != int(netlink.SCOPE_UNIVERSE) {
			continue
		}
		// skip Egress IPs
		if address.Label == GetAssignedAddressLabel(link.Attrs().Name) {
			continue
		}
		addr, err := netip.ParsePrefix(address.IPNet.String())
		if err != nil {
			klog.Errorf("Link Manager: unable to parse address %s on link %s: %v", address.String(), link.Attrs().Name, err)
			continue
		}
		validAddresses = append(validAddresses, addr)
	}
	return validAddresses, nil
}

func getAllLinkAddressesByIPFamily(link netlink.Link, v4, v6 bool) ([]netlink.Addr, error) {
	links := make([]netlink.Addr, 0)
	if v4 {
		linksFound, err := netlinkOps.AddrList(link, netlink.FAMILY_V4)
		if err != nil {
			return links, fmt.Errorf("failed to list link addresses: %v", err)
		}
		links = linksFound
	}
	if v6 {
		linksFound, err := netlinkOps.AddrList(link, netlink.FAMILY_V6)
		if err != nil {
			return links, fmt.Errorf("failed to list link addresses: %v", err)
		}
		links = append(links, linksFound...)
	}
	return links, nil
}

func containsAddress(addresses []netlink.Addr, candidate netlink.Addr) bool {
	for _, address := range addresses {
		if address.Equal(candidate) {
			return true
		}
	}
	return false
}

type NetLinkOps interface {
	LinkList() ([]netlink.Link, error)
	LinkByIndex(index int) (netlink.Link, error)
	AddrAdd(link netlink.Link, addr *netlink.Addr) error
	AddrDel(link netlink.Link, addr *netlink.Addr) error
	AddrList(link netlink.Link, family int) ([]netlink.Addr, error)
}

type defaultNetLinkOps struct {
}

func (defaultNetLinkOps) LinkList() ([]netlink.Link, error) {
	return netlink.LinkList()
}

func (defaultNetLinkOps) LinkByIndex(index int) (netlink.Link, error) {
	return netlink.LinkByIndex(index)
}

func (defaultNetLinkOps) AddrDel(link netlink.Link, addr *netlink.Addr) error {
	return netlink.AddrDel(link, addr)
}

func (defaultNetLinkOps) AddrAdd(link netlink.Link, addr *netlink.Addr) error {
	return netlink.AddrAdd(link, addr)
}

func (defaultNetLinkOps) AddrList(link netlink.Link, family int) ([]netlink.Addr, error) {
	return netlink.AddrList(link, family)
}
