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

package banzaicloud

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"

	"github.com/jinzhu/gorm"
)

type NodePools []NodePool

type NodePool struct {
	Model

	Name             string                 `yaml:"name" gorm:"unique_index:idx_cluster_id_name"`
	Roles            Roles                  `yaml:"roles" gorm:"-"`
	Hosts            Hosts                  `yaml:"hosts" gorm:"foreignkey:Name"`
	Provider         NodePoolProvider       `yaml:"provider"`
	ProviderConfig   map[string]interface{} `yaml:"providerConfig" gorm:"-"`
	ProviderConfigDB string                 `gorm:"column:provider_config;type:text"`
}

// TableName changes the default table name.
func (NodePool) TableName() string {
	return "topology_nodepools"
}

func (n NodePool) String() string {
	return fmt.Sprintf(
		"ID: %d, CreatedAt: %v, CreatedBy: %d, ClusterID: %d, Name: %s, Roles: %s, Hosts: %s, Provider: %s",
		n.ID,
		n.CreatedAt,
		n.CreatedBy,
		n.ClusterID,
		n.Name,
		n.Roles,
		n.Hosts,
		n.Provider,
	)
}

// BeforeCreate marshals fields.
func (n *NodePool) BeforeCreate(scope *gorm.Scope) error {
	j, err := json.Marshal(n.ProviderConfig)
	if err != nil {
		return err
	}
	return scope.SetColumn("ProviderConfigDB", string(j))
}

func (n *NodePool) BeforeUpdate(scope *gorm.Scope) error {
	return n.BeforeCreate(scope)
}

// AfterFind unmarshals fields.
func (n *NodePool) AfterFind() error {
	var cfg map[string]interface{}
	err := json.Unmarshal([]byte(n.ProviderConfigDB), &cfg)
	if err != nil {
		return err
	}
	n.ProviderConfig = cfg
	return nil
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
	Model

	Name             string `yaml:"name"`
	PrivateIP        string `yaml:"privateIP"`
	NetworkInterface string `yaml:"networkInterface"`
	Roles            Roles  `yaml:"roles" gorm:"-"`
	RolesDB          string `gorm:"column:roles;type:varchar(255)"`
	Labels           Labels `yaml:"labels" gorm:"-"`
	LabelsDB         string `gorm:"column:labels;type:varchar(255)"`
	Taints           Taints `yaml:"taint" gorm:"-"`
	TaintsDB         string `gorm:"column:taints;type:varchar(255)"`
}

// TableName changes the default table name.
func (Host) TableName() string {
	return "cluster_topology_nodepool_host"
}

// String prints row contents.
func (h Host) String() string {
	return fmt.Sprintf(
		"ID: %d, createdAt: %v, createdBy: %d, ClusterID: %d, Name: %s, PrivateIP: %s, NetworkInterface: %s, Roles: %s, Labels: %s, Taints: %s",
		h.ID,
		h.CreatedAt,
		h.CreatedBy,
		h.ClusterID,
		h.Name,
		h.PrivateIP,
		h.NetworkInterface,
		h.Roles,
		h.Labels,
		h.Taints,
	)
}

// BeforeCreate marshals fields.
func (h *Host) BeforeCreate(scope *gorm.Scope) error {
	var (
		roles, labels, taints []byte
		err                   error
	)

	roles, err = json.Marshal(h.Roles)
	if err != nil {
		return err
	}
	labels, err = json.Marshal(h.Labels)
	if err != nil {
		return err
	}
	taints, err = json.Marshal(h.Taints)
	if err != nil {
		return err
	}

	err = scope.SetColumn("Roles", string(roles))
	if err != nil {
		return err
	}
	err = scope.SetColumn("Labels", string(labels))
	if err != nil {
		return err
	}
	err = scope.SetColumn("Taints", string(taints))
	if err != nil {
		return err
	}

	return nil
}

// BeforeUpdate marshals fields.
func (h *Host) BeforeUpdate(scope *gorm.Scope) error {
	return h.BeforeCreate(scope)
}

// AfterFind unmarshals fields.
func (h *Host) AfterFind() error {
	var (
		roles  Roles
		labels Labels
		taints Taints
		err    error
	)

	err = json.Unmarshal([]byte(h.RolesDB), &roles)
	if err != nil {
		return err
	}
	err = json.Unmarshal([]byte(h.LabelsDB), &labels)
	if err != nil {
		return err
	}
	err = json.Unmarshal([]byte(h.TaintsDB), &taints)
	if err != nil {
		return err
	}

	h.Roles = roles
	h.Labels = labels
	h.Taints = taints

	return nil
}

type Labels map[string]string

type Taints []Taint
type Taint string
