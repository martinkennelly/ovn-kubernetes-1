package provider

import (
	"context"
	"fmt"
	"github.com/openshift/api/machine/v1beta1"
	machineclient "github.com/openshift/client-go/machine/clientset/versioned"
	"github.com/ovn-org/ovn-kubernetes/test/e2e/containerruntime"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	utilnet "k8s.io/utils/net"
	"k8s.io/utils/ptr"
	"strings"
	"sync"
	"time"
)

const (
	OpenShift               Name = "openshift"
	nodeLabelSelectorWorker      = "node-role.kubernetes.io/worker"
	machineAPINamespace          = "openshift-machine-api"
	machineLabelRole             = "machine.openshift.io/cluster-api-machine-role"
	machineLabelCAPI             = "machine.openshift.io/cluster-api-machineset"
	testMachineSetName           = "ovn-kubernetes-e2e"
	machineUserName              = "core"
	machineCreationLimit         = 5
)

// machine maps 1-1 with OpenShift machine and therefore represents either a VM or BM host
type machine struct {
	name string
	ipv4 string
	ipv6 string
	// container that is hosted by this machine
	container ExternalContainer
	active    bool
}

func (m *machine) hasIPv4Addr() bool {
	return m.ipv4 != ""
}

func (m *machine) hasIPv6Addr() bool {
	return m.ipv6 != ""
}

func (m *machine) getValidIP() string {
	if m.hasIPv4Addr() {
		return m.ipv4
	}
	if m.hasIPv6Addr() {
		return m.ipv6
	}
	panic("machine has no valid IP address set")
}

func (m *machine) isHostingContainer() bool {
	return m.active
}

func (m *machine) addHostNetContainer(container ExternalContainer) error {
	if m.isHostingContainer() {
		panic("unable to add container to a machine which already hosts a container")
	}
	cmd := buildStartContainerCmd(container.Name, container.Image, container.CMD, container.ContainerPort, container.hostPort)
	cmd = addElevatedPrivileges(cmd)
	if result, err := m.execCmd(cmd); err != nil || result.isError() {
		return fmt.Errorf("failed to execute command on machine %s: %s", m.name, result)
	}
	m.container = container
	m.active = true
	return nil
}

func (m *machine) deleteHostNetContainer(container ExternalContainer) error {
	if !m.isHostingContainer() {
		panic("attempted to delete a container when the machine doesnt host one")
	}
	isRunning, err := m.isContainerRunning(container.Name)
	if err != nil {
		return fmt.Errorf("failed to check if container is running: %v", err)
	}
	if !isRunning {
		m.active = false
		return nil
	}
	// remove the container
	cmd := buildRemoveContainerCmd(container.Name)
	cmd = addElevatedPrivileges(cmd)
	if result, err := m.execCmd(cmd); err != nil || result.isError() {
		return fmt.Errorf("failed to execute command on machine %s: %s", m.name, result)
	}
	m.active = false
	return nil
}

func (m *machine) execCmd(cmd string) (result, error) {
	var result result
	signer, err := getSigner()
	if err != nil {
		return result, fmt.Errorf("error getting signer: %v", err)
	}
	result, err = runSSHCommand(cmd, m.getValidIP(), signer)
	if err != nil {
		return result, fmt.Errorf("failed to run SSH command (%s) for %s@%s: %v", cmd, machineUserName, m.getValidIP(), err)
	}
	return result, nil
}

func (m *machine) isContainerRunning(name string) (bool, error) {
	// check to see if the container is running before attempting to delete it
	isPresentCmd := buildContainerCheckCmd(name)
	isPresentCmd = addElevatedPrivileges(isPresentCmd)
	r, err := m.execCmd(isPresentCmd)
	if err != nil || r.isError() {
		return false, fmt.Errorf("failed to execute command on machine %s: %s", m.name, r)
	}
	if r.getStdOut() != "" {
		return true, nil
	}
	return false, nil
}

type machines struct {
	mu   *sync.Mutex
	list []*machine
}

type openshift struct {
	kubeClient             *kubernetes.Clientset
	machineClient          *machineclient.Clientset
	sharedExternalMachines *machines
}

func newOpenShiftProvider(config *rest.Config) Provider {
	machineClient, err := machineclient.NewForConfig(config)
	if err != nil {
		panic(fmt.Sprintf("failed to create new OpenShift provider because unable to create machine client: %v", err))
	}
	kubeClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(fmt.Sprintf("failed to create new OpenShift provider because unable to create kubernetes client: %v", err))
	}
	return openshift{machineClient: machineClient, kubeClient: kubeClient, sharedExternalMachines: &machines{mu: &sync.Mutex{}, list: make([]*machine, 0)}}
}

func (o openshift) GetName() Name {
	return OpenShift
}

func (o openshift) NewTestContext() Context {
	return &contextOpenshift{32700, o.kubeClient, o.machineClient, o.sharedExternalMachines,
		make([]ExternalContainer, 0), make([]func() error, 0)}
}

func (o openshift) CleanUp() error {
	return o.machineClient.MachineV1beta1().MachineSets(machineAPINamespace).Delete(context.TODO(), testMachineSetName, metav1.DeleteOptions{})
}

type contextOpenshift struct {
	containerPort     int
	kubeClient        *kubernetes.Clientset
	machineClient     *machineclient.Clientset
	sharedMachines    *machines
	cleanUpContainers []ExternalContainer
	cleanUpFns        []func() error
}

func (c *contextOpenshift) GetContainerPort() int {
	port := c.containerPort
	c.containerPort += 1
	return port
}

func (c contextOpenshift) ExecK8NodeCommand(nodeName string, cmd []string) (string, error) {
	if len(cmd) == 0 {
		panic("ExecK8NodeCommand(): insufficient command arguments")
	}
	signer, err := getSigner()
	if err != nil {
		return "", fmt.Errorf("error getting signer: %v", err)
	}
	cmdStr := strings.Join(cmd, " ")
	cmdStr = addElevatedPrivileges(cmdStr)
	r, err := runSSHCommand(cmdStr, nodeName, signer)
	if err != nil {
		return "", fmt.Errorf("failed to run SSH command (%s) for %s@%s: %v", cmd, machineUserName, nodeName, err)
	}
	if r.isError() {
		return "", fmt.Errorf("failed run executed command: %s", r)
	}
	return r.stdout, nil
}

func (c contextOpenshift) ExecExternalContainerCommand(container ExternalContainer, cmd []string) (string, error) {
	if !container.IsIPv6() && !container.IsIPv4() {
		return "", fmt.Errorf("expected either IPv4 or IPv6 address to be set")
	}
	m, err := c.getMachineForExternalContainer(container)
	if err != nil {
		return "", err
	}
	r, err := m.execCmd(strings.Join(cmd, " "))
	if err != nil || r.isError() {
		return "", fmt.Errorf("failed to execute command on remote machine: %v - result: %q", err, r)
	}
	return r.getStdOut(), nil
}

func (c contextOpenshift) CreateExternalContainer(container ExternalContainer) (ExternalContainer, error) {
	if valid, err := container.IsValidPreCreate(); !valid {
		return container, fmt.Errorf("failed to create external container: %v", err)
	}
	if container.Network.Type != Primary {
		return container, fmt.Errorf("non-primary network container is not supported")
	}
	c.sharedMachines.mu.Lock()
	defer c.sharedMachines.mu.Unlock()
	if err := c.ensureTestMachineSet(); err != nil {
		return container, fmt.Errorf("failed to create external container (%s) because unable to create test MachineSet: %v", container.String(), err)
	}
	container.hostPort = int(rand.Int63nRange(c.getSupportedPortRange()))
	m, err := c.getInActiveMachine()
	if err != nil {
		return container, fmt.Errorf("failed to find an available machine to host the container: %v", err)
	}
	if err = m.addHostNetContainer(container); err != nil {
		return container, fmt.Errorf("failed to add container to machine %s: %v", m.name, err)
	}
	container.ipv4 = m.ipv4
	container.ipv6 = m.ipv6
	if valid, err := container.IsValidPostCreate(); !valid {
		return container, fmt.Errorf("failed to validate external container post creation: %v", err)
	}
	c.cleanUpContainers = append(c.cleanUpContainers, container)
	return container, nil
}

func (c contextOpenshift) DeleteExternalContainer(container ExternalContainer) error {
	if valid, err := container.IsValidPreDelete(); !valid {
		return fmt.Errorf("external container is invalid: %v", err)
	}
	c.sharedMachines.mu.Lock()
	defer c.sharedMachines.mu.Unlock()
	machine, err := c.getMachineForExternalContainer(container)
	if err != nil {
		return err
	}
	return machine.deleteHostNetContainer(container)
}

func (c contextOpenshift) CreateSecondaryNetwork(name string, subnets ...string) (Network, error) {
	panic("not implemented")
}

func (c contextOpenshift) DeleteSecondaryNetwork(network Network) error {
	panic("not implemented")
}

func (c contextOpenshift) AttachSecondaryNetwork(network Network, node string) error {
	panic("not implemented")
}

func (c contextOpenshift) DetachSecondaryNetwork(network Network, node string) error {
	panic("not implemented")
}

func (c contextOpenshift) AttachSecondaryNetworkExternalContainer(network Network, container ExternalContainer) error {
	panic("not implemented")
}

func (c contextOpenshift) DetachSecondaryNetworkExternalContainer(network Network, container ExternalContainer) error {
	panic("not implemented")
}

func (c contextOpenshift) AddCleanUpFn(cleanUpFn func() error) {
	c.cleanUpFns = append(c.cleanUpFns, cleanUpFn)
}

func (c contextOpenshift) CleanUp() error {
	var errs []error
	// remove containers
	for _, container := range c.cleanUpContainers {
		if err := c.DeleteExternalContainer(container); err != nil {
			errs = append(errs, err)
		}
	}
	c.cleanUpContainers = nil
	// generic cleanup activities
	for i := len(c.cleanUpFns) - 1; i >= 0; i-- {
		if err := c.cleanUpFns[i](); err != nil {
			errs = append(errs, err)
		}
	}
	c.cleanUpFns = nil
	return condenseErrors(errs)
}

func (c contextOpenshift) isSupportedExternalContainer(container ExternalContainer) bool {
	if container.Network.Type != Primary {
		return false
	}
	return true
}

// getInActiveMachine finds a machine that is inactive or creates a new machine
func (c contextOpenshift) getInActiveMachine() (*machine, error) {
	// check if there's a machine that is free
	for _, m := range c.sharedMachines.list {
		if !m.active {
			return m, nil
		}
	}
	if len(c.sharedMachines.list) >= machineCreationLimit {
		return nil, fmt.Errorf("cannot create more machines because limit (%d) reached", machineCreationLimit)
	}
	newMachine, err := c.addMachine()
	if err != nil {
		return nil, fmt.Errorf("failed to scale up: %v", err)
	}
	c.sharedMachines.list = append(c.sharedMachines.list, newMachine)
	return newMachine, nil
}

func (c contextOpenshift) addMachine() (*machine, error) {
	newMachine := &machine{}
	// get initial count of nodes before scaling
	initWorkerNodeNames, err := getWorkerNodeNameSet(c.kubeClient)
	if err != nil {
		return newMachine, fmt.Errorf("failed to get worker node name: %v", err)
	}
	if err = scaleMachineSetReplicas(c.machineClient, testMachineSetName); err != nil {
		return newMachine, fmt.Errorf("failed to scale machineset %s: %v", testMachineSetName, err)
	}
	if err = waitUntilNodeCountIncByOne(c.kubeClient, initWorkerNodeNames.Len()); err != nil {
		return newMachine, fmt.Errorf("failed to observe new machine being added as a kuberetes node: %v", err)
	}
	if newMachine.name, err = getNewNodeName(c.kubeClient, initWorkerNodeNames); err != nil {
		return newMachine, fmt.Errorf("failed to get newly added node name: %v", err)
	}
	if err = waitUntilNodeReady(c.kubeClient, newMachine.name); err != nil {
		return newMachine, err
	}
	if err = setMachineIPs(c.kubeClient, newMachine); err != nil {
		return newMachine, fmt.Errorf("failed to get node %s IPs: %v", newMachine.name, err)
	}
	if err = c.kubeClient.CoreV1().Nodes().Delete(context.TODO(), newMachine.name, metav1.DeleteOptions{}); err != nil {
		return newMachine, fmt.Errorf("failed to delete kubernetes node %s: %v", newMachine.name, err)
	}
	return newMachine, nil
}

func (c contextOpenshift) getMachineForExternalContainer(container ExternalContainer) (*machine, error) {
	for _, machine := range c.sharedMachines.list {
		if !machine.active {
			continue
		}
		if machine.ipv4 == container.ipv4 || machine.ipv6 == container.ipv6 {
			return machine, nil
		}
	}
	return nil, fmt.Errorf("failed to find machine which hosts external container: %q", container.String())
}

// listWorkerMachineSets list all worker machineSets
func (c contextOpenshift) ensureTestMachineSet() error {
	machineSets, err := c.machineClient.MachineV1beta1().MachineSets(machineAPINamespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list machine sets: %v", machineSets)
	}
	if len(machineSets.Items) == 0 {
		return fmt.Errorf("at least one machine set must be present")
	}
	for _, machineSet := range machineSets.Items {
		if machineSet.Name == testMachineSetName {
			return nil
		}
	}

	for _, machineSet := range machineSets.Items {
		// skip machine sets that maybe owned by other tests
		if strings.Contains(machineSet.Name, testMachineSetName) {
			continue
		}
		if val, ok := machineSet.Labels[machineLabelRole]; ok {
			if val == "worker" {
				newMachineSet := patchMachineSet(machineSet.DeepCopy())
				_, err = c.machineClient.MachineV1beta1().MachineSets(machineAPINamespace).Create(context.TODO(), newMachineSet, metav1.CreateOptions{})
				return err
			}
		}
	}
	return fmt.Errorf("failed to ensure a test machine set exists")
}

func (c contextOpenshift) getSupportedPortRange() (int64, int64) {
	return 32700, 32767
}

func addElevatedPrivileges(cmd string) string {
	return fmt.Sprintf("sudo su && %s", cmd)
}

func buildStartContainerCmd(name, image string, cmd []string, containerPort, hostPort int) string {
	return fmt.Sprintf("%s run -itd --privileged --name %s --network host -p %d:%d %s %s",
		containerruntime.GetContainerRuntime(), name, hostPort, containerPort, image, strings.Join(cmd, " "))
}

func buildContainerCheckCmd(name string) string {
	return fmt.Sprintf("%s ps -f Name=^%s$ -q", containerruntime.GetContainerRuntime(), name)
}

func buildRemoveContainerCmd(name string) string {
	return fmt.Sprintf("%s rm -f %s", containerruntime.GetContainerRuntime(), name)
}

func setMachineIPs(kubeClient *kubernetes.Clientset, machine *machine) error {
	node, err := kubeClient.CoreV1().Nodes().Get(context.TODO(), machine.name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get kubernetes node %s: %v", machine.name, err)
	}
	machine.ipv4, machine.ipv6 = getNodeIPAddressForEachFamily(node)
	return nil
}

func isNodeReady(kubeClient *kubernetes.Clientset, name string) (bool, error) {
	node, err := kubeClient.CoreV1().Nodes().Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		return false, fmt.Errorf("failed to get kubernetes node %s: %v", name, err)
	}
	for _, condition := range node.Status.Conditions {
		if condition.Type == corev1.NodeReady && condition.Status == corev1.ConditionTrue {
			return true, nil
		}
	}
	return false, nil
}

func getWorkerNodeNameSet(kubeClient *kubernetes.Clientset) (sets.Set[string], error) {
	workerNodeNames := sets.New[string]()
	workerNodeList, err := kubeClient.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{
		LabelSelector: nodeLabelSelectorWorker,
	})
	if err != nil {
		return workerNodeNames, fmt.Errorf("failed to list worker nodes: %v", err)
	}
	for _, workerNode := range workerNodeList.Items {
		workerNodeNames.Insert(workerNode.Name)
	}
	return workerNodeNames, nil
}

func scaleMachineSetReplicas(machineClient *machineclient.Clientset, machineSetName string) error {
	testMachineSet, err := machineClient.MachineV1beta1().MachineSets(machineAPINamespace).Get(context.TODO(), machineSetName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get test machine set %s: %v", machineSetName, err)
	}
	replicas := *testMachineSet.Spec.Replicas
	replicas++
	testMachineSet.Spec.Replicas = &replicas
	_, err = machineClient.MachineV1beta1().MachineSets(machineAPINamespace).Update(context.TODO(), testMachineSet, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to get increment replicate count for test machine set: %v", err)
	}
	return nil
}

func waitUntilNodeReady(kubeClient *kubernetes.Clientset, nodeName string) error {
	timeout := 5 * time.Minute
	err := wait.PollUntilContextTimeout(context.TODO(), 10*time.Second, timeout, true, func(ctx context.Context) (done bool, err error) {
		return isNodeReady(kubeClient, nodeName)
	})
	if err != nil {
		return fmt.Errorf("node %s failed to become Ready (timeout %s): %v", nodeName, timeout.String(), err)
	}
	return nil
}

func waitUntilNodeCountIncByOne(kubeClient *kubernetes.Clientset, initNodeCount int) error {
	expectedNodeCount := initNodeCount + 1
	timeout := 5 * time.Minute
	// wait for node count to increment
	err := wait.PollUntilContextTimeout(context.TODO(), 10*time.Second, timeout, true, func(ctx context.Context) (done bool, err error) {
		currentWorkerNodeNames, err := getWorkerNodeNameSet(kubeClient)
		if err != nil {
			return false, fmt.Errorf("failed to get worker node names: %v", err)
		}
		if currentWorkerNodeNames.Len() == expectedNodeCount {
			return true, nil
		}
		return false, nil
	})
	if err != nil {
		return fmt.Errorf("nodes failed to scale to %d (timeout %s): %v", expectedNodeCount, timeout.String(), err)
	}
	return nil
}

func getNewNodeName(kubeClient *kubernetes.Clientset, initNodeNames sets.Set[string]) (string, error) {
	currentNodeNames, err := getWorkerNodeNameSet(kubeClient)
	if err != nil {
		return "", fmt.Errorf("failed to get worker node names: %v", err)
	}
	deltaNodeNames := initNodeNames.Difference(currentNodeNames).UnsortedList()
	if len(deltaNodeNames) != 1 {
		return "", fmt.Errorf("expected one machine to be added but found %d", len(deltaNodeNames))
	}
	return deltaNodeNames[0], nil
}

func patchMachineSet(machineSet *v1beta1.MachineSet) *v1beta1.MachineSet {
	machineSet.Name = testMachineSetName
	machineSet.Spec.Replicas = ptr.To(int32(0))
	machineSet.Spec.Selector.MatchLabels[machineLabelCAPI] = testMachineSetName
	machineSet.Spec.Template.ObjectMeta.Labels[machineLabelCAPI] = testMachineSetName
	return machineSet
}

func getNodeIPAddressForEachFamily(node *corev1.Node) (string, string) {
	var ipv4, ipv6 string
	for _, nodeAddress := range node.Status.Addresses {
		if nodeAddress.Type != corev1.NodeInternalIP && nodeAddress.Type != corev1.NodeExternalIP {
			continue
		}
		if nodeAddress.Address == "" {
			continue
		}
		if ipv4 == "" && !utilnet.IsIPv6String(nodeAddress.Address) {
			ipv4 = nodeAddress.Address
		}
		if ipv6 == "" && utilnet.IsIPv6String(nodeAddress.Address) {
			ipv6 = nodeAddress.Address
		}
	}
	return ipv4, ipv6
}
