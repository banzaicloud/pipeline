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

package config

import (
	"github.com/banzaicloud/pipeline/internal/platform/database"
	"github.com/pkg/errors"
)

// CasbinDSN returns the connection string for Casbin gorm adapter.
func CasbinDSN() (string, error) {
	config := NewDBConfig()

	err := config.Validate()
	if err != nil {
		return "", errors.Wrap(err, "invalid database config")
	}

	dsn, err := database.GetDSN(config)
	if err != nil {
		return "", errors.Wrap(err, "could not get DSN for casbin gorm adapter")
	}

	return dsn, nil
}
