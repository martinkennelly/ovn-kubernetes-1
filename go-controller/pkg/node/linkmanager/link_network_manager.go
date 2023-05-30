package link_manager

import (
	"fmt"
	"github.com/vishvananda/netlink"

	// "os"
	// "strings"
	"sync"
	"time"

	"k8s.io/klog/v2"
	// utilnet "k8s.io/utils/net"
)

// gather all interfaces data address + mask and offer this as a service
// offer to take address and assign to interface. Ensure its there.

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

func NewController(name string, v4, v6 bool) *Controller {
	return &Controller{
		mu:          &sync.Mutex{},
		name:        name,
		ipv4Enabled: v4,
		ipv6Enabled: v6,
		store:       make(map[string][]netlink.Addr, 0),
	}
}

func (c *Controller) run(stopCh <-chan struct{}, doneWg *sync.WaitGroup) {
	go func() {
		defer doneWg.Done()
		ticker := time.NewTicker(2 * time.Minute)
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
	}()
}

func (c *Controller) addAddress(address netlink.Addr) error {
	if address.LinkIndex == 0 {
		return fmt.Errorf("link index must be non-zero")
	}
	if address.IPNet == nil {
		return fmt.Errorf("IP must be non-nil")
	}
	if address.IPNet.IP.IsUnspecified() {
		return fmt.Errorf("IP must be specified")
	}
	if _, err := netlink.LinkByIndex(address.LinkIndex); err != nil {
		return fmt.Errorf("no valid link associated with addresses %s: %v", address.String(), err)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	// overwrite label to the name of this component in-order to aid address ownership
	address.Label = c.name
	if err := c.addAddressToStore(address); err != nil {
		return fmt.Errorf("failed to add address %q: %v", address.String(), err)
	}
	return nil
}

func (c *Controller) reconcile() {
	// 1. get all the links on the node
	// 2. iterate over the links and get the addresses associated with it
	// 3. cleanup any stale addresses from link that we no longer managed
	// 4. remove any stale addresses from links that we do manage
	// 5. add addresses that are missing from a link that we managed
	links, err := netlink.LinkList()
	if err != nil {
		klog.Errorf("Link Network Manager: failed to list links: %v", err)
		return
	}
	for _, link := range links {
		// get all addresses associated with the link depending on which IP families we support
		addressesFound, err := c.getLinkAddressesByIPFamily(link)
		if err != nil {
			klog.Errorf("Link Network Manager: failed to get address from link %q", link.Attrs().Name)
			continue
		}
		addressesWanted, found := c.store[link.Attrs().Name]
		if !found {
			// we dont managed this link, check we  used to manage it and if a stale address is present, delete it
			for _, address := range addressesFound {
				// we label any address we create, so if we aren't managed a link, we must remove any stale addresses
				if address.Label == c.name {
					if err := netlink.AddrDel(link, &address); err != nil {
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
				if addrFound.Label == c.name {
					// delete an unmanaged address that we used to managed
					if err = netlink.AddrDel(link, &addrFound); err != nil {
						klog.Errorf("Link Network Manager: failed to delete stale address %q from link %q: %v",
							addrFound, link.Attrs().Name, err)
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
				if err = netlink.AddrAdd(link, &addressWanted); err != nil {
					klog.Errorf("Link Network Manager: failed to add address %q to link %q: %v",
						addressWanted.String(), link.Attrs().Name, err)
				}
			}
		}
	}
}

func containsAddress(addresses []netlink.Addr, candidate netlink.Addr) bool {
	for _, address := range addresses {
		if address.Equal(candidate) {
			return true
		}
	}
	return false
}

func (c *Controller) getLinkAddressesByIPFamily(link netlink.Link) ([]netlink.Addr, error) {
	links := make([]netlink.Addr, 0)
	if c.ipv4Enabled {
		linksFound, err := netlink.AddrList(link, netlink.FAMILY_V4)
		if err != nil {
			return links, fmt.Errorf("failed to list link addresses: %v", err)
		}
		links = linksFound
	}
	if c.ipv6Enabled {
		linksFound, err := netlink.AddrList(link, netlink.FAMILY_V6)
		if err != nil {
			return links, fmt.Errorf("failed to list link addresses: %v", err)
		}
		links = append(links, linksFound...)
	}
	return links, nil
}

func (c *Controller) addAddressToStore(newAddress netlink.Addr) error {
	link, err := netlink.LinkByIndex(newAddress.LinkIndex)
	if err != nil {
		return fmt.Errorf("failed to find link by index: %v", err)
	}
	addressesSaved, found := c.store[link.Attrs().Name]
	if !found {
		c.store[link.Attrs().Name] = []netlink.Addr{newAddress}
		return nil
	}
	// check if the address already exists
	found = false
	for _, addressSaved := range addressesSaved {
		if addressSaved.Equal(newAddress) {
			found = true
			break
		}
	}
	// add it to store if not found
	if !found {
		c.store[link.Attrs().Name] = append(addressesSaved, newAddress)
	}
	return nil
}

/*

// detects if the IP is valid for a node
// excludes things like local IPs, mgmt port ip, special masquerade IP
func (c *egressIpNodeManager) isValidNodeIP(addr net.IP) bool {
	if addr == nil {
		return false
	}
	if addr.IsLinkLocalUnicast() {
		return false
	}
	if addr.IsLoopback() {
		return false
	}

	// if utilnet.IsIPv4(addr) {
	// 	if c.mgmtPortConfig.ipv4 != nil && c.mgmtPortConfig.ipv4.ifAddr.IP.Equal(addr) {
	// 		return false
	// 	}
	// } else if utilnet.IsIPv6(addr) {
	// 	if c.mgmtPortConfig.ipv6 != nil && c.mgmtPortConfig.ipv6.ifAddr.IP.Equal(addr) {
	// 		return false
	// 	}
	// }

	if util.IsAddressReservedForInternalUse(addr) {
		return false
	}

	return true
}

*/
