//go:build !race

package egressip

import (
	"context"
	"fmt"
	"hash/fnv"
	"net"
	"os"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/containernetworking/plugins/pkg/ns"
	"github.com/containernetworking/plugins/pkg/testutils"
	"github.com/ovn-org/ovn-kubernetes/go-controller/pkg/config"
	egressipv1 "github.com/ovn-org/ovn-kubernetes/go-controller/pkg/crd/egressip/v1"
	egressipfake "github.com/ovn-org/ovn-kubernetes/go-controller/pkg/crd/egressip/v1/apis/clientset/versioned/fake"
	"github.com/ovn-org/ovn-kubernetes/go-controller/pkg/factory"
	"github.com/ovn-org/ovn-kubernetes/go-controller/pkg/node/iptables"
	"github.com/ovn-org/ovn-kubernetes/go-controller/pkg/node/linkmanager"
	"github.com/ovn-org/ovn-kubernetes/go-controller/pkg/node/routemanager"
	"github.com/ovn-org/ovn-kubernetes/go-controller/pkg/util"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
	utiliptables "k8s.io/kubernetes/pkg/util/iptables"

	"github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	"github.com/onsi/gomega"
	"github.com/vishvananda/netlink"
)

type inf struct {
	name string
	addr string
}

type node struct {
	name   string
	labels map[string]string
	annots map[string]string
	infs   []inf
}

type egressIPSelected struct {
	egressipv1.EgressIP
	expectedPods     []string
	expectedRoutes   []netlink.Route
	expectedIPTRules []string
	expectedRules    []string
	expectedInf      string
}

const (
	namespace1            = "egressip1"
	namespace2            = "egressip2"
	namespace3            = "egressip3"
	namespace4            = "egressip4"
	node1Name             = "node1"
	node2Name             = "node2"
	egressIP1Name         = "egressip-1"
	egressIP2Name         = "egressip-2"
	egressIP1IP           = "5.5.5.50"
	egressIP2IP           = "5.5.10.55"
	oneSec                = time.Second
	dummyLink1Name        = "dummy1"
	dummyLink2Name        = "dummy2"
	dummyLink3Name        = "dummy3"
	dummyLink4Name        = "dummy4"
	dummy1IPv4CIDR        = "5.5.5.10/24"
	dummy1IPv4CIDRNetwork = "5.5.5.0/24"
	dummy2IPv4CIDR        = "5.5.10.15/24"
	dummy2IPv4CIDRNetwork = "5.5.10.0/24"
	dummy3IPv4CIDR        = "8.8.10.3/16"
	dummy3IPv4CIDRNetwork = "8.8.0.0/16"
	dummy4IPv4CIDR        = "9.8.10.3/16"
	dummy4IPv4CIDRNetwork = "9.8.0.0/16"
	pod1Name              = "testpod1"
	pod1IPv4              = "192.168.100.2"
	pod1IPv4CIDR          = "192.168.100.2/32"
	pod2Name              = "testpod2"
	pod2IPv4              = "192.168.100.7"
	pod2IPv4CIDR          = "192.168.100.7/32"
	pod3Name              = "testpod3"
	pod3IPv4              = "192.168.100.9"
	pod3IPv4CIDR          = "192.168.100.9/32"
	pod4Name              = "testpod4"
	pod4IPv4              = "192.168.100.67"
	pod4IPv4CIDR          = "192.168.100.67/32"
	ovnManagedNetwork     = "11.11.0.0/16"
)

var (
	nodeAnnotations = map[string]string{
		"k8s.ovn.org/node-primary-ifaddr": fmt.Sprintf("{\"ipv4\": \"%s\", \"ipv6\": \"%s\"}", ovnManagedNetwork, ""),
	}
	nodeLabels = map[string]string{
		"k8s.ovn.org/egress-assignable": "",
	}
	egressPodLabel  = map[string]string{"egress": "needed"}
	namespace1Label = map[string]string{"prod1": ""}
	namespace2Label = map[string]string{"prod2": ""}
)

type cleanupFn func() error

func setupTestEnvironment(stopCh chan struct{}, wg *sync.WaitGroup, namespaces []corev1.Namespace, pods []corev1.Pod, egressIPs []egressipv1.EgressIP, node node) (ns.NetNS, *Controller, *util.OVNNodeClientset, cleanupFn, error) {
	testNS, err := testutils.NewNS()
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("failed to setup test environment because failed to create new namespace: %v", err)
	}
	for _, i := range node.infs {
		if i.name == "" || i.addr == "" {
			continue
		}
		err = testNS.Do(func(netNS ns.NetNS) error {
			if err = addInfAndAddr(i.name, i.addr); err != nil {
				return err
			}
			return nil
		})
		if err != nil {
			return nil, nil, nil, nil, err
		}
	}
	kubeClient := fake.NewSimpleClientset(&corev1.NodeList{Items: []corev1.Node{getNodeObj(node.name, node.annots, node.labels)}},
		&corev1.NamespaceList{Items: namespaces}, &corev1.PodList{Items: pods})
	egressIPClient := egressipfake.NewSimpleClientset(&egressipv1.EgressIPList{Items: egressIPs})
	ovnNodeClient := &util.OVNNodeClientset{
		KubeClient:     kubeClient,
		EgressIPClient: egressIPClient,
	}
	rm := routemanager.NewController()
	config.OVNKubernetesFeature.EnableEgressIP = true
	watchFactory, err := factory.NewNodeWatchFactory(ovnNodeClient, node.name)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("failed to setup test environment because failed to create new node watch factory: %v", err)
	}
	if err = watchFactory.Start(); err != nil {
		return nil, nil, nil, nil, fmt.Errorf("failed to setup test environment because failed to start watch factory: %v", err)
	}
	cleanupFn := func() error {
		watchFactory.Shutdown()
		if err := testutils.UnmountNS(testNS); err != nil {
			return err
		}
		return nil
	}

	c, err := NewController(watchFactory.EgressIPInformer(), watchFactory.NodeInformer(), watchFactory.NamespaceInformer(),
		watchFactory.PodCoreInformer(), rm, true, false, node.name)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("failed to setup test environment because failed to create new egress IP controller: %v", err)
	}
	wg.Add(2)
	// we do not call start for our controller because the newly created goroutines will not be set to the correct network namespace,
	// so we invoke them manually here and call reconcile manually
	go testNS.Do(func(netNS ns.NetNS) error {
		c.routeManager.Run(stopCh, 10*time.Millisecond)
		wg.Done()
		return nil
	})

	// normally executed during Run but we call it manually here because run spawns a go routine that we cannot control its netns during test
	err = testNS.Do(func(netNS ns.NetNS) error {
		return c.ruleManager.OwnPriority(ipRulePriority)
	})

	go testNS.Do(func(netNS ns.NetNS) error {
		var err error
		for {
			select {
			case <-stopCh:
				wg.Done()
				return nil
			case <-c.triggerReconcileCh:
				if err = c.reconcile(); err != nil {
					klog.Errorf("Failed to reconcile egress IP: %v", err)
				}
			}
		}
	})
	if err != nil {
		return nil, nil, nil, nil, err
	}

	return testNS, c, ovnNodeClient, cleanupFn, nil
}

func checkForCleanSlate(testNS ns.NetNS, c *Controller, node node) error {
	// check iptables if empty

	err := testNS.Do(func(netNS ns.NetNS) error {
		c.triggerReconcile()
		ruleArgs, err := c.iptablesManager.GetIPv4ChainRuleArgs(utiliptables.TableNAT, multiNICChain)
		if err != nil {
			return err
		}
		if len(ruleArgs) != 0 {
			return fmt.Errorf("expected 0 rules but found %d (%v)", len(ruleArgs), ruleArgs)
		}
		return nil
	})
	if err != nil {
		return err
	}

	// check default routes are removed
	err = testNS.Do(func(netNS ns.NetNS) error {
		for _, inf := range node.infs {
			linkIndex := getLinkIndex(inf.name)
			tableID := getRouteTableID(linkIndex)
			filter, mask := filterRouteByLinkTable(linkIndex, tableID)
			existingRoutes, err := netlink.RouteListFiltered(netlink.FAMILY_ALL, filter, mask)
			if err != nil {
				return fmt.Errorf("failed to list routes for link %s: %v", dummyLink1Name, err)
			}
			if len(existingRoutes) != 0 {
				return fmt.Errorf("expecting no routes at table ID %d but found %d", tableID, len(existingRoutes))
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	// check rules are removed
	err = testNS.Do(func(netNS ns.NetNS) error {
		rules, err := netlink.RuleList(netlink.FAMILY_ALL)
		if err != nil {
			return err
		}
		for _, rule := range rules {
			if rule.Priority != ipRulePriority {
				continue
			}
			return fmt.Errorf("unexpected stale rule found: %s", rule.String())
		}
		return nil
	})
	if err != nil {
		return err
	}

	// check that egress IP aren't assigned to any interface aka stale link address
	err = testNS.Do(func(netNS ns.NetNS) error {
		for _, inf := range node.infs {
			link, err := netlink.LinkByName(inf.name)
			if err != nil {
				return err
			}
			addresses, err := netlink.AddrList(link, netlink.FAMILY_ALL)
			if err != nil {
				return err
			}
			for _, address := range addresses {
				if address.Label == linkmanager.GetAssignedAddressLabel(inf.name) {
					return fmt.Errorf("stale address %s found on interface %s", address.String(), inf.name)
				}
			}
		}
		return nil
	})
	return err
}

var _ = table.DescribeTable("EgressIP selectors",
	func(egressIPSelecteds []egressIPSelected, pods []corev1.Pod, namespaces []corev1.Namespace, node node) {
		defer ginkgo.GinkgoRecover()
		if os.Getenv("NOROOT") == "TRUE" {
			ginkgo.Skip("Test requires root privileges")
		}
		//	var node2IPv4 = "5.5.5.20"
		stopCh := make(chan struct{}, 1)
		var wg sync.WaitGroup
		// construct test dependencies

		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		var err error
		egressIPList := make([]egressipv1.EgressIP, 0)
		for _, egressIPSelected := range egressIPSelecteds {
			egressIPList = append(egressIPList, egressIPSelected.EgressIP)
		}
		testNS, c, _, cleanupFn, err := setupTestEnvironment(stopCh, &wg, namespaces, pods, egressIPList, node)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
		defer gomega.Expect(cleanupFn()).Should(gomega.Succeed())

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
				gomega.PanicWith(fmt.Sprintf("timed out waiting for %q caches to sync", resource.name))
			}
		}

		// Ensure only the iptables rules we expect are present on the chain
		gomega.Eventually(func() error {
			var foundIPTRules []iptables.RuleArg
			err := testNS.Do(func(netNS ns.NetNS) error {
				c.triggerReconcile()
				foundIPTRules, err = c.iptablesManager.GetIPv4ChainRuleArgs(utiliptables.TableNAT, multiNICChain)
				return err
			})
			if err != nil {
				return err
			}
			// refator all this to account for new egress selected refactor
			// since iptables rules found must be unique and since each expected rule is also unique, it is sufficient
			// to say they are equal by comparing their length and ensuring we find all expected rules
			var ruleCount int
			for _, egressIPSelected := range egressIPSelecteds {
				ruleCount += len(egressIPSelected.expectedIPTRules)
			}

			if ruleCount != len(foundIPTRules) {
				return fmt.Errorf("expected and found rule count do not match: expected %d rules but got %d", ruleCount,
					len(foundIPTRules))
			}

			for _, egressIPSelected := range egressIPSelecteds {
				for _, expectedIPTRule := range egressIPSelected.expectedIPTRules {
					var found bool
					for _, foundIPTRule := range foundIPTRules {
						if expectedIPTRule == strings.Join(foundIPTRule.Args, " ") {
							found = true
							break
						}
					}
					if !found {
						return fmt.Errorf("failed to find expected rule %s", expectedIPTRule)
					}
				}
			}
			return nil
		}).WithTimeout(oneSec).Should(gomega.Succeed())

		// Ensure only the routes we expect are present in a specific route table
		gomega.Eventually(func() error {
			return testNS.Do(func(netNS ns.NetNS) error {
				// loop over egress IPs
				for _, egressIPSelected := range egressIPSelecteds {
					if egressIPSelected.Status.Items[0].Node != node1Name {
						continue
					}
					expectedInf := getExpectedInterfaceForEIP(egressIPSelected.Status.Items[0].EgressIP)
					filter, mask := filterRouteByLinkTable(getLinkIndex(expectedInf), getRouteTableID(getLinkIndex(expectedInf)))
					existingRoutes, err := netlink.RouteListFiltered(netlink.FAMILY_ALL, filter, mask)
					if err != nil {
						return fmt.Errorf("failed to list routes for link %s: %v", expectedInf, err)
					}
					if len(egressIPSelected.expectedPods) > 0 {
						if len(existingRoutes) == 0 {
							return fmt.Errorf("expected route but failed to find any")
						}
					}
					// expecting only one entry
					for _, existingRoute := range existingRoutes {
						expectedRoute := getExpectedDefaultRoute(getLinkIndex(dummyLink1Name))
						if expectedRoute.Dst != existingRoute.Dst {
							return fmt.Errorf("failed to find route (%s) but found route with unexpected destination %q", getExpectedDefaultRoute(getLinkIndex(dummyLink1Name)), existingRoutes[0].String())
						}
					}
				}
				return nil
			})
		}).WithTimeout(oneSec).Should(gomega.Succeed())

		// verify the correct rules are present and also no rule entries we do not expect
		egressIPRules := make(map[string][]*netlink.Rule, 0)
		for _, egressIPSelected := range egressIPSelecteds {
			var expectedRules []*netlink.Rule
			if egressIPSelected.Status.Items[0].Node != node1Name {
				continue
			}
			for _, expectedPod := range egressIPSelected.expectedPods {
				pod := getPod(pods, expectedPod)
				if pod.Spec.NodeName != node1Name {
					continue
				}
				cidr := pod.Status.PodIP + "/32"
				ip, ipNet, _ := net.ParseCIDR(cidr)
				ipNet.IP = ip
				r := netlink.NewRule()
				r.Priority = ipRulePriority
				r.Src = ipNet
				r.Table = getRouteTableID(getLinkIndex(egressIPSelected.expectedInf))
				expectedRules = append(expectedRules, r)
			}
			egressIPRules[egressIPSelected.Name] = expectedRules
		}
		// verify expected versus found
		gomega.Eventually(func() error {
			return testNS.Do(func(netNS ns.NetNS) error {
				for _, egressIPSelected := range egressIPSelecteds {
					if egressIPSelected.Status.Items[0].Node != node1Name {
						continue
					}
					foundRules, err := netlink.RuleList(netlink.FAMILY_ALL)
					if err != nil {
						return err
					}
					temp := foundRules[:0]
					for _, rule := range foundRules {
						if rule.Priority == ipRulePriority {
							temp = append(temp, rule)
						}
					}
					foundRules = temp
					expectedRules := egressIPRules[egressIPSelected.Name]
					if len(foundRules) != len(expectedRules) {
						return fmt.Errorf("failed to the correct number of rules (expected %d, but got %d)", len(expectedRules), len(foundRules))
					}

					var found bool
					for _, foundRule := range foundRules {
						found = false
						for _, expectedRule := range expectedRules {
							if expectedRule.Src.IP.Equal(foundRule.Src.IP) {
								if expectedRule.Table == foundRule.Table {
									found = true
								}
							}
						}
						if !found {
							return fmt.Errorf("unexpected rule found: %v", foundRule)
						}
					}
				}
				return nil
			})
		})

		// Ensure no stale routes exist in any possible tables we may own
		// For testing, the link index is derived from the name of the device with an upper bound of 205 and lower bound of 5
		gomega.Eventually(func() error {
			return testNS.Do(func(netNS ns.NetNS) error {
				for i := 5; i < 250; i++ {
					// skip route table IDs where we have configured routes
					var skip bool
					for _, inf := range node.infs {
						if getLinkIndex(inf.name) == i {
							skip = true
						}
					}
					if skip {
						continue
					}
					for _, inf := range node.infs {
						filter, mask := filterRouteByLinkTable(getLinkIndex(inf.name), i)
						existingRoutes, err := netlink.RouteListFiltered(netlink.FAMILY_ALL, filter, mask)
						if err != nil {
							return fmt.Errorf("failed to list routes for link %s: %v", dummyLink1Name, err)
						}
						if len(existingRoutes) != 0 {
							return fmt.Errorf("expecting no routes at table ID %d but found %d", i, len(existingRoutes))
						}
					}
				}
				return nil
			})
		}).WithTimeout(oneSec).Should(gomega.Succeed())
		close(stopCh)
		wg.Wait()
	},
	table.Entry("configures nothing when EIPs dont select anything",
		[]egressIPSelected{
			{
				newEgressIP(egressIP1Name, egressIP1IP, dummy1IPv4CIDRNetwork, node1Name, namespace1Label, egressPodLabel),
				[]string{},
				[]netlink.Route{},
				[]string{},
				[]string{},
				"",
			},
		},
		[]corev1.Pod{newPodWithLabels(namespace1, pod1Name, node1Name, pod1IPv4, map[string]string{})},
		[]corev1.Namespace{newNamespaceWithLabels(namespace1, namespace1Label)},
		node{
			node1Name,
			nodeLabels,
			nodeAnnotations,
			[]inf{{dummyLink1Name, dummy1IPv4CIDR}, {dummyLink2Name, dummy2IPv4CIDR}},
		},
	),
	table.Entry("configures one EIP and one Pod",
		[]egressIPSelected{
			{
				newEgressIP(egressIP1Name, egressIP1IP, dummy1IPv4CIDRNetwork, node1Name, namespace1Label, egressPodLabel),
				[]string{pod1Name},
				[]netlink.Route{getExpectedDefaultRoute(getLinkIndex(dummyLink1Name))},
				[]string{getExpectedIPTableMasqRule(pod1IPv4CIDR, dummyLink1Name, egressIP1IP)},
				[]string{getExpectedRule(pod1IPv4, getRouteTableID(getLinkIndex(dummyLink1Name)))},
				dummyLink1Name,
			},
		},
		[]corev1.Pod{newPodWithLabels(namespace1, pod1Name, node1Name, pod1IPv4, egressPodLabel)},
		[]corev1.Namespace{newNamespaceWithLabels(namespace1, namespace1Label)},
		node{
			node1Name,
			nodeLabels,
			nodeAnnotations,
			[]inf{{dummyLink1Name, dummy1IPv4CIDR}, {dummyLink2Name, dummy2IPv4CIDR}},
		},
	),
	table.Entry("configures one EIP and multiple pods",
		// Test pod and namespace selection -
		[]egressIPSelected{
			{
				newEgressIP(egressIP1Name, egressIP1IP, dummy1IPv4CIDRNetwork, node1Name, namespace1Label, egressPodLabel),
				[]string{pod1Name, pod2Name},
				[]netlink.Route{getExpectedDefaultRoute(getLinkIndex(dummyLink1Name))},
				[]string{getExpectedIPTableMasqRule(pod1IPv4CIDR, dummyLink1Name, egressIP1IP),
					getExpectedIPTableMasqRule(pod2IPv4CIDR, dummyLink1Name, egressIP1IP)},
				[]string{getExpectedRule(pod1IPv4, getRouteTableID(getLinkIndex(dummyLink1Name))),
					getExpectedRule(pod2IPv4, getRouteTableID(getLinkIndex(dummyLink1Name)))},
				dummyLink1Name,
			},
		},
		[]corev1.Pod{newPodWithLabels(namespace1, pod1Name, node1Name, pod1IPv4, egressPodLabel),
			newPodWithLabels(namespace1, pod2Name, node1Name, pod2IPv4, egressPodLabel),
			newPodWithLabels(namespace1, pod3Name, node1Name, pod3IPv4, map[string]string{})},
		[]corev1.Namespace{newNamespaceWithLabels(namespace1, namespace1Label), newNamespaceWithLabels(namespace2, map[string]string{})},
		node{
			node1Name,
			nodeLabels,
			nodeAnnotations,
			[]inf{{dummyLink1Name, dummy1IPv4CIDR}, {dummyLink2Name, dummy2IPv4CIDR}},
		},
	),
	table.Entry("configures one EIP and multiple namespaces and multiple pods",
		[]egressIPSelected{
			{
				newEgressIP(egressIP1Name, egressIP1IP, dummy1IPv4CIDRNetwork, node1Name, namespace1Label, egressPodLabel),
				[]string{pod1Name, pod2Name},
				[]netlink.Route{getExpectedDefaultRoute(getLinkIndex(dummyLink1Name))},
				[]string{getExpectedIPTableMasqRule(pod1IPv4CIDR, dummyLink1Name, egressIP1IP),
					getExpectedIPTableMasqRule(pod2IPv4CIDR, dummyLink1Name, egressIP1IP)},
				[]string{getExpectedRule(pod1IPv4, getRouteTableID(getLinkIndex(dummyLink1Name))),
					getExpectedRule(pod2IPv4, getRouteTableID(getLinkIndex(dummyLink1Name)))},
				dummyLink1Name,
			},
		},
		[]corev1.Pod{newPodWithLabels(namespace1, pod1Name, node1Name, pod1IPv4, egressPodLabel),
			newPodWithLabels(namespace2, pod2Name, node1Name, pod2IPv4, egressPodLabel),
			newPodWithLabels(namespace2, pod3Name, node1Name, pod3IPv4, map[string]string{}),
			newPodWithLabels(namespace3, pod4Name, node1Name, pod4IPv4, egressPodLabel)},
		[]corev1.Namespace{newNamespaceWithLabels(namespace1, namespace1Label), newNamespaceWithLabels(namespace2, namespace1Label),
			newNamespaceWithLabels(namespace3, map[string]string{})},
		node{
			node1Name,
			nodeLabels,
			nodeAnnotations,
			[]inf{{dummyLink1Name, dummy1IPv4CIDR}, {dummyLink2Name, dummy2IPv4CIDR}},
		},
	),
	table.Entry("configures one EIP and multiple namespaces and multiple pods",
		[]egressIPSelected{
			{
				newEgressIP(egressIP1Name, egressIP1IP, dummy1IPv4CIDRNetwork, node1Name, namespace1Label, egressPodLabel),
				[]string{pod1Name, pod2Name},
				[]netlink.Route{getExpectedDefaultRoute(getLinkIndex(dummyLink1Name))},
				[]string{getExpectedIPTableMasqRule(pod1IPv4CIDR, dummyLink1Name, egressIP1IP),
					getExpectedIPTableMasqRule(pod2IPv4CIDR, dummyLink1Name, egressIP1IP)},
				[]string{getExpectedRule(pod1IPv4, getRouteTableID(getLinkIndex(dummyLink1Name))),
					getExpectedRule(pod2IPv4, getRouteTableID(getLinkIndex(dummyLink1Name)))},
				dummyLink1Name,
			},
		},
		[]corev1.Pod{newPodWithLabels(namespace1, pod1Name, node1Name, pod1IPv4, egressPodLabel),
			newPodWithLabels(namespace2, pod2Name, node1Name, pod2IPv4, egressPodLabel),
			newPodWithLabels(namespace2, pod3Name, node1Name, pod3IPv4, map[string]string{}),
			newPodWithLabels(namespace3, pod4Name, node1Name, pod4IPv4, egressPodLabel)},
		[]corev1.Namespace{newNamespaceWithLabels(namespace1, namespace1Label), newNamespaceWithLabels(namespace2, namespace1Label),
			newNamespaceWithLabels(namespace3, map[string]string{})},
		node{
			node1Name,
			nodeLabels,
			nodeAnnotations,
			[]inf{{dummyLink1Name, dummy1IPv4CIDR}, {dummyLink2Name, dummy2IPv4CIDR}},
		},
	),
	table.Entry("configures multiple EIPs and multiple namespaces and multiple pods",
		[]egressIPSelected{
			{
				newEgressIP(egressIP1Name, egressIP1IP, dummy1IPv4CIDRNetwork, node1Name, namespace1Label, egressPodLabel),
				[]string{pod1Name, pod2Name},
				[]netlink.Route{getExpectedDefaultRoute(getLinkIndex(dummyLink1Name))},
				[]string{getExpectedIPTableMasqRule(pod1IPv4CIDR, dummyLink1Name, egressIP1IP),
					getExpectedIPTableMasqRule(pod2IPv4CIDR, dummyLink1Name, egressIP1IP)},
				[]string{getExpectedRule(pod1IPv4, getRouteTableID(getLinkIndex(dummyLink1Name))),
					getExpectedRule(pod2IPv4, getRouteTableID(getLinkIndex(dummyLink1Name)))},
				dummyLink1Name,
			},
			{
				newEgressIP(egressIP2Name, egressIP2IP, dummy2IPv4CIDRNetwork, node1Name, namespace2Label, egressPodLabel),
				[]string{pod3Name, pod4Name},
				[]netlink.Route{getExpectedDefaultRoute(getLinkIndex(dummyLink2Name))},
				[]string{getExpectedIPTableMasqRule(pod3IPv4CIDR, dummyLink2Name, egressIP2IP),
					getExpectedIPTableMasqRule(pod4IPv4CIDR, dummyLink2Name, egressIP2IP)},
				[]string{getExpectedRule(pod3IPv4, getRouteTableID(getLinkIndex(dummyLink2Name))),
					getExpectedRule(pod4IPv4, getRouteTableID(getLinkIndex(dummyLink2Name)))},
				dummyLink2Name,
			},
		},
		[]corev1.Pod{newPodWithLabels(namespace1, pod1Name, node1Name, pod1IPv4, egressPodLabel),
			newPodWithLabels(namespace2, pod2Name, node1Name, pod2IPv4, egressPodLabel),
			newPodWithLabels(namespace3, pod3Name, node1Name, pod3IPv4, egressPodLabel),
			newPodWithLabels(namespace4, pod4Name, node1Name, pod4IPv4, egressPodLabel)},
		[]corev1.Namespace{newNamespaceWithLabels(namespace1, namespace1Label), newNamespaceWithLabels(namespace2, namespace1Label),
			newNamespaceWithLabels(namespace3, namespace2Label), newNamespaceWithLabels(namespace4, namespace2Label)},
		node{
			node1Name,
			nodeLabels,
			nodeAnnotations,
			[]inf{{dummyLink1Name, dummy1IPv4CIDR}, {dummyLink2Name, dummy2IPv4CIDR},
				{dummyLink3Name, dummy3IPv4CIDR}, {dummyLink4Name, dummy4IPv4CIDR}},
		},
	),
)

var _ = ginkgo.Describe("Reconcile", func() {
	defer ginkgo.GinkgoRecover()
	if os.Getenv("NOROOT") == "TRUE" {
		ginkgo.Skip("Test requires root privileges")
	}
	var netNS ns.NetNS
	var c *Controller
	var ovnClient *util.OVNNodeClientset
	var cleanupFn cleanupFn
	var err error
	var wg *sync.WaitGroup
	var stopCh chan struct{}
	var n node

	ginkgo.BeforeEach(func() {
		runtime.LockOSThread()
		stopCh = make(chan struct{}, 0)
		wg = &sync.WaitGroup{}
		ns := newNamespaceWithLabels(namespace1, namespace1Label)
		n = node{node1Name, nodeLabels, nodeAnnotations, []inf{{dummyLink1Name, dummy1IPv4CIDR}}}
		pod := newPodWithLabels(namespace1, pod1Name, node1Name, pod1IPv4, egressPodLabel)
		netNS, c, ovnClient, cleanupFn, err = setupTestEnvironment(stopCh, wg, []corev1.Namespace{ns}, []corev1.Pod{pod}, []egressipv1.EgressIP{
			newEgressIP(egressIP1Name, egressIP1IP, dummy1IPv4CIDRNetwork, node1Name, namespace1Label, egressPodLabel)}, n)
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
				panic(fmt.Sprintf("timed out waiting for %q caches to sync", resource.name))
			}
		}
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
	})

	ginkgo.AfterEach(func() {
		close(stopCh)
		defer runtime.UnlockOSThread()
		wg.Wait()
		gomega.Expect(cleanupFn()).Should(gomega.Succeed())
	})

	ginkgo.Context("nodes", func() {
		ginkgo.It("removes config when node is no longer labeled egress-able", func() {
			// Cluster manager should then reassign the egress IP but here we do it manually
			egressIP, err := ovnClient.EgressIPClient.K8sV1().EgressIPs().Get(context.TODO(), egressIP1Name, metav1.GetOptions{})
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			egressIP = egressIP.DeepCopy()
			egressIP.Status.Items[0].Node = node2Name
			egressIP, err = ovnClient.EgressIPClient.K8sV1().EgressIPs().Update(context.TODO(), egressIP, metav1.UpdateOptions{})
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			gomega.Expect(egressIP.Status.Items[0].Node).Should(gomega.Equal(node2Name))
			gomega.Eventually(func() error {
				if err := checkForCleanSlate(netNS, c, n); err != nil {
					return err
				}
				return nil
			}).WithTimeout(oneSec).Should(gomega.Succeed())
		})
	})
})

func newPodWithLabels(namespace, name, node, podIP string, additionalLabels map[string]string) corev1.Pod {
	podIPs := []corev1.PodIP{}
	if podIP != "" {
		podIPs = append(podIPs, corev1.PodIP{IP: podIP})
	}
	return corev1.Pod{
		ObjectMeta: newPodMeta(namespace, name, additionalLabels),
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "containerName",
					Image: "containerImage",
				},
			},
			NodeName: node,
		},
		Status: corev1.PodStatus{
			Phase:  corev1.PodRunning,
			PodIP:  podIP,
			PodIPs: podIPs,
		},
	}
}

func newPodMeta(namespace, name string, additionalLabels map[string]string) metav1.ObjectMeta {
	labels := map[string]string{
		"name": name,
	}
	for k, v := range additionalLabels {
		labels[k] = v
	}
	return metav1.ObjectMeta{
		Name:      name,
		UID:       types.UID(name),
		Namespace: namespace,
		Labels:    labels,
	}
}

func newNamespaceWithLabels(namespace string, additionalLabels map[string]string) corev1.Namespace {
	return corev1.Namespace{
		ObjectMeta: newNamespaceMeta(namespace, additionalLabels),
		Spec:       corev1.NamespaceSpec{},
		Status:     corev1.NamespaceStatus{},
	}
}

func newNamespace(namespace string) corev1.Namespace {
	return corev1.Namespace{
		ObjectMeta: newNamespaceMeta(namespace, nil),
		Spec:       corev1.NamespaceSpec{},
		Status:     corev1.NamespaceStatus{},
	}
}

func newNamespaceMeta(namespace string, additionalLabels map[string]string) metav1.ObjectMeta {
	labels := map[string]string{
		"name": namespace,
	}
	for k, v := range additionalLabels {
		labels[k] = v
	}
	return metav1.ObjectMeta{
		UID:         types.UID(namespace),
		Name:        namespace,
		Labels:      labels,
		Annotations: map[string]string{},
	}
}

func getNodeObj(nodeName string, annotations, labels map[string]string) corev1.Node {
	return corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name:        nodeName,
			Annotations: annotations,
			Labels:      labels,
		},
		Status: corev1.NodeStatus{
			Conditions: []corev1.NodeCondition{
				{
					Type:   corev1.NodeReady,
					Status: corev1.ConditionTrue,
				},
			},
		},
	}
}

func newEgressIPMeta(name string) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		UID:  types.UID(name),
		Name: name,
		Labels: map[string]string{
			"name": name,
		},
	}
}

func newEgressIP(name, ip, network, node string, namespaceLabels, podLabels map[string]string) egressipv1.EgressIP {
	_, ipnet, err := net.ParseCIDR(network)
	if err != nil {
		panic(err.Error())
	}
	return egressipv1.EgressIP{
		ObjectMeta: newEgressIPMeta(name),
		Spec: egressipv1.EgressIPSpec{
			EgressIPs: []string{ip},
			PodSelector: metav1.LabelSelector{
				MatchLabels: podLabels,
			},
			NamespaceSelector: metav1.LabelSelector{
				MatchLabels: namespaceLabels,
			},
		},
		Status: egressipv1.EgressIPStatus{
			Items: []egressipv1.EgressIPStatusItem{{
				node,
				ip,
				ipnet.String(),
			}},
		},
	}
}

func validateChainJump(ipt *iptables.Controller, chain utiliptables.Chain, jumpChain string) (bool, error) {
	ruleArgs, err := ipt.GetIPv4ChainRuleArgs(utiliptables.TableNAT, multiNICChain)
	if err != nil {
		return false, err
	}
	expectedArgs := []string{"-A", string(chain), "-j", jumpChain}

	for _, ruleArg := range ruleArgs {
		if reflect.DeepEqual(ruleArg.Args, expectedArgs) {
			return true, nil
		}
	}
	return false, fmt.Errorf("failed to find IP masq rule")
}

func validateEgressIPMasqueradeRule(ipt *iptables.Controller, chain iptables.Chain, podIP, egressIP string) (bool, error) {
	ruleArgs, err := ipt.GetIPv4ChainRuleArgs(utiliptables.TableNAT, multiNICChain)
	if err != nil {
		return false, err
	}
	expectedArgs := []string{"-A", string(chain.Chain), "-s", podIP, "-j", "SNAT", "--to", egressIP}
	for _, ruleArg := range ruleArgs {
		if reflect.DeepEqual(ruleArg.Args, expectedArgs) {
			return true, nil
		}
	}
	return false, fmt.Errorf("failed to find IP masq rule")
}

var index = 5

func addInfAndAddr(name string, address string) error {
	index += 1
	dummy := &netlink.Dummy{LinkAttrs: netlink.LinkAttrs{
		Index:        getLinkIndex(name),
		MTU:          1500,
		Name:         name,
		HardwareAddr: util.IPAddrToHWAddr(net.ParseIP(address)),
	}}
	if err := netlink.LinkAdd(dummy); err != nil {
		return fmt.Errorf("failed to add dummy link %q: %v", dummy.Name, err)
	}
	link, err := netlink.LinkByName(name)
	if err != nil {
		return err
	}
	if err = netlink.LinkSetUp(link); err != nil {
		return err
	}
	ip, ipNet, err := net.ParseCIDR(address)
	if err != nil {
		return err
	}
	ipNet.IP = ip
	addr := &netlink.Addr{IPNet: ipNet, LinkIndex: dummy.Index, Scope: int(netlink.SCOPE_UNIVERSE)}
	return netlink.AddrAdd(dummy, addr)
}

func getExpectedIPTableMasqRule(podIP, egressInterfaceName, snatIP string) string {
	return fmt.Sprintf("-s %s -o %s -j SNAT --to-source %s", podIP, egressInterfaceName, snatIP)
}

func getExpectedDefaultRoute(linkIndex int) netlink.Route {
	return netlink.Route{LinkIndex: linkIndex, Table: getRouteTableID(linkIndex)}
}

func getExpectedRule(podIP string, tableID int) string {
	return fmt.Sprintf("from %s lookup %d", podIP, tableID)
}

func getLinkIndex(linkName string) int {
	return hash(linkName)%200 + 5
}

func hash(s string) int {
	h := fnv.New32a()
	h.Write([]byte(s))
	return int(h.Sum32())
}

func getExpectedRouteRule(linkIndex int) netlink.Route {
	_, defaultCIDRIPNet, _ := net.ParseCIDR("0.0.0.0/0")
	return netlink.Route{
		LinkIndex: linkIndex,
		Dst:       defaultCIDRIPNet,
		Table:     getRouteTableID(linkIndex),
	}
}

func getExpectedInterfaceForEIP(eip string) string {
	ip := net.ParseIP(eip)
	_, dummy1IPNet, _ := net.ParseCIDR(dummy1IPv4CIDR)
	if dummy1IPNet.Contains(ip) {
		return dummyLink1Name
	}
	_, dummy2IPNet, _ := net.ParseCIDR(dummy2IPv4CIDR)
	if dummy2IPNet.Contains(ip) {
		return dummyLink2Name
	}
	panic("failed to find network or interface for EIP " + eip)
}

func getPod(pods []corev1.Pod, name string) corev1.Pod {
	for _, pod := range pods {
		if pod.Name == name {
			return pod
		}
	}
	panic("failed to find a pod")
}
