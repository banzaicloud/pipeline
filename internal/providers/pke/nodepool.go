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
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	pkgAuth "github.com/banzaicloud/pipeline/pkg/auth"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
)

type NodePools []NodePool

type NodePool struct {
	NodePoolID uint `gorm:"primary_key;name:id"`
	CreatedAt  time.Time
	CreatedBy  pkgAuth.UserID

	ClusterID pkgCluster.ClusterID `gorm:"foreignkey:ClusterIDl;association_foreignkey:ClusterID;unique_index:idx_cluster_id_name"`

	Name           string            `yaml:"name" gorm:"unique_index:idx_cluster_id_name"`
	Roles          Roles             `yaml:"roles" gorm:"type:varchar(255)"`
	Hosts          Hosts             `yaml:"hosts" gorm:"foreignkey:NodePoolID"`
	Provider       NodePoolProvider  `yaml:"provider"`
	ProviderConfig Config            `yaml:"providerConfig" gorm:"column:provider_config;type:text"`
	Labels         map[string]string `yaml:"labels" gorm:"-"`
}

// TableName changes the default table name.
func (NodePool) TableName() string {
	return "topology_nodepools"
}

func (n NodePool) String() string {
	return fmt.Sprintf(
		"ID: %d, CreatedAt: %v, CreatedBy: %d, ClusterID: %d, Name: %s, Roles: %s, Hosts: %s, Provider: %s",
		n.NodePoolID,
		n.CreatedAt,
		n.CreatedBy,
		n.ClusterID,
		n.Name,
		n.Roles,
		n.Hosts,
		n.Provider,
	)
}

type NodePoolProvider string

const (
	NPPAmazon NodePoolProvider = "amazon"
)

var _ driver.Valuer = (*NodePoolProvider)(nil)

// Value implements the driver.Valuer interface
func (n NodePoolProvider) Value() (driver.Value, error) {
	return string(n), nil
}

var _ sql.Scanner = (*NodePoolProvider)(nil)

// Scan implements the sql.Scanner interface
func (n *NodePoolProvider) Scan(src interface{}) error {
	*n = NodePoolProvider(string(src.([]uint8)))
	return nil
}

type Roles []Role
type Role string

const (
	RoleMaster         Role = "master"
	RoleWorker         Role = "worker"
	RolePipelineSystem Role = "pipeline-system"
)

var _ driver.Valuer = (*Roles)(nil)

// Value implements the driver.Valuer interface
func (n Roles) Value() (driver.Value, error) {
	r, err := json.Marshal(n)
	if err != nil {
		return "", err
	}
	return string(r), nil
}

var _ sql.Scanner = (*Roles)(nil)

// Scan implements the sql.Scanner interface
func (n *Roles) Scan(src interface{}) error {
	return json.Unmarshal([]byte(string(src.([]uint8))), &n)
}

var _ driver.Valuer = (*Role)(nil)

// Value implements the driver.Valuer interface
func (r Role) Value() (driver.Value, error) {
	return string(r), nil
}

var _ sql.Scanner = (*Role)(nil)

// Scan implements the sql.Scanner interface
func (r *Role) Scan(src interface{}) error {
	*r = Role(string(src.(string)))
	return nil
}

type Hosts []Host
type Host struct {
	ID        uint `gorm:"primary_key"`
	CreatedAt time.Time
	CreatedBy pkgAuth.UserID

	NodePoolID uint `gorm:"name:nodepool_id;foreignkey:NodePoolID"`

	Name             string `yaml:"name"`
	PrivateIP        string `yaml:"privateIP"`
	NetworkInterface string `yaml:"networkInterface"`
	Roles            Roles  `yaml:"roles" gorm:"type:varchar(255)"`
	Labels           Labels `yaml:"labels" gorm:"type:varchar(255)"`
	Taints           Taints `yaml:"taint" gorm:"type:varchar(255)"`
}

// TableName changes the default table name.
func (Host) TableName() string {
	return "topology_nodepool_hosts"
}

// String prints row contents.
func (h Host) String() string {
	return fmt.Sprintf(
		"ID: %d, createdAt: %v, createdBy: %d, NodePoolID: %d, Name: %s, PrivateIP: %s, NetworkInterface: %s, Roles: %s, Labels: %s, Taints: %s",
		h.ID,
		h.CreatedAt,
		h.CreatedBy,
		h.NodePoolID,
		h.Name,
		h.PrivateIP,
		h.NetworkInterface,
		h.Roles,
		h.Labels,
		h.Taints,
	)
}

type Labels map[string]string

var _ driver.Valuer = (*Labels)(nil)

// Value implements the driver.Valuer interface
func (n Labels) Value() (driver.Value, error) {
	r, err := json.Marshal(n)
	if err != nil {
		return "", err
	}
	return string(r), nil
}

var _ sql.Scanner = (*Labels)(nil)

// Scan implements the sql.Scanner interface
func (n *Labels) Scan(src interface{}) error {
	return json.Unmarshal([]byte(string(src.([]uint8))), &n)
}

type Taints []Taint
type Taint string

var _ driver.Valuer = (*Taints)(nil)

// Value implements the driver.Valuer interface
func (n Taints) Value() (driver.Value, error) {
	r, err := json.Marshal(n)
	if err != nil {
		return "", err
	}
	return string(r), nil
}

var _ sql.Scanner = (*Taints)(nil)

// Scan implements the sql.Scanner interface
func (n *Taints) Scan(src interface{}) error {
	return json.Unmarshal([]byte(string(src.([]uint8))), &n)
}
