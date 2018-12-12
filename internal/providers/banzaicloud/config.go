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
)

type Config map[string]interface{}

var _ driver.Valuer = (*Config)(nil)

// Value implements the driver.Valuer interface
func (n Config) Value() (driver.Value, error) {
	r, err := json.Marshal(n)
	if err != nil {
		return "", err
	}
	return string(r), nil
}

var _ sql.Scanner = (*Config)(nil)

// Scan implements the sql.Scanner interface
func (n *Config) Scan(src interface{}) error {
	return json.Unmarshal([]byte(string(src.([]uint8))), &n)
}
