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
	"fmt"

	"github.com/jinzhu/gorm"
)

// Kubernetes is the schema for the DB.
type Kubernetes struct {
	Model

	Version     string `yaml:"version"`
	RBAC        RBAC   `yaml:"rbac" gorm:"-"`
	OIDC        OIDC   `yaml:"oidc" gorm:"-"`
	RBACEnabled bool   `gorm:"column:rbac_enabled"`
}

// TableName changes the default table name.
func (Kubernetes) TableName() string {
	return "topology_kubernetes"
}

func (k Kubernetes) String() string {
	return fmt.Sprintf(
		"ID: %d, CreatedAt: %v, CreatedBy: %d, ClusterID: %d, Version: %s, RBAC.Enabled: %t, OIDC.Enabled: %t",
		k.ID,
		k.CreatedAt,
		k.CreatedBy,
		k.ClusterID,
		k.Version,
		k.RBAC.Enabled,
		k.OIDC.Enabled,
	)
}

// BeforeCreate marshals fields.
func (k *Kubernetes) BeforeCreate(scope *gorm.Scope) error {
	err := scope.SetColumn("OIDCEnabled", k.OIDC.Enabled)
	if err != nil {
		return err
	}
	return scope.SetColumn("RBACEnabled", k.RBAC.Enabled)
}

func (k *Kubernetes) BeforeUpdate(scope *gorm.Scope) error {
	return k.BeforeCreate(scope)
}

// AfterFind unmarshals fields.
func (k *Kubernetes) AfterFind() error {
	k.RBAC.Enabled = k.RBACEnabled
	return nil
}

// RBAC is the schema for the DB.
type RBAC struct {
	Enabled bool `yaml:"enabled"`
}

// OIDC is the schema for the DB.
type OIDC struct {
	Enabled bool `yaml:"enabled"`
}
