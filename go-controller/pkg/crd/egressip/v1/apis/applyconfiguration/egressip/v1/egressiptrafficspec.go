/*


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
// Code generated by applyconfiguration-gen. DO NOT EDIT.

package v1

import (
	v1 "github.com/ovn-org/ovn-kubernetes/go-controller/pkg/crd/egressip/v1"
)

// EgressIPTrafficSpecApplyConfiguration represents an declarative configuration of the EgressIPTrafficSpec type for use
// with apply.
type EgressIPTrafficSpecApplyConfiguration struct {
	DestinationNetworks []v1.CIDR `json:"destinationNetworks,omitempty"`
}

// EgressIPTrafficSpecApplyConfiguration constructs an declarative configuration of the EgressIPTrafficSpec type for use with
// apply.
func EgressIPTrafficSpec() *EgressIPTrafficSpecApplyConfiguration {
	return &EgressIPTrafficSpecApplyConfiguration{}
}

// WithDestinationNetworks adds the given value to the DestinationNetworks field in the declarative configuration
// and returns the receiver, so that objects can be build by chaining "With" function invocations.
// If called multiple times, values provided by each call will be appended to the DestinationNetworks field.
func (b *EgressIPTrafficSpecApplyConfiguration) WithDestinationNetworks(values ...v1.CIDR) *EgressIPTrafficSpecApplyConfiguration {
	for i := range values {
		b.DestinationNetworks = append(b.DestinationNetworks, values[i])
	}
	return b
}
