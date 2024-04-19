package provider

import (
	"fmt"
	"github.com/ovn-org/ovn-kubernetes/test/e2e/containerruntime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/kubernetes/test/e2e/framework"
	utilnet "k8s.io/utils/net"
	"os/exec"
	"strings"
	"time"
)

const (
	KinD                   Name = "kind"
	kindPrimaryNetworkName      = "kind"
)

type kind struct {
}

func newKinDProvider() Provider {
	return &kind{}
}

func (k *kind) GetName() Name {
	return KinD
}

func (k *kind) NewTestContext() Context {
	return &contextKind{containerPort: 12000}
}

func (k *kind) CleanUp() error {
	// no-op for KinD
	return nil
}

type contextKind struct {
	containerPort                      int
	cleanUpSecondaryNetworkAttachments NetworkAttachments
	cleanUpSecondaryNetworks           Networks
	cleanUpContainers                  []ExternalContainer
	cleanUpFns                         []func() error
}

func (c *contextKind) GetContainerPort() int {
	port := c.containerPort
	c.containerPort += 1
	return port
}

func (c *contextKind) ExecK8NodeCommand(nodeName string, cmd []string) (string, error) {
	if len(cmd) == 0 {
		panic("ExecK8NodeCommand(): insufficient command arguments")
	}
	cmdArgs := append([]string{"exec", nodeName}, cmd...)
	output, err := exec.Command(containerruntime.GetContainerRuntime().String(), cmdArgs...).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to run %q: %s (%s)", strings.Join(cmd, " "), err, output)
	}
	return string(output), nil
}

func (c *contextKind) ExecExternalContainerCommand(container ExternalContainer, cmd []string) (string, error) {
	cmdArgs := append([]string{"exec", container.Name}, cmd...)
	out, err := exec.Command(containerruntime.GetContainerRuntime().String(), cmdArgs...).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to exec container command (%s): err: %v, stdout: %q", strings.Join(cmdArgs, " "), err, out)
	}
	return string(out), nil
}

func (c *contextKind) CreateExternalContainer(container ExternalContainer) (ExternalContainer, error) {
	if valid, err := container.IsValidPreCreate(); !valid {
		return container, err
	}
	var networkName string
	if container.Network.Type == Primary {
		networkName = kindPrimaryNetworkName
	} else {
		networkName = container.Network.Name
	}
	container.hostPort = container.ContainerPort
	cmdArgs := []string{"run", "-itd", "--privileged", "--name", container.Name, "--network", networkName, "--hostname", container.Name,
		container.Image}
	cmdArgs = append(cmdArgs, container.CMD...)
	fmt.Printf("creating container with command: %q\n", strings.Join(cmdArgs, " "))
	output, err := exec.Command(containerruntime.GetContainerRuntime().String(), cmdArgs...).CombinedOutput()
	if err != nil {
		return container, fmt.Errorf("failed to create container %s: %s (%s)", container, err, output)
	}
	ipv4Bytes, err := exec.Command(containerruntime.GetContainerRuntime().String(), "inspect", "-f", "{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}", container.Name).CombinedOutput()
	if err != nil {
		return container, fmt.Errorf("failed to retrieve IPv4 address from container %s: %s (%s)", container, err, output)
	}
	ipv6Bytes, err := exec.Command(containerruntime.GetContainerRuntime().String(), "inspect", "-f", "{{range .NetworkSettings.Networks}}{{.GlobalIPv6Address}}{{end}}", container.Name).CombinedOutput()
	if err != nil {
		return container, fmt.Errorf("failed to retrieve IPv6 address from container %s: %s (%s)", container, err, output)
	}
	container.ipv4 = strings.Trim(string(ipv4Bytes), "\n")
	container.ipv6 = strings.Trim(string(ipv6Bytes), "\n")
	if valid, err := container.IsValidPostCreate(); !valid {
		return container, err
	}
	c.cleanUpContainers = append(c.cleanUpContainers, container)
	return container, nil
}

func (c *contextKind) DeleteExternalContainer(container ExternalContainer) error {
	// check if it is present before deleting
	output, err := exec.Command(containerruntime.GetContainerRuntime().String(), "ps", "-f", fmt.Sprintf("Name=^%s$", container.Name), "-q").CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to check if external container (%s) is already deleted: %v (%s)", container, err, output)
	}
	if string(output) == "" {
		return nil
	}
	output, err = exec.Command(containerruntime.GetContainerRuntime().String(), "rm", "-f", container.Name).CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to delete external container (%s): %v (%s)", container, err, output)
	}
	err = wait.ExponentialBackoff(wait.Backoff{Duration: 1 * time.Second, Factor: 5, Steps: 5}, wait.ConditionFunc(func() (done bool, err error) {
		output, err = exec.Command(containerruntime.GetContainerRuntime().String(), "ps", "-f", fmt.Sprintf("Name=^%s$", container.Name), "-q").CombinedOutput()
		if err != nil {
			return false, fmt.Errorf("failed to check if external container (%s) is deleted: %v (%s)", container, err, output)
		}
		if string(output) != "" {
			return false, nil
		}
		return true, nil
	}))
	if err != nil {
		return fmt.Errorf("failed to delete external container (%s): %v", container, err)
	}
	return nil
}

func (c *contextKind) GetExternalContainerLogs(container ExternalContainer) (string, error) {
	// check if it is present before retrieving logs
	output, err := exec.Command(containerruntime.GetContainerRuntime().String(), "ps", "-f", fmt.Sprintf("Name=^%s$", container.Name), "-q").CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to check if external container (%s) exists: %v (%s)", container, err, output)
	}
	if string(output) == "" {
		return "", fmt.Errorf("external container (%s) does not exist", container.String())
	}
	output, err = exec.Command(containerruntime.GetContainerRuntime().String(), "logs", container.Name).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to get logs of external container (%s): %v (%s)", container, err, output)
	}
	return string(output), nil
}

func (c *contextKind) CreateSecondaryNetwork(name string, subnets ...string) (Network, error) {
	network := Network{Secondary, name}
	cmdArgs := []string{"network", "create", "--internal", "--driver", "bridge", name}
	var v6 bool
	// detect if IPv6 flag is required
	for _, subnet := range subnets {
		cmdArgs = append(cmdArgs, "--subnet", subnet)
		if utilnet.IsIPv6CIDRString(subnet) {
			v6 = true
		}
	}
	if v6 {
		cmdArgs = append(cmdArgs, "--ipv6")
	}
	output, err := exec.Command(containerruntime.GetContainerRuntime().String(), cmdArgs...).CombinedOutput()
	if err != nil {
		return network, fmt.Errorf("failed to create Network with command %q: %s (%s)", strings.Join(cmdArgs, " "), err, output)
	}
	c.cleanUpSecondaryNetworks.InsertNoDupe(network)
	return network, nil
}

func (c *contextKind) AttachSecondaryNetwork(network Network, node string) error {
	if network.Type == Primary {
		panic("must be secondary network")
	}
	output, err := exec.Command(containerruntime.GetContainerRuntime().String(), "network", "connect", network.Name, node).CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to attach secondary network %s to node %s: %s (%s)", network.Type, node, err, output)
	}
	c.cleanUpSecondaryNetworkAttachments.InsertNoDupe(NetworkAttachment{network, node})
	return nil
}

func (c *contextKind) DetachSecondaryNetwork(network Network, node string) error {
	if network.Type == Primary {
		panic("must be secondary network")
	}
	output, err := exec.Command(containerruntime.GetContainerRuntime().String(), "network", "disconnect", network.Name, node).CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to detach secondary network %s from node %s: %s (%s)", network, node, err, output)
	}
	return nil
}

func (c *contextKind) AttachSecondaryNetworkExternalContainer(network Network, container ExternalContainer) error {
	return c.AttachSecondaryNetwork(network, container.Name)
}

func (c *contextKind) DetachSecondaryNetworkExternalContainer(network Network, container ExternalContainer) error {
	return c.DetachSecondaryNetwork(network, container.Name)
}

func (c *contextKind) DeleteSecondaryNetwork(network Network) error {
	if network.Type == Primary {
		panic("must be secondary network")
	}
	// check if it is present before deleting
	output, err := exec.Command(containerruntime.GetContainerRuntime().String(), "network", "ls", "-f", fmt.Sprintf("Name=^%s$", network.Name), "-q").CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to check if secondary network (%s) is already deleted: %v (%s)", network.Name, err, output)
	}
	if string(output) == "" {
		return nil
	}
	output, err = exec.Command(containerruntime.GetContainerRuntime().String(), "network", "rm", network.Name).CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to delete secondary network %s: %s (%s)", network.Name, err, output)
	}
	return nil
}

func (c *contextKind) AddCleanUpFn(cleanUpFn func() error) {
	c.cleanUpFns = append(c.cleanUpFns, cleanUpFn)
}

func (c *contextKind) CleanUp() error {
	var errs []error
	// detach network(s) from nodes
	for _, networkAttachment := range c.cleanUpSecondaryNetworkAttachments.List {
		framework.Logf("CleanUp: detach network %s from node %s", networkAttachment.Network, networkAttachment.Node)
		if err := c.DetachSecondaryNetwork(networkAttachment.Network, networkAttachment.Node); err != nil {
			errs = append(errs, err)
		}
	}
	// remove containers
	for _, container := range c.cleanUpContainers {
		framework.Logf("CleanUp: deleting container %s", container.Name)
		if err := c.DeleteExternalContainer(container); err != nil {
			errs = append(errs, err)
		}
	}
	c.cleanUpContainers = nil
	// delete secondary networks
	for _, network := range c.cleanUpSecondaryNetworks.List {
		framework.Logf("CleanUp: delete network %s", network.Name)
		if err := c.DeleteSecondaryNetwork(network); err != nil {
			errs = append(errs, err)
		}
	}
	c.cleanUpSecondaryNetworks.List = nil
	// generic cleanup activities
	for i := len(c.cleanUpFns) - 1; i >= 0; i-- {
		framework.Logf("CleanUp: exec cleanup func %d of %d", i+1, len(c.cleanUpFns))
		if err := c.cleanUpFns[i](); err != nil {
			errs = append(errs, err)
		}
	}
	c.cleanUpFns = nil
	return condenseErrors(errs)
}
