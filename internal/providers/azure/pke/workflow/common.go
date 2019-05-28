// Copyright © 2019 Banzai Cloud
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package workflow

// ResourceIDProvider describes the interface of the provider of a single resource ID
type ResourceIDProvider interface {
	Get() string
}

// ConstantResourceIDProvider implements a ResourceIDProvider with a constant value
type ConstantResourceIDProvider string

// Get returns the constant resource ID stored in the provider
func (p ConstantResourceIDProvider) Get() string {
	return string(p)
}

// ResourceIDByNameProvider describes the interface of the by-name provider of a resource ID
type ResourceIDByNameProvider interface {
	Get(name string) string
}

// MapResourceIDByNameProvider implements a ResourceIDByNameProvider with a map of string to string
type MapResourceIDByNameProvider map[string]string

// Get returns the resource ID associated with the specified name in the mapping of the provider
func (p MapResourceIDByNameProvider) Get(name string) string {
	return p[name]
}

// Put associates the specified name and resource ID pair in the mapping of the provider
func (p MapResourceIDByNameProvider) Put(name string, resourceID string) {
	p[name] = resourceID
}

// IPAddressProvider describes the interface of the provider of a single IP address
type IPAddressProvider interface {
	Get() string
}

// ConstantIPAddressProvider implements an IPAddressProvider with a constant value
type ConstantIPAddressProvider string

// Get returns the constant IP address stored in the provider
func (p ConstantIPAddressProvider) Get() string {
	return string(p)
}
