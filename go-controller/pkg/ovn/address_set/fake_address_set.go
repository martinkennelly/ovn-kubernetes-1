package addressset

import (
	"fmt"
	"net"
	"sync"
	"sync/atomic"

	"github.com/onsi/gomega"

	"github.com/ovn-org/libovsdb/ovsdb"
	"github.com/ovn-org/ovn-kubernetes/go-controller/pkg/config"
	libovsdbops "github.com/ovn-org/ovn-kubernetes/go-controller/pkg/libovsdb/ops"

	"k8s.io/klog/v2"
	utilnet "k8s.io/utils/net"
)

func NewFakeAddressSetFactoryForIPs(controllerName string) *FakeAddressSetFactoryForIPs {
	return &FakeAddressSetFactoryForIPs{
		ControllerName: controllerName,
		asf:            &ovnAddressSetFactory{},
		sets:           make(map[string]*fakeAddressSets),
	}
}

type FakeAddressSetFactoryForIPs struct {
	// ControllerName is stored here for convenience, it is used to build dbIDs for fake-only methods like
	// AddressSetExists, EventuallyExpectAddressSet, etc.
	ControllerName string
	asf            *ovnAddressSetFactory
	sync.Mutex
	// maps address set name to object
	sets                map[string]*fakeAddressSets
	errOnNextNewAddrSet bool
}

// fakeFactory implements the AddressSetFactoryIPs interface
var _ AddressSetFactoryIPs = &FakeAddressSetFactoryForIPs{}

const FakeASFError = "fake asf error"

// ErrOnNextNewASCall will make FakeAddressSetFactoryForIPs return FakeASFError on the next NewAddressSet call
func (f *FakeAddressSetFactoryForIPs) ErrOnNextNewASCall() {
	f.errOnNextNewAddrSet = true
}

// NewAddressSet returns a new address set object
func (f *FakeAddressSetFactoryForIPs) NewAddressSet(dbIDs *libovsdbops.DbObjectIDs, ips []net.IP) (AddressSetIPs, error) {
	if f.errOnNextNewAddrSet {
		f.errOnNextNewAddrSet = false
		return nil, fmt.Errorf(FakeASFError)
	}
	if err := f.asf.validateDbIDs(dbIDs); err != nil {
		return nil, fmt.Errorf("failed to create address set: %w", err)
	}
	f.Lock()
	defer f.Unlock()
	name := getOvnAddressSetsName(dbIDs)

	_, ok := f.sets[name]
	gomega.Expect(ok).To(gomega.BeFalse(), fmt.Sprintf("new address set %s already exists", name))
	set, err := f.newFakeAddressSetsIPs(ips, dbIDs, f.removeAddressSet)
	if err != nil {
		return nil, err
	}
	f.sets[name] = set
	return set, nil
}

// NewAddressSetOps returns a new address set object
func (f *FakeAddressSetFactoryForIPs) NewAddressSetOps(dbIDs *libovsdbops.DbObjectIDs, ips []net.IP) (AddressSetIPs, []ovsdb.Operation, error) {
	if f.errOnNextNewAddrSet {
		f.errOnNextNewAddrSet = false
		return nil, nil, fmt.Errorf(FakeASFError)
	}
	if err := f.asf.validateDbIDs(dbIDs); err != nil {
		return nil, nil, fmt.Errorf("failed to create address set: %w", err)
	}
	f.Lock()
	defer f.Unlock()
	name := getOvnAddressSetsName(dbIDs)

	_, ok := f.sets[name]
	gomega.Expect(ok).To(gomega.BeFalse(), fmt.Sprintf("new address set %s already exists", name))
	set, err := f.newFakeAddressSetsIPs(ips, dbIDs, f.removeAddressSet)
	if err != nil {
		return nil, nil, err
	}
	f.sets[name] = set
	return set, nil, nil
}

// EnsureAddressSet returns set object
func (f *FakeAddressSetFactoryForIPs) EnsureAddressSet(dbIDs *libovsdbops.DbObjectIDs) (AddressSetIPs, error) {
	if err := f.asf.validateDbIDs(dbIDs); err != nil {
		return nil, fmt.Errorf("failed to ensure address set: %w", err)
	}
	f.Lock()
	defer f.Unlock()
	name := getOvnAddressSetsName(dbIDs)
	set, ok := f.sets[name]
	if ok {
		return set, nil
	}
	set, err := f.newFakeAddressSetsIPs([]net.IP{}, dbIDs, f.removeAddressSet)
	if err != nil {
		return nil, err
	}
	f.sets[name] = set
	return set, nil
}

// GetAddressSet returns set object
func (f *FakeAddressSetFactoryForIPs) GetAddressSet(dbIDs *libovsdbops.DbObjectIDs) (AddressSetIPs, error) {
	if err := f.asf.validateDbIDs(dbIDs); err != nil {
		return nil, fmt.Errorf("failed to get address set: %w", err)
	}
	f.Lock()
	defer f.Unlock()
	name := getOvnAddressSetsName(dbIDs)
	set, ok := f.sets[name]
	if ok {
		return set, nil
	}
	return nil, fmt.Errorf("error fetching address set")
}

func (f *FakeAddressSetFactoryForIPs) ProcessEachAddressSet(ownerController string, indexT *libovsdbops.ObjectIDsType, iteratorFn AddressSetIterFunc) error {
	f.Lock()
	asNames := map[string]*libovsdbops.DbObjectIDs{}
	for _, set := range f.sets {
		if !set.dbIDs.HasSameOwner(ownerController, indexT) {
			continue
		}
		// set.dbIDs doesn't have ip family
		addrSetName := getOvnAddressSetsName(set.dbIDs)
		if _, ok := asNames[addrSetName]; ok {
			continue
		}
		asNames[addrSetName] = set.dbIDs
	}
	f.Unlock()
	for _, dbIDs := range asNames {
		if err := iteratorFn(dbIDs); err != nil {
			return err
		}
	}
	return nil
}

func (f *FakeAddressSetFactoryForIPs) DestroyAddressSet(dbIDs *libovsdbops.DbObjectIDs) error {
	if err := f.asf.validateDbIDs(dbIDs); err != nil {
		return fmt.Errorf("failed to destroy address set: %w", err)
	}
	name := getOvnAddressSetsName(dbIDs)
	if _, ok := f.sets[name]; ok {
		f.removeAddressSet(name)
		return nil
	}
	return nil
}

func (f *FakeAddressSetFactoryForIPs) getAddressSet(dbIDs *libovsdbops.DbObjectIDs) *fakeAddressSets {
	f.Lock()
	defer f.Unlock()
	name := getOvnAddressSetsName(dbIDs)
	if as, ok := f.sets[name]; ok {
		as.Lock()
		return as
	}
	return nil
}

// removeAddressSet removes the address set from the factory
func (f *FakeAddressSetFactoryForIPs) removeAddressSet(name string) {
	f.Lock()
	defer f.Unlock()
	delete(f.sets, name)
}

// ExpectAddressSetWithIPs ensures the named address set exists with the given set of IPs
func (f *FakeAddressSetFactoryForIPs) expectAddressSetWithIPs(g gomega.Gomega, dbIDs *libovsdbops.DbObjectIDs, ips []string) {
	var lenAddressSet int
	as := f.getAddressSet(dbIDs)
	gomega.Expect(as).ToNot(gomega.BeNil(), fmt.Sprintf("expected address set %s to exist", dbIDs.String()))
	defer as.Unlock()
	as4 := as.ipv4
	if as4 != nil {
		lenAddressSet = lenAddressSet + len(as4.ips)
	}
	as6 := as.ipv6
	if as6 != nil {
		lenAddressSet = lenAddressSet + len(as6.ips)
	}

	for _, ip := range ips {
		if utilnet.IsIPv6(net.ParseIP(ip)) {
			g.Expect(as6).NotTo(gomega.BeNil())
			g.Expect(as6.ips).To(gomega.HaveKey(ip))
		} else {
			g.Expect(as4).NotTo(gomega.BeNil())
			g.Expect(as4.ips).To(gomega.HaveKey(ip))
		}
	}
	if lenAddressSet != len(ips) {
		var addrs []string
		if as4 != nil {
			for _, v := range as4.ips {
				addrs = append(addrs, v.String())
			}
		}
		if as6 != nil {
			for _, v := range as6.ips {
				addrs = append(addrs, v.String())
			}
		}

		klog.Errorf("IPv4 addresses mismatch in cache: %#v, expected: %#v", addrs, ips)
	}

	g.Expect(lenAddressSet).To(gomega.Equal(len(ips)))
}

func (f *FakeAddressSetFactoryForIPs) getDbIDsFromNsNameOrDbIDs(dbIDsOrNsName any) *libovsdbops.DbObjectIDs {
	var dbIDs *libovsdbops.DbObjectIDs
	if nsName, ok := dbIDsOrNsName.(string); ok {
		dbIDs = libovsdbops.NewDbObjectIDs(libovsdbops.AddressSetNamespace, f.ControllerName, map[libovsdbops.ExternalIDKey]string{
			libovsdbops.ObjectNameKey: nsName,
		})
	} else if dbIDs, ok = dbIDsOrNsName.(*libovsdbops.DbObjectIDs); !ok {
		panic("unexpected type of argument passed to ExpectAddressSetWithIPs")
	}
	return dbIDs
}

// ExpectAddressSetWithIPs ensure address set exists with the given set of ips.
// Address set is identified by dbIDsOrNsName, which may be a namespace name (string) or a *libovsdbops.DbObjectIDs.
func (f *FakeAddressSetFactoryForIPs) ExpectAddressSetWithIPs(dbIDsOrNsName any, ips []string) {
	dbIDs := f.getDbIDsFromNsNameOrDbIDs(dbIDsOrNsName)
	g := gomega.Default
	f.expectAddressSetWithIPs(g, dbIDs, ips)
}

func (f *FakeAddressSetFactoryForIPs) EventuallyExpectAddressSetWithIPs(dbIDsOrNsName any, ips []string) {
	dbIDs := f.getDbIDsFromNsNameOrDbIDs(dbIDsOrNsName)
	gomega.Eventually(func(g gomega.Gomega) {
		f.expectAddressSetWithIPs(g, dbIDs, ips)
	}).Should(gomega.Succeed())
}

// ExpectEmptyAddressSet ensures the address set owned by dbIDsOrNsName exists with no IPs
func (f *FakeAddressSetFactoryForIPs) ExpectEmptyAddressSet(dbIDsOrNsName any) {
	dbIDs := f.getDbIDsFromNsNameOrDbIDs(dbIDsOrNsName)
	f.ExpectAddressSetWithIPs(dbIDs, nil)
}

// EventuallyExpectEmptyAddressSetExist ensures the named address set eventually exists with no IPs
func (f *FakeAddressSetFactoryForIPs) EventuallyExpectEmptyAddressSetExist(dbIDsOrNsName any) {
	dbIDs := f.getDbIDsFromNsNameOrDbIDs(dbIDsOrNsName)
	f.EventuallyExpectAddressSetWithIPs(dbIDs, nil)
}

func (f *FakeAddressSetFactoryForIPs) AddressSetExists(dbIDsOrNsName any) bool {
	dbIDs := f.getDbIDsFromNsNameOrDbIDs(dbIDsOrNsName)
	name := getOvnAddressSetsName(dbIDs)
	f.Lock()
	defer f.Unlock()
	_, ok := f.sets[name]
	return ok
}

// EventuallyExpectAddressSet ensures the named address set eventually exists
func (f *FakeAddressSetFactoryForIPs) EventuallyExpectAddressSet(dbIDsOrNsName any) {
	dbIDs := f.getDbIDsFromNsNameOrDbIDs(dbIDsOrNsName)
	gomega.Eventually(func() bool {
		return f.AddressSetExists(dbIDs)
	}).Should(gomega.BeTrue())
}

// EventuallyExpectNoAddressSet ensures the named address set eventually does not exist
// For namespaces address set deletion is delayed by 20 seconds, it is only tested once in namespace_test
// to not slow down tests. Don't use for namespace-owned address sets
func (f *FakeAddressSetFactoryForIPs) EventuallyExpectNoAddressSet(dbIDsOrNsName any) {
	dbIDs := f.getDbIDsFromNsNameOrDbIDs(dbIDsOrNsName)
	gomega.Eventually(func() bool {
		return f.AddressSetExists(dbIDs)
	}).Should(gomega.BeFalse())
}

// ExpectNumberOfAddressSets ensures the number of created address sets equals given number
func (f *FakeAddressSetFactoryForIPs) ExpectNumberOfAddressSets(n int) {
	gomega.Expect(len(f.sets)).To(gomega.Equal(n))
}

func (f *FakeAddressSetFactoryForIPs) newFakeAddressSetsIPs(ips []net.IP, dbIDs *libovsdbops.DbObjectIDs, removeFn removeFunc) (*fakeAddressSets, error) {
	var v4set, v6set *fakeAddressSet
	v4Ips := make([]net.IP, 0)
	v6Ips := make([]net.IP, 0)
	for _, ip := range ips {
		if utilnet.IsIPv6(ip) {
			v6Ips = append(v6Ips, ip)
		} else {
			v4Ips = append(v4Ips, ip)
		}
	}
	if config.IPv4Mode {
		v4set = f.newFakeAddressSetIPs(v4Ips, dbIDs, ipv4InternalID)
	}
	if config.IPv6Mode {
		v6set = f.newFakeAddressSetIPs(v6Ips, dbIDs, ipv6InternalID)
	}
	name := getOvnAddressSetsName(dbIDs)
	return &fakeAddressSets{name: name, ipv4: v4set, ipv6: v6set, dbIDs: dbIDs, removeFn: removeFn}, nil
}

func (f *FakeAddressSetFactoryForIPs) newFakeAddressSetIPs(ips []net.IP, dbIDs *libovsdbops.DbObjectIDs, ipFamily string) *fakeAddressSet {
	name := getDbIDsWithIPFamily(dbIDs, ipFamily).String()

	as := &fakeAddressSet{
		name:     name,
		hashName: hashedAddressSet(name),
		ips:      make(map[string]net.IP),
	}
	for _, ip := range ips {
		as.ips[ip.String()] = ip
	}
	return as
}

type removeFunc func(string)

type fakeAddressSet struct {
	name      string
	hashName  string
	ips       map[string]net.IP
	cidrs     map[string]*net.IPNet
	destroyed uint32
}

// fakeAddressSets implements the AddressSet interface for IPs
var _ AddressSetIPs = &fakeAddressSets{}

// fakeAddressSets implements the AddressSet interface for CIDRs
var _ AddressSetCIDRs = &fakeAddressSets{}

type fakeAddressSets struct {
	sync.Mutex
	// name without ip family
	name     string
	ipv4     *fakeAddressSet
	ipv6     *fakeAddressSet
	dbIDs    *libovsdbops.DbObjectIDs
	removeFn removeFunc
}

func (as *fakeAddressSets) GetASHashNames() (string, string) {
	var ipv4AS string
	var ipv6AS string
	if as.ipv4 != nil {
		ipv4AS = as.ipv4.getHashName()
	}
	if as.ipv6 != nil {
		ipv6AS = as.ipv6.getHashName()
	}
	return ipv4AS, ipv6AS
}

func (as *fakeAddressSets) GetName() string {
	return as.name
}

func (as *fakeAddressSets) AddIPs(ips []net.IP) error {
	_, err := as.AddIPsReturnOps(ips)
	return err
}

func (as *fakeAddressSets) AddIPsReturnOps(ips []net.IP) ([]ovsdb.Operation, error) {
	var ops []ovsdb.Operation
	var err error
	as.Lock()
	defer as.Unlock()
	for _, ip := range ips {
		if as.ipv6 != nil && utilnet.IsIPv6(ip) {
			ops, err = as.ipv6.addIP(ip)
		} else if as.ipv4 != nil && !utilnet.IsIPv6(ip) {
			ops, err = as.ipv4.addIP(ip)
		}
		if err != nil {
			return nil, err
		}
	}
	return ops, nil
}

func (as *fakeAddressSets) GetIPs() ([]string, []string) {
	as.Lock()
	defer as.Unlock()

	var v4ips []string
	var v6ips []string

	if as.ipv6 != nil {
		v6ips, _ = as.ipv6.getIPs()
	}
	if as.ipv4 != nil {
		v4ips, _ = as.ipv4.getIPs()
	}

	return v4ips, v6ips
}

func (as *fakeAddressSets) SetIPs(ips []net.IP) error {
	allIPs := []net.IP{}
	if as.ipv4 != nil {
		for _, ip := range as.ipv4.ips {
			allIPs = append(allIPs, ip)
		}
	}

	if as.ipv6 != nil {
		for _, ip := range as.ipv6.ips {
			allIPs = append(allIPs, ip)
		}
	}

	err := as.DeleteIPs(allIPs)
	if err != nil {
		return err
	}

	return as.AddIPs(ips)
}

func (as *fakeAddressSets) DeleteIPs(ips []net.IP) error {
	_, err := as.DeleteIPsReturnOps(ips)
	return err
}

func (as *fakeAddressSets) DeleteIPsReturnOps(ips []net.IP) ([]ovsdb.Operation, error) {
	var ops []ovsdb.Operation
	var err error
	as.Lock()
	defer as.Unlock()

	for _, ip := range ips {
		if as.ipv6 != nil && utilnet.IsIPv6(ip) {
			ops, err = as.ipv6.deleteIP(ip)
		} else if as.ipv4 != nil && !utilnet.IsIPv6(ip) {
			ops, err = as.ipv4.deleteIP(ip)
		}
		if err != nil {
			return nil, err
		}
	}
	return ops, nil
}

func (as *fakeAddressSets) AddCIDRs(cidrs []*net.IPNet) error {
	_, err := as.AddCIDRsReturnOps(cidrs)
	return err
}

func (as *fakeAddressSets) AddCIDRsReturnOps(cidrs []*net.IPNet) ([]ovsdb.Operation, error) {
	var ops []ovsdb.Operation
	var err error
	as.Lock()
	defer as.Unlock()
	for _, cidr := range cidrs {
		if as.ipv6 != nil && utilnet.IsIPv6CIDR(cidr) {
			ops, err = as.ipv6.addCIDR(cidr)
		} else if as.ipv4 != nil && !utilnet.IsIPv6CIDR(cidr) {
			ops, err = as.ipv4.addCIDR(cidr)
		}
		if err != nil {
			return nil, err
		}
	}
	return ops, nil
}

func (as *fakeAddressSets) GetCIDRs() ([]string, []string) {
	as.Lock()
	defer as.Unlock()

	var v4CIDRs []string
	var v6CIDRs []string

	if as.ipv6 != nil {
		v6CIDRs, _ = as.ipv6.getCIDRs()
	}
	if as.ipv4 != nil {
		v4CIDRs, _ = as.ipv4.getCIDRs()
	}

	return v4CIDRs, v6CIDRs
}

func (as *fakeAddressSets) SetCIDRs(cidrs []*net.IPNet) error {
	allCIDRs := []*net.IPNet{}
	if as.ipv4 != nil {
		for _, cidr := range as.ipv4.cidrs {
			allCIDRs = append(allCIDRs, cidr)
		}
	}

	if as.ipv6 != nil {
		for _, cidr := range as.ipv6.cidrs {
			allCIDRs = append(allCIDRs, cidr)
		}
	}

	err := as.DeleteCIDRs(allCIDRs)
	if err != nil {
		return err
	}

	return as.AddCIDRs(cidrs)
}

func (as *fakeAddressSets) DeleteCIDRs(cidrs []*net.IPNet) error {
	_, err := as.DeleteCIDRsReturnOps(cidrs)
	return err
}

func (as *fakeAddressSets) DeleteCIDRsReturnOps(cidrs []*net.IPNet) ([]ovsdb.Operation, error) {
	var ops []ovsdb.Operation
	var err error
	as.Lock()
	defer as.Unlock()

	for _, cidr := range cidrs {
		if as.ipv6 != nil && utilnet.IsIPv6CIDR(cidr) {
			ops, err = as.ipv6.deleteCIDR(cidr)
		} else if as.ipv4 != nil && !utilnet.IsIPv6CIDR(cidr) {
			ops, err = as.ipv4.deleteCIDR(cidr)
		}
		if err != nil {
			return nil, err
		}
	}
	return ops, nil
}

func (as *fakeAddressSets) Destroy() error {
	as.Lock()
	defer func() {
		as.Unlock()
		as.removeFn(as.name)
	}()

	if as.ipv4 != nil {
		err := as.ipv4.destroy()
		if err != nil {
			return err
		}
	}
	if as.ipv6 != nil {
		return as.ipv6.destroy()
	}
	return nil
}

func (as *fakeAddressSets) GetUUIDs() (string, string) {
	var ipv4UUID, ipv6UUID string
	if as.ipv4 != nil {
		ipv4UUID = as.ipv4.getUUID()
	}
	if as.ipv6 != nil {
		ipv6UUID = as.ipv6.getUUID()
	}
	return ipv4UUID, ipv6UUID
}

func (as *fakeAddressSet) getHashName() string {
	gomega.Expect(atomic.LoadUint32(&as.destroyed)).To(gomega.Equal(uint32(0)))
	return as.hashName
}

func (as *fakeAddressSet) addIP(ip net.IP) ([]ovsdb.Operation, error) {
	gomega.Expect(atomic.LoadUint32(&as.destroyed)).To(gomega.Equal(uint32(0)))
	ipStr := ip.String()
	if _, ok := as.ips[ipStr]; !ok {
		as.ips[ip.String()] = ip
	}
	return nil, nil
}

func (as *fakeAddressSet) getIPs() ([]string, error) {
	gomega.Expect(atomic.LoadUint32(&as.destroyed)).To(gomega.Equal(uint32(0)))
	uniqIPs := make([]string, 0, len(as.ips))
	for _, ip := range as.ips {
		uniqIPs = append(uniqIPs, ip.String())
	}
	return uniqIPs, nil
}

func (as *fakeAddressSet) deleteIP(ip net.IP) ([]ovsdb.Operation, error) {
	gomega.Expect(atomic.LoadUint32(&as.destroyed)).To(gomega.Equal(uint32(0)))
	delete(as.ips, ip.String())
	return nil, nil
}

func (as *fakeAddressSet) addCIDR(cidr *net.IPNet) ([]ovsdb.Operation, error) {
	gomega.Expect(atomic.LoadUint32(&as.destroyed)).To(gomega.Equal(uint32(0)))
	cidrStr := cidr.String()
	if _, ok := as.cidrs[cidrStr]; !ok {
		as.cidrs[cidr.String()] = cidr
	}
	return nil, nil
}

func (as *fakeAddressSet) getCIDRs() ([]string, error) {
	gomega.Expect(atomic.LoadUint32(&as.destroyed)).To(gomega.Equal(uint32(0)))
	uniqCIDRs := make([]string, 0, len(as.cidrs))
	for _, cidr := range as.cidrs {
		uniqCIDRs = append(uniqCIDRs, cidr.String())
	}
	return uniqCIDRs, nil
}

func (as *fakeAddressSet) deleteCIDR(cidr *net.IPNet) ([]ovsdb.Operation, error) {
	gomega.Expect(atomic.LoadUint32(&as.destroyed)).To(gomega.Equal(uint32(0)))
	delete(as.cidrs, cidr.String())
	return nil, nil
}

func (as *fakeAddressSet) destroy() error {
	// Don't check here if the address set was already destroyed as it should be
	// a thread safe, idempotent operation anyway.
	atomic.StoreUint32(&as.destroyed, 1)
	return nil
}

func (as *fakeAddressSet) getUUID() string {
	return as.name + "-UUID"
}

func NewFakeAddressSetFactoryForCIDRs(controllerName string) *FakeAddressSetFactoryForCIDRs {
	return &FakeAddressSetFactoryForCIDRs{
		ControllerName: controllerName,
		asf:            &ovnAddressSetFactory{},
		sets:           make(map[string]*fakeAddressSets),
	}
}

type FakeAddressSetFactoryForCIDRs struct {
	// ControllerName is stored here for convenience, it is used to build dbIDs for fake-only methods like
	// AddressSetExists, EventuallyExpectAddressSet, etc.
	ControllerName string
	asf            *ovnAddressSetFactory
	sync.Mutex
	// maps address set name to object
	sets                map[string]*fakeAddressSets
	errOnNextNewAddrSet bool
}

// fakeFactory implements the AddressSetFactoryIPs interface
var _ AddressSetFactoryCIDRs = &FakeAddressSetFactoryForCIDRs{}

// ErrOnNextNewASCall will make FakeAddressSetFactoryForCIDRs return FakeASFError on the next NewAddressSet call
func (f *FakeAddressSetFactoryForCIDRs) ErrOnNextNewASCall() {
	f.errOnNextNewAddrSet = true
}

// NewAddressSet returns a new address set object
func (f *FakeAddressSetFactoryForCIDRs) NewAddressSet(dbIDs *libovsdbops.DbObjectIDs, cidrs []*net.IPNet) (AddressSetCIDRs, error) {
	if f.errOnNextNewAddrSet {
		f.errOnNextNewAddrSet = false
		return nil, fmt.Errorf(FakeASFError)
	}
	if err := f.asf.validateDbIDs(dbIDs); err != nil {
		return nil, fmt.Errorf("failed to create address set: %w", err)
	}
	f.Lock()
	defer f.Unlock()
	name := getOvnAddressSetsName(dbIDs)

	_, ok := f.sets[name]
	gomega.Expect(ok).To(gomega.BeFalse(), fmt.Sprintf("new address set %s already exists", name))
	set, err := f.newFakeAddressSetsCIDRs(cidrs, dbIDs, f.removeAddressSet)
	if err != nil {
		return nil, err
	}
	f.sets[name] = set
	return set, nil
}

// NewAddressSetOps returns a new address set object
func (f *FakeAddressSetFactoryForCIDRs) NewAddressSetOps(dbIDs *libovsdbops.DbObjectIDs, cidrs []*net.IPNet) (AddressSetCIDRs, []ovsdb.Operation, error) {
	if f.errOnNextNewAddrSet {
		f.errOnNextNewAddrSet = false
		return nil, nil, fmt.Errorf(FakeASFError)
	}
	if err := f.asf.validateDbIDs(dbIDs); err != nil {
		return nil, nil, fmt.Errorf("failed to create address set: %w", err)
	}
	f.Lock()
	defer f.Unlock()
	name := getOvnAddressSetsName(dbIDs)

	_, ok := f.sets[name]
	gomega.Expect(ok).To(gomega.BeFalse(), fmt.Sprintf("new address set %s already exists", name))
	set, err := f.newFakeAddressSetsCIDRs(cidrs, dbIDs, f.removeAddressSet)
	if err != nil {
		return nil, nil, err
	}
	f.sets[name] = set
	return set, nil, nil
}

// EnsureAddressSet returns set object
func (f *FakeAddressSetFactoryForCIDRs) EnsureAddressSet(dbIDs *libovsdbops.DbObjectIDs) (AddressSetCIDRs, error) {
	if err := f.asf.validateDbIDs(dbIDs); err != nil {
		return nil, fmt.Errorf("failed to ensure address set: %w", err)
	}
	f.Lock()
	defer f.Unlock()
	name := getOvnAddressSetsName(dbIDs)
	set, ok := f.sets[name]
	if ok {
		return set, nil
	}
	set, err := f.newFakeAddressSetsCIDRs([]*net.IPNet{}, dbIDs, f.removeAddressSet)
	if err != nil {
		return nil, err
	}
	f.sets[name] = set
	return set, nil
}

// GetAddressSet returns set object
func (f *FakeAddressSetFactoryForCIDRs) GetAddressSet(dbIDs *libovsdbops.DbObjectIDs) (AddressSetCIDRs, error) {
	if err := f.asf.validateDbIDs(dbIDs); err != nil {
		return nil, fmt.Errorf("failed to get address set: %w", err)
	}
	f.Lock()
	defer f.Unlock()
	name := getOvnAddressSetsName(dbIDs)
	set, ok := f.sets[name]
	if ok {
		return set, nil
	}
	return nil, fmt.Errorf("error fetching address set")
}

func (f *FakeAddressSetFactoryForCIDRs) ProcessEachAddressSet(ownerController string, indexT *libovsdbops.ObjectIDsType, iteratorFn AddressSetIterFunc) error {
	f.Lock()
	asNames := map[string]*libovsdbops.DbObjectIDs{}
	for _, set := range f.sets {
		if !set.dbIDs.HasSameOwner(ownerController, indexT) {
			continue
		}
		// set.dbIDs doesn't have ip family
		addrSetName := getOvnAddressSetsName(set.dbIDs)
		if _, ok := asNames[addrSetName]; ok {
			continue
		}
		asNames[addrSetName] = set.dbIDs
	}
	f.Unlock()
	for _, dbIDs := range asNames {
		if err := iteratorFn(dbIDs); err != nil {
			return err
		}
	}
	return nil
}

func (f *FakeAddressSetFactoryForCIDRs) DestroyAddressSet(dbIDs *libovsdbops.DbObjectIDs) error {
	if err := f.asf.validateDbIDs(dbIDs); err != nil {
		return fmt.Errorf("failed to destroy address set: %w", err)
	}
	name := getOvnAddressSetsName(dbIDs)
	if _, ok := f.sets[name]; ok {
		f.removeAddressSet(name)
		return nil
	}
	return nil
}

func (f *FakeAddressSetFactoryForCIDRs) getAddressSet(dbIDs *libovsdbops.DbObjectIDs) *fakeAddressSets {
	f.Lock()
	defer f.Unlock()
	name := getOvnAddressSetsName(dbIDs)
	if as, ok := f.sets[name]; ok {
		as.Lock()
		return as
	}
	return nil
}

// removeAddressSet removes the address set from the factory
func (f *FakeAddressSetFactoryForCIDRs) removeAddressSet(name string) {
	f.Lock()
	defer f.Unlock()
	delete(f.sets, name)
}

// expectAddressSetWithCIDRs ensures the named address set exists with the given set of CIDRs
func (f *FakeAddressSetFactoryForCIDRs) expectAddressSetWithCIDRs(g gomega.Gomega, dbIDs *libovsdbops.DbObjectIDs, cidrs []string) {
	var lenAddressSet int
	as := f.getAddressSet(dbIDs)
	gomega.Expect(as).ToNot(gomega.BeNil(), fmt.Sprintf("expected address set %s to exist", dbIDs.String()))
	defer as.Unlock()
	as4 := as.ipv4
	if as4 != nil {
		lenAddressSet = lenAddressSet + len(as4.ips)
	}
	as6 := as.ipv6
	if as6 != nil {
		lenAddressSet = lenAddressSet + len(as6.ips)
	}

	for _, cidr := range cidrs {
		if utilnet.IsIPv6String(cidr) {
			g.Expect(as6).NotTo(gomega.BeNil())
			g.Expect(as6.cidrs).To(gomega.HaveKey(cidr))
		} else {
			g.Expect(as4).NotTo(gomega.BeNil())
			g.Expect(as4.cidrs).To(gomega.HaveKey(cidr))
		}
	}
	if lenAddressSet != len(cidrs) {
		var addrs []string
		if as4 != nil {
			for _, v := range as4.cidrs {
				addrs = append(addrs, v.String())
			}
		}
		if as6 != nil {
			for _, v := range as6.cidrs {
				addrs = append(addrs, v.String())
			}
		}

		klog.Errorf("IPv4 addresses mismatch in cache: %#v, expected: %#v", addrs, cidrs)
	}

	g.Expect(lenAddressSet).To(gomega.Equal(len(cidrs)))
}

func (f *FakeAddressSetFactoryForCIDRs) getDbIDsFromNsNameOrDbIDs(dbIDsOrNsName any) *libovsdbops.DbObjectIDs {
	var dbIDs *libovsdbops.DbObjectIDs
	if nsName, ok := dbIDsOrNsName.(string); ok {
		dbIDs = libovsdbops.NewDbObjectIDs(libovsdbops.AddressSetNamespace, f.ControllerName, map[libovsdbops.ExternalIDKey]string{
			libovsdbops.ObjectNameKey: nsName,
		})
	} else if dbIDs, ok = dbIDsOrNsName.(*libovsdbops.DbObjectIDs); !ok {
		panic("unexpected type of argument passed to ExpectAddressSetWithIPs")
	}
	return dbIDs
}

// ExpectAddressSetWithCIDRs ensure address set exists with the given set of CIDRs.
// Address set is identified by dbIDsOrNsName, which may be a namespace name (string) or a *libovsdbops.DbObjectIDs.
func (f *FakeAddressSetFactoryForCIDRs) ExpectAddressSetWithCIDRs(dbIDsOrNsName any, cidrs []string) {
	dbIDs := f.getDbIDsFromNsNameOrDbIDs(dbIDsOrNsName)
	g := gomega.Default
	f.expectAddressSetWithCIDRs(g, dbIDs, cidrs)
}

func (f *FakeAddressSetFactoryForCIDRs) EventuallyExpectAddressSetWithCIDRs(dbIDsOrNsName any, ips []string) {
	dbIDs := f.getDbIDsFromNsNameOrDbIDs(dbIDsOrNsName)
	gomega.Eventually(func(g gomega.Gomega) {
		f.expectAddressSetWithCIDRs(g, dbIDs, ips)
	}).Should(gomega.Succeed())
}

// ExpectEmptyAddressSet ensures the address set owned by dbIDsOrNsName exists with no CIDRs
func (f *FakeAddressSetFactoryForCIDRs) ExpectEmptyAddressSet(dbIDsOrNsName any) {
	dbIDs := f.getDbIDsFromNsNameOrDbIDs(dbIDsOrNsName)
	f.ExpectAddressSetWithCIDRs(dbIDs, nil)
}

// EventuallyExpectEmptyAddressSetExist ensures the named address set eventually exists with no CIDRs
func (f *FakeAddressSetFactoryForCIDRs) EventuallyExpectEmptyAddressSetExist(dbIDsOrNsName any) {
	dbIDs := f.getDbIDsFromNsNameOrDbIDs(dbIDsOrNsName)
	f.EventuallyExpectAddressSetWithCIDRs(dbIDs, nil)
}

func (f *FakeAddressSetFactoryForCIDRs) AddressSetExists(dbIDsOrNsName any) bool {
	dbIDs := f.getDbIDsFromNsNameOrDbIDs(dbIDsOrNsName)
	name := getOvnAddressSetsName(dbIDs)
	f.Lock()
	defer f.Unlock()
	_, ok := f.sets[name]
	return ok
}

// EventuallyExpectAddressSet ensures the named address set eventually exists
func (f *FakeAddressSetFactoryForCIDRs) EventuallyExpectAddressSet(dbIDsOrNsName any) {
	dbIDs := f.getDbIDsFromNsNameOrDbIDs(dbIDsOrNsName)
	gomega.Eventually(func() bool {
		return f.AddressSetExists(dbIDs)
	}).Should(gomega.BeTrue())
}

// EventuallyExpectNoAddressSet ensures the named address set eventually does not exist
// For namespaces address set deletion is delayed by 20 seconds, it is only tested once in namespace_test
// to not slow down tests. Don't use for namespace-owned address sets
func (f *FakeAddressSetFactoryForCIDRs) EventuallyExpectNoAddressSet(dbIDsOrNsName any) {
	dbIDs := f.getDbIDsFromNsNameOrDbIDs(dbIDsOrNsName)
	gomega.Eventually(func() bool {
		return f.AddressSetExists(dbIDs)
	}).Should(gomega.BeFalse())
}

// ExpectNumberOfAddressSets ensures the number of created address sets equals given number
func (f *FakeAddressSetFactoryForCIDRs) ExpectNumberOfAddressSets(n int) {
	gomega.Expect(len(f.sets)).To(gomega.Equal(n))
}

func (f *FakeAddressSetFactoryForCIDRs) newFakeAddressSetsCIDRs(cidrs []*net.IPNet, dbIDs *libovsdbops.DbObjectIDs, removeFn removeFunc) (*fakeAddressSets, error) {
	var v4set, v6set *fakeAddressSet
	v4CIDRs := make([]*net.IPNet, 0)
	v6CIDRs := make([]*net.IPNet, 0)
	for _, cidr := range cidrs {
		if utilnet.IsIPv6CIDR(cidr) {
			v6CIDRs = append(v6CIDRs, cidr)
		} else {
			v4CIDRs = append(v4CIDRs, cidr)
		}
	}
	if config.IPv4Mode {
		v4set = f.newFakeAddressSetCIDRs(v4CIDRs, dbIDs, ipv4InternalID)
	}
	if config.IPv6Mode {
		v6set = f.newFakeAddressSetCIDRs(v6CIDRs, dbIDs, ipv6InternalID)
	}
	name := getOvnAddressSetsName(dbIDs)
	return &fakeAddressSets{name: name, ipv4: v4set, ipv6: v6set, dbIDs: dbIDs, removeFn: removeFn}, nil
}

func (f *FakeAddressSetFactoryForCIDRs) newFakeAddressSetCIDRs(cidrs []*net.IPNet, dbIDs *libovsdbops.DbObjectIDs, ipFamily string) *fakeAddressSet {
	name := getDbIDsWithIPFamily(dbIDs, ipFamily).String()

	as := &fakeAddressSet{
		name:     name,
		hashName: hashedAddressSet(name),
		cidrs:    make(map[string]*net.IPNet),
	}
	for _, cidr := range cidrs {
		as.cidrs[cidr.String()] = cidr
	}
	return as
}
