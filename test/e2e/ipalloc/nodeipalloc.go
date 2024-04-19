package ipalloc

import (
	"context"
	"fmt"
	"github.com/ovn-org/ovn-kubernetes/go-controller/pkg/util"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"math/big"
	"net"
	"sync"
)

type network struct {
	net *net.IPNet
	// base is a cached version of the start IP in the CIDR range as a *big.Int
	base *big.Int
	// max is the maximum size of the usable addresses in the range
	max   int
	count int
}

func (n *network) allocateNext() (net.IP, error) {
	if n.count >= n.max {
		return net.IP{}, fmt.Errorf("limit of %d reached", n.max)
	}
	n.base.Add(n.base, big.NewInt(1))
	n.count += 1
	b := n.base.Bytes()
	b = append(make([]byte, 16), b...)
	return b[len(b)-16:], nil
}

type ipAllocator struct {
	mu      *sync.Mutex
	network *network
}

func newIPAllocation(cidr *net.IPNet) ipAllocator {
	return ipAllocator{&sync.Mutex{}, &network{net: cidr, base: getBaseInt(cidr.IP), max: limit(cidr)}}
}

func (ia ipAllocator) AllocateNext() (net.IP, error) {
	ia.mu.Lock()
	defer ia.mu.Unlock()
	return ia.network.allocateNext()
}

func getBaseInt(ip net.IP) *big.Int {
	return big.NewInt(0).SetBytes(ip.To16())
}

func limit(subnet *net.IPNet) int {
	ones, bits := subnet.Mask.Size()
	if bits == 32 && (bits-ones) >= 31 || bits == 128 && (bits-ones) >= 127 {
		return 0
	}
	// limit to 2^8 (256) IPs for e2es
	if bits == 128 && (bits-ones) >= 8 {
		return int(1) << uint(8)
	}
	return int(1) << uint(bits-ones)
}

// NodeIPAllocator attempts to allocate an IP in the same subnet as a node which also check if its already allocated on any node
type NodeIPAllocator struct {
	ipAllocV4  ipAllocator
	ipAllocV6  ipAllocator
	nodeClient v1.NodeInterface
}

func NewNodePrimaryIPAllocator(nodeClient v1.NodeInterface, nodeName string) (NodeIPAllocator, error) {
	nIPAlloc := NodeIPAllocator{nodeClient: nodeClient}
	node, err := nodeClient.Get(context.TODO(), nodeName, metav1.GetOptions{})
	if err != nil {
		return nIPAlloc, fmt.Errorf("failed to get node %s: %v", nodeName, err)
	}
	nodePrimaryIPs, err := util.ParseNodePrimaryIfAddr(node)
	if nodePrimaryIPs.V4.Net != nil {
		nIPAlloc.ipAllocV4 = newIPAllocation(nodePrimaryIPs.V4.Net)
	}
	if nodePrimaryIPs.V6.Net != nil {
		nIPAlloc.ipAllocV6 = newIPAllocation(nodePrimaryIPs.V6.Net)
	}
	return nIPAlloc, nil
}

func (nia NodeIPAllocator) AllocateNextV4() (net.IP, error) {
	if nia.ipAllocV4.network == nil {
		return nil, fmt.Errorf("ipv4 network is not set")
	}
	return allocateNonConflictingIP(nia.nodeClient, nia.ipAllocV4.AllocateNext)
}

func (nia NodeIPAllocator) AllocateNextV6() (net.IP, error) {
	if nia.ipAllocV6.network == nil {
		return nil, fmt.Errorf("ipv6 network is not set")
	}
	return allocateNonConflictingIP(nia.nodeClient, nia.ipAllocV6.AllocateNext)
}

type allocNextFn func() (net.IP, error)

func allocateNonConflictingIP(nodeClient v1.NodeInterface, allocateFn allocNextFn) (net.IP, error) {
	nodeList, err := nodeClient.List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list nodes: %v", err)
	}
	for {
		nextIP, err := allocateFn()
		if err != nil {
			return nil, fmt.Errorf("failed to allocated next IPv4 address")
		}
		isConflict, err := isConflictWithExistingHostIPs(nodeList.Items, nextIP)
		if err != nil {
			return nil, fmt.Errorf("failed to determine if IP conflicts with existing IPs")
		}
		if !isConflict {
			return nextIP, nil
		}
	}
}

func isConflictWithExistingHostIPs(nodes []corev1.Node, ip net.IP) (bool, error) {
	ipStr := ip.String()
	for _, node := range nodes {
		nodeIPsSet, err := util.ParseNodeHostAddressesDropNetMask(&node)
		if err != nil {
			return false, fmt.Errorf("failed to parse node %s primary annotation info: %v", node.Name, err)
		}
		if nodeIPsSet.Has(ipStr) {
			return true, nil
		}
	}
	return false, nil
}
