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

// EgressIPStatusItemApplyConfiguration represents a declarative configuration of the EgressIPStatusItem type for use
// with apply.
type EgressIPStatusItemApplyConfiguration struct {
	Node     *string `json:"node,omitempty"`
	EgressIP *string `json:"egressIP,omitempty"`
}

// EgressIPStatusItemApplyConfiguration constructs a declarative configuration of the EgressIPStatusItem type for use with
// apply.
func EgressIPStatusItem() *EgressIPStatusItemApplyConfiguration {
	return &EgressIPStatusItemApplyConfiguration{}
}

// WithNode sets the Node field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Node field is set to the value of the last call.
func (b *EgressIPStatusItemApplyConfiguration) WithNode(value string) *EgressIPStatusItemApplyConfiguration {
	b.Node = &value
	return b
}

// WithEgressIP sets the EgressIP field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the EgressIP field is set to the value of the last call.
func (b *EgressIPStatusItemApplyConfiguration) WithEgressIP(value string) *EgressIPStatusItemApplyConfiguration {
	b.EgressIP = &value
	return b
}
