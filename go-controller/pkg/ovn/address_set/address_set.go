package addressset

import (
	"fmt"
	"net"

	"github.com/pkg/errors"

	libovsdbclient "github.com/ovn-org/libovsdb/client"
	"github.com/ovn-org/libovsdb/ovsdb"
	libovsdbops "github.com/ovn-org/ovn-kubernetes/go-controller/pkg/libovsdb/ops"
	"github.com/ovn-org/ovn-kubernetes/go-controller/pkg/nbdb"
	"github.com/ovn-org/ovn-kubernetes/go-controller/pkg/util"

	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/klog/v2"
	utilnet "k8s.io/utils/net"
)

const (
	ipv4InternalID = "v4"
	ipv6InternalID = "v6"
)

type AddressSetIterFunc func(dbIDs *libovsdbops.DbObjectIDs) error

// AddressSetFactoryIPs is an interface for managing address set objects for IP addresses.
type AddressSetFactoryIPs interface {
	// NewAddressSet returns a new object that implements AddressSet
	// and contains the given IPs, or an error. Internally it creates
	// an address set for IPv4 and IPv6 each.
	NewAddressSet(dbIDs *libovsdbops.DbObjectIDs, ips []net.IP) (AddressSetIPs, error)
	// NewAddressSetOps returns a new object that implements AddressSet
	// and contains the given IPs, or an error. Internally it creates
	// ops to create an address set for IPv4 and IPv6 each.
	NewAddressSetOps(dbIDs *libovsdbops.DbObjectIDs, ips []net.IP) (AddressSetIPs, []ovsdb.Operation, error)
	// EnsureAddressSet makes sure that an address set object exists in ovn
	// with the given dbIDs.
	EnsureAddressSet(dbIDs *libovsdbops.DbObjectIDs) (AddressSetIPs, error)
	// ProcessEachAddressSet calls the given function for each address set of type dbIDsType owned by given ownerController.
	ProcessEachAddressSet(ownerController string, dbIDsType *libovsdbops.ObjectIDsType, iteratorFn AddressSetIterFunc) error
	// DestroyAddressSet deletes the address sets with given dbIDs.
	DestroyAddressSet(dbIDs *libovsdbops.DbObjectIDs) error
	// GetAddressSet returns the address-set that matches the given dbIDs
	GetAddressSet(dbIDs *libovsdbops.DbObjectIDs) (AddressSetIPs, error)
}

// AddressSetIPs is an interface for address set objects which contain IP addresses
type AddressSetIPs interface {
	// GetASHashNames returns the hashed name for ipv6 and ipv4 addressSets
	GetASHashNames() (string, string)
	// GetName returns the descriptive name of the address set
	GetName() string
	// AddIPs adds the array of IPs to the address set
	AddIPs(ip []net.IP) error
	// AddIPsReturnOps returns the ops needed to add the array of IPs to the address set
	AddIPsReturnOps(ip []net.IP) ([]ovsdb.Operation, error)
	// GetIPs gets the list of v4 & v6 IPs from the address set
	GetIPs() ([]string, []string)
	// SetIPs sets the address set to the given array of addresses
	SetIPs(ip []net.IP) error
	DeleteIPs(ip []net.IP) error
	// DeleteIPsReturnOps returns the ops needed to delete the array of IPs from the address set
	DeleteIPsReturnOps(ip []net.IP) ([]ovsdb.Operation, error)
	Destroy() error
}

// AddressSetFactoryCIDRs is an interface for managing address set objects for CIDRs
type AddressSetFactoryCIDRs interface {
	// NewAddressSet returns a new object that implements AddressSet
	// and contains the given CIDRs, or an error. Internally it creates
	// an address set for IPv4 and IPv6 each.
	NewAddressSet(dbIDs *libovsdbops.DbObjectIDs, cidrs []*net.IPNet) (AddressSetCIDRs, error)
	// NewAddressSetOps returns a new object that implements AddressSet
	// and contains the given CIDRs, or an error. Internally it creates
	// ops to create an address set for IPv4 and IPv6 each.
	NewAddressSetOps(dbIDs *libovsdbops.DbObjectIDs, cidrs []*net.IPNet) (AddressSetCIDRs, []ovsdb.Operation, error)
	// EnsureAddressSet makes sure that an address set object exists in ovn
	// with the given dbIDs.
	EnsureAddressSet(dbIDs *libovsdbops.DbObjectIDs) (AddressSetCIDRs, error)
	// ProcessEachAddressSet calls the given function for each address set of type dbIDsType owned by given ownerController.
	ProcessEachAddressSet(ownerController string, dbIDsType *libovsdbops.ObjectIDsType, iteratorFn AddressSetIterFunc) error
	// DestroyAddressSet deletes the address sets with given dbIDs.
	DestroyAddressSet(dbIDs *libovsdbops.DbObjectIDs) error
	// GetAddressSet returns the address-set that matches the given dbIDs
	GetAddressSet(dbIDs *libovsdbops.DbObjectIDs) (AddressSetCIDRs, error)
}

// AddressSetCIDRs is an interface for address set objects which contain CIDRs
type AddressSetCIDRs interface {
	// GetASHashNames returns the hashed name for ipv6 and ipv4 addressSets
	GetASHashNames() (string, string)
	// GetName returns the descriptive name of the address set
	GetName() string
	// AddCIDRs adds the array of CIDRs to the address set
	AddCIDRs(cidrs []*net.IPNet) error
	// AddCIDRsReturnOps returns the ops needed to add the array of CIDRs to the address set
	AddCIDRsReturnOps(ip []*net.IPNet) ([]ovsdb.Operation, error)
	// GetCIDRs gets the list of v4 & v6 CIDRs from the address set
	GetCIDRs() ([]string, []string)
	// SetCIDRs sets the address set to the given array of addresses
	SetCIDRs(cidrs []*net.IPNet) error
	DeleteCIDRs(cidrs []*net.IPNet) error
	// DeleteCIDRsReturnOps returns the ops needed to delete the array of CIDRs from the address set
	DeleteCIDRsReturnOps(cidrs []*net.IPNet) ([]ovsdb.Operation, error)
	Destroy() error
	GetUUIDs() (string, string)
}

type ovnAddressSetFactory struct {
	nbClient libovsdbclient.Client
	ipv4Mode bool
	ipv6Mode bool
}

type ovnAddressSetFactoryIPs struct {
	ovnAddressSetFactory
}

// NewOvnAddressSetFactoryForIPs creates a new AddressSetFactoryIPs backed by
// address set objects that execute OVN commands
func NewOvnAddressSetFactoryForIPs(nbClient libovsdbclient.Client, ipv4Mode, ipv6Mode bool) AddressSetFactoryIPs {
	return &ovnAddressSetFactoryIPs{
		ovnAddressSetFactory{
			nbClient: nbClient,
			ipv4Mode: ipv4Mode,
			ipv6Mode: ipv6Mode,
		},
	}
}

// ovnAddressSetFactory implements the AddressSetFactoryIPs interface
var _ AddressSetFactoryIPs = &ovnAddressSetFactoryIPs{}

// NewAddressSet returns a new object that implements AddressSet
// and contains the given IPs, or an error. Internally it creates
// an address set for IPv4 and IPv6 each.
func (asf *ovnAddressSetFactoryIPs) NewAddressSet(dbIDs *libovsdbops.DbObjectIDs, ips []net.IP) (AddressSetIPs, error) {
	as, ops, err := asf.NewAddressSetOps(dbIDs, ips)
	if err != nil {
		return nil, err
	}
	_, err = libovsdbops.TransactAndCheck(asf.nbClient, ops)
	if err != nil {
		return nil, err
	}
	return as, nil
}

// NewAddressSetOps returns a new object that implements AddressSet
// and contains the given IPs, or an error. Internally it creates
// address set ops for IPv4 and IPv6 each.
func (asf *ovnAddressSetFactoryIPs) NewAddressSetOps(dbIDs *libovsdbops.DbObjectIDs, ips []net.IP) (AddressSetIPs, []ovsdb.Operation, error) {
	if err := asf.validateDbIDs(dbIDs); err != nil {
		return nil, nil, fmt.Errorf("failed to create address set ops: %w", err)
	}
	return asf.ensureOvnAddressSetsOps(ips, dbIDs, true)
}

// EnsureAddressSet makes sure that an address set object exists in ovn
// with the given dbIDs.
func (asf *ovnAddressSetFactoryIPs) EnsureAddressSet(dbIDs *libovsdbops.DbObjectIDs) (AddressSetIPs, error) {
	if err := asf.validateDbIDs(dbIDs); err != nil {
		return nil, fmt.Errorf("failed to ensure address set: %w", err)
	}
	as, ops, err := asf.ensureOvnAddressSetsOps(nil, dbIDs, false)
	if err != nil {
		return nil, err
	}
	_, err = libovsdbops.TransactAndCheck(asf.nbClient, ops)
	if err != nil {
		return nil, err
	}
	return as, nil
}

// GetAddressSet returns the address-set that matches the given dbIDs
func (asf *ovnAddressSetFactoryIPs) GetAddressSet(dbIDs *libovsdbops.DbObjectIDs) (AddressSetIPs, error) {
	if err := asf.validateDbIDs(dbIDs); err != nil {
		return nil, fmt.Errorf("failed to get address set: %w", err)
	}
	var (
		v4set, v6set *ovnAddressSet
	)
	p := libovsdbops.GetPredicate[*nbdb.AddressSet](dbIDs, nil)
	addrSetList, err := libovsdbops.FindAddressSetsWithPredicate(asf.nbClient, p)
	if err != nil {
		return nil, fmt.Errorf("error getting address sets: %w", err)
	}
	for i := range addrSetList {
		addrSet := addrSetList[i]
		if addrSet.ExternalIDs[libovsdbops.AddressSetIPFamilyKey.String()] == ipv4InternalID {
			v4set = asf.newOvnAddressSet(addrSet)
		}
		if addrSet.ExternalIDs[libovsdbops.AddressSetIPFamilyKey.String()] == ipv6InternalID {
			v6set = asf.newOvnAddressSet(addrSet)
		}
	}
	return asf.newOvnAddressSets(v4set, v6set, dbIDs), nil
}

// if updateAS is false, ips will be ignored, only empty address sets will be created or existing address sets will
// be returned
func (asf *ovnAddressSetFactoryIPs) ensureOvnAddressSetsOps(ips []net.IP, dbIDs *libovsdbops.DbObjectIDs,
	updateAS bool) (*ovnAddressSets, []ovsdb.Operation, error) {
	var (
		v4set, v6set *ovnAddressSet
		v4IPs, v6IPs []net.IP
		err          error
	)
	if ips != nil {
		v4IPs, v6IPs = splitIPsByFamily(ips)
	}
	var ops []ovsdb.Operation
	if asf.ipv4Mode {
		v4set, ops, err = asf.ensureOvnAddressSetOps(v4IPs, dbIDs, ipv4InternalID, updateAS, ops)
		if err != nil {
			return nil, nil, err
		}
	}
	if asf.ipv6Mode {
		v6set, ops, err = asf.ensureOvnAddressSetOps(v6IPs, dbIDs, ipv6InternalID, updateAS, ops)
		if err != nil {
			return nil, nil, err
		}
	}
	return asf.newOvnAddressSets(v4set, v6set, dbIDs), ops, nil
}

func (asf *ovnAddressSetFactoryIPs) ensureOvnAddressSetOps(ips []net.IP, dbIDs *libovsdbops.DbObjectIDs,
	ipFamily string, updateAS bool, ops []ovsdb.Operation) (*ovnAddressSet, []ovsdb.Operation, error) {
	addrSet := buildAddressSet(dbIDs, ipFamily)
	var err error
	if updateAS {
		// overwrite ips, EnsureAddressSet doesn't do that
		uniqIPs := ipsToStringUnique(ips)
		addrSet.Addresses = uniqIPs
		ops, err = libovsdbops.CreateOrUpdateAddressSetsOps(asf.nbClient, ops, addrSet)
	} else {
		ops, err = libovsdbops.CreateAddressSetsOps(asf.nbClient, ops, addrSet)
	}

	// UUID should always be set if no error, check anyway
	if err != nil {
		// NOTE: While ovsdb transactions get serialized by libovsdb, the decision to create vs. update
		// the address set takes place before that serialization is done. Because of that, it is feasible
		// that one of the go threads attempting to call this routine at the same time will fail.
		// This is described in https://bugzilla.redhat.com/show_bug.cgi?id=2108026 . While we could
		// handle that failure here by retrying, a higher level retry (see retry_obj.go) is already
		// present in the codepath, so no additional handling for that condition has been added here.
		return nil, nil, fmt.Errorf("failed to create or update address set ops %+v: %w", addrSet, err)
	}
	as := asf.newOvnAddressSet(addrSet)
	klog.V(5).Infof("New(%s) with %v", asDetail(as), ips)
	return as, ops, nil
}

type ovnAddressSetFactoryCIDRs struct {
	ovnAddressSetFactory
}

// NewOvnAddressSetFactoryForCIDRs creates a new AddressSetFactoryCIDRs backed by
// address set objects that execute OVN commands
func NewOvnAddressSetFactoryForCIDRs(nbClient libovsdbclient.Client, v4, v6 bool) AddressSetFactoryCIDRs {
	return &ovnAddressSetFactoryCIDRs{
		ovnAddressSetFactory{
			nbClient: nbClient,
			ipv4Mode: v4,
			ipv6Mode: v6,
		},
	}
}

// ovnAddressSetFactory implements the AddressSetFactoryCIDRs interface
var _ AddressSetFactoryCIDRs = &ovnAddressSetFactoryCIDRs{}

// NewAddressSet returns a new object that implements AddressSet
// and contains the given IPs, or an error. Internally it creates
// an address set for IPv4 and IPv6 each.
func (asf *ovnAddressSetFactoryCIDRs) NewAddressSet(dbIDs *libovsdbops.DbObjectIDs, cidrs []*net.IPNet) (AddressSetCIDRs, error) {
	as, ops, err := asf.NewAddressSetOps(dbIDs, cidrs)
	if err != nil {
		return nil, err
	}
	_, err = libovsdbops.TransactAndCheck(asf.nbClient, ops)
	if err != nil {
		return nil, err
	}
	return as, nil
}

// NewAddressSetOps returns a new object that implements AddressSet
// and contains the given CIDRs, or an error. Internally it creates
// address set ops for IPv4 and IPv6 each.
func (asf *ovnAddressSetFactoryCIDRs) NewAddressSetOps(dbIDs *libovsdbops.DbObjectIDs, cidrs []*net.IPNet) (AddressSetCIDRs, []ovsdb.Operation, error) {
	if err := asf.validateDbIDs(dbIDs); err != nil {
		return nil, nil, fmt.Errorf("failed to create address set ops: %w", err)
	}
	return asf.ensureOvnAddressSetsOps(cidrs, dbIDs, true)
}

// EnsureAddressSet makes sure that an address set object exists in ovn
// with the given dbIDs.
func (asf *ovnAddressSetFactoryCIDRs) EnsureAddressSet(dbIDs *libovsdbops.DbObjectIDs) (AddressSetCIDRs, error) {
	if err := asf.validateDbIDs(dbIDs); err != nil {
		return nil, fmt.Errorf("failed to ensure address set: %w", err)
	}
	klog.Infof("MARTIN: tranxing ensure addr set %+v", dbIDs)
	as, ops, err := asf.ensureOvnAddressSetsOps(nil, dbIDs, false)
	if err != nil {
		return nil, fmt.Errorf("failed to generate ops: %v", err)
	}
	klog.Infof("MARTIN: tranxing ensure addr set and %d ops", len(ops))
	_, err = libovsdbops.TransactAndCheck(asf.nbClient, ops)
	if err != nil {
		return nil, fmt.Errorf("failed to transact: %v", err)
	}
	return as, nil
}

// GetAddressSet returns the address-set that matches the given dbIDs
func (asf *ovnAddressSetFactoryCIDRs) GetAddressSet(dbIDs *libovsdbops.DbObjectIDs) (AddressSetCIDRs, error) {
	if err := asf.validateDbIDs(dbIDs); err != nil {
		return nil, fmt.Errorf("failed to get address set: %w", err)
	}
	var (
		v4set, v6set *ovnAddressSet
	)
	p := libovsdbops.GetPredicate[*nbdb.AddressSet](dbIDs, nil)
	addrSetList, err := libovsdbops.FindAddressSetsWithPredicate(asf.nbClient, p)
	if err != nil {
		return nil, fmt.Errorf("error getting address sets: %w", err)
	}
	for i := range addrSetList {
		addrSet := addrSetList[i]
		if addrSet.ExternalIDs[libovsdbops.AddressSetIPFamilyKey.String()] == ipv4InternalID {
			v4set = asf.newOvnAddressSet(addrSet)
		}
		if addrSet.ExternalIDs[libovsdbops.AddressSetIPFamilyKey.String()] == ipv6InternalID {
			v6set = asf.newOvnAddressSet(addrSet)
		}
	}
	return asf.newOvnAddressSets(v4set, v6set, dbIDs), nil
}

// if updateAS is false, cidrs will be ignored, only empty address sets will be created or existing address sets will
// be returned
func (asf *ovnAddressSetFactoryCIDRs) ensureOvnAddressSetsOps(cidrs []*net.IPNet, dbIDs *libovsdbops.DbObjectIDs,
	updateAS bool) (*ovnAddressSets, []ovsdb.Operation, error) {
	var (
		v4set, v6set     *ovnAddressSet
		v4CIDRs, v6CIDRs []*net.IPNet
		err              error
	)
	if cidrs != nil {
		v4CIDRs, v6CIDRs = splitCIDRsByFamily(cidrs)
	}
	var ops []ovsdb.Operation
	if asf.ipv4Mode {
		v4set, ops, err = asf.ensureOvnAddressSetOps(v4CIDRs, dbIDs, ipv4InternalID, updateAS, ops)
		if err != nil {
			return nil, nil, err
		}
	}
	if asf.ipv6Mode {
		v6set, ops, err = asf.ensureOvnAddressSetOps(v6CIDRs, dbIDs, ipv6InternalID, updateAS, ops)
		if err != nil {
			return nil, nil, err
		}
	}
	return asf.newOvnAddressSets(v4set, v6set, dbIDs), ops, nil
}

func (asf *ovnAddressSetFactoryCIDRs) ensureOvnAddressSetOps(cidrs []*net.IPNet, dbIDs *libovsdbops.DbObjectIDs,
	ipFamily string, updateAS bool, ops []ovsdb.Operation) (*ovnAddressSet, []ovsdb.Operation, error) {
	addrSet := buildAddressSet(dbIDs, ipFamily)
	var err error
	if updateAS {
		// overwrite CIDRs, EnsureAddressSet doesn't do that
		uniqIPs := cidrsToStringUnique(cidrs)
		addrSet.Addresses = uniqIPs
		ops, err = libovsdbops.CreateOrUpdateAddressSetsOps(asf.nbClient, ops, addrSet)
	} else {
		ops, err = libovsdbops.CreateAddressSetsOps(asf.nbClient, ops, addrSet)
	}

	// UUID should always be set if no error, check anyway
	if err != nil {
		// NOTE: While ovsdb transactions get serialized by libovsdb, the decision to create vs. update
		// the address set takes place before that serialization is done. Because of that, it is feasible
		// that one of the go threads attempting to call this routine at the same time will fail.
		// This is described in https://bugzilla.redhat.com/show_bug.cgi?id=2108026 . While we could
		// handle that failure here by retrying, a higher level retry (see retry_obj.go) is already
		// present in the codepath, so no additional handling for that condition has been added here.
		return nil, nil, fmt.Errorf("failed to create or update address set ops %+v: %w", addrSet, err)
	}
	as := asf.newOvnAddressSet(addrSet)
	klog.V(5).Infof("New(%s) with %v", asDetail(as), cidrs)
	return as, ops, nil
}

func getDbIDsWithIPFamily(dbIDs *libovsdbops.DbObjectIDs, ipFamily string) *libovsdbops.DbObjectIDs {
	return dbIDs.AddIDs(map[libovsdbops.ExternalIDKey]string{libovsdbops.AddressSetIPFamilyKey: ipFamily})
}

func buildAddressSet(dbIDs *libovsdbops.DbObjectIDs, ipFamily string) *nbdb.AddressSet {
	dbIDsWithIPFam := getDbIDsWithIPFamily(dbIDs, ipFamily)
	externalIDs := dbIDsWithIPFam.GetExternalIDs()
	name := externalIDs[libovsdbops.PrimaryIDKey.String()]
	as := &nbdb.AddressSet{
		Name:        hashedAddressSet(name),
		ExternalIDs: externalIDs,
	}
	return as
}

func (asf *ovnAddressSetFactory) newOvnAddressSet(addrSet *nbdb.AddressSet) *ovnAddressSet {
	return &ovnAddressSet{
		nbClient: asf.nbClient,
		name:     addrSet.ExternalIDs[libovsdbops.PrimaryIDKey.String()],
		hashName: addrSet.Name,
		uuid:     addrSet.UUID,
	}
}

func (asf *ovnAddressSetFactory) validateDbIDs(dbIDs *libovsdbops.DbObjectIDs) error {
	unsetKeys := dbIDs.GetUnsetKeys()
	if len(unsetKeys) == 1 && unsetKeys[0] == libovsdbops.AddressSetIPFamilyKey {
		return nil
	}
	return fmt.Errorf("wrong set of keys is unset %v", unsetKeys)
}

// forEachAddressSet executes a do function on each address set owned by ovnAddressSetFactory.ControllerName
func (asf *ovnAddressSetFactory) forEachAddressSet(ownerController string, dbIDsType *libovsdbops.ObjectIDsType,
	do func(*nbdb.AddressSet) error) error {
	predIDs := libovsdbops.NewDbObjectIDs(dbIDsType, ownerController, nil)
	p := libovsdbops.GetPredicate[*nbdb.AddressSet](predIDs, nil)
	addrSetList, err := libovsdbops.FindAddressSetsWithPredicate(asf.nbClient, p)
	if err != nil {
		return fmt.Errorf("error reading address sets: %+v", err)
	}

	var errs []error
	for _, addrSet := range addrSetList {
		if err = do(addrSet); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("failed to iterate address sets: %v", utilerrors.NewAggregate(errs))
	}

	return nil
}

// ProcessEachAddressSet calls the given function for each address set of type dbIDsType owned by given ownerController.
func (asf *ovnAddressSetFactory) ProcessEachAddressSet(ownerController string, dbIDsType *libovsdbops.ObjectIDsType,
	iteratorFn AddressSetIterFunc) error {
	processedAddressSets := sets.Set[string]{}
	return asf.forEachAddressSet(ownerController, dbIDsType, func(as *nbdb.AddressSet) error {
		dbIDs, err := libovsdbops.NewDbObjectIDsFromExternalIDs(dbIDsType, as.ExternalIDs)
		if err != nil {
			return fmt.Errorf("failed to get objectIDs for %+v address set from ExternalIDs: %w", as, err)
		}
		// remove ipFamily to process address set only once
		dbIDsWithoutIPFam := dbIDs.RemoveIDs(libovsdbops.AddressSetIPFamilyKey)
		nameWithoutIPFam := getOvnAddressSetsName(dbIDsWithoutIPFam)
		if processedAddressSets.Has(nameWithoutIPFam) {
			// We have already processed the address set. In case of dual stack we will have _v4 and _v6
			// suffixes for address sets. Since we are normalizing these two address sets through this API
			// we will process only one normalized address set name.
			return nil
		}
		processedAddressSets.Insert(nameWithoutIPFam)
		return iteratorFn(dbIDsWithoutIPFam)
	})
}

// DestroyAddressSet deletes the address sets with given dbIDs.
func (asf *ovnAddressSetFactory) DestroyAddressSet(dbIDs *libovsdbops.DbObjectIDs) error {
	if err := asf.validateDbIDs(dbIDs); err != nil {
		return fmt.Errorf("failed to destroy address set: %w", err)
	}
	asv4 := &nbdb.AddressSet{
		Name: buildAddressSet(dbIDs, ipv4InternalID).Name,
	}
	asv6 := &nbdb.AddressSet{
		Name: buildAddressSet(dbIDs, ipv6InternalID).Name,
	}
	err := libovsdbops.DeleteAddressSets(asf.nbClient, asv4, asv6)
	if err != nil {
		return fmt.Errorf("failed to delete address sets %s: %v", getOvnAddressSetsName(dbIDs), err)
	}
	return nil
}

// GetHashNamesForAS returns hashed address set names for given dbIDs for both ip families.
// Can be used to cleanup e.g. address set references if the address set was deleted.
func GetHashNamesForAS(dbIDs *libovsdbops.DbObjectIDs) (string, string) {
	return buildAddressSet(dbIDs, ipv4InternalID).Name,
		buildAddressSet(dbIDs, ipv6InternalID).Name
}

// GetTestDbAddrSetsIPs returns nbdb.AddressSet objects both for ipv4 and ipv6, regardless of current config.
// May only be used for testing.
func GetTestDbAddrSetsIPs(dbIDs *libovsdbops.DbObjectIDs, ips []net.IP) (*nbdb.AddressSet, *nbdb.AddressSet) {
	var v4set, v6set *nbdb.AddressSet
	v4IPs, v6IPs := splitIPsByFamily(ips)
	// v4 address set
	v4set = buildAddressSet(dbIDs, ipv4InternalID)
	uniqIPs := ipsToStringUnique(v4IPs)
	v4set.Addresses = uniqIPs
	v4set.UUID = v4set.Name + "-UUID"
	// v6 address set
	v6set = buildAddressSet(dbIDs, ipv6InternalID)
	uniqIPs = ipsToStringUnique(v6IPs)
	v6set.Addresses = uniqIPs
	v6set.UUID = v6set.Name + "-UUID"
	return v4set, v6set
}

// GetTestDbAddrSetsCIDRs returns nbdb.AddressSet objects both for ipv4 and ipv6, regardless of current config.
// May only be used for testing.
func GetTestDbAddrSetsCIDRs(dbIDs *libovsdbops.DbObjectIDs, cidrs []*net.IPNet) (*nbdb.AddressSet, *nbdb.AddressSet) {
	var v4set, v6set *nbdb.AddressSet
	v4CIDRs, v6CIDRs := splitCIDRsByFamily(cidrs)
	// v4 address set
	v4set = buildAddressSet(dbIDs, ipv4InternalID)
	uniqCIDRs := cidrsToStringUnique(v4CIDRs)
	v4set.Addresses = uniqCIDRs
	v4set.UUID = v4set.Name + "-UUID"
	// v6 address set
	v6set = buildAddressSet(dbIDs, ipv6InternalID)
	uniqCIDRs = cidrsToStringUnique(v6CIDRs)
	v6set.Addresses = uniqCIDRs
	v6set.UUID = v6set.Name + "-UUID"
	return v4set, v6set
}

// ovnAddressSet is ipFamily-specific address set
type ovnAddressSet struct {
	nbClient libovsdbclient.Client
	// name is based on dbIDs and ipFamily
	name string
	// hashName = hashedAddressSet(name) and is set to AddressSet.Name
	hashName string
	uuid     string
}

// getOvnAddressSetsName returns the name for ovnAddressSets, that contains both ipv4 and ipv6 address sets,
// therefore the name should not include ipFamily information. DbObjectIDs without ipFamily is what is used by
// the AddressSetFactoryIPs functions.
func getOvnAddressSetsName(dbIDs *libovsdbops.DbObjectIDs) string {
	if dbIDs.GetObjectID(libovsdbops.AddressSetIPFamilyKey) != "" {
		dbIDs = dbIDs.RemoveIDs(libovsdbops.AddressSetIPFamilyKey)
	}
	return dbIDs.String()
}

// ovnAddressSets is an abstraction for ipv4 and ipv6 address sets
type ovnAddressSets struct {
	nbClient libovsdbclient.Client
	// name is based on dbIDs without ipFamily
	name string
	ipv4 *ovnAddressSet
	ipv6 *ovnAddressSet
}

// ovnAddressSets implements the AddressSet interface
var _ AddressSetIPs = &ovnAddressSets{}

func (asf *ovnAddressSetFactory) newOvnAddressSets(v4set, v6set *ovnAddressSet, dbIDs *libovsdbops.DbObjectIDs) *ovnAddressSets {
	return &ovnAddressSets{nbClient: asf.nbClient, name: getOvnAddressSetsName(dbIDs), ipv4: v4set, ipv6: v6set}
}

// hash the provided input to make it a valid ovnAddressSet name.
func hashedAddressSet(s string) string {
	return util.HashForOVN(s)
}

func asDetail(as *ovnAddressSet) string {
	return fmt.Sprintf("%s/%s/%s", as.uuid, as.name, as.hashName)
}

func (as *ovnAddressSets) GetASHashNames() (string, string) {
	var ipv4AS string
	var ipv6AS string
	if as.ipv4 != nil {
		ipv4AS = as.ipv4.hashName
	}
	if as.ipv6 != nil {
		ipv6AS = as.ipv6.hashName
	}
	return ipv4AS, ipv6AS
}

func (as *ovnAddressSets) GetName() string {
	return as.name
}

func (as *ovnAddressSets) GetUUIDs() (string, string) {
	var ipv4UUID string
	var ipv6UUID string
	if as.ipv4 != nil {
		ipv4UUID = as.ipv4.uuid
	}
	if as.ipv6 != nil {
		ipv6UUID = as.ipv6.uuid
	}
	return ipv4UUID, ipv6UUID
}

// SetIPs replaces the address set's IP addresses with the given slice.
// NOTE: this function is not thread-safe when when run concurrently with other
// IP add/delete operations.
func (as *ovnAddressSets) SetIPs(ips []net.IP) error {
	var err error

	v4ips, v6ips := splitIPsByFamily(ips)

	if as.ipv6 != nil {
		err = as.ipv6.setIPs(v6ips)
	}
	if as.ipv4 != nil {
		err = errors.Wrapf(err, "%v", as.ipv4.setIPs(v4ips))
	}

	return err
}

// SetCIDRs replaces the address set's CIDR addresses with the given slice.
// NOTE: this function is not thread-safe when when run concurrently with other
// IP add/delete operations.
func (as *ovnAddressSets) SetCIDRs(cidrs []*net.IPNet) error {
	var err error

	v4cidrs, v6cidrs := splitCIDRsByFamily(cidrs)

	if as.ipv6 != nil {
		err = as.ipv6.setCIDRs(v6cidrs)
	}
	if as.ipv4 != nil {
		err = errors.Wrapf(err, "%v", as.ipv4.setCIDRs(v4cidrs))
	}

	return err
}

func (as *ovnAddressSets) GetIPs() ([]string, []string) {
	return as.getAddresses()
}

func (as *ovnAddressSets) GetCIDRs() ([]string, []string) {
	return as.getAddresses()
}

func (as *ovnAddressSets) AddIPs(ips []net.IP) error {
	if len(ips) == 0 {
		return nil
	}
	var ops []ovsdb.Operation
	var err error
	if ops, err = as.AddIPsReturnOps(ips); err != nil {
		return err
	}
	_, err = libovsdbops.TransactAndCheck(as.nbClient, ops)
	if err != nil {
		return fmt.Errorf("failed add ips to address set %s (%v)",
			as.name, err)
	}
	return nil
}

func (as *ovnAddressSets) AddCIDRs(cidrs []*net.IPNet) error {
	if len(cidrs) == 0 {
		return nil
	}
	var ops []ovsdb.Operation
	var err error
	if ops, err = as.AddCIDRsReturnOps(cidrs); err != nil {
		return err
	}
	_, err = libovsdbops.TransactAndCheck(as.nbClient, ops)
	if err != nil {
		return fmt.Errorf("failed add cidrs to address set %s (%v)",
			as.name, err)
	}
	return nil
}

func (as *ovnAddressSets) AddIPsReturnOps(ips []net.IP) ([]ovsdb.Operation, error) {
	var ops []ovsdb.Operation
	var err error
	if len(ips) == 0 {
		return ops, nil
	}

	v4ips, v6ips := splitIPsByFamily(ips)
	var op []ovsdb.Operation
	if as.ipv6 != nil {
		if op, err = as.ipv6.addIPs(v6ips); err != nil {
			return nil, fmt.Errorf("failed to AddIPs to the v6 set: %w", err)
		}
		ops = append(ops, op...)
	}
	if as.ipv4 != nil {
		if op, err = as.ipv4.addIPs(v4ips); err != nil {
			return nil, fmt.Errorf("failed to AddIPs to the v4 set: %w", err)
		}
		ops = append(ops, op...)
	}

	return ops, nil
}

func (as *ovnAddressSets) AddCIDRsReturnOps(cidrs []*net.IPNet) ([]ovsdb.Operation, error) {
	var ops []ovsdb.Operation
	var err error
	if len(cidrs) == 0 {
		return ops, nil
	}

	v4ips, v6ips := splitCIDRsByFamily(cidrs)
	var op []ovsdb.Operation
	if as.ipv6 != nil {
		if op, err = as.ipv6.addCIDRs(v6ips); err != nil {
			return nil, fmt.Errorf("failed to CIDRs to the v6 set: %w", err)
		}
		ops = append(ops, op...)
	}
	if as.ipv4 != nil {
		if op, err = as.ipv4.addCIDRs(v4ips); err != nil {
			return nil, fmt.Errorf("failed to CIDRs to the v4 set: %w", err)
		}
		ops = append(ops, op...)
	}

	return ops, nil
}

func (as *ovnAddressSets) DeleteIPs(ips []net.IP) error {
	if len(ips) == 0 {
		return nil
	}
	var ops []ovsdb.Operation
	var err error
	if ops, err = as.DeleteIPsReturnOps(ips); err != nil {
		return err
	}

	_, err = libovsdbops.TransactAndCheck(as.nbClient, ops)
	if err != nil {
		return fmt.Errorf("failed to delete ips from address set %s (%v)",
			as.name, err)
	}
	return nil
}

func (as *ovnAddressSets) DeleteCIDRs(cidrs []*net.IPNet) error {
	if len(cidrs) == 0 {
		return nil
	}
	var ops []ovsdb.Operation
	var err error
	if ops, err = as.DeleteCIDRsReturnOps(cidrs); err != nil {
		return err
	}

	_, err = libovsdbops.TransactAndCheck(as.nbClient, ops)
	if err != nil {
		return fmt.Errorf("failed to delete cidrs from address set %s (%v)",
			as.name, err)
	}
	return nil
}

func (as *ovnAddressSets) DeleteIPsReturnOps(ips []net.IP) ([]ovsdb.Operation, error) {
	var ops []ovsdb.Operation
	var err error
	if len(ips) == 0 {
		return ops, nil
	}

	v4ips, v6ips := splitIPsByFamily(ips)
	var op []ovsdb.Operation
	if as.ipv6 != nil {
		if op, err = as.ipv6.deleteIPs(v6ips); err != nil {
			return nil, fmt.Errorf("failed to IPs from the v6 set: %w", err)
		}
		ops = append(ops, op...)
	}
	if as.ipv4 != nil {
		if op, err = as.ipv4.deleteIPs(v4ips); err != nil {
			return nil, fmt.Errorf("failed to IPs from the v4 set: %w", err)
		}
		ops = append(ops, op...)
	}
	return ops, nil
}

func (as *ovnAddressSets) DeleteCIDRsReturnOps(cidrs []*net.IPNet) ([]ovsdb.Operation, error) {
	var ops []ovsdb.Operation
	var err error
	if len(cidrs) == 0 {
		return ops, nil
	}

	v4ips, v6ips := splitCIDRsByFamily(cidrs)
	var op []ovsdb.Operation
	if as.ipv6 != nil {
		if op, err = as.ipv6.deleteCIDRs(v6ips); err != nil {
			return nil, fmt.Errorf("failed to CIDRs from the v6 set: %w", err)
		}
		ops = append(ops, op...)
	}
	if as.ipv4 != nil {
		if op, err = as.ipv4.deleteCIDRs(v4ips); err != nil {
			return nil, fmt.Errorf("failed to CIDRs from the v4 set: %w", err)
		}
		ops = append(ops, op...)
	}
	return ops, nil
}

func (as *ovnAddressSets) Destroy() error {
	if as.ipv4 != nil {
		err := as.ipv4.destroy()
		if err != nil {
			return err
		}
	}
	if as.ipv6 != nil {
		err := as.ipv6.destroy()
		if err != nil {
			return err
		}
	}
	return nil
}

func (as *ovnAddressSets) getAddresses() ([]string, []string) {
	var v4 []string
	var v6 []string

	if as.ipv6 != nil {
		v6, _ = as.ipv6.getAddresses()
	}
	if as.ipv4 != nil {
		v4, _ = as.ipv4.getAddresses()
	}

	return v4, v6
}

// setIP updates the given address set in OVN to be only the given IPs, disregarding
// existing state.
func (as *ovnAddressSet) setIPs(ips []net.IP) error {
	uniqIPs := ipsToStringUnique(ips)
	return as.setAddresses(uniqIPs)
}

// setIP updates the given address set in OVN to be only the given IPs, disregarding
// existing state.
func (as *ovnAddressSet) setCIDRs(cidrs []*net.IPNet) error {
	uniqCIDRs := cidrsToStringUnique(cidrs)
	return as.setAddresses(uniqCIDRs)
}

// setIP updates the given address set in OVN to be only the given IPs, disregarding
// existing state.
func (as *ovnAddressSet) setAddresses(addresses []string) error {
	addrset := nbdb.AddressSet{
		UUID:      as.uuid,
		Name:      as.hashName,
		Addresses: addresses,
	}
	err := libovsdbops.UpdateAddressSets(as.nbClient, &addrset)
	if err != nil {
		return fmt.Errorf("failed to update address set IPs %+v: %v", addrset, err)
	}

	return nil
}

// getAddresses gets the IPs of a given address set in OVN from libovsdb cache
func (as *ovnAddressSet) getAddresses() ([]string, error) {
	addrset := &nbdb.AddressSet{
		UUID: as.uuid,
		Name: as.hashName,
	}
	addrset, err := libovsdbops.GetAddressSet(as.nbClient, addrset)
	if err != nil {
		return nil, err
	}

	return addrset.Addresses, nil
}

// addIPs appends the set of IPs to the existing address_set.
func (as *ovnAddressSet) addIPs(ips []net.IP) ([]ovsdb.Operation, error) {
	if len(ips) == 0 {
		return nil, nil
	}
	return as.addAddresses(ipsToStringUnique(ips))
}

// addCIDRs appends the set of CIDRs to the existing address_set.
func (as *ovnAddressSet) addCIDRs(cidrs []*net.IPNet) ([]ovsdb.Operation, error) {
	if len(cidrs) == 0 {
		return nil, nil
	}
	return as.addAddresses(cidrsToStringUnique(cidrs))
}

func (as *ovnAddressSet) addAddresses(addresses []string) ([]ovsdb.Operation, error) {
	if as.hasAddresses(addresses...) {
		return nil, nil
	}
	klog.V(5).Infof("(%s) adding addresses (%s) to address set", asDetail(as), addresses)
	addrset := nbdb.AddressSet{
		UUID: as.uuid,
		Name: as.hashName,
	}
	ops, err := libovsdbops.AddAddressesToAddressSetOps(as.nbClient, nil, &addrset, addresses...)
	if err != nil {
		return nil, fmt.Errorf("failed to add addresses %v to address set %+v: %v", addresses, addrset, err)
	}
	return ops, nil
}

// hasAddresses returns true if an address set contains all given addresses
func (as *ovnAddressSet) hasAddresses(addresses ...string) bool {
	existingAddresses, err := as.getAddresses()
	if err != nil {
		return false
	}
	if len(existingAddresses) == 0 {
		return false
	}
	return sets.NewString(existingAddresses...).HasAll(addresses...)
}

// deleteIPs removes selected IPs from the existing address_set
func (as *ovnAddressSet) deleteIPs(ips []net.IP) ([]ovsdb.Operation, error) {
	if len(ips) == 0 {
		return nil, nil
	}
	return as.deleteAddresses(ipsToStringUnique(ips))

}

// deleteIPs removes selected IPs from the existing address_set
func (as *ovnAddressSet) deleteCIDRs(cidrs []*net.IPNet) ([]ovsdb.Operation, error) {
	if len(cidrs) == 0 {
		return nil, nil
	}
	return as.deleteAddresses(cidrsToStringUnique(cidrs))
}

func (as *ovnAddressSet) deleteAddresses(addresses []string) ([]ovsdb.Operation, error) {
	klog.V(5).Infof("(%s) deleting address %s from address set", asDetail(as), addresses)
	addrset := nbdb.AddressSet{
		UUID: as.uuid,
		Name: as.hashName,
	}
	ops, err := libovsdbops.DeleteAddressesFromAddressSetOps(as.nbClient, nil, &addrset, addresses...)
	if err != nil {
		return nil, fmt.Errorf("failed to delete IPs %v to address set %+v: %v", addresses, addrset, err)
	}

	return ops, nil
}

func (as *ovnAddressSet) destroy() error {
	klog.V(5).Infof("Destroy(%s)", asDetail(as))
	addrset := nbdb.AddressSet{
		UUID: as.uuid,
		Name: as.hashName,
	}
	err := libovsdbops.DeleteAddressSets(as.nbClient, &addrset)
	if err != nil {
		return fmt.Errorf("failed to delete address set %+v: %v", addrset, err)
	}

	return nil
}

func (as *ovnAddressSet) GetUUIDs() (string, error) {
	if as.uuid == "" {
		return "", fmt.Errorf("no UUID is set")
	}
	return as.uuid, nil
}

// splitIPsByFamily takes a slice of IPs and returns two slices, with
// v4 and v6 addresses collated accordingly.
func splitIPsByFamily(ips []net.IP) (v4 []net.IP, v6 []net.IP) {
	for _, ip := range ips {
		if utilnet.IsIPv6(ip) {
			v6 = append(v6, ip)
		} else {
			v4 = append(v4, ip)
		}
	}
	return
}

// splitCIDRsByFamily takes a slice of CIDRs and returns two slices, with
// v4 and v6 addresses collated accordingly.
func splitCIDRsByFamily(cidrs []*net.IPNet) (v4 []*net.IPNet, v6 []*net.IPNet) {
	for _, cidr := range cidrs {
		if utilnet.IsIPv6CIDR(cidr) {
			v6 = append(v6, cidr)
		} else {
			v4 = append(v4, cidr)
		}
	}
	return
}

// Takes a slice of IPs and returns a slice with unique IPs
func ipsToStringUnique(ips []net.IP) []string {
	s := sets.New[string]()
	for _, ip := range ips {
		s.Insert(ip.String())
	}
	return s.UnsortedList()
}

// Takes a slice of IPs and returns a slice with unique IPs
func cidrsToStringUnique(cidrs []*net.IPNet) []string {
	s := sets.New[string]()
	for _, cidr := range cidrs {
		s.Insert(cidr.String())
	}
	return s.UnsortedList()
}
