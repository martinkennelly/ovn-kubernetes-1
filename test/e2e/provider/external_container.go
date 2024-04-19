package provider

import (
	"errors"
	"fmt"
	"strings"
)

type ExternalContainer struct {
	Name          string
	ipv4          string
	ipv6          string
	Image         string
	Network       Network
	CMD           []string
	ContainerPort int
}

func (ec ExternalContainer) GetName() string {
	return ec.Name
}

func (ec ExternalContainer) GetIPv4() string {
	return ec.ipv4
}

func (ec ExternalContainer) GetIPv6() string {
	return ec.ipv6
}

func (ec ExternalContainer) IsIPv4() bool {
	return ec.ipv4 != ""
}

func (ec ExternalContainer) IsIPv6() bool {
	return ec.ipv6 != ""
}

func (ec ExternalContainer) String() string {
	str := fmt.Sprintf("Name: %q, Image: %q, Network: %q, Command: %q", ec.Name, ec.Image, ec.Network, strings.Join(ec.CMD, " "))
	if ec.IsIPv4() {
		str = fmt.Sprintf("%s, IPv4 address: %q", str, ec.GetIPv4())
	}
	if ec.IsIPv6() {
		str = fmt.Sprintf("%s, IPv6 address: %s", str, ec.GetIPv6())
	}
	str = fmt.Sprintf("%s, container port %d", str, ec.ContainerPort)
	return str
}

func (ec ExternalContainer) IsValidPreCreate() (bool, error) {
	var errs []error
	if ec.Name == "" {
		errs = append(errs, errors.New("name is not set"))
	}
	if ec.Image == "" {
		errs = append(errs, errors.New("image is not set"))
	}
	if ec.Network.Type.String() == "" {
		errs = append(errs, errors.New("network is not set"))
	}
	if ec.Network.Type == Secondary && ec.Network.Name == "" {
		errs = append(errs, errors.New("secondary networks must have name defined"))
	}
	if ec.ContainerPort == 0 {
		errs = append(errs, errors.New("container port is not set"))
	}
	if ec.ipv4 != "" {
		errs = append(errs, errors.New("do not set IPv4 address - let your provider populate it"))
	}
	if ec.ipv6 != "" {
		errs = append(errs, errors.New("do not set IPv6 address - let your provider populate it"))
	}
	if len(errs) == 0 {
		return true, nil
	}
	return false, condenseErrors(errs)
}

func (ec ExternalContainer) IsValidPostCreate() (bool, error) {
	var errs []error
	if ec.ipv4 == "" && ec.ipv6 == "" {
		errs = append(errs, errors.New("provider did not populate an IPv4 or an IPv6 address"))
	}
	if len(errs) == 0 {
		return true, nil
	}
	return false, condenseErrors(errs)
}

func (ec ExternalContainer) IsValidPreDelete() (bool, error) {
	if ec.ipv4 == "" && ec.ipv6 == "" {
		return false, fmt.Errorf("IPv4 or IPv6 must be set")
	}
	return true, nil
}

func condenseErrors(errs []error) error {
	switch len(errs) {
	case 0:
		return nil
	case 1:
		return errs[0]
	}
	err := errs[0]
	for _, e := range errs[1:] {
		err = errors.Join(err, e)
	}
	return err
}
