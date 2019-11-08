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

import "testing"

func TestConfig_Validate(t *testing.T) {
	tests := map[string]Config{
		"database host is required": {
			Dialect:  "postgres",
			Port:     3306,
			User:     "root",
			Password: "",
			Name:     "database",
		},
		"database port is required": {
			Dialect:  "mysql",
			Host:     "localhost",
			User:     "root",
			Password: "",
			Name:     "database",
		},
		"database user is required if no secret role is provided": {
			Dialect:  "postgres",
			Host:     "localhost",
			Port:     3306,
			Password: "",
			Name:     "database",
		},
		"database name is required": {
			Dialect:  "mysql",
			Host:     "localhost",
			Port:     3306,
			User:     "root",
			Password: "",
		},
		"database dialect is required": {
			Host:     "localhost",
			Port:     3306,
			User:     "root",
			Password: "",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			err := test.Validate()

			if err.Error() != name {
				t.Errorf("expected error %q, got: %q", name, err.Error())
			}
		})
	}
}

func TestGetDSN(t *testing.T) {

	configs := map[string]Config{
		"root:@tcp(host:3306)/database?parseTime=true": {
			Dialect:  "mysql",
			Host:     "host",
			Port:     3306,
			User:     "root",
			Password: "",
			Name:     "database",
			Params: map[string]string{
				"parseTime": "true",
			},
		},
		"postgresql://root:@host:5432/database": {
			Dialect:  "postgres",
			Host:     "host",
			Port:     5432,
			User:     "root",
			Password: "",
			Name:     "database",
		},
	}

	for expectedDsn, config := range configs {
		dsn, err := GetDSN(config)
		if err != nil {
			t.Fatal("unexpected error:", err.Error())
		}

		if dsn != expectedDsn {
			t.Errorf("expected DSN to be %q, got: %q", expectedDsn, dsn)
		}
	}
}
