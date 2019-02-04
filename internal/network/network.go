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

package network

// Network is an interface that cloud specific VPC network implementations must implement
type Network interface {
	CIDRs() []string
	ID() string
	Name() string
}

// Subnet is an interface that cloud specific VPC subnetwork implementations must implement
type Subnet interface {
	CIDRs() []string
	ID() string
	Location() string
	Name() string
}

// RouteTable is an interface that cloud specific VPC route table implementations must implement
type RouteTable interface {
	ID() string
	Name() string
}

// Service defines the interface of provider specific network service implementations
type Service interface {
	ListNetworks() ([]Network, error)
	ListRouteTables(networkID string) ([]RouteTable, error)
	ListSubnets(networkID string) ([]Subnet, error)
}
