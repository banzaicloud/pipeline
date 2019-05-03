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

package database

import (
	"fmt"

	database "github.com/banzaicloud/bank-vaults/pkg/db"
	"github.com/pkg/errors"
)

// GetDSN returns a DSN string from a config.
func GetDSN(c Config) (string, error) {
	var dsn string

	switch c.Dialect {
	case "mysql":
		dsn = fmt.Sprintf("@tcp(%s:%d)/%s", c.Host, c.Port, c.Name)

		if c.Role != "" {
			var err error

			dsn, err = database.DynamicSecretDataSource(c.Dialect, c.Role+dsn)
			if err != nil {
				return "", err
			}
		} else {
			dsn = fmt.Sprintf("%s:%s%s", c.User, c.Pass, dsn)
		}

	case "postgres":
		dsn = fmt.Sprintf("@%s:%d/%s", c.Host, c.Port, c.Name)

		if c.Role != "" {
			var err error

			dsn, err = database.DynamicSecretDataSource(c.Dialect, "postgresql://"+c.Role+dsn)
			if err != nil {
				return "", err
			}
		} else {
			dsn = fmt.Sprintf("postgresql://%s:%s%s", c.User, c.Pass, dsn)
		}

	default:
		return "", errors.Errorf("unsupported db dialect: %s", c.Dialect)
	}

	var params string

	if len(c.Params) > 0 {
		var query string

		for key, value := range c.Params {
			if query != "" {
				query += "&"
			}

			query += key + "=" + value
		}

		params = "?" + query
	}

	dsn = dsn + params

	return dsn, nil
}
