package provider

import (
	"k8s.io/client-go/rest"
)

type Provider interface {
	GetName() Name
	NewTestContext() Context
	CleanUp() error
}

type Context interface {
	GetContainerPort() int
	ExecK8NodeCommand(nodeName string, cmd []string) (string, error)
	ExecExternalContainerCommand(container ExternalContainer, cmd []string) (string, error)

	CreateExternalContainer(container ExternalContainer) (ExternalContainer, error)
	DeleteExternalContainer(container ExternalContainer) error
	GetExternalContainerLogs(container ExternalContainer) (string, error)

	CreateSecondaryNetwork(name string, subnets ...string) (Network, error)
	DeleteSecondaryNetwork(network Network) error
	AttachSecondaryNetwork(network Network, node string) error
	DetachSecondaryNetwork(network Network, node string) error
	AttachSecondaryNetworkExternalContainer(network Network, container ExternalContainer) error
	DetachSecondaryNetworkExternalContainer(network Network, container ExternalContainer) error

	AddCleanUpFn(func() error)
	CleanUp() error
}

type Name string

func (n Name) String() string {
	return string(n)
}

var provider Provider

func Set(config *rest.Config, depType string) {
	if provider != nil {
		panic("provider already set")
	}
	switch depType {
	case OpenShift.String():
		provider = newOpenShiftProvider(config)
	case KinD.String(), "skeleton":
		provider = newKinDProvider()
	default:
		panic("provider is not specified")
	}
}

func Get() Provider {
	if provider == nil {
		panic("provider not set")
	}
	return provider
}
