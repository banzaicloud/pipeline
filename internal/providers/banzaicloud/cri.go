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

// CRI is the schema for the DB.
type CRI struct {
	Model

	Runtime         Runtime                `yaml:"runtime"`
	RuntimeConfig   map[string]interface{} `yaml:"runtimeConfig" gorm:"-"`
	RuntimeConfigDB string                 `gorm:"column:runtime_config;type:text"`
}

// TableName changes the default table name.
func (CRI) TableName() string {
	return "topology_cris"
}

func (c CRI) String() string {
	return fmt.Sprintf(
		"ID: %d, CreatedAt: %v, CreatedBy: %d, ClusterID: %d, Runtime: %s",
		c.ID,
		c.CreatedAt,
		c.CreatedBy,
		c.ClusterID,
		c.Runtime,
	)
}

// BeforeCreate marshals fields.
func (c *CRI) BeforeCreate(scope *gorm.Scope) error {
	j, err := json.Marshal(c.RuntimeConfig)
	if err != nil {
		return err
	}
	return scope.SetColumn("RuntimeConfigDB", string(j))
}

// BeforeUpdate marshals fields.
func (c *CRI) BeforeUpdate(scope *gorm.Scope) error {
	return c.BeforeCreate(scope)
}

// AfterFind unmarshals fields.
func (c *CRI) AfterFind() error {
	var cfg map[string]interface{}
	err := json.Unmarshal([]byte(c.RuntimeConfigDB), &cfg)
	if err != nil {
		return err
	}
	c.RuntimeConfig = cfg
	return nil
}

// Runtime is the schema for the DB.
type Runtime string

const (
	CRIDocker     Runtime = "docker"
	CRIContainerd Runtime = "containerd"
)

var _ driver.Valuer = (*Runtime)(nil)

// Value implements the driver.Valuer interface
func (n Runtime) Value() (driver.Value, error) {
	return string(n), nil
}

var _ sql.Scanner = (*Runtime)(nil)

// Scan implements the sql.Scanner interface
func (n *Runtime) Scan(src interface{}) error {
	*n = Runtime(string(src.([]uint8)))
	return nil
}
