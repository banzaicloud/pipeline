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

	"github.com/banzaicloud/pipeline/internal/database/sql/json"
)

// KubeADM is the schema for the DB.
type KubeADM struct {
	Model

	ExtraArgs ExtraArgs `yaml:"extraArgs" gorm:"type:json"`
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

// ExtraArgs is the schema for the DB.
type ExtraArgs []ExtraArg

func (m *ExtraArgs) Scan(src interface{}) error {
	return json.Scan(src, m)
}

func (m ExtraArgs) Value() (driver.Value, error) {
	return json.Value(m)
}

// ExtraArg is the schema for the DB.
type ExtraArg string

// Value implements the driver.Valuer interface
func (e ExtraArg) Value() (driver.Value, error) {
	return string(e), nil
}

// Scan implements the sql.Scanner interface
func (e *ExtraArg) Scan(src interface{}) error {
	value, err := cast.ToStringE(src)
	*e = ExtraArg(value)
	return err
}
