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

	"github.com/jinzhu/gorm"
	"github.com/spf13/cast"
)

// KubeADM is the schema for the DB.
type KubeADM struct {
	Model

	ExtraArgs   ExtraArgs `yaml:"extraArgs" gorm:"-"`
	ExtraArgsDB string    `gorm:"column:extra_args;type:varchar(255)"`
}

// TableName changes the default table name.
func (KubeADM) TableName() string {
	return "topology_kubeadms"
}

func (k KubeADM) String() string {
	return fmt.Sprintf(
		"ID: %d, CreatedAt: %v, CreatedBy: %d, ClusterID: %d, ExtraArgs: %s",
		k.ID,
		k.CreatedAt,
		k.CreatedBy,
		k.ClusterID,
		k.ExtraArgs,
	)
}

// BeforeCreate marshals fields.
func (k *KubeADM) BeforeCreate(scope *gorm.Scope) error {
	var (
		extraArgs []byte
		err       error
	)

	extraArgs, err = json.Marshal(k.ExtraArgs)
	if err != nil {
		return err
	}
	err = scope.SetColumn("ExtraArgs", string(extraArgs))
	if err != nil {
		return err
	}

	return nil
}

// BeforeUpdate marshals fields.
func (k *KubeADM) BeforeUpdate(scope *gorm.Scope) error {
	return k.BeforeCreate(scope)
}

// AfterFind unmarshals fields.
func (k *KubeADM) AfterFind() error {
	var (
		extraArgs ExtraArgs
		err       error
	)

	err = json.Unmarshal([]byte(k.ExtraArgsDB), &extraArgs)
	if err != nil {
		return err
	}

	k.ExtraArgs = extraArgs

	return nil
}

// ExtraArgs is the schema for the DB.
type ExtraArgs []ExtraArg

// ExtraArg is the schema for the DB.
type ExtraArg string

var _ driver.Valuer = (*ExtraArg)(nil)

// Value implements the driver.Valuer interface
func (e ExtraArg) Value() (driver.Value, error) {
	return string(e), nil
}

var _ sql.Scanner = (*ExtraArg)(nil)

// Scan implements the sql.Scanner interface
func (e *ExtraArg) Scan(src interface{}) error {
	value, err := cast.ToStringE(src)
	*e = ExtraArg(value)
	return err
}
