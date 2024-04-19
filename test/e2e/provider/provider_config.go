package provider

func GetExternalBridgeName() string {
	switch Get().GetName() {
	case OpenShift:
		return "br-ex"
	case KinD:
		return "breth0"
	default:
		panic("unknown provider name")
	}
}

func GetOVNKubernetesNamespace() string {
	switch Get().GetName() {
	case OpenShift:
		return "openshift-ovn-kubernetes"
	case KinD:
		return "ovn-kubernetes"
	default:
		panic("unknown provider name")
	}
}

func GetPrimaryInfName() string {
	switch Get().GetName() {
	case OpenShift:
		return "ens5"
	case KinD:
		return "eth0"
	default:
		panic("unknown provider name")
	}
}
