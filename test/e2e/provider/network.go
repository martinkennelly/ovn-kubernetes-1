package provider

import "fmt"

type NetworkType int

const (
	Primary NetworkType = iota
	Secondary
)

func (n NetworkType) String() string {
	switch n {
	case Primary:
		return "primary"
	case Secondary:
		return "secondary"
	default:
		panic("unknown network type")
	}
}

type Network struct {
	Type NetworkType
	Name string
}

func (n Network) Equal(candidate Network) bool {
	if n.Name != candidate.Name {
		return false
	}
	if n.Type != candidate.Type {
		return false
	}
	return true
}

func (n Network) String() string {
	return fmt.Sprintf("Name: %s, Type: %s", n.Name, n.Type)
}

type Networks struct {
	List []Network
}

func (n *Networks) InsertNoDupe(candidate Network) {
	var found bool
	for _, network := range n.List {
		if network.Equal(candidate) {
			found = true
			break
		}
	}
	if !found {
		n.List = append(n.List, candidate)
	}
}

type NetworkAttachment struct {
	Network Network
	Node    string
}

func (na NetworkAttachment) Equal(candidate NetworkAttachment) bool {
	if na.Node != candidate.Node {
		return false
	}
	if !na.Network.Equal(candidate.Network) {
		return false
	}
	return true
}

type NetworkAttachments struct {
	List []NetworkAttachment
}

func (na *NetworkAttachments) InsertNoDupe(candidate NetworkAttachment) {
	var found bool
	for _, existingNetworkAttachment := range na.List {
		if existingNetworkAttachment.Equal(candidate) {
			found = true
			break
		}
	}
	if !found {
		na.List = append(na.List, candidate)
	}
}
