package generated

import (
	"fmt"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/ginkgo/v2/types"
)

var Annotations = map[string]string{
	"ACL Logging for AdminNetworkPolicy and BaselineAdminNetworkPolicy the ANP ACL logs have the expected log level": "[sig-network][Suite:openshift/conformance/parallel]",

	"ACL Logging for EgressFirewall when an invalid value is provided to the allow rule when the allowed destination is poked there should be no trace in the ACL logs": "[sig-network][Suite:openshift/conformance/parallel]",

	"ACL Logging for EgressFirewall when an invalid value is provided to the allow rule when the denied destination is poked the logs should have the expected log level": "[sig-network][Suite:openshift/conformance/parallel]",

	"ACL Logging for EgressFirewall when both the namespace's ACL logging deny and allow annotation are set to \"\" when the allowed destination is poked there should be no trace in the ACL logs": "[sig-network][Suite:openshift/conformance/parallel]",

	"ACL Logging for EgressFirewall when both the namespace's ACL logging deny and allow annotation are set to \"\" when the denied destination is poked there should be no trace in the ACL logs": "[sig-network][Suite:openshift/conformance/parallel]",

	"ACL Logging for EgressFirewall when both the namespace's ACL logging deny and allow annotation are set to \"invalid\" when the allowed destination is poked there should be no trace in the ACL logs": "[sig-network][Suite:openshift/conformance/parallel]",

	"ACL Logging for EgressFirewall when both the namespace's ACL logging deny and allow annotation are set to \"invalid\" when the denied destination is poked there should be no trace in the ACL logs": "[sig-network][Suite:openshift/conformance/parallel]",

	"ACL Logging for EgressFirewall when the namespace is brought up with the initial ACL log severity when the allowed destination is poked the logs should have the expected log level": "[sig-network][Suite:openshift/conformance/parallel]",

	"ACL Logging for EgressFirewall when the namespace is brought up with the initial ACL log severity when the denied destination is poked the logs should have the expected log level": "[sig-network][Suite:openshift/conformance/parallel]",

	"ACL Logging for EgressFirewall when the namespace's ACL logging allow annotation is removed when the allowed destination is poked there should be no trace in the ACL logs": "[sig-network][Suite:openshift/conformance/parallel]",

	"ACL Logging for EgressFirewall when the namespace's ACL logging allow annotation is removed when the denied destination is poked the logs should have the expected log level": "[sig-network][Suite:openshift/conformance/parallel]",

	"ACL Logging for EgressFirewall when the namespace's ACL logging annotation cannot be parsed when the allowed destination is poked there should be no trace in the ACL logs": "[sig-network][Suite:openshift/conformance/parallel]",

	"ACL Logging for EgressFirewall when the namespace's ACL logging annotation cannot be parsed when the denied destination is poked there should be no trace in the ACL logs": "[sig-network][Suite:openshift/conformance/parallel]",

	"ACL Logging for EgressFirewall when the namespace's ACL logging annotation is updated when the allowed destination is poked the logs should have the expected log level": "[sig-network][Suite:openshift/conformance/parallel]",

	"ACL Logging for EgressFirewall when the namespace's ACL logging annotation is updated when the denied destination is poked the logs should have the expected log level": "[sig-network][Suite:openshift/conformance/parallel]",

	"ACL Logging for EgressFirewall when the namespace's entire ACL logging annotation is removed when the allowed destination is poked there should be no trace in the ACL logs": "[sig-network][Suite:openshift/conformance/parallel]",

	"ACL Logging for EgressFirewall when the namespace's entire ACL logging annotation is removed when the denied destination is poked there should be no trace in the ACL logs": "[sig-network][Suite:openshift/conformance/parallel]",

	"ACL Logging for EgressFirewall when the namespace's entire ACL logging annotation is set to {} when the allowed destination is poked there should be no trace in the ACL logs": "[sig-network][Suite:openshift/conformance/parallel]",

	"ACL Logging for EgressFirewall when the namespace's entire ACL logging annotation is set to {} when the denied destination is poked there should be no trace in the ACL logs": "[sig-network][Suite:openshift/conformance/parallel]",

	"ACL Logging for NetworkPolicy the logs have the expected log level": "[sig-network][Suite:openshift/conformance/parallel]",

	"ACL Logging for NetworkPolicy when the namespace's ACL allow and deny logging annotations are set to invalid values ACL logging is disabled": "[sig-network][Suite:openshift/conformance/parallel]",

	"ACL Logging for NetworkPolicy when the namespace's ACL logging annotation is removed ACL logging is disabled": "[sig-network][Suite:openshift/conformance/parallel]",

	"ACL Logging for NetworkPolicy when the namespace's ACL logging annotation is updated the ACL logs are updated accordingly": "[sig-network][Suite:openshift/conformance/parallel]",

	"Check whether gateway-mtu-support annotation on node is set based on disable-pkt-mtu-check value when DisablePacketMTUCheck is either not set or set to false Verify whether gateway-mtu-support annotation is not set on nodes when DisablePacketMTUCheck is either not set or set to false": "[sig-network][Suite:openshift/conformance/parallel]",

	"Creating a static pod on a node Should successfully create then remove a static pod": "[sig-network][Suite:openshift/conformance/parallel]",

	"EgressService Multiple Networks, external clients sharing ip [LGW] Should validate pods on different networks can reach different clients with same ip without SNAT ipv4 pods": "[sig-network][Suite:openshift/conformance/parallel]",

	"EgressService Multiple Networks, external clients sharing ip [LGW] Should validate pods on different networks can reach different clients with same ip without SNAT ipv6 pods": "[sig-network][Suite:openshift/conformance/parallel]",

	"EgressService Should validate a node with a local ep is selected when ETP=Local ipv4 pods": "[sig-network][Suite:openshift/conformance/parallel]",

	"EgressService Should validate a node with a local ep is selected when ETP=Local ipv6 pods": "[sig-network][Suite:openshift/conformance/parallel]",

	"EgressService Should validate egress service has higher priority than EgressIP when not assigned to the same node ipv4 pods": "[sig-network][Suite:openshift/conformance/parallel]",

	"EgressService Should validate egress service has higher priority than EgressIP when not assigned to the same node ipv6 pods": "[sig-network][Suite:openshift/conformance/parallel]",

	"EgressService Should validate pods' egress is SNATed to the LB's ingress ip with selectors ipv4 pods": "[sig-network][Suite:openshift/conformance/parallel]",

	"EgressService Should validate pods' egress is SNATed to the LB's ingress ip with selectors ipv6 pods": "[sig-network][Suite:openshift/conformance/parallel]",

	"EgressService Should validate pods' egress is SNATed to the LB's ingress ip without selectors ipv4 pods": "[sig-network][Suite:openshift/conformance/parallel]",

	"EgressService Should validate pods' egress is SNATed to the LB's ingress ip without selectors ipv6 pods": "[sig-network][Suite:openshift/conformance/parallel]",

	"EgressService Should validate the egress SVC SNAT functionality against host-networked pods ipv4 pods": "[sig-network][Suite:openshift/conformance/parallel]",

	"EgressService Should validate the egress SVC SNAT functionality against host-networked pods ipv6 pods": "[sig-network][Suite:openshift/conformance/parallel]",

	"EgressService [LGW] Should validate ingress reply traffic uses the Network ipv4 pods": "[sig-network][Suite:openshift/conformance/parallel]",

	"EgressService [LGW] Should validate ingress reply traffic uses the Network ipv6 pods": "[sig-network][Suite:openshift/conformance/parallel]",

	"EgressService [LGW] Should validate pods' egress uses node's IP when setting Network without SNAT ipv4 pods": "[sig-network][Suite:openshift/conformance/parallel]",

	"EgressService [LGW] Should validate pods' egress uses node's IP when setting Network without SNAT ipv6 pods": "[sig-network][Suite:openshift/conformance/parallel]",

	"External Gateway When migrating from Annotations to Admin Policy Based External Route CRs e2e multiple external gateway stale conntrack entry deletion validation ExternalGWPod annotation: Should validate conntrack entry remains unchanged when deleting the annotation in the pods while the CR dynamic hop still references the same pods with the pod selector IPV4 tcp": "[sig-network][Suite:openshift/conformance/parallel]",

	"External Gateway When migrating from Annotations to Admin Policy Based External Route CRs e2e multiple external gateway stale conntrack entry deletion validation ExternalGWPod annotation: Should validate conntrack entry remains unchanged when deleting the annotation in the pods while the CR dynamic hop still references the same pods with the pod selector IPV4 udp": "[sig-network][Suite:openshift/conformance/parallel]",

	"External Gateway When migrating from Annotations to Admin Policy Based External Route CRs e2e multiple external gateway stale conntrack entry deletion validation ExternalGWPod annotation: Should validate conntrack entry remains unchanged when deleting the annotation in the pods while the CR dynamic hop still references the same pods with the pod selector IPV6 tcp": "[sig-network][Suite:openshift/conformance/parallel]",

	"External Gateway When migrating from Annotations to Admin Policy Based External Route CRs e2e multiple external gateway stale conntrack entry deletion validation ExternalGWPod annotation: Should validate conntrack entry remains unchanged when deleting the annotation in the pods while the CR dynamic hop still references the same pods with the pod selector IPV6 udp": "[sig-network][Suite:openshift/conformance/parallel]",

	"External Gateway When migrating from Annotations to Admin Policy Based External Route CRs e2e multiple external gateway stale conntrack entry deletion validation Namespace annotation: Should validate conntrack entry remains unchanged when deleting the annotation in the namespace while the CR static hop still references the same namespace in the policy IPV4 tcp": "[sig-network][Suite:openshift/conformance/parallel]",

	"External Gateway When migrating from Annotations to Admin Policy Based External Route CRs e2e multiple external gateway stale conntrack entry deletion validation Namespace annotation: Should validate conntrack entry remains unchanged when deleting the annotation in the namespace while the CR static hop still references the same namespace in the policy IPV4 udp": "[sig-network][Suite:openshift/conformance/parallel]",

	"External Gateway When migrating from Annotations to Admin Policy Based External Route CRs e2e multiple external gateway stale conntrack entry deletion validation Namespace annotation: Should validate conntrack entry remains unchanged when deleting the annotation in the namespace while the CR static hop still references the same namespace in the policy IPV6 tcp": "[sig-network][Suite:openshift/conformance/parallel]",

	"External Gateway When migrating from Annotations to Admin Policy Based External Route CRs e2e multiple external gateway stale conntrack entry deletion validation Namespace annotation: Should validate conntrack entry remains unchanged when deleting the annotation in the namespace while the CR static hop still references the same namespace in the policy IPV6 udp": "[sig-network][Suite:openshift/conformance/parallel]",

	"External Gateway When migrating from Annotations to Admin Policy Based External Route CRs e2e non-vxlan external gateway through a gateway pod Should validate ICMP connectivity to an external gateway's loopback address via a pod with external gateway annotations and a policy CR and after the annotations are removed ipv4": "[sig-network][Suite:openshift/conformance/parallel]",

	"External Gateway When migrating from Annotations to Admin Policy Based External Route CRs e2e non-vxlan external gateway through a gateway pod Should validate TCP/UDP connectivity to an external gateway's loopback address via a pod when deleting the annotation and supported by a CR with the same gateway IPs TCP ipv4": "[sig-network][Suite:openshift/conformance/parallel]",

	"External Gateway When migrating from Annotations to Admin Policy Based External Route CRs e2e non-vxlan external gateway through a gateway pod Should validate TCP/UDP connectivity to an external gateway's loopback address via a pod when deleting the annotation and supported by a CR with the same gateway IPs TCP ipv6": "[sig-network][Suite:openshift/conformance/parallel]",

	"External Gateway When migrating from Annotations to Admin Policy Based External Route CRs e2e non-vxlan external gateway through a gateway pod Should validate TCP/UDP connectivity to an external gateway's loopback address via a pod when deleting the annotation and supported by a CR with the same gateway IPs UDP ipv4": "[sig-network][Suite:openshift/conformance/parallel]",

	"External Gateway When migrating from Annotations to Admin Policy Based External Route CRs e2e non-vxlan external gateway through a gateway pod Should validate TCP/UDP connectivity to an external gateway's loopback address via a pod when deleting the annotation and supported by a CR with the same gateway IPs UDP ipv6": "[sig-network][Suite:openshift/conformance/parallel]",

	"External Gateway When validating the Admin Policy Based External Route status Should update the status of a successful and failed CRs": "[sig-network][Suite:openshift/conformance/parallel]",

	"External Gateway With Admin Policy Based External Route CRs BFD e2e multiple external gateway validation Should validate ICMP connectivity to multiple external gateways for an ECMP scenario IPV4": "[sig-network][Suite:openshift/conformance/parallel]",

	"External Gateway With Admin Policy Based External Route CRs BFD e2e multiple external gateway validation Should validate ICMP connectivity to multiple external gateways for an ECMP scenario IPV6": "[sig-network][Suite:openshift/conformance/parallel]",

	"External Gateway With Admin Policy Based External Route CRs BFD e2e multiple external gateway validation Should validate TCP/UDP connectivity to multiple external gateways for a UDP / TCP scenario IPV4 tcp": "[sig-network][Suite:openshift/conformance/parallel]",

	"External Gateway With Admin Policy Based External Route CRs BFD e2e multiple external gateway validation Should validate TCP/UDP connectivity to multiple external gateways for a UDP / TCP scenario IPV4 udp": "[sig-network][Suite:openshift/conformance/parallel]",

	"External Gateway With Admin Policy Based External Route CRs BFD e2e multiple external gateway validation Should validate TCP/UDP connectivity to multiple external gateways for a UDP / TCP scenario IPV6 tcp": "[sig-network][Suite:openshift/conformance/parallel]",

	"External Gateway With Admin Policy Based External Route CRs BFD e2e multiple external gateway validation Should validate TCP/UDP connectivity to multiple external gateways for a UDP / TCP scenario IPV6 udp": "[sig-network][Suite:openshift/conformance/parallel]",

	"External Gateway With Admin Policy Based External Route CRs BFD e2e non-vxlan external gateway through a dynamic hop Should validate ICMP connectivity to an external gateway's loopback address via a pod with dynamic hop ipv4": "[sig-network][Suite:openshift/conformance/parallel]",

	"External Gateway With Admin Policy Based External Route CRs BFD e2e non-vxlan external gateway through a dynamic hop Should validate ICMP connectivity to an external gateway's loopback address via a pod with dynamic hop ipv6": "[sig-network][Suite:openshift/conformance/parallel]",

	"External Gateway With Admin Policy Based External Route CRs BFD e2e non-vxlan external gateway through a dynamic hop Should validate TCP/UDP connectivity to an external gateway's loopback address via a pod with a dynamic hop TCP ipv4": "[sig-network][Suite:openshift/conformance/parallel]",

	"External Gateway With Admin Policy Based External Route CRs BFD e2e non-vxlan external gateway through a dynamic hop Should validate TCP/UDP connectivity to an external gateway's loopback address via a pod with a dynamic hop TCP ipv6": "[sig-network][Suite:openshift/conformance/parallel]",

	"External Gateway With Admin Policy Based External Route CRs BFD e2e non-vxlan external gateway through a dynamic hop Should validate TCP/UDP connectivity to an external gateway's loopback address via a pod with a dynamic hop UDP ipv4": "[sig-network][Suite:openshift/conformance/parallel]",

	"External Gateway With Admin Policy Based External Route CRs BFD e2e non-vxlan external gateway through a dynamic hop Should validate TCP/UDP connectivity to an external gateway's loopback address via a pod with a dynamic hop UDP ipv6": "[sig-network][Suite:openshift/conformance/parallel]",

	"External Gateway With Admin Policy Based External Route CRs e2e multiple external gateway stale conntrack entry deletion validation Dynamic Hop: Should validate conntrack entry deletion for TCP/UDP traffic via multiple external gateways a.k.a ECMP routes IPV4 tcp": "[sig-network][Suite:openshift/conformance/parallel]",

	"External Gateway With Admin Policy Based External Route CRs e2e multiple external gateway stale conntrack entry deletion validation Dynamic Hop: Should validate conntrack entry deletion for TCP/UDP traffic via multiple external gateways a.k.a ECMP routes IPV4 udp": "[sig-network][Suite:openshift/conformance/parallel]",

	"External Gateway With Admin Policy Based External Route CRs e2e multiple external gateway stale conntrack entry deletion validation Dynamic Hop: Should validate conntrack entry deletion for TCP/UDP traffic via multiple external gateways a.k.a ECMP routes IPV6 tcp": "[sig-network][Suite:openshift/conformance/parallel]",

	"External Gateway With Admin Policy Based External Route CRs e2e multiple external gateway stale conntrack entry deletion validation Dynamic Hop: Should validate conntrack entry deletion for TCP/UDP traffic via multiple external gateways a.k.a ECMP routes IPV6 udp": "[sig-network][Suite:openshift/conformance/parallel]",

	"External Gateway With Admin Policy Based External Route CRs e2e multiple external gateway stale conntrack entry deletion validation Static Hop: Should validate conntrack entry deletion for TCP/UDP traffic via multiple external gateways a.k.a ECMP routes IPV4 tcp": "[sig-network][Suite:openshift/conformance/parallel]",

	"External Gateway With Admin Policy Based External Route CRs e2e multiple external gateway stale conntrack entry deletion validation Static Hop: Should validate conntrack entry deletion for TCP/UDP traffic via multiple external gateways a.k.a ECMP routes IPV4 udp": "[sig-network][Suite:openshift/conformance/parallel]",

	"External Gateway With Admin Policy Based External Route CRs e2e multiple external gateway stale conntrack entry deletion validation Static Hop: Should validate conntrack entry deletion for TCP/UDP traffic via multiple external gateways a.k.a ECMP routes IPV6 tcp": "[sig-network][Suite:openshift/conformance/parallel]",

	"External Gateway With Admin Policy Based External Route CRs e2e multiple external gateway stale conntrack entry deletion validation Static Hop: Should validate conntrack entry deletion for TCP/UDP traffic via multiple external gateways a.k.a ECMP routes IPV6 udp": "[sig-network][Suite:openshift/conformance/parallel]",

	"External Gateway With Admin Policy Based External Route CRs e2e multiple external gateway validation Should validate ICMP connectivity to multiple external gateways for an ECMP scenario IPV4": "[sig-network][Suite:openshift/conformance/parallel]",

	"External Gateway With Admin Policy Based External Route CRs e2e multiple external gateway validation Should validate ICMP connectivity to multiple external gateways for an ECMP scenario IPV6": "[sig-network][Suite:openshift/conformance/parallel]",

	"External Gateway With Admin Policy Based External Route CRs e2e multiple external gateway validation Should validate TCP/UDP connectivity to multiple external gateways for a UDP / TCP scenario IPV4 tcp": "[sig-network][Suite:openshift/conformance/parallel]",

	"External Gateway With Admin Policy Based External Route CRs e2e multiple external gateway validation Should validate TCP/UDP connectivity to multiple external gateways for a UDP / TCP scenario IPV4 udp": "[sig-network][Suite:openshift/conformance/parallel]",

	"External Gateway With Admin Policy Based External Route CRs e2e multiple external gateway validation Should validate TCP/UDP connectivity to multiple external gateways for a UDP / TCP scenario IPV6 tcp": "[sig-network][Suite:openshift/conformance/parallel]",

	"External Gateway With Admin Policy Based External Route CRs e2e multiple external gateway validation Should validate TCP/UDP connectivity to multiple external gateways for a UDP / TCP scenario IPV6 udp": "[sig-network][Suite:openshift/conformance/parallel]",

	"External Gateway With Admin Policy Based External Route CRs e2e non-vxlan external gateway through a gateway pod Should validate ICMP connectivity to an external gateway's loopback address via a gateway pod ipv4": "[sig-network][Suite:openshift/conformance/parallel]",

	"External Gateway With Admin Policy Based External Route CRs e2e non-vxlan external gateway through a gateway pod Should validate ICMP connectivity to an external gateway's loopback address via a gateway pod ipv6": "[sig-network][Suite:openshift/conformance/parallel]",

	"External Gateway With Admin Policy Based External Route CRs e2e non-vxlan external gateway through a gateway pod Should validate TCP/UDP connectivity even after MAC change (gateway migration) for egress TCP ipv4": "[sig-network][Suite:openshift/conformance/parallel]",

	"External Gateway With Admin Policy Based External Route CRs e2e non-vxlan external gateway through a gateway pod Should validate TCP/UDP connectivity even after MAC change (gateway migration) for egress TCP ipv6": "[sig-network][Suite:openshift/conformance/parallel]",

	"External Gateway With Admin Policy Based External Route CRs e2e non-vxlan external gateway through a gateway pod Should validate TCP/UDP connectivity even after MAC change (gateway migration) for egress UDP ipv4": "[sig-network][Suite:openshift/conformance/parallel]",

	"External Gateway With Admin Policy Based External Route CRs e2e non-vxlan external gateway through a gateway pod Should validate TCP/UDP connectivity even after MAC change (gateway migration) for egress UDP ipv6": "[sig-network][Suite:openshift/conformance/parallel]",

	"External Gateway With Admin Policy Based External Route CRs e2e non-vxlan external gateway through a gateway pod Should validate TCP/UDP connectivity to an external gateway's loopback address via a gateway pod TCP ipv4": "[sig-network][Suite:openshift/conformance/parallel]",

	"External Gateway With Admin Policy Based External Route CRs e2e non-vxlan external gateway through a gateway pod Should validate TCP/UDP connectivity to an external gateway's loopback address via a gateway pod TCP ipv6": "[sig-network][Suite:openshift/conformance/parallel]",

	"External Gateway With Admin Policy Based External Route CRs e2e non-vxlan external gateway through a gateway pod Should validate TCP/UDP connectivity to an external gateway's loopback address via a gateway pod UDP ipv4": "[sig-network][Suite:openshift/conformance/parallel]",

	"External Gateway With Admin Policy Based External Route CRs e2e non-vxlan external gateway through a gateway pod Should validate TCP/UDP connectivity to an external gateway's loopback address via a gateway pod UDP ipv6": "[sig-network][Suite:openshift/conformance/parallel]",

	"External Gateway With annotations BFD e2e multiple external gateway validation Should validate ICMP connectivity to multiple external gateways for an ECMP scenario IPV4": "[sig-network][Suite:openshift/conformance/parallel]",

	"External Gateway With annotations BFD e2e multiple external gateway validation Should validate ICMP connectivity to multiple external gateways for an ECMP scenario IPV6": "[sig-network][Suite:openshift/conformance/parallel]",

	"External Gateway With annotations BFD e2e multiple external gateway validation Should validate TCP/UDP connectivity to multiple external gateways for a UDP / TCP scenario IPV4 tcp": "[sig-network][Suite:openshift/conformance/parallel]",

	"External Gateway With annotations BFD e2e multiple external gateway validation Should validate TCP/UDP connectivity to multiple external gateways for a UDP / TCP scenario IPV4 udp": "[sig-network][Suite:openshift/conformance/parallel]",

	"External Gateway With annotations BFD e2e multiple external gateway validation Should validate TCP/UDP connectivity to multiple external gateways for a UDP / TCP scenario IPV6 tcp": "[sig-network][Suite:openshift/conformance/parallel]",

	"External Gateway With annotations BFD e2e multiple external gateway validation Should validate TCP/UDP connectivity to multiple external gateways for a UDP / TCP scenario IPV6 udp": "[sig-network][Suite:openshift/conformance/parallel]",

	"External Gateway With annotations BFD e2e non-vxlan external gateway through an annotated gateway pod Should validate ICMP connectivity to an external gateway's loopback address via a pod with external gateway annotations enabled ipv4": "[sig-network][Suite:openshift/conformance/parallel]",

	"External Gateway With annotations BFD e2e non-vxlan external gateway through an annotated gateway pod Should validate ICMP connectivity to an external gateway's loopback address via a pod with external gateway annotations enabled ipv6": "[sig-network][Suite:openshift/conformance/parallel]",

	"External Gateway With annotations BFD e2e non-vxlan external gateway through an annotated gateway pod Should validate TCP/UDP connectivity to an external gateway's loopback address via a pod with external gateway annotations enabled TCP ipv4": "[sig-network][Suite:openshift/conformance/parallel]",

	"External Gateway With annotations BFD e2e non-vxlan external gateway through an annotated gateway pod Should validate TCP/UDP connectivity to an external gateway's loopback address via a pod with external gateway annotations enabled TCP ipv6": "[sig-network][Suite:openshift/conformance/parallel]",

	"External Gateway With annotations BFD e2e non-vxlan external gateway through an annotated gateway pod Should validate TCP/UDP connectivity to an external gateway's loopback address via a pod with external gateway annotations enabled UDP ipv4": "[sig-network][Suite:openshift/conformance/parallel]",

	"External Gateway With annotations BFD e2e non-vxlan external gateway through an annotated gateway pod Should validate TCP/UDP connectivity to an external gateway's loopback address via a pod with external gateway annotations enabled UDP ipv6": "[sig-network][Suite:openshift/conformance/parallel]",

	"External Gateway With annotations e2e multiple external gateway stale conntrack entry deletion validation ExternalGWPod annotation: Should validate conntrack entry deletion for TCP/UDP traffic via multiple external gateways a.k.a ECMP routes IPV4 tcp": "[sig-network][Suite:openshift/conformance/parallel]",

	"External Gateway With annotations e2e multiple external gateway stale conntrack entry deletion validation ExternalGWPod annotation: Should validate conntrack entry deletion for TCP/UDP traffic via multiple external gateways a.k.a ECMP routes IPV4 udp + pod delete": "[sig-network][Suite:openshift/conformance/parallel]",

	"External Gateway With annotations e2e multiple external gateway stale conntrack entry deletion validation ExternalGWPod annotation: Should validate conntrack entry deletion for TCP/UDP traffic via multiple external gateways a.k.a ECMP routes IPV4 udp": "[sig-network][Suite:openshift/conformance/parallel]",

	"External Gateway With annotations e2e multiple external gateway stale conntrack entry deletion validation ExternalGWPod annotation: Should validate conntrack entry deletion for TCP/UDP traffic via multiple external gateways a.k.a ECMP routes IPV6 tcp + pod delete": "[sig-network][Suite:openshift/conformance/parallel]",

	"External Gateway With annotations e2e multiple external gateway stale conntrack entry deletion validation ExternalGWPod annotation: Should validate conntrack entry deletion for TCP/UDP traffic via multiple external gateways a.k.a ECMP routes IPV6 tcp": "[sig-network][Suite:openshift/conformance/parallel]",

	"External Gateway With annotations e2e multiple external gateway stale conntrack entry deletion validation ExternalGWPod annotation: Should validate conntrack entry deletion for TCP/UDP traffic via multiple external gateways a.k.a ECMP routes IPV6 udp": "[sig-network][Suite:openshift/conformance/parallel]",

	"External Gateway With annotations e2e multiple external gateway stale conntrack entry deletion validation Namespace annotation: Should validate conntrack entry deletion for TCP/UDP traffic via multiple external gateways a.k.a ECMP routes IPV4 tcp": "[sig-network][Suite:openshift/conformance/parallel]",

	"External Gateway With annotations e2e multiple external gateway stale conntrack entry deletion validation Namespace annotation: Should validate conntrack entry deletion for TCP/UDP traffic via multiple external gateways a.k.a ECMP routes IPV4 udp": "[sig-network][Suite:openshift/conformance/parallel]",

	"External Gateway With annotations e2e multiple external gateway stale conntrack entry deletion validation Namespace annotation: Should validate conntrack entry deletion for TCP/UDP traffic via multiple external gateways a.k.a ECMP routes IPV6 tcp": "[sig-network][Suite:openshift/conformance/parallel]",

	"External Gateway With annotations e2e multiple external gateway stale conntrack entry deletion validation Namespace annotation: Should validate conntrack entry deletion for TCP/UDP traffic via multiple external gateways a.k.a ECMP routes IPV6 udp": "[sig-network][Suite:openshift/conformance/parallel]",

	"External Gateway With annotations e2e multiple external gateway validation Should validate ICMP connectivity to multiple external gateways for an ECMP scenario IPV4": "[sig-network][Suite:openshift/conformance/parallel]",

	"External Gateway With annotations e2e multiple external gateway validation Should validate ICMP connectivity to multiple external gateways for an ECMP scenario IPV6": "[sig-network][Suite:openshift/conformance/parallel]",

	"External Gateway With annotations e2e multiple external gateway validation Should validate TCP/UDP connectivity to multiple external gateways for a UDP / TCP scenario IPV4 tcp": "[sig-network][Suite:openshift/conformance/parallel]",

	"External Gateway With annotations e2e multiple external gateway validation Should validate TCP/UDP connectivity to multiple external gateways for a UDP / TCP scenario IPV4 udp": "[sig-network][Suite:openshift/conformance/parallel]",

	"External Gateway With annotations e2e multiple external gateway validation Should validate TCP/UDP connectivity to multiple external gateways for a UDP / TCP scenario IPV6 tcp": "[sig-network][Suite:openshift/conformance/parallel]",

	"External Gateway With annotations e2e multiple external gateway validation Should validate TCP/UDP connectivity to multiple external gateways for a UDP / TCP scenario IPV6 udp": "[sig-network][Suite:openshift/conformance/parallel]",

	"External Gateway With annotations e2e non-vxlan external gateway through a gateway pod Should validate ICMP connectivity to an external gateway's loopback address via a pod with external gateway CR ipv4": "[sig-network][Suite:openshift/conformance/parallel]",

	"External Gateway With annotations e2e non-vxlan external gateway through a gateway pod Should validate ICMP connectivity to an external gateway's loopback address via a pod with external gateway CR ipv6": "[sig-network][Suite:openshift/conformance/parallel]",

	"External Gateway With annotations e2e non-vxlan external gateway through a gateway pod Should validate TCP/UDP connectivity to an external gateway's loopback address via a pod with external gateway annotations enabled TCP ipv4": "[sig-network][Suite:openshift/conformance/parallel]",

	"External Gateway With annotations e2e non-vxlan external gateway through a gateway pod Should validate TCP/UDP connectivity to an external gateway's loopback address via a pod with external gateway annotations enabled TCP ipv6": "[sig-network][Suite:openshift/conformance/parallel]",

	"External Gateway With annotations e2e non-vxlan external gateway through a gateway pod Should validate TCP/UDP connectivity to an external gateway's loopback address via a pod with external gateway annotations enabled UDP ipv4": "[sig-network][Suite:openshift/conformance/parallel]",

	"External Gateway With annotations e2e non-vxlan external gateway through a gateway pod Should validate TCP/UDP connectivity to an external gateway's loopback address via a pod with external gateway annotations enabled UDP ipv6": "[sig-network][Suite:openshift/conformance/parallel]",

	"External Gateway e2e ingress gateway traffic validation Should validate ingress connectivity from an external gateway": "[sig-network][Suite:openshift/conformance/parallel]",

	"External Gateway e2e non-vxlan external gateway and update validation Should validate connectivity without vxlan before and after updating the namespace annotation to a new external gateway": "[sig-network][Suite:openshift/conformance/parallel]",

	"Kubevirt Virtual Machines with default pod network when live migration with post-copy succeeds, should keep connectivity": "[sig-network][Suite:openshift/conformance/parallel]",

	"Kubevirt Virtual Machines with default pod network when live migration with pre-copy fails, should keep connectivity": "[sig-network][Suite:openshift/conformance/parallel]",

	"Kubevirt Virtual Machines with default pod network when live migration with pre-copy succeeds, should keep connectivity": "[sig-network][Suite:openshift/conformance/parallel]",

	"Kubevirt Virtual Machines with kubevirt VM using layer2 UDPN should configure IPv4 and IPv6 using DHCP and NDP": "[sig-network][Suite:openshift/conformance/parallel]",

	"Kubevirt Virtual Machines with user defined networks and persistent ips configured should keep ip after live migration failed of VirtualMachineInstance with interface binding for UDN with primary/layer2": "[sig-network][Suite:openshift/conformance/parallel]",

	"Kubevirt Virtual Machines with user defined networks and persistent ips configured should keep ip after live migration failed of VirtualMachineInstance with secondary/localnet": "[sig-network][Suite:openshift/conformance/parallel]",

	"Kubevirt Virtual Machines with user defined networks and persistent ips configured should keep ip after live migration of VirtualMachine with interface binding for UDN with primary/layer2": "[sig-network][Suite:openshift/conformance/parallel]",

	"Kubevirt Virtual Machines with user defined networks and persistent ips configured should keep ip after live migration of VirtualMachine with secondary/layer2": "[sig-network][Suite:openshift/conformance/parallel]",

	"Kubevirt Virtual Machines with user defined networks and persistent ips configured should keep ip after live migration of VirtualMachine with secondary/localnet": "[sig-network][Suite:openshift/conformance/parallel]",

	"Kubevirt Virtual Machines with user defined networks and persistent ips configured should keep ip after live migration of VirtualMachineInstance with interface binding for UDN with primary/layer2": "[sig-network][Suite:openshift/conformance/parallel]",

	"Kubevirt Virtual Machines with user defined networks and persistent ips configured should keep ip after live migration of VirtualMachineInstance with secondary/layer2": "[sig-network][Suite:openshift/conformance/parallel]",

	"Kubevirt Virtual Machines with user defined networks and persistent ips configured should keep ip after live migration of VirtualMachineInstance with secondary/localnet": "[sig-network][Suite:openshift/conformance/parallel]",

	"Kubevirt Virtual Machines with user defined networks and persistent ips configured should keep ip after restart of VirtualMachine with interface binding for UDN with primary/layer2": "[sig-network][Suite:openshift/conformance/parallel]",

	"Kubevirt Virtual Machines with user defined networks and persistent ips configured should keep ip after restart of VirtualMachine with secondary/layer2": "[sig-network][Suite:openshift/conformance/parallel]",

	"Kubevirt Virtual Machines with user defined networks and persistent ips configured should keep ip after restart of VirtualMachine with secondary/localnet": "[sig-network][Suite:openshift/conformance/parallel]",

	"Load Balancer Service Tests with MetalLB Should ensure connectivity works on an external service when mtu changes in intermediate node": "[sig-network][Suite:openshift/conformance/parallel]",

	"Load Balancer Service Tests with MetalLB Should ensure load balancer service works when ETP=local and backend pods are also egressIP served pods": "[sig-network][Suite:openshift/conformance/parallel]",

	"Load Balancer Service Tests with MetalLB Should ensure load balancer service works when ETP=local and session affinity is set": "[sig-network][Suite:openshift/conformance/parallel]",

	"Load Balancer Service Tests with MetalLB Should ensure load balancer service works with 0 node ports when ETP=local": "[sig-network][Suite:openshift/conformance/parallel]",

	"Load Balancer Service Tests with MetalLB Should ensure load balancer service works with pmtud": "[sig-network][Suite:openshift/conformance/parallel]",

	"Multi Homing A pod with multiple attachments to the same OVN-K networks features two different IPs from the same subnet": "[sig-network][Suite:openshift/conformance/parallel]",

	"Multi Homing A single pod with an OVN-K secondary network is able to get to the Running phase when attaching to an L2 - switched - network featuring `excludeCIDR`s": "[sig-network][Suite:openshift/conformance/parallel]",

	"Multi Homing A single pod with an OVN-K secondary network is able to get to the Running phase when attaching to an L2 - switched - network with a dual stack configuration": "[sig-network][Suite:openshift/conformance/parallel]",

	"Multi Homing A single pod with an OVN-K secondary network is able to get to the Running phase when attaching to an L2 - switched - network with an IPv6 subnet": "[sig-network][Suite:openshift/conformance/parallel]",

	"Multi Homing A single pod with an OVN-K secondary network is able to get to the Running phase when attaching to an L2 - switched - network without IPAM": "[sig-network][Suite:openshift/conformance/parallel]",

	"Multi Homing A single pod with an OVN-K secondary network is able to get to the Running phase when attaching to an L2 - switched - network": "[sig-network][Suite:openshift/conformance/parallel]",

	"Multi Homing A single pod with an OVN-K secondary network is able to get to the Running phase when attaching to an L3 - routed - network with IPv6 network": "[sig-network][Suite:openshift/conformance/parallel]",

	"Multi Homing A single pod with an OVN-K secondary network is able to get to the Running phase when attaching to an L3 - routed - network": "[sig-network][Suite:openshift/conformance/parallel]",

	"Multi Homing A single pod with an OVN-K secondary network is able to get to the Running phase when attaching to an Localnet - switched - network featuring `excludeCIDR`s": "[sig-network][Suite:openshift/conformance/parallel]",

	"Multi Homing A single pod with an OVN-K secondary network is able to get to the Running phase when attaching to an localnet - switched - network with an IPv6 subnet": "[sig-network][Suite:openshift/conformance/parallel]",

	"Multi Homing A single pod with an OVN-K secondary network is able to get to the Running phase when attaching to an localnet - switched - network without IPAM": "[sig-network][Suite:openshift/conformance/parallel]",

	"Multi Homing A single pod with an OVN-K secondary network is able to get to the Running phase when attaching to an localnet - switched - network": "[sig-network][Suite:openshift/conformance/parallel]",

	"Multi Homing multiple pods connected to the same OVN-K secondary network can communicate over the secondary network can communicate over an L2 - switched - secondary network with `excludeCIDR`s": "[sig-network][Suite:openshift/conformance/parallel]",

	"Multi Homing multiple pods connected to the same OVN-K secondary network can communicate over the secondary network can communicate over an L2 - switched - secondary network without IPAM": "[sig-network][Suite:openshift/conformance/parallel]",

	"Multi Homing multiple pods connected to the same OVN-K secondary network can communicate over the secondary network can communicate over an L2 secondary network when the pods are scheduled in different nodes": "[sig-network][Suite:openshift/conformance/parallel]",

	"Multi Homing multiple pods connected to the same OVN-K secondary network can communicate over the secondary network can communicate over an L2 secondary network with a dual stack configuration": "[sig-network][Suite:openshift/conformance/parallel]",

	"Multi Homing multiple pods connected to the same OVN-K secondary network can communicate over the secondary network can communicate over an L2 secondary network with an IPv6 subnet when pods are scheduled in different nodes": "[sig-network][Suite:openshift/conformance/parallel]",

	"Multi Homing multiple pods connected to the same OVN-K secondary network can communicate over the secondary network can communicate over an L2 secondary network without IPAM, with static IPs configured via network selection elements": "[sig-network][Suite:openshift/conformance/parallel]",

	"Multi Homing multiple pods connected to the same OVN-K secondary network can communicate over the secondary network can communicate over an L3 - routed - secondary network with IPv6 subnet": "[sig-network][Suite:openshift/conformance/parallel]",

	"Multi Homing multiple pods connected to the same OVN-K secondary network can communicate over the secondary network can communicate over an L3 - routed - secondary network with a dual stack configuration": "[sig-network][Suite:openshift/conformance/parallel]",

	"Multi Homing multiple pods connected to the same OVN-K secondary network can communicate over the secondary network can communicate over an L3 - routed - secondary network": "[sig-network][Suite:openshift/conformance/parallel]",

	"Multi Homing multiple pods connected to the same OVN-K secondary network can communicate over the secondary network can communicate over an localnet secondary network when the pods are scheduled on different nodes": "[sig-network][Suite:openshift/conformance/parallel]",

	"Multi Homing multiple pods connected to the same OVN-K secondary network can communicate over the secondary network can communicate over an localnet secondary network with a dual stack configuration when pods are scheduled on different nodes": "[sig-network][Suite:openshift/conformance/parallel]",

	"Multi Homing multiple pods connected to the same OVN-K secondary network can communicate over the secondary network can communicate over an localnet secondary network with an IPv6 subnet when pods are scheduled on different nodes": "[sig-network][Suite:openshift/conformance/parallel]",

	"Multi Homing multiple pods connected to the same OVN-K secondary network can communicate over the secondary network can communicate over an localnet secondary network without IPAM when the pods are scheduled on different nodes": "[sig-network][Suite:openshift/conformance/parallel]",

	"Multi Homing multiple pods connected to the same OVN-K secondary network can communicate over the secondary network can communicate over an localnet secondary network without IPAM when the pods are scheduled on different nodes, with static IPs configured via network selection elements": "[sig-network][Suite:openshift/conformance/parallel]",

	"Multi Homing multiple pods connected to the same OVN-K secondary network eventually configures pods that were added to an already existing network before the nad": "[sig-network][Suite:openshift/conformance/parallel]",

	"Multi Homing multiple pods connected to the same OVN-K secondary network localnet OVN-K secondary network with a service running on the underlay can communicate over a localnet secondary network from pod to the underlay service": "[sig-network][Suite:openshift/conformance/parallel]",

	"Multi Homing multiple pods connected to the same OVN-K secondary network localnet OVN-K secondary network with a service running on the underlay when a policy is provisioned can communicate over a localnet secondary network from pod to gw egress allow all": "[sig-network][Suite:openshift/conformance/parallel]",

	"Multi Homing multiple pods connected to the same OVN-K secondary network localnet OVN-K secondary network with a service running on the underlay when a policy is provisioned can communicate over a localnet secondary network from pod to gw egress deny all": "[sig-network][Suite:openshift/conformance/parallel]",

	"Multi Homing multiple pods connected to the same OVN-K secondary network localnet OVN-K secondary network with a service running on the underlay when a policy is provisioned can communicate over a localnet secondary network from pod to gw ingress denyall, egress allow all, ingress policy should have no impact on egress": "[sig-network][Suite:openshift/conformance/parallel]",

	"Multi Homing multiple pods connected to the same OVN-K secondary network localnet OVN-K secondary network with a service running on the underlay when a policy is provisioned can communicate over a localnet secondary network from pod to gw ingress denyall, ingress policy should have no impact on egress": "[sig-network][Suite:openshift/conformance/parallel]",

	"Multi Homing multiple pods connected to the same OVN-K secondary network localnet OVN-K secondary network with a service running on the underlay with multi network policy blocking the traffic can not communicate over a localnet secondary network from pod to the underlay service": "[sig-network][Suite:openshift/conformance/parallel]",

	"Multi Homing multiple pods connected to the same OVN-K secondary network localnet OVN-K secondary network with a trunked configuration the same bridge mapping can be shared by a separate VLAN by using the physical network name attribute": "[sig-network][Suite:openshift/conformance/parallel]",

	"Multi Homing multiple pods connected to the same OVN-K secondary network multi-network policies multi-network ingress allow all for a localnet topology when the multi-net policy is egress deny-all, ingress allow-all": "[sig-network][Suite:openshift/conformance/parallel]",

	"Multi Homing multiple pods connected to the same OVN-K secondary network multi-network policies multi-network ingress allow all for a localnet topology when the multi-net policy is egress deny-all, should not affect ingress": "[sig-network][Suite:openshift/conformance/parallel]",

	"Multi Homing multiple pods connected to the same OVN-K secondary network multi-network policies multi-network ingress allow all for a localnet topology when the multi-net policy is ingress allow-all": "[sig-network][Suite:openshift/conformance/parallel]",

	"Multi Homing multiple pods connected to the same OVN-K secondary network multi-network policies multi-network ingress deny all policies for a localnet topology when the multi-net policy is ingress deny-all": "[sig-network][Suite:openshift/conformance/parallel]",

	"Multi Homing multiple pods connected to the same OVN-K secondary network multi-network policies multi-network policies configure traffic allow lists for a localnet topology when the multi-net policy describes the allow-list using IPBlock": "[sig-network][Suite:openshift/conformance/parallel]",

	"Multi Homing multiple pods connected to the same OVN-K secondary network multi-network policies multi-network policies configure traffic allow lists for a localnet topology when the multi-net policy describes the allow-list using pod selectors": "[sig-network][Suite:openshift/conformance/parallel]",

	"Multi Homing multiple pods connected to the same OVN-K secondary network multi-network policies multi-network policies configure traffic allow lists for a localnet topology when the multi-net policy describes the allow-list via namespace selectors": "[sig-network][Suite:openshift/conformance/parallel]",

	"Multi Homing multiple pods connected to the same OVN-K secondary network multi-network policies multi-network policies configure traffic allow lists for a pure L2 overlay when the multi-net policy describes the allow-list using IPBlock": "[sig-network][Suite:openshift/conformance/parallel]",

	"Multi Homing multiple pods connected to the same OVN-K secondary network multi-network policies multi-network policies configure traffic allow lists for a pure L2 overlay when the multi-net policy describes the allow-list using pod selectors": "[sig-network][Suite:openshift/conformance/parallel]",

	"Multi Homing multiple pods connected to the same OVN-K secondary network multi-network policies multi-network policies configure traffic allow lists for a pure L2 overlay when the multi-net policy describes the allow-list via namespace selectors": "[sig-network][Suite:openshift/conformance/parallel]",

	"Multi Homing multiple pods connected to the same OVN-K secondary network multi-network policies multi-network policies configure traffic allow lists for a routed topology when the multi-net policy describes the allow-list using IPBlock": "[sig-network][Suite:openshift/conformance/parallel]",

	"Multi Homing multiple pods connected to the same OVN-K secondary network multi-network policies multi-network policies configure traffic allow lists for a routed topology when the multi-net policy describes the allow-list using pod selectors": "[sig-network][Suite:openshift/conformance/parallel]",

	"Multi Homing multiple pods connected to the same OVN-K secondary network multi-network policies multi-network policies configure traffic allow lists for a routed topology when the multi-net policy describes the allow-list via namespace selectors": "[sig-network][Suite:openshift/conformance/parallel]",

	"Multi Homing multiple pods connected to the same OVN-K secondary network multi-network policies multi-network policies configure traffic allow lists for an IPAMless pure L2 overlay when the multi-net policy describes the allow-list using IPBlock": "[sig-network][Suite:openshift/conformance/parallel]",

	"Multi node zones interconnect Pod interconnectivity": "[sig-network][Suite:openshift/conformance/parallel]",

	"Multicast when multicast enabled for namespace should be able to receive multicast IGMP query": "[sig-network][Suite:openshift/conformance/parallel]",

	"Multicast when multicast enabled for namespace should be able to send multicast UDP traffic between nodes": "[sig-network][Suite:openshift/conformance/parallel]",

	"Network Segmentation ClusterUserDefinedNetwork CRD Controller pod connected to ClusterUserDefinedNetwork CR & managed NADs cannot be deleted when being used": "[sig-network][OCPFeatureGate:NetworkSegmentation][Feature:UserDefinedPrimaryNetworks][Suite:openshift/conformance/parallel]",

	"Network Segmentation ClusterUserDefinedNetwork CRD Controller should create NAD according to spec in each target namespace and report active namespaces": "[sig-network][OCPFeatureGate:NetworkSegmentation][Feature:UserDefinedPrimaryNetworks][Suite:openshift/conformance/parallel]",

	"Network Segmentation ClusterUserDefinedNetwork CRD Controller should create NAD in new created namespaces that apply to namespace-selector": "[sig-network][OCPFeatureGate:NetworkSegmentation][Feature:UserDefinedPrimaryNetworks][Suite:openshift/conformance/parallel]",

	"Network Segmentation ClusterUserDefinedNetwork CRD Controller when CR is deleted, should delete all managed NAD in each target namespace": "[sig-network][OCPFeatureGate:NetworkSegmentation][Feature:UserDefinedPrimaryNetworks][Suite:openshift/conformance/parallel]",

	"Network Segmentation ClusterUserDefinedNetwork CRD Controller when namespace-selector is mutated should create NAD in namespaces that apply to mutated namespace-selector": "[sig-network][OCPFeatureGate:NetworkSegmentation][Feature:UserDefinedPrimaryNetworks][Suite:openshift/conformance/parallel]",

	"Network Segmentation ClusterUserDefinedNetwork CRD Controller when namespace-selector is mutated should delete managed NAD in namespaces that no longer apply to namespace-selector": "[sig-network][OCPFeatureGate:NetworkSegmentation][Feature:UserDefinedPrimaryNetworks][Suite:openshift/conformance/parallel]",

	"Network Segmentation EndpointSlices mirroring a user defined primary network created using NetworkAttachmentDefinitions does not mirror EndpointSlices in namespaces not using user defined primary networks L2 secondary UDN": "[sig-network][OCPFeatureGate:NetworkSegmentation][Feature:UserDefinedPrimaryNetworks][Suite:openshift/conformance/parallel]",

	"Network Segmentation EndpointSlices mirroring a user defined primary network created using NetworkAttachmentDefinitions does not mirror EndpointSlices in namespaces not using user defined primary networks L3 secondary UDN": "[sig-network][OCPFeatureGate:NetworkSegmentation][Feature:UserDefinedPrimaryNetworks][Suite:openshift/conformance/parallel]",

	"Network Segmentation EndpointSlices mirroring a user defined primary network created using NetworkAttachmentDefinitions mirrors EndpointSlices managed by the default controller for namespaces with user defined primary networks L2 primary UDN, cluster-networked pods": "[sig-network][OCPFeatureGate:NetworkSegmentation][Feature:UserDefinedPrimaryNetworks][Suite:openshift/conformance/parallel]",

	"Network Segmentation EndpointSlices mirroring a user defined primary network created using NetworkAttachmentDefinitions mirrors EndpointSlices managed by the default controller for namespaces with user defined primary networks L2 primary UDN, host-networked pods": "[sig-network][OCPFeatureGate:NetworkSegmentation][Feature:UserDefinedPrimaryNetworks][Suite:openshift/conformance/parallel]",

	"Network Segmentation EndpointSlices mirroring a user defined primary network created using NetworkAttachmentDefinitions mirrors EndpointSlices managed by the default controller for namespaces with user defined primary networks L3 primary UDN, cluster-networked pods": "[sig-network][OCPFeatureGate:NetworkSegmentation][Feature:UserDefinedPrimaryNetworks][Suite:openshift/conformance/parallel]",

	"Network Segmentation EndpointSlices mirroring a user defined primary network created using NetworkAttachmentDefinitions mirrors EndpointSlices managed by the default controller for namespaces with user defined primary networks L3 primary UDN, host-networked pods": "[sig-network][OCPFeatureGate:NetworkSegmentation][Feature:UserDefinedPrimaryNetworks][Suite:openshift/conformance/parallel]",

	"Network Segmentation EndpointSlices mirroring a user defined primary network created using UserDefinedNetwork does not mirror EndpointSlices in namespaces not using user defined primary networks L2 secondary UDN": "[sig-network][OCPFeatureGate:NetworkSegmentation][Feature:UserDefinedPrimaryNetworks][Suite:openshift/conformance/parallel]",

	"Network Segmentation EndpointSlices mirroring a user defined primary network created using UserDefinedNetwork does not mirror EndpointSlices in namespaces not using user defined primary networks L3 secondary UDN": "[sig-network][OCPFeatureGate:NetworkSegmentation][Feature:UserDefinedPrimaryNetworks][Suite:openshift/conformance/parallel]",

	"Network Segmentation EndpointSlices mirroring a user defined primary network created using UserDefinedNetwork mirrors EndpointSlices managed by the default controller for namespaces with user defined primary networks L2 primary UDN, cluster-networked pods": "[sig-network][OCPFeatureGate:NetworkSegmentation][Feature:UserDefinedPrimaryNetworks][Suite:openshift/conformance/parallel]",

	"Network Segmentation EndpointSlices mirroring a user defined primary network created using UserDefinedNetwork mirrors EndpointSlices managed by the default controller for namespaces with user defined primary networks L2 primary UDN, host-networked pods": "[sig-network][OCPFeatureGate:NetworkSegmentation][Feature:UserDefinedPrimaryNetworks][Suite:openshift/conformance/parallel]",

	"Network Segmentation EndpointSlices mirroring a user defined primary network created using UserDefinedNetwork mirrors EndpointSlices managed by the default controller for namespaces with user defined primary networks L3 primary UDN, cluster-networked pods": "[sig-network][OCPFeatureGate:NetworkSegmentation][Feature:UserDefinedPrimaryNetworks][Suite:openshift/conformance/parallel]",

	"Network Segmentation EndpointSlices mirroring a user defined primary network created using UserDefinedNetwork mirrors EndpointSlices managed by the default controller for namespaces with user defined primary networks L3 primary UDN, host-networked pods": "[sig-network][OCPFeatureGate:NetworkSegmentation][Feature:UserDefinedPrimaryNetworks][Suite:openshift/conformance/parallel]",

	"Network Segmentation Sync perform east/west traffic between nodes following OVN Kube node pod restart L2": "[sig-network][OCPFeatureGate:NetworkSegmentation][Feature:UserDefinedPrimaryNetworks][Suite:openshift/conformance/parallel]",

	"Network Segmentation Sync perform east/west traffic between nodes following OVN Kube node pod restart L3": "[sig-network][OCPFeatureGate:NetworkSegmentation][Feature:UserDefinedPrimaryNetworks][Suite:openshift/conformance/parallel]",

	"Network Segmentation UDN Pod should react to k8s.ovn.org/open-default-ports annotations changes": "[sig-network][OCPFeatureGate:NetworkSegmentation][Feature:UserDefinedPrimaryNetworks][Suite:openshift/conformance/parallel]",

	"Network Segmentation UserDefinedNetwork CRD Controller for L2 secondary network pod connected to UserDefinedNetwork cannot be deleted when being used": "[sig-network][OCPFeatureGate:NetworkSegmentation][Feature:UserDefinedPrimaryNetworks][Suite:openshift/conformance/parallel]",

	"Network Segmentation UserDefinedNetwork CRD Controller for L2 secondary network should create NetworkAttachmentDefinition according to spec": "[sig-network][OCPFeatureGate:NetworkSegmentation][Feature:UserDefinedPrimaryNetworks][Suite:openshift/conformance/parallel]",

	"Network Segmentation UserDefinedNetwork CRD Controller for L2 secondary network should delete NetworkAttachmentDefinition when UserDefinedNetwork is deleted": "[sig-network][OCPFeatureGate:NetworkSegmentation][Feature:UserDefinedPrimaryNetworks][Suite:openshift/conformance/parallel]",

	"Network Segmentation UserDefinedNetwork CRD Controller for primary UDN without required namespace label should be able to create pod and it will attach to the cluster default network": "[sig-network][OCPFeatureGate:NetworkSegmentation][Feature:UserDefinedPrimaryNetworks][Suite:openshift/conformance/parallel]",

	"Network Segmentation UserDefinedNetwork CRD Controller for primary UDN without required namespace label should not be able to update the namespace and add the UDN label": "[sig-network][OCPFeatureGate:NetworkSegmentation][Feature:UserDefinedPrimaryNetworks][Suite:openshift/conformance/parallel]",

	"Network Segmentation UserDefinedNetwork CRD Controller for primary UDN without required namespace label should not be able to update the namespace and remove the UDN label": "[sig-network][OCPFeatureGate:NetworkSegmentation][Feature:UserDefinedPrimaryNetworks][Suite:openshift/conformance/parallel]",

	"Network Segmentation UserDefinedNetwork CRD Controller should correctly report subsystem error on node subnet allocation": "[sig-network][OCPFeatureGate:NetworkSegmentation][Feature:UserDefinedPrimaryNetworks][Suite:openshift/conformance/parallel]",

	"Network Segmentation a user defined primary network created using ClusterUserDefinedNetwork can perform east/west traffic between nodes two pods connected over a L2 primary UDN": "[sig-network][OCPFeatureGate:NetworkSegmentation][Feature:UserDefinedPrimaryNetworks][Suite:openshift/conformance/parallel]",

	"Network Segmentation a user defined primary network created using ClusterUserDefinedNetwork can perform east/west traffic between nodes two pods connected over a L3 primary UDN": "[sig-network][OCPFeatureGate:NetworkSegmentation][Feature:UserDefinedPrimaryNetworks][Suite:openshift/conformance/parallel]",

	"Network Segmentation a user defined primary network created using ClusterUserDefinedNetwork creates a networkStatus Annotation with UDN interface L2 primary UDN": "[sig-network][OCPFeatureGate:NetworkSegmentation][Feature:UserDefinedPrimaryNetworks][Suite:openshift/conformance/parallel]",

	"Network Segmentation a user defined primary network created using ClusterUserDefinedNetwork creates a networkStatus Annotation with UDN interface L3 primary UDN": "[sig-network][OCPFeatureGate:NetworkSegmentation][Feature:UserDefinedPrimaryNetworks][Suite:openshift/conformance/parallel]",

	"Network Segmentation a user defined primary network created using ClusterUserDefinedNetwork is isolated from the default network with L2 primary UDN": "[sig-network][OCPFeatureGate:NetworkSegmentation][Feature:UserDefinedPrimaryNetworks][Suite:openshift/conformance/parallel]",

	"Network Segmentation a user defined primary network created using ClusterUserDefinedNetwork is isolated from the default network with L3 primary UDN": "[sig-network][OCPFeatureGate:NetworkSegmentation][Feature:UserDefinedPrimaryNetworks][Suite:openshift/conformance/parallel]",

	"Network Segmentation a user defined primary network created using ClusterUserDefinedNetwork isolates overlapping CIDRs with L2 primary UDN": "[sig-network][OCPFeatureGate:NetworkSegmentation][Feature:UserDefinedPrimaryNetworks][Suite:openshift/conformance/parallel]",

	"Network Segmentation a user defined primary network created using ClusterUserDefinedNetwork isolates overlapping CIDRs with L3 primary UDN": "[sig-network][OCPFeatureGate:NetworkSegmentation][Feature:UserDefinedPrimaryNetworks][Suite:openshift/conformance/parallel]",

	"Network Segmentation a user defined primary network created using NetworkAttachmentDefinitions can perform east/west traffic between nodes two pods connected over a L2 primary UDN": "[sig-network][OCPFeatureGate:NetworkSegmentation][Feature:UserDefinedPrimaryNetworks][Suite:openshift/conformance/parallel]",

	"Network Segmentation a user defined primary network created using NetworkAttachmentDefinitions can perform east/west traffic between nodes two pods connected over a L3 primary UDN": "[sig-network][OCPFeatureGate:NetworkSegmentation][Feature:UserDefinedPrimaryNetworks][Suite:openshift/conformance/parallel]",

	"Network Segmentation a user defined primary network created using NetworkAttachmentDefinitions creates a networkStatus Annotation with UDN interface L2 primary UDN": "[sig-network][OCPFeatureGate:NetworkSegmentation][Feature:UserDefinedPrimaryNetworks][Suite:openshift/conformance/parallel]",

	"Network Segmentation a user defined primary network created using NetworkAttachmentDefinitions creates a networkStatus Annotation with UDN interface L3 primary UDN": "[sig-network][OCPFeatureGate:NetworkSegmentation][Feature:UserDefinedPrimaryNetworks][Suite:openshift/conformance/parallel]",

	"Network Segmentation a user defined primary network created using NetworkAttachmentDefinitions is isolated from the default network with L2 primary UDN": "[sig-network][OCPFeatureGate:NetworkSegmentation][Feature:UserDefinedPrimaryNetworks][Suite:openshift/conformance/parallel]",

	"Network Segmentation a user defined primary network created using NetworkAttachmentDefinitions is isolated from the default network with L3 primary UDN": "[sig-network][OCPFeatureGate:NetworkSegmentation][Feature:UserDefinedPrimaryNetworks][Suite:openshift/conformance/parallel]",

	"Network Segmentation a user defined primary network created using NetworkAttachmentDefinitions isolates overlapping CIDRs with L2 primary UDN": "[sig-network][OCPFeatureGate:NetworkSegmentation][Feature:UserDefinedPrimaryNetworks][Suite:openshift/conformance/parallel]",

	"Network Segmentation a user defined primary network created using NetworkAttachmentDefinitions isolates overlapping CIDRs with L3 primary UDN": "[sig-network][OCPFeatureGate:NetworkSegmentation][Feature:UserDefinedPrimaryNetworks][Suite:openshift/conformance/parallel]",

	"Network Segmentation a user defined primary network created using UserDefinedNetwork can perform east/west traffic between nodes two pods connected over a L2 primary UDN": "[sig-network][OCPFeatureGate:NetworkSegmentation][Feature:UserDefinedPrimaryNetworks][Suite:openshift/conformance/parallel]",

	"Network Segmentation a user defined primary network created using UserDefinedNetwork can perform east/west traffic between nodes two pods connected over a L3 primary UDN": "[sig-network][OCPFeatureGate:NetworkSegmentation][Feature:UserDefinedPrimaryNetworks][Suite:openshift/conformance/parallel]",

	"Network Segmentation a user defined primary network created using UserDefinedNetwork creates a networkStatus Annotation with UDN interface L2 primary UDN": "[sig-network][OCPFeatureGate:NetworkSegmentation][Feature:UserDefinedPrimaryNetworks][Suite:openshift/conformance/parallel]",

	"Network Segmentation a user defined primary network created using UserDefinedNetwork creates a networkStatus Annotation with UDN interface L3 primary UDN": "[sig-network][OCPFeatureGate:NetworkSegmentation][Feature:UserDefinedPrimaryNetworks][Suite:openshift/conformance/parallel]",

	"Network Segmentation a user defined primary network created using UserDefinedNetwork is isolated from the default network with L2 primary UDN": "[sig-network][OCPFeatureGate:NetworkSegmentation][Feature:UserDefinedPrimaryNetworks][Suite:openshift/conformance/parallel]",

	"Network Segmentation a user defined primary network created using UserDefinedNetwork is isolated from the default network with L3 primary UDN": "[sig-network][OCPFeatureGate:NetworkSegmentation][Feature:UserDefinedPrimaryNetworks][Suite:openshift/conformance/parallel]",

	"Network Segmentation a user defined primary network created using UserDefinedNetwork isolates overlapping CIDRs with L2 primary UDN": "[sig-network][OCPFeatureGate:NetworkSegmentation][Feature:UserDefinedPrimaryNetworks][Suite:openshift/conformance/parallel]",

	"Network Segmentation a user defined primary network created using UserDefinedNetwork isolates overlapping CIDRs with L3 primary UDN": "[sig-network][OCPFeatureGate:NetworkSegmentation][Feature:UserDefinedPrimaryNetworks][Suite:openshift/conformance/parallel]",

	"Network Segmentation a user defined primary network doesn't cause network name conflict": "[sig-network][OCPFeatureGate:NetworkSegmentation][Feature:UserDefinedPrimaryNetworks][Suite:openshift/conformance/parallel]",

	"Network Segmentation a user defined primary network with multicast feature enabled for namespace should be able to receive multicast IGMP query with primary layer2 UDN": "[sig-network][OCPFeatureGate:NetworkSegmentation][Feature:UserDefinedPrimaryNetworks][Suite:openshift/conformance/parallel]",

	"Network Segmentation a user defined primary network with multicast feature enabled for namespace should be able to receive multicast IGMP query with primary layer3 UDN": "[sig-network][OCPFeatureGate:NetworkSegmentation][Feature:UserDefinedPrimaryNetworks][Suite:openshift/conformance/parallel]",

	"Network Segmentation a user defined primary network with multicast feature enabled for namespace should be able to send multicast UDP traffic between nodes with primary layer2 UDN": "[sig-network][OCPFeatureGate:NetworkSegmentation][Feature:UserDefinedPrimaryNetworks][Suite:openshift/conformance/parallel]",

	"Network Segmentation a user defined primary network with multicast feature enabled for namespace should be able to send multicast UDP traffic between nodes with primary layer3 UDN": "[sig-network][OCPFeatureGate:NetworkSegmentation][Feature:UserDefinedPrimaryNetworks][Suite:openshift/conformance/parallel]",

	"Network Segmentation pod2Egress on a user defined primary network created using ClusterUserDefinedNetwork can be accessed to from the pods running in the Kubernetes cluster by one pod over a layer2 network": "[sig-network][OCPFeatureGate:NetworkSegmentation][Feature:UserDefinedPrimaryNetworks][Suite:openshift/conformance/parallel]",

	"Network Segmentation pod2Egress on a user defined primary network created using ClusterUserDefinedNetwork can be accessed to from the pods running in the Kubernetes cluster by one pod over a layer3 network": "[sig-network][OCPFeatureGate:NetworkSegmentation][Feature:UserDefinedPrimaryNetworks][Suite:openshift/conformance/parallel]",

	"Network Segmentation pod2Egress on a user defined primary network created using NetworkAttachmentDefinitions can be accessed to from the pods running in the Kubernetes cluster by one pod over a layer2 network": "[sig-network][OCPFeatureGate:NetworkSegmentation][Feature:UserDefinedPrimaryNetworks][Suite:openshift/conformance/parallel]",

	"Network Segmentation pod2Egress on a user defined primary network created using NetworkAttachmentDefinitions can be accessed to from the pods running in the Kubernetes cluster by one pod over a layer3 network": "[sig-network][OCPFeatureGate:NetworkSegmentation][Feature:UserDefinedPrimaryNetworks][Suite:openshift/conformance/parallel]",

	"Network Segmentation pod2Egress on a user defined primary network created using UserDefinedNetwork can be accessed to from the pods running in the Kubernetes cluster by one pod over a layer2 network": "[sig-network][OCPFeatureGate:NetworkSegmentation][Feature:UserDefinedPrimaryNetworks][Suite:openshift/conformance/parallel]",

	"Network Segmentation pod2Egress on a user defined primary network created using UserDefinedNetwork can be accessed to from the pods running in the Kubernetes cluster by one pod over a layer3 network": "[sig-network][OCPFeatureGate:NetworkSegmentation][Feature:UserDefinedPrimaryNetworks][Suite:openshift/conformance/parallel]",

	"Network Segmentation when primary network exist, ClusterUserDefinedNetwork status should report not-ready": "[sig-network][OCPFeatureGate:NetworkSegmentation][Feature:UserDefinedPrimaryNetworks][Suite:openshift/conformance/parallel]",

	"Network Segmentation when primary network exist, UserDefinedNetwork status should report not-ready": "[sig-network][OCPFeatureGate:NetworkSegmentation][Feature:UserDefinedPrimaryNetworks][Suite:openshift/conformance/parallel]",

	"Network Segmentation: Network Policies on a user defined primary network allow ingress traffic to one pod from a particular namespace in L2 primary UDN": "[sig-network][OCPFeatureGate:NetworkSegmentation][Feature:UserDefinedPrimaryNetworks][Suite:openshift/conformance/parallel]",

	"Network Segmentation: Network Policies on a user defined primary network allow ingress traffic to one pod from a particular namespace in L3 primary UDN": "[sig-network][OCPFeatureGate:NetworkSegmentation][Feature:UserDefinedPrimaryNetworks][Suite:openshift/conformance/parallel]",

	"Network Segmentation: Network Policies on a user defined primary network pods within namespace should be isolated when deny policy is present in L2 dualstack primary UDN": "[sig-network][OCPFeatureGate:NetworkSegmentation][Feature:UserDefinedPrimaryNetworks][Suite:openshift/conformance/parallel]",

	"Network Segmentation: Network Policies on a user defined primary network pods within namespace should be isolated when deny policy is present in L3 dualstack primary UDN": "[sig-network][OCPFeatureGate:NetworkSegmentation][Feature:UserDefinedPrimaryNetworks][Suite:openshift/conformance/parallel]",

	"Network Segmentation: services on a user defined primary network should be reachable through their cluster IP, node port and load balancer L2 primary UDN, cluster-networked pods, NodePort service": "[sig-network][OCPFeatureGate:NetworkSegmentation][Feature:UserDefinedPrimaryNetworks][Suite:openshift/conformance/parallel]",

	"Network Segmentation: services on a user defined primary network should be reachable through their cluster IP, node port and load balancer L3 primary UDN, cluster-networked pods, NodePort service": "[sig-network][OCPFeatureGate:NetworkSegmentation][Feature:UserDefinedPrimaryNetworks][Suite:openshift/conformance/parallel]",

	"Node IP and MAC address migration when the node IPv4 address is updated when ETP=Local service with host network backend is configured makes sure that the flows are updated with new IP address (update kubelet first, the IP address later)": "[sig-network][Suite:openshift/conformance/parallel]",

	"Node IP and MAC address migration when the node IPv4 address is updated when ETP=Local service with host network backend is configured makes sure that the flows are updated with new IP address (update the IP address first, kubelet later)": "[sig-network][Suite:openshift/conformance/parallel]",

	"Node IP and MAC address migration when the node IPv4 address is updated when EgressIPs are configured makes sure that the EgressIP is still operational (update kubelet first, the IP address later)": "[sig-network][Suite:openshift/conformance/parallel]",

	"Node IP and MAC address migration when the node IPv4 address is updated when EgressIPs are configured makes sure that the EgressIP is still operational (update the IP address first, kubelet later)": "[sig-network][Suite:openshift/conformance/parallel]",

	"Node IP and MAC address migration when the node IPv4 address is updated when no EgressIPs are configured makes sure that the cluster is still operational (update kubelet first, the IP address later)": "[sig-network][Suite:openshift/conformance/parallel]",

	"Node IP and MAC address migration when the node IPv4 address is updated when no EgressIPs are configured makes sure that the cluster is still operational (update the IP address first, kubelet later)": "[sig-network][Suite:openshift/conformance/parallel]",

	"Node IP and MAC address migration when the node IPv6 address is updated when ETP=Local service with host network backend is configured makes sure that the flows are updated with new IP address (update kubelet first, the IP address later)": "[sig-network][Suite:openshift/conformance/parallel]",

	"Node IP and MAC address migration when the node IPv6 address is updated when ETP=Local service with host network backend is configured makes sure that the flows are updated with new IP address (update the IP address first, kubelet later)": "[sig-network][Suite:openshift/conformance/parallel]",

	"Node IP and MAC address migration when the node IPv6 address is updated when EgressIPs are configured makes sure that the EgressIP is still operational (update kubelet first, the IP address later)": "[sig-network][Suite:openshift/conformance/parallel]",

	"Node IP and MAC address migration when the node IPv6 address is updated when EgressIPs are configured makes sure that the EgressIP is still operational (update the IP address first, kubelet later)": "[sig-network][Suite:openshift/conformance/parallel]",

	"Node IP and MAC address migration when the node IPv6 address is updated when no EgressIPs are configured makes sure that the cluster is still operational (update kubelet first, the IP address later)": "[sig-network][Suite:openshift/conformance/parallel]",

	"Node IP and MAC address migration when the node IPv6 address is updated when no EgressIPs are configured makes sure that the cluster is still operational (update the IP address first, kubelet later)": "[sig-network][Suite:openshift/conformance/parallel]",

	"Node IP and MAC address migration when when MAC address changes when a nodeport service is configured Ensures flows are updated when MAC address changes": "[sig-network][Suite:openshift/conformance/parallel]",

	"OVS CPU affinity pinning can be enabled on specific nodes by creating enable_dynamic_cpu_affinity file": "[sig-network][Suite:openshift/conformance/parallel]",

	"Pod to external server PMTUD when a client ovnk pod targeting an external server is created when tests are run towards the agnhost echo server queries to the hostNetworked server pod on another node shall work for TCP": "[sig-network][Suite:openshift/conformance/parallel]",

	"Pod to external server PMTUD when a client ovnk pod targeting an external server is created when tests are run towards the agnhost echo server queries to the hostNetworked server pod on another node shall work for UDP": "[sig-network][Suite:openshift/conformance/parallel]",

	"Pod to pod TCP with low MTU when a client ovnk pod targeting an ovnk pod server(running on another node) with low mtu when MTU is lowered between the two nodes large queries to the server pod on another node shall work for TCP": "[sig-network][Suite:openshift/conformance/parallel]",

	"Service Hairpin SNAT Should ensure service hairpin traffic is NOT SNATed to hairpin masquerade IP; GR LB": "[sig-network][Suite:openshift/conformance/parallel]",

	"Service Hairpin SNAT Should ensure service hairpin traffic is SNATed to hairpin masquerade IP; Switch LB": "[sig-network][Suite:openshift/conformance/parallel]",

	"Services All service features work when manually listening on a non-default address": "[sig-network][Suite:openshift/conformance/parallel]",

	"Services Allow connection to an external IP using a source port that is equal to a node port": "[sig-network][Suite:openshift/conformance/parallel]",

	"Services Creates a host-network service, and ensures that host-network pods can connect to it": "[sig-network][Suite:openshift/conformance/parallel]",

	"Services Creates a service with session-affinity, and ensures it works after backend deletion": "[sig-network][Suite:openshift/conformance/parallel]",

	"Services does not use host masquerade address as source IP address when communicating externally": "[sig-network][Suite:openshift/conformance/parallel]",

	"Services of type NodePort should handle IP fragments": "[sig-network][Suite:openshift/conformance/parallel]",

	"Services of type NodePort should listen on each host addresses": "[sig-network][Suite:openshift/conformance/parallel]",

	"Services of type NodePort should work on secondary node interfaces for ETP=local and ETP=cluster when backend pods are also served by EgressIP": "[sig-network][Suite:openshift/conformance/parallel]",

	"Services when a nodePort service targeting a pod with hostNetwork:false is created when tests are run towards the agnhost echo service queries to the nodePort service shall work for TCP": "[sig-network][Suite:openshift/conformance/parallel]",

	"Services when a nodePort service targeting a pod with hostNetwork:false is created when tests are run towards the agnhost echo service queries to the nodePort service shall work for UDP": "[sig-network][Suite:openshift/conformance/parallel]",

	"Services when a nodePort service targeting a pod with hostNetwork:true is created when tests are run towards the agnhost echo service queries to the nodePort service shall work for TCP": "[sig-network][Suite:openshift/conformance/parallel]",

	"Services when a nodePort service targeting a pod with hostNetwork:true is created when tests are run towards the agnhost echo service queries to the nodePort service shall work for UDP": "[sig-network][Suite:openshift/conformance/parallel]",

	"Status manager validation Should validate the egress firewall status when adding a new zone": "[sig-network][Suite:openshift/conformance/parallel]",

	"Status manager validation Should validate the egress firewall status when adding an unknown zone": "[sig-network][Suite:openshift/conformance/parallel]",

	"Unidling Should generate a NeedPods event for traffic destined to idled services": "[sig-network][Suite:openshift/conformance/parallel]",

	"Unidling With annotated service Should connect to an unidled backend at the first attempt": "[sig-network][Suite:openshift/conformance/parallel]",

	"Unidling With annotated service Should generate a NeedPods event for traffic destined to idled services": "[sig-network][Suite:openshift/conformance/parallel]",

	"Unidling With annotated service Should generate a NeedPods event when backends were added and then removed": "[sig-network][Suite:openshift/conformance/parallel]",

	"Unidling With annotated service Should not generate a NeedPods event when has backend": "[sig-network][Suite:openshift/conformance/parallel]",

	"Unidling With annotated service Should not generate a NeedPods event when removing the annotation": "[sig-network][Suite:openshift/conformance/parallel]",

	"Unidling With non annotated service Should generate a NeedPods event when adding the annotation": "[sig-network][Suite:openshift/conformance/parallel]",

	"Unidling With non annotated service Should not generate a NeedPods event for traffic destined to idled services": "[sig-network][Suite:openshift/conformance/parallel]",

	"Unidling With non annotated service Should not generate a NeedPods event when backends were added and then removed": "[sig-network][Suite:openshift/conformance/parallel]",

	"Unidling With non annotated service Should not generate a NeedPods event when has backend": "[sig-network][Suite:openshift/conformance/parallel]",

	"e2e EgressQoS validation Should deny resources with bad values": "[sig-network][Suite:openshift/conformance/parallel]",

	"e2e EgressQoS validation Should validate correct DSCP value on EgressQoS resource changes ipv4 pod after resource": "[sig-network][Suite:openshift/conformance/parallel]",

	"e2e EgressQoS validation Should validate correct DSCP value on EgressQoS resource changes ipv4 pod before resource": "[sig-network][Suite:openshift/conformance/parallel]",

	"e2e EgressQoS validation Should validate correct DSCP value on EgressQoS resource changes ipv6 pod after resource": "[sig-network][Suite:openshift/conformance/parallel]",

	"e2e EgressQoS validation Should validate correct DSCP value on EgressQoS resource changes ipv6 pod before resource": "[sig-network][Suite:openshift/conformance/parallel]",

	"e2e EgressQoS validation Should validate correct DSCP value on pod labels changes ipv4 pod": "[sig-network][Suite:openshift/conformance/parallel]",

	"e2e EgressQoS validation Should validate correct DSCP value on pod labels changes ipv6 pod": "[sig-network][Suite:openshift/conformance/parallel]",

	"e2e br-int flow monitoring export validation Should validate flow data of br-int is sent to an external gateway with netflow v5": "[sig-network][Suite:openshift/conformance/parallel]",

	"e2e br-int flow monitoring export validation Should validate flow data of br-int is sent to an external gateway with sflow": "[sig-network][Suite:openshift/conformance/parallel]",

	"e2e control plane should provide Internet connection continuously when all ovnkube-control-plane pods are killed": "[sig-network][Suite:openshift/conformance/parallel]",

	"e2e control plane should provide Internet connection continuously when all pods are killed on node running master instance of ovnkube-control-plane": "[sig-network][Suite:openshift/conformance/parallel]",

	"e2e control plane should provide Internet connection continuously when ovnkube-node pod is killed": "[sig-network][Suite:openshift/conformance/parallel]",

	"e2e control plane should provide Internet connection continuously when pod running master instance of ovnkube-control-plane is killed": "[sig-network][Suite:openshift/conformance/parallel]",

	"e2e control plane should provide connection to external host by DNS name from a pod": "[sig-network][Suite:openshift/conformance/parallel]",

	"e2e control plane test node readiness according to its defaults interface MTU size should get node not ready with a too small MTU": "[sig-network][Suite:openshift/conformance/parallel]",

	"e2e control plane test node readiness according to its defaults interface MTU size should get node ready with a big enough MTU": "[sig-network][Suite:openshift/conformance/parallel]",

	"e2e delete databases Should validate connectivity before and after deleting all the db-pods at once in HA mode": "[sig-network][Suite:openshift/conformance/parallel]",

	"e2e delete databases Should validate connectivity before and after deleting all the db-pods at once in Non-HA mode": "[sig-network][Suite:openshift/conformance/parallel]",

	"e2e delete databases recovering from deleting db files while maintaining connectivity when deleting both db files on ovnkube-db-0": "[sig-network][Suite:openshift/conformance/parallel]",

	"e2e delete databases recovering from deleting db files while maintaining connectivity when deleting both db files on ovnkube-db-1": "[sig-network][Suite:openshift/conformance/parallel]",

	"e2e delete databases recovering from deleting db files while maintaining connectivity when deleting both db files on ovnkube-db-2": "[sig-network][Suite:openshift/conformance/parallel]",

	"e2e egress IP validation Cluster Default Network Should re-assign egress IPs when node readiness / reachability goes down/up": "[sig-network][Suite:openshift/conformance/parallel]",

	"e2e egress IP validation Cluster Default Network Should validate egress IP logic when one pod is managed by more than one egressIP object": "[sig-network][Suite:openshift/conformance/parallel]",

	"e2e egress IP validation Cluster Default Network Should validate the egress IP SNAT functionality for stateful-sets": "[sig-network][Suite:openshift/conformance/parallel]",

	"e2e egress IP validation Cluster Default Network Should validate the egress IP functionality against remote hosts with egress firewall applied": "[sig-network][Suite:openshift/conformance/parallel]",

	"e2e egress IP validation Cluster Default Network [OVN network] Should validate the egress IP SNAT functionality against host-networked pods": "[sig-network][Suite:openshift/conformance/parallel]",

	"e2e egress IP validation Cluster Default Network [OVN network] Using different methods to disable a node's availability for egress Should validate the egress IP functionality against remote hosts disabling egress nodes impeding GRCP health check": "[sig-network][Suite:openshift/conformance/parallel]",

	"e2e egress IP validation Cluster Default Network [OVN network] Using different methods to disable a node's availability for egress Should validate the egress IP functionality against remote hosts disabling egress nodes impeding Legacy health check": "[sig-network][Suite:openshift/conformance/parallel]",

	"e2e egress IP validation Cluster Default Network [OVN network] Using different methods to disable a node's availability for egress Should validate the egress IP functionality against remote hosts disabling egress nodes with egress-assignable label": "[sig-network][Suite:openshift/conformance/parallel]",

	"e2e egress IP validation Cluster Default Network [OVN network] multiple namespaces sharing a role primary network": "[sig-network][Suite:openshift/conformance/parallel]",

	"e2e egress IP validation Cluster Default Network [OVN network] multiple namespaces with different primary networks L2 Primary UDN": "[sig-network][Suite:openshift/conformance/parallel]",

	"e2e egress IP validation Cluster Default Network [OVN network] multiple namespaces with different primary networks L3 Primary UDN": "[sig-network][Suite:openshift/conformance/parallel]",

	"e2e egress IP validation Cluster Default Network [secondary-host-eip] Multiple EgressIP objects and their Egress IP hosted on the same interface": "[sig-network][Suite:openshift/conformance/parallel]",

	"e2e egress IP validation Cluster Default Network [secondary-host-eip] Using different methods to disable a node or pod availability for egress IPv4": "[sig-network][Suite:openshift/conformance/parallel]",

	"e2e egress IP validation Cluster Default Network [secondary-host-eip] Using different methods to disable a node or pod availability for egress IPv6 compressed": "[sig-network][Suite:openshift/conformance/parallel]",

	"e2e egress IP validation Cluster Default Network [secondary-host-eip] Using different methods to disable a node or pod availability for egress IPv6 uncompressed": "[sig-network][Suite:openshift/conformance/parallel]",

	"e2e egress IP validation Cluster Default Network [secondary-host-eip] Using different methods to disable a node or pod availability for egress": "[sig-network][Suite:openshift/conformance/parallel]",

	"e2e egress IP validation Cluster Default Network [secondary-host-eip] uses VRF routing table if EIP assigned interface is VRF slave": "[sig-network][Suite:openshift/conformance/parallel]",

	"e2e egress IP validation Cluster Default Network of replies to egress IP packets that require fragmentation [LGW][IPv4]": "[sig-network][Suite:openshift/conformance/parallel]",

	"e2e egress IP validation Network Segmentation: IPv4 L2 role primary Should re-assign egress IPs when node readiness / reachability goes down/up": "[sig-network][OCPFeatureGate:NetworkSegmentation][Feature:UserDefinedPrimaryNetworks][Suite:openshift/conformance/parallel]",

	"e2e egress IP validation Network Segmentation: IPv4 L2 role primary Should validate egress IP logic when one pod is managed by more than one egressIP object": "[sig-network][OCPFeatureGate:NetworkSegmentation][Feature:UserDefinedPrimaryNetworks][Suite:openshift/conformance/parallel]",

	"e2e egress IP validation Network Segmentation: IPv4 L2 role primary Should validate the egress IP SNAT functionality for stateful-sets": "[sig-network][OCPFeatureGate:NetworkSegmentation][Feature:UserDefinedPrimaryNetworks][Suite:openshift/conformance/parallel]",

	"e2e egress IP validation Network Segmentation: IPv4 L2 role primary Should validate the egress IP functionality against remote hosts with egress firewall applied": "[sig-network][OCPFeatureGate:NetworkSegmentation][Feature:UserDefinedPrimaryNetworks][Suite:openshift/conformance/parallel]",

	"e2e egress IP validation Network Segmentation: IPv4 L2 role primary [OVN network] Should validate the egress IP SNAT functionality against host-networked pods": "[sig-network][OCPFeatureGate:NetworkSegmentation][Feature:UserDefinedPrimaryNetworks][Suite:openshift/conformance/parallel]",

	"e2e egress IP validation Network Segmentation: IPv4 L2 role primary [OVN network] Using different methods to disable a node's availability for egress Should validate the egress IP functionality against remote hosts disabling egress nodes impeding GRCP health check": "[sig-network][OCPFeatureGate:NetworkSegmentation][Feature:UserDefinedPrimaryNetworks][Suite:openshift/conformance/parallel]",

	"e2e egress IP validation Network Segmentation: IPv4 L2 role primary [OVN network] Using different methods to disable a node's availability for egress Should validate the egress IP functionality against remote hosts disabling egress nodes impeding Legacy health check": "[sig-network][OCPFeatureGate:NetworkSegmentation][Feature:UserDefinedPrimaryNetworks][Suite:openshift/conformance/parallel]",

	"e2e egress IP validation Network Segmentation: IPv4 L2 role primary [OVN network] Using different methods to disable a node's availability for egress Should validate the egress IP functionality against remote hosts disabling egress nodes with egress-assignable label": "[sig-network][OCPFeatureGate:NetworkSegmentation][Feature:UserDefinedPrimaryNetworks][Suite:openshift/conformance/parallel]",

	"e2e egress IP validation Network Segmentation: IPv4 L2 role primary [OVN network] multiple namespaces sharing a role primary network": "[sig-network][OCPFeatureGate:NetworkSegmentation][Feature:UserDefinedPrimaryNetworks][Suite:openshift/conformance/parallel]",

	"e2e egress IP validation Network Segmentation: IPv4 L2 role primary [OVN network] multiple namespaces with different primary networks L2 Primary UDN": "[sig-network][OCPFeatureGate:NetworkSegmentation][Feature:UserDefinedPrimaryNetworks][Suite:openshift/conformance/parallel]",

	"e2e egress IP validation Network Segmentation: IPv4 L2 role primary [OVN network] multiple namespaces with different primary networks L3 Primary UDN": "[sig-network][OCPFeatureGate:NetworkSegmentation][Feature:UserDefinedPrimaryNetworks][Suite:openshift/conformance/parallel]",

	"e2e egress IP validation Network Segmentation: IPv4 L2 role primary [secondary-host-eip] Multiple EgressIP objects and their Egress IP hosted on the same interface": "[sig-network][OCPFeatureGate:NetworkSegmentation][Feature:UserDefinedPrimaryNetworks][Suite:openshift/conformance/parallel]",

	"e2e egress IP validation Network Segmentation: IPv4 L2 role primary [secondary-host-eip] Using different methods to disable a node or pod availability for egress IPv4": "[sig-network][OCPFeatureGate:NetworkSegmentation][Feature:UserDefinedPrimaryNetworks][Suite:openshift/conformance/parallel]",

	"e2e egress IP validation Network Segmentation: IPv4 L2 role primary [secondary-host-eip] Using different methods to disable a node or pod availability for egress IPv6 compressed": "[sig-network][OCPFeatureGate:NetworkSegmentation][Feature:UserDefinedPrimaryNetworks][Suite:openshift/conformance/parallel]",

	"e2e egress IP validation Network Segmentation: IPv4 L2 role primary [secondary-host-eip] Using different methods to disable a node or pod availability for egress IPv6 uncompressed": "[sig-network][OCPFeatureGate:NetworkSegmentation][Feature:UserDefinedPrimaryNetworks][Suite:openshift/conformance/parallel]",

	"e2e egress IP validation Network Segmentation: IPv4 L2 role primary [secondary-host-eip] Using different methods to disable a node or pod availability for egress": "[sig-network][OCPFeatureGate:NetworkSegmentation][Feature:UserDefinedPrimaryNetworks][Suite:openshift/conformance/parallel]",

	"e2e egress IP validation Network Segmentation: IPv4 L2 role primary [secondary-host-eip] uses VRF routing table if EIP assigned interface is VRF slave": "[sig-network][OCPFeatureGate:NetworkSegmentation][Feature:UserDefinedPrimaryNetworks][Suite:openshift/conformance/parallel]",

	"e2e egress IP validation Network Segmentation: IPv4 L2 role primary of replies to egress IP packets that require fragmentation [LGW][IPv4]": "[sig-network][OCPFeatureGate:NetworkSegmentation][Feature:UserDefinedPrimaryNetworks][Suite:openshift/conformance/parallel]",

	"e2e egress IP validation Network Segmentation: IPv4 L3 role primary Should re-assign egress IPs when node readiness / reachability goes down/up": "[sig-network][OCPFeatureGate:NetworkSegmentation][Feature:UserDefinedPrimaryNetworks][Suite:openshift/conformance/parallel]",

	"e2e egress IP validation Network Segmentation: IPv4 L3 role primary Should validate egress IP logic when one pod is managed by more than one egressIP object": "[sig-network][OCPFeatureGate:NetworkSegmentation][Feature:UserDefinedPrimaryNetworks][Suite:openshift/conformance/parallel]",

	"e2e egress IP validation Network Segmentation: IPv4 L3 role primary Should validate the egress IP SNAT functionality for stateful-sets": "[sig-network][OCPFeatureGate:NetworkSegmentation][Feature:UserDefinedPrimaryNetworks][Suite:openshift/conformance/parallel]",

	"e2e egress IP validation Network Segmentation: IPv4 L3 role primary Should validate the egress IP functionality against remote hosts with egress firewall applied": "[sig-network][OCPFeatureGate:NetworkSegmentation][Feature:UserDefinedPrimaryNetworks][Suite:openshift/conformance/parallel]",

	"e2e egress IP validation Network Segmentation: IPv4 L3 role primary [OVN network] Should validate the egress IP SNAT functionality against host-networked pods": "[sig-network][OCPFeatureGate:NetworkSegmentation][Feature:UserDefinedPrimaryNetworks][Suite:openshift/conformance/parallel]",

	"e2e egress IP validation Network Segmentation: IPv4 L3 role primary [OVN network] Using different methods to disable a node's availability for egress Should validate the egress IP functionality against remote hosts disabling egress nodes impeding GRCP health check": "[sig-network][OCPFeatureGate:NetworkSegmentation][Feature:UserDefinedPrimaryNetworks][Suite:openshift/conformance/parallel]",

	"e2e egress IP validation Network Segmentation: IPv4 L3 role primary [OVN network] Using different methods to disable a node's availability for egress Should validate the egress IP functionality against remote hosts disabling egress nodes impeding Legacy health check": "[sig-network][OCPFeatureGate:NetworkSegmentation][Feature:UserDefinedPrimaryNetworks][Suite:openshift/conformance/parallel]",

	"e2e egress IP validation Network Segmentation: IPv4 L3 role primary [OVN network] Using different methods to disable a node's availability for egress Should validate the egress IP functionality against remote hosts disabling egress nodes with egress-assignable label": "[sig-network][OCPFeatureGate:NetworkSegmentation][Feature:UserDefinedPrimaryNetworks][Suite:openshift/conformance/parallel]",

	"e2e egress IP validation Network Segmentation: IPv4 L3 role primary [OVN network] multiple namespaces sharing a role primary network": "[sig-network][OCPFeatureGate:NetworkSegmentation][Feature:UserDefinedPrimaryNetworks][Suite:openshift/conformance/parallel]",

	"e2e egress IP validation Network Segmentation: IPv4 L3 role primary [OVN network] multiple namespaces with different primary networks L2 Primary UDN": "[sig-network][OCPFeatureGate:NetworkSegmentation][Feature:UserDefinedPrimaryNetworks][Suite:openshift/conformance/parallel]",

	"e2e egress IP validation Network Segmentation: IPv4 L3 role primary [OVN network] multiple namespaces with different primary networks L3 Primary UDN": "[sig-network][OCPFeatureGate:NetworkSegmentation][Feature:UserDefinedPrimaryNetworks][Suite:openshift/conformance/parallel]",

	"e2e egress IP validation Network Segmentation: IPv4 L3 role primary [secondary-host-eip] Multiple EgressIP objects and their Egress IP hosted on the same interface": "[sig-network][OCPFeatureGate:NetworkSegmentation][Feature:UserDefinedPrimaryNetworks][Suite:openshift/conformance/parallel]",

	"e2e egress IP validation Network Segmentation: IPv4 L3 role primary [secondary-host-eip] Using different methods to disable a node or pod availability for egress IPv4": "[sig-network][OCPFeatureGate:NetworkSegmentation][Feature:UserDefinedPrimaryNetworks][Suite:openshift/conformance/parallel]",

	"e2e egress IP validation Network Segmentation: IPv4 L3 role primary [secondary-host-eip] Using different methods to disable a node or pod availability for egress IPv6 compressed": "[sig-network][OCPFeatureGate:NetworkSegmentation][Feature:UserDefinedPrimaryNetworks][Suite:openshift/conformance/parallel]",

	"e2e egress IP validation Network Segmentation: IPv4 L3 role primary [secondary-host-eip] Using different methods to disable a node or pod availability for egress IPv6 uncompressed": "[sig-network][OCPFeatureGate:NetworkSegmentation][Feature:UserDefinedPrimaryNetworks][Suite:openshift/conformance/parallel]",

	"e2e egress IP validation Network Segmentation: IPv4 L3 role primary [secondary-host-eip] Using different methods to disable a node or pod availability for egress": "[sig-network][OCPFeatureGate:NetworkSegmentation][Feature:UserDefinedPrimaryNetworks][Suite:openshift/conformance/parallel]",

	"e2e egress IP validation Network Segmentation: IPv4 L3 role primary [secondary-host-eip] uses VRF routing table if EIP assigned interface is VRF slave": "[sig-network][OCPFeatureGate:NetworkSegmentation][Feature:UserDefinedPrimaryNetworks][Suite:openshift/conformance/parallel]",

	"e2e egress IP validation Network Segmentation: IPv4 L3 role primary of replies to egress IP packets that require fragmentation [LGW][IPv4]": "[sig-network][OCPFeatureGate:NetworkSegmentation][Feature:UserDefinedPrimaryNetworks][Suite:openshift/conformance/parallel]",

	"e2e egress IP validation Network Segmentation: IPv6 L2 role primary Should re-assign egress IPs when node readiness / reachability goes down/up": "[sig-network][OCPFeatureGate:NetworkSegmentation][Feature:UserDefinedPrimaryNetworks][Suite:openshift/conformance/parallel]",

	"e2e egress IP validation Network Segmentation: IPv6 L2 role primary Should validate egress IP logic when one pod is managed by more than one egressIP object": "[sig-network][OCPFeatureGate:NetworkSegmentation][Feature:UserDefinedPrimaryNetworks][Suite:openshift/conformance/parallel]",

	"e2e egress IP validation Network Segmentation: IPv6 L2 role primary Should validate the egress IP SNAT functionality for stateful-sets": "[sig-network][OCPFeatureGate:NetworkSegmentation][Feature:UserDefinedPrimaryNetworks][Suite:openshift/conformance/parallel]",

	"e2e egress IP validation Network Segmentation: IPv6 L2 role primary Should validate the egress IP functionality against remote hosts with egress firewall applied": "[sig-network][OCPFeatureGate:NetworkSegmentation][Feature:UserDefinedPrimaryNetworks][Suite:openshift/conformance/parallel]",

	"e2e egress IP validation Network Segmentation: IPv6 L2 role primary [OVN network] Should validate the egress IP SNAT functionality against host-networked pods": "[sig-network][OCPFeatureGate:NetworkSegmentation][Feature:UserDefinedPrimaryNetworks][Suite:openshift/conformance/parallel]",

	"e2e egress IP validation Network Segmentation: IPv6 L2 role primary [OVN network] Using different methods to disable a node's availability for egress Should validate the egress IP functionality against remote hosts disabling egress nodes impeding GRCP health check": "[sig-network][OCPFeatureGate:NetworkSegmentation][Feature:UserDefinedPrimaryNetworks][Suite:openshift/conformance/parallel]",

	"e2e egress IP validation Network Segmentation: IPv6 L2 role primary [OVN network] Using different methods to disable a node's availability for egress Should validate the egress IP functionality against remote hosts disabling egress nodes impeding Legacy health check": "[sig-network][OCPFeatureGate:NetworkSegmentation][Feature:UserDefinedPrimaryNetworks][Suite:openshift/conformance/parallel]",

	"e2e egress IP validation Network Segmentation: IPv6 L2 role primary [OVN network] Using different methods to disable a node's availability for egress Should validate the egress IP functionality against remote hosts disabling egress nodes with egress-assignable label": "[sig-network][OCPFeatureGate:NetworkSegmentation][Feature:UserDefinedPrimaryNetworks][Suite:openshift/conformance/parallel]",

	"e2e egress IP validation Network Segmentation: IPv6 L2 role primary [OVN network] multiple namespaces sharing a role primary network": "[sig-network][OCPFeatureGate:NetworkSegmentation][Feature:UserDefinedPrimaryNetworks][Suite:openshift/conformance/parallel]",

	"e2e egress IP validation Network Segmentation: IPv6 L2 role primary [OVN network] multiple namespaces with different primary networks L2 Primary UDN": "[sig-network][OCPFeatureGate:NetworkSegmentation][Feature:UserDefinedPrimaryNetworks][Suite:openshift/conformance/parallel]",

	"e2e egress IP validation Network Segmentation: IPv6 L2 role primary [OVN network] multiple namespaces with different primary networks L3 Primary UDN": "[sig-network][OCPFeatureGate:NetworkSegmentation][Feature:UserDefinedPrimaryNetworks][Suite:openshift/conformance/parallel]",

	"e2e egress IP validation Network Segmentation: IPv6 L2 role primary [secondary-host-eip] Multiple EgressIP objects and their Egress IP hosted on the same interface": "[sig-network][OCPFeatureGate:NetworkSegmentation][Feature:UserDefinedPrimaryNetworks][Suite:openshift/conformance/parallel]",

	"e2e egress IP validation Network Segmentation: IPv6 L2 role primary [secondary-host-eip] Using different methods to disable a node or pod availability for egress IPv4": "[sig-network][OCPFeatureGate:NetworkSegmentation][Feature:UserDefinedPrimaryNetworks][Suite:openshift/conformance/parallel]",

	"e2e egress IP validation Network Segmentation: IPv6 L2 role primary [secondary-host-eip] Using different methods to disable a node or pod availability for egress IPv6 compressed": "[sig-network][OCPFeatureGate:NetworkSegmentation][Feature:UserDefinedPrimaryNetworks][Suite:openshift/conformance/parallel]",

	"e2e egress IP validation Network Segmentation: IPv6 L2 role primary [secondary-host-eip] Using different methods to disable a node or pod availability for egress IPv6 uncompressed": "[sig-network][OCPFeatureGate:NetworkSegmentation][Feature:UserDefinedPrimaryNetworks][Suite:openshift/conformance/parallel]",

	"e2e egress IP validation Network Segmentation: IPv6 L2 role primary [secondary-host-eip] Using different methods to disable a node or pod availability for egress": "[sig-network][OCPFeatureGate:NetworkSegmentation][Feature:UserDefinedPrimaryNetworks][Suite:openshift/conformance/parallel]",

	"e2e egress IP validation Network Segmentation: IPv6 L2 role primary [secondary-host-eip] uses VRF routing table if EIP assigned interface is VRF slave": "[sig-network][OCPFeatureGate:NetworkSegmentation][Feature:UserDefinedPrimaryNetworks][Suite:openshift/conformance/parallel]",

	"e2e egress IP validation Network Segmentation: IPv6 L2 role primary of replies to egress IP packets that require fragmentation [LGW][IPv4]": "[sig-network][OCPFeatureGate:NetworkSegmentation][Feature:UserDefinedPrimaryNetworks][Suite:openshift/conformance/parallel]",

	"e2e egress IP validation Network Segmentation: IPv6 L3 role primary Should re-assign egress IPs when node readiness / reachability goes down/up": "[sig-network][OCPFeatureGate:NetworkSegmentation][Feature:UserDefinedPrimaryNetworks][Suite:openshift/conformance/parallel]",

	"e2e egress IP validation Network Segmentation: IPv6 L3 role primary Should validate egress IP logic when one pod is managed by more than one egressIP object": "[sig-network][OCPFeatureGate:NetworkSegmentation][Feature:UserDefinedPrimaryNetworks][Suite:openshift/conformance/parallel]",

	"e2e egress IP validation Network Segmentation: IPv6 L3 role primary Should validate the egress IP SNAT functionality for stateful-sets": "[sig-network][OCPFeatureGate:NetworkSegmentation][Feature:UserDefinedPrimaryNetworks][Suite:openshift/conformance/parallel]",

	"e2e egress IP validation Network Segmentation: IPv6 L3 role primary Should validate the egress IP functionality against remote hosts with egress firewall applied": "[sig-network][OCPFeatureGate:NetworkSegmentation][Feature:UserDefinedPrimaryNetworks][Suite:openshift/conformance/parallel]",

	"e2e egress IP validation Network Segmentation: IPv6 L3 role primary [OVN network] Should validate the egress IP SNAT functionality against host-networked pods": "[sig-network][OCPFeatureGate:NetworkSegmentation][Feature:UserDefinedPrimaryNetworks][Suite:openshift/conformance/parallel]",

	"e2e egress IP validation Network Segmentation: IPv6 L3 role primary [OVN network] Using different methods to disable a node's availability for egress Should validate the egress IP functionality against remote hosts disabling egress nodes impeding GRCP health check": "[sig-network][OCPFeatureGate:NetworkSegmentation][Feature:UserDefinedPrimaryNetworks][Suite:openshift/conformance/parallel]",

	"e2e egress IP validation Network Segmentation: IPv6 L3 role primary [OVN network] Using different methods to disable a node's availability for egress Should validate the egress IP functionality against remote hosts disabling egress nodes impeding Legacy health check": "[sig-network][OCPFeatureGate:NetworkSegmentation][Feature:UserDefinedPrimaryNetworks][Suite:openshift/conformance/parallel]",

	"e2e egress IP validation Network Segmentation: IPv6 L3 role primary [OVN network] Using different methods to disable a node's availability for egress Should validate the egress IP functionality against remote hosts disabling egress nodes with egress-assignable label": "[sig-network][OCPFeatureGate:NetworkSegmentation][Feature:UserDefinedPrimaryNetworks][Suite:openshift/conformance/parallel]",

	"e2e egress IP validation Network Segmentation: IPv6 L3 role primary [OVN network] multiple namespaces sharing a role primary network": "[sig-network][OCPFeatureGate:NetworkSegmentation][Feature:UserDefinedPrimaryNetworks][Suite:openshift/conformance/parallel]",

	"e2e egress IP validation Network Segmentation: IPv6 L3 role primary [OVN network] multiple namespaces with different primary networks L2 Primary UDN": "[sig-network][OCPFeatureGate:NetworkSegmentation][Feature:UserDefinedPrimaryNetworks][Suite:openshift/conformance/parallel]",

	"e2e egress IP validation Network Segmentation: IPv6 L3 role primary [OVN network] multiple namespaces with different primary networks L3 Primary UDN": "[sig-network][OCPFeatureGate:NetworkSegmentation][Feature:UserDefinedPrimaryNetworks][Suite:openshift/conformance/parallel]",

	"e2e egress IP validation Network Segmentation: IPv6 L3 role primary [secondary-host-eip] Multiple EgressIP objects and their Egress IP hosted on the same interface": "[sig-network][OCPFeatureGate:NetworkSegmentation][Feature:UserDefinedPrimaryNetworks][Suite:openshift/conformance/parallel]",

	"e2e egress IP validation Network Segmentation: IPv6 L3 role primary [secondary-host-eip] Using different methods to disable a node or pod availability for egress IPv4": "[sig-network][OCPFeatureGate:NetworkSegmentation][Feature:UserDefinedPrimaryNetworks][Suite:openshift/conformance/parallel]",

	"e2e egress IP validation Network Segmentation: IPv6 L3 role primary [secondary-host-eip] Using different methods to disable a node or pod availability for egress IPv6 compressed": "[sig-network][OCPFeatureGate:NetworkSegmentation][Feature:UserDefinedPrimaryNetworks][Suite:openshift/conformance/parallel]",

	"e2e egress IP validation Network Segmentation: IPv6 L3 role primary [secondary-host-eip] Using different methods to disable a node or pod availability for egress IPv6 uncompressed": "[sig-network][OCPFeatureGate:NetworkSegmentation][Feature:UserDefinedPrimaryNetworks][Suite:openshift/conformance/parallel]",

	"e2e egress IP validation Network Segmentation: IPv6 L3 role primary [secondary-host-eip] Using different methods to disable a node or pod availability for egress": "[sig-network][OCPFeatureGate:NetworkSegmentation][Feature:UserDefinedPrimaryNetworks][Suite:openshift/conformance/parallel]",

	"e2e egress IP validation Network Segmentation: IPv6 L3 role primary [secondary-host-eip] uses VRF routing table if EIP assigned interface is VRF slave": "[sig-network][OCPFeatureGate:NetworkSegmentation][Feature:UserDefinedPrimaryNetworks][Suite:openshift/conformance/parallel]",

	"e2e egress IP validation Network Segmentation: IPv6 L3 role primary of replies to egress IP packets that require fragmentation [LGW][IPv4]": "[sig-network][OCPFeatureGate:NetworkSegmentation][Feature:UserDefinedPrimaryNetworks][Suite:openshift/conformance/parallel]",

	"e2e egress firewall policy validation with external containers Should validate that egressfirewall supports DNS name in caps": "[sig-network][Suite:openshift/conformance/parallel]",

	"e2e egress firewall policy validation with external containers Should validate the egress firewall allows inbound connections": "[sig-network][Suite:openshift/conformance/parallel]",

	"e2e egress firewall policy validation with external containers Should validate the egress firewall doesn't affect internal connections": "[sig-network][Suite:openshift/conformance/parallel]",

	"e2e egress firewall policy validation with external containers Should validate the egress firewall policy functionality for allowed CIDR and port": "[sig-network][Suite:openshift/conformance/parallel]",

	"e2e egress firewall policy validation with external containers Should validate the egress firewall policy functionality for allowed IP": "[sig-network][Suite:openshift/conformance/parallel]",

	"e2e ingress to host-networked pods traffic validation Validating ingress traffic to Host Networked pods with externalTrafficPolicy=local Should be allowed to node local host-networked endpoints by nodeport services": "[sig-network][Suite:openshift/conformance/parallel]",

	"e2e ingress traffic validation Validating ingress traffic Should be allowed by externalip services": "[sig-network][Suite:openshift/conformance/parallel]",

	"e2e ingress traffic validation Validating ingress traffic Should be allowed by nodeport services after upgrade to DualStack": "[sig-network][Suite:openshift/conformance/parallel]",

	"e2e ingress traffic validation Validating ingress traffic Should be allowed by nodeport services": "[sig-network][Suite:openshift/conformance/parallel]",

	"e2e ingress traffic validation Validating ingress traffic Should be allowed to node local cluster-networked endpoints by nodeport services with externalTrafficPolicy=local": "[sig-network][Suite:openshift/conformance/parallel]",

	"e2e ingress traffic validation Validating ingress traffic to manually added node IPs Should be allowed by externalip services to a new node ip": "[sig-network][Suite:openshift/conformance/parallel]",

	"e2e network policy hairpinning validation Should validate the hairpinned traffic is always allowed": "[sig-network][Suite:openshift/conformance/parallel]",

	"test e2e inter-node connectivity between worker nodes Should validate connectivity within a namespace of pods on separate nodes": "[sig-network][Suite:openshift/conformance/parallel]",

	"test e2e pod connectivity to host addresses Should validate connectivity from a pod to a non-node host address on same node": "[sig-network][Suite:openshift/conformance/parallel]",
}

func init() {
	ginkgo.GetSuite().SetAnnotateFn(func(name string, node types.TestSpec) {
		if newLabels, ok := Annotations[name]; ok {
			node.AppendText(newLabels)
		} else {
			panic(fmt.Sprintf("unable to find test %s", name))
		}
	})
}
