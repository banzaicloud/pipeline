// Copyright © 2018 Banzai Cloud
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

package pke

import (
	"database/sql/driver"
	"fmt"

	"github.com/spf13/cast"
)

// Network is the schema for the DB.
type Network struct {
	Model

	ServiceCIDR         string               `yaml:"serviceCIDR" gorm:"column:service_cidr"`
	PodCIDR             string               `yaml:"podCIDR" gorm:"column:pod_cidr"`
	Provider            NetworkProvider      `yaml:"provider"`
	APIServerAddress    string               `yaml:"apiServerAddress"`
	CloudProvider       CloudNetworkProvider `yaml:"cloudProvider" gorm:"column:cloud_provider"`
	CloudProviderConfig Config               `yaml:"cloudProviderConfig" gorm:"column:cloud_provider_config;type:text"`
}

// TableName changes the default table name.
func (Network) TableName() string {
	return "topology_networks"
}

func (n Network) String() string {
	return fmt.Sprintf(
		"ID: %d, CreatedAt: %v, CreatedBy: %d, ClusterID: %d, ServiceCIDR: %s, PodCIDR: %s, Provider: %s, APIServerAddress: %s",
		n.ID,
		n.CreatedAt,
		n.CreatedBy,
		n.ClusterID,
		n.ServiceCIDR,
		n.PodCIDR,
		n.Provider,
		n.APIServerAddress,
	)
}

// NetworkProvider is the schema for the DB.
type NetworkProvider string

const (
	NPWeave NetworkProvider = "weave" // Weave network provider.
)

// Value implements the driver.Valuer interface
func (n NetworkProvider) Value() (driver.Value, error) {
	return string(n), nil
}

// Scan implements the sql.Scanner interface
func (n *NetworkProvider) Scan(src interface{}) error {
	value, err := cast.ToStringE(src)
	*n = NetworkProvider(value)
	return err
}

// NetworkProvider is the schema for the DB.
type CloudNetworkProvider string

const (
	CNPAmazon CloudNetworkProvider = "ec2" // Amazon EC2 network provider.
)

// Value implements the driver.Valuer interface
func (n CloudNetworkProvider) Value() (driver.Value, error) {
	return string(n), nil
}

// Scan implements the sql.Scanner interface
func (n *CloudNetworkProvider) Scan(src interface{}) error {
	value, err := cast.ToStringE(src)
	*n = CloudNetworkProvider(value)
	return err
}
