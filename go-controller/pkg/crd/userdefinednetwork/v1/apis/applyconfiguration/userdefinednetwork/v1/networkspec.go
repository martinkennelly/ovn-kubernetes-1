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
	v1 "github.com/ovn-org/ovn-kubernetes/go-controller/pkg/crd/userdefinednetwork/v1"
)

// NetworkSpecApplyConfiguration represents a declarative configuration of the NetworkSpec type for use
// with apply.
type NetworkSpecApplyConfiguration struct {
	Topology *v1.NetworkTopology             `json:"topology,omitempty"`
	Layer3   *Layer3ConfigApplyConfiguration `json:"layer3,omitempty"`
	Layer2   *Layer2ConfigApplyConfiguration `json:"layer2,omitempty"`
}

// NetworkSpecApplyConfiguration constructs a declarative configuration of the NetworkSpec type for use with
// apply.
func NetworkSpec() *NetworkSpecApplyConfiguration {
	return &NetworkSpecApplyConfiguration{}
}

// WithTopology sets the Topology field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Topology field is set to the value of the last call.
func (b *NetworkSpecApplyConfiguration) WithTopology(value v1.NetworkTopology) *NetworkSpecApplyConfiguration {
	b.Topology = &value
	return b
}

// WithLayer3 sets the Layer3 field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Layer3 field is set to the value of the last call.
func (b *NetworkSpecApplyConfiguration) WithLayer3(value *Layer3ConfigApplyConfiguration) *NetworkSpecApplyConfiguration {
	b.Layer3 = value
	return b
}

// WithLayer2 sets the Layer2 field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Layer2 field is set to the value of the last call.
func (b *NetworkSpecApplyConfiguration) WithLayer2(value *Layer2ConfigApplyConfiguration) *NetworkSpecApplyConfiguration {
	b.Layer2 = value
	return b
}
