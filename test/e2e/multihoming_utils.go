package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/rand"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/kubernetes/test/e2e/framework"
	e2epod "k8s.io/kubernetes/test/e2e/framework/pod"
	"net"
	"strings"

	mnpapi "github.com/k8snetworkplumbingwg/multi-networkpolicy/pkg/apis/k8s.cni.cncf.io/v1beta1"
	nadapi "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
)

func netCIDR(netCIDR string, netPrefixLengthPerNode int) string {
	return fmt.Sprintf("%s/%d", netCIDR, netPrefixLengthPerNode)
}

func getNetCIDRSubnet(netCIDR string) (string, error) {
	subStrings := strings.Split(netCIDR, "/")
	if len(subStrings) == 3 {
		return subStrings[0] + "/" + subStrings[1], nil
	} else if len(subStrings) == 2 {
		return netCIDR, nil
	}
	return "", fmt.Errorf("invalid network cidr %s", netCIDR)
}

type networkAttachmentConfig struct {
	cidr         string
	excludeCIDRs []string
	namespace    string
	name         string
	topology     string
	networkName  string
	vlanID       int
	isIPv6       bool
}

func (nac networkAttachmentConfig) attachmentName() string {
	if nac.networkName != "" {
		return nac.networkName
	}
	return uniqueNadName(nac.name)
}

func uniqueNadName(originalNetName string) string {
	const randomStringLength = 5
	return fmt.Sprintf("%s_%s", rand.String(randomStringLength), originalNetName)
}

func generateNAD(config networkAttachmentConfig) *nadapi.NetworkAttachmentDefinition {
	nadSpec := fmt.Sprintf(
		`
{
        "cniVersion": "0.3.0",
        "name": %q,
        "type": "ovn-k8s-cni-overlay",
        "topology":%q,
        "subnets": %q,
        "excludeSubnets": %q,
        "mtu": 1300,
        "netAttachDefName": %q,
        "vlanID": %d
}
`,
		config.attachmentName(),
		config.topology,
		config.cidr,
		strings.Join(config.excludeCIDRs, ","),
		namespacedName(config.namespace, config.name),
		config.vlanID,
	)
	return generateNetAttachDef(config.namespace, config.name, nadSpec)
}

func generateNetAttachDef(namespace, nadName, nadSpec string) *nadapi.NetworkAttachmentDefinition {
	return &nadapi.NetworkAttachmentDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nadName,
			Namespace: namespace,
		},
		Spec: nadapi.NetworkAttachmentDefinitionSpec{Config: nadSpec},
	}
}

type podConfiguration struct {
	attachments            []nadapi.NetworkSelectionElement
	containerCmd           []string
	name                   string
	namespace              string
	nodeSelector           map[string]string
	isPrivileged           bool
	labels                 map[string]string
	requiresExtraNamespace bool
}

func generatePodSpec(config podConfiguration) *v1.Pod {
	podSpec := e2epod.NewAgnhostPod(config.namespace, config.name, nil, nil, nil, config.containerCmd...)
	podSpec.Annotations = networkSelectionElements(config.attachments...)
	podSpec.Spec.NodeSelector = config.nodeSelector
	podSpec.Labels = config.labels
	if config.isPrivileged {
		privileged := true
		podSpec.Spec.Containers[0].SecurityContext.Privileged = &privileged
	}
	return podSpec
}

func networkSelectionElements(elements ...nadapi.NetworkSelectionElement) map[string]string {
	marshalledElements, err := json.Marshal(elements)
	if err != nil {
		panic(fmt.Errorf("programmer error: you've provided wrong input to the test data: %v", err))
	}
	return map[string]string{
		nadapi.NetworkAttachmentAnnot: string(marshalledElements),
	}
}

func httpServerContainerCmd(port uint16) []string {
	return []string{"netexec", "--http-port", fmt.Sprintf("%d", port)}
}

func podNetworkStatus(pod *v1.Pod, predicates ...func(nadapi.NetworkStatus) bool) ([]nadapi.NetworkStatus, error) {
	podNetStatus, found := pod.Annotations[nadapi.NetworkStatusAnnot]
	if !found {
		return nil, fmt.Errorf("the pod must feature the `networks-status` annotation")
	}

	var netStatus []nadapi.NetworkStatus
	if err := json.Unmarshal([]byte(podNetStatus), &netStatus); err != nil {
		return nil, err
	}

	var netStatusMeetingPredicates []nadapi.NetworkStatus
	for i := range netStatus {
		for _, predicate := range predicates {
			if predicate(netStatus[i]) {
				netStatusMeetingPredicates = append(netStatusMeetingPredicates, netStatus[i])
				continue
			}
		}
	}
	return netStatusMeetingPredicates, nil
}

func namespacedName(namespace, name string) string {
	return fmt.Sprintf("%s/%s", namespace, name)
}

func inRange(cidr string, ip string) error {
	_, cidrRange, err := net.ParseCIDR(cidr)
	if err != nil {
		return err
	}

	if cidrRange.Contains(net.ParseIP(ip)) {
		return nil
	}

	return fmt.Errorf("ip [%s] is NOT in range %s", ip, cidr)
}

func connectToServer(clientPodConfig podConfiguration, serverIP string, port int) error {
	_, err := framework.RunKubectl(
		clientPodConfig.namespace,
		"exec",
		clientPodConfig.name,
		"--",
		"curl",
		"--connect-timeout",
		"2",
		net.JoinHostPort(serverIP, fmt.Sprintf("%d", port)),
	)
	return err
}

func newAttachmentConfigWithOverriddenName(name, namespace, networkName, topology, cidr string) networkAttachmentConfig {
	return networkAttachmentConfig{
		cidr:        cidr,
		name:        name,
		namespace:   namespace,
		networkName: networkName,
		topology:    topology,
	}
}

func configurePodStaticIP(podNamespace string, podName string, staticIP string) error {
	_, err := framework.RunKubectl(
		podNamespace, "exec", podName, "--",
		"ip", "addr", "add", staticIP, "dev", "net1",
	)
	return err
}

func areStaticIPsConfiguredViaCNI(podConfig podConfiguration) bool {
	for _, attachment := range podConfig.attachments {
		if len(attachment.IPRequest) > 0 {
			return true
		}
	}
	return false
}

func podIPForAttachment(k8sClient clientset.Interface, podNamespace string, podName string, attachmentName string, ipIndex int) (string, error) {
	pod, err := k8sClient.CoreV1().Pods(podNamespace).Get(context.Background(), podName, metav1.GetOptions{})
	if err != nil {
		return "", err
	}
	netStatus, err := podNetworkStatus(pod, func(status nadapi.NetworkStatus) bool {
		return status.Name == namespacedName(podNamespace, attachmentName)
	})
	if err != nil {
		return "", err
	}
	if len(netStatus) != 1 {
		return "", fmt.Errorf("more than one status entry for attachment %s on pod %s", attachmentName, namespacedName(podNamespace, podName))
	}
	if len(netStatus[0].IPs) == 0 {
		return "", fmt.Errorf("no IPs for attachment %s on pod %s", attachmentName, namespacedName(podNamespace, podName))
	}
	return netStatus[0].IPs[ipIndex], nil
}

func allowedClient(podName string) string {
	return "allowed-" + podName
}

func blockedClient(podName string) string {
	return "blocked-" + podName
}

func multiNetIngressLimitingPolicy(policyFor string, appliesFor metav1.LabelSelector, allowForSelector metav1.LabelSelector, allowPorts ...int) *mnpapi.MultiNetworkPolicy {
	var (
		portAllowlist []mnpapi.MultiNetworkPolicyPort
	)
	tcp := v1.ProtocolTCP
	for _, port := range allowPorts {
		p := intstr.FromInt(port)
		portAllowlist = append(portAllowlist, mnpapi.MultiNetworkPolicyPort{
			Protocol: &tcp,
			Port:     &p,
		})
	}
	return &mnpapi.MultiNetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{
			PolicyForAnnotation: policyFor,
		},
			Name: "allow-ports-via-pod-selector",
		},

		Spec: mnpapi.MultiNetworkPolicySpec{
			PodSelector: appliesFor,
			Ingress: []mnpapi.MultiNetworkPolicyIngressRule{
				{
					Ports: portAllowlist,
					From: []mnpapi.MultiNetworkPolicyPeer{
						{
							PodSelector: &allowForSelector,
						},
					},
				},
			},
			PolicyTypes: []mnpapi.MultiPolicyType{mnpapi.PolicyTypeIngress},
		},
	}
}

func multiNetIngressLimitingIPBlockPolicy(
	policyFor string,
	appliesFor metav1.LabelSelector,
	allowForIPBlock mnpapi.IPBlock,
	allowPorts ...int,
) *mnpapi.MultiNetworkPolicy {
	var (
		portAllowlist []mnpapi.MultiNetworkPolicyPort
	)
	tcp := v1.ProtocolTCP
	for _, port := range allowPorts {
		p := intstr.FromInt(port)
		portAllowlist = append(portAllowlist, mnpapi.MultiNetworkPolicyPort{
			Protocol: &tcp,
			Port:     &p,
		})
	}
	return &mnpapi.MultiNetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{
			PolicyForAnnotation: policyFor,
		},
			Name: "allow-ports-ip-block",
		},

		Spec: mnpapi.MultiNetworkPolicySpec{
			PodSelector: appliesFor,
			Ingress: []mnpapi.MultiNetworkPolicyIngressRule{
				{
					Ports: portAllowlist,
					From: []mnpapi.MultiNetworkPolicyPeer{
						{
							IPBlock: &allowForIPBlock,
						},
					},
				},
			},
			PolicyTypes: []mnpapi.MultiPolicyType{mnpapi.PolicyTypeIngress},
		},
	}
}

func doesPolicyFeatAnIPBlock(policy *mnpapi.MultiNetworkPolicy) bool {
	for _, rule := range policy.Spec.Ingress {
		for _, peer := range rule.From {
			if peer.IPBlock != nil {
				return true
			}
		}
	}
	for _, rule := range policy.Spec.Egress {
		for _, peer := range rule.To {
			if peer.IPBlock != nil {
				return true
			}
		}
	}
	return false
}

func setBlockedClientIPInPolicyIPBlockExcludedRanges(policy *mnpapi.MultiNetworkPolicy, blockedIP string) {
	if policy.Spec.Ingress != nil {
		for _, rule := range policy.Spec.Ingress {
			for _, peer := range rule.From {
				if peer.IPBlock != nil {
					peer.IPBlock.Except = []string{blockedIP}
				}
			}
		}
	}
	if policy.Spec.Egress != nil {
		for _, rule := range policy.Spec.Egress {
			for _, peer := range rule.To {
				if peer.IPBlock != nil {
					peer.IPBlock.Except = []string{blockedIP}
				}
			}
		}
	}
}

func multiNetIngressLimitingPolicyAllowFromNamespace(
	policyFor string, appliesFor metav1.LabelSelector, allowForSelector metav1.LabelSelector, allowPorts ...int,
) *mnpapi.MultiNetworkPolicy {
	return &mnpapi.MultiNetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{
			PolicyForAnnotation: policyFor,
		},
			Name: "allow-ports-same-ns",
		},

		Spec: mnpapi.MultiNetworkPolicySpec{
			PodSelector: appliesFor,
			Ingress: []mnpapi.MultiNetworkPolicyIngressRule{
				{
					Ports: allowedTCPPortsForPolicy(allowPorts...),
					From: []mnpapi.MultiNetworkPolicyPeer{
						{
							NamespaceSelector: &allowForSelector,
						},
					},
				},
			},
			PolicyTypes: []mnpapi.MultiPolicyType{mnpapi.PolicyTypeIngress},
		},
	}
}

func allowedTCPPortsForPolicy(allowPorts ...int) []mnpapi.MultiNetworkPolicyPort {
	var (
		portAllowlist []mnpapi.MultiNetworkPolicyPort
	)
	tcp := v1.ProtocolTCP
	for _, port := range allowPorts {
		p := intstr.FromInt(port)
		portAllowlist = append(portAllowlist, mnpapi.MultiNetworkPolicyPort{
			Protocol: &tcp,
			Port:     &p,
		})
	}
	return portAllowlist
}

func reachToServerPodFromClient(cs clientset.Interface, serverConfig podConfiguration, clientConfig podConfiguration, serverIP string, serverPort int) error {
	updatedPod, err := cs.CoreV1().Pods(serverConfig.namespace).Get(context.Background(), serverConfig.name, metav1.GetOptions{})
	if err != nil {
		return err
	}

	if updatedPod.Status.Phase == v1.PodRunning {
		return connectToServer(clientConfig, serverIP, serverPort)
	}

	return fmt.Errorf("pod not running. /me is sad")
}