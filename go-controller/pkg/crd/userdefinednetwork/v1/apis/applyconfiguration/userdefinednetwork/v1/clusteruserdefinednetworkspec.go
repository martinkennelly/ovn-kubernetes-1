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
	v1 "k8s.io/client-go/applyconfigurations/meta/v1"
)

// ClusterUserDefinedNetworkSpecApplyConfiguration represents a declarative configuration of the ClusterUserDefinedNetworkSpec type for use
// with apply.
type ClusterUserDefinedNetworkSpecApplyConfiguration struct {
	NamespaceSelector *v1.LabelSelectorApplyConfiguration `json:"namespaceSelector,omitempty"`
	Network           *NetworkSpecApplyConfiguration      `json:"network,omitempty"`
}

// ClusterUserDefinedNetworkSpecApplyConfiguration constructs a declarative configuration of the ClusterUserDefinedNetworkSpec type for use with
// apply.
func ClusterUserDefinedNetworkSpec() *ClusterUserDefinedNetworkSpecApplyConfiguration {
	return &ClusterUserDefinedNetworkSpecApplyConfiguration{}
}

// WithNamespaceSelector sets the NamespaceSelector field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the NamespaceSelector field is set to the value of the last call.
func (b *ClusterUserDefinedNetworkSpecApplyConfiguration) WithNamespaceSelector(value *v1.LabelSelectorApplyConfiguration) *ClusterUserDefinedNetworkSpecApplyConfiguration {
	b.NamespaceSelector = value
	return b
}

// WithNetwork sets the Network field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Network field is set to the value of the last call.
func (b *ClusterUserDefinedNetworkSpecApplyConfiguration) WithNetwork(value *NetworkSpecApplyConfiguration) *ClusterUserDefinedNetworkSpecApplyConfiguration {
	b.Network = value
	return b
}
