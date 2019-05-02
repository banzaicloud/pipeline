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

import "github.com/pkg/errors"

// Config holds information necessary for connecting to a database.
type Config struct {
	Dialect string
	Host    string
	Port    int
	User    string
	Pass    string
	Name    string

	Role string

	Params map[string]string

	EnableLog bool
}

// Validate checks that the configuration is valid.
func (c Config) Validate() error {
	if c.Dialect == "" {
		return errors.New("database dialect is required")
	}

	if c.Host == "" {
		return errors.New("database host is required")
	}

	if c.Port == 0 {
		return errors.New("database port is required")
	}

	if c.Role == "" {
		if c.User == "" {
			return errors.New("database user is required if no secret role is provided")
		}
	}

	if c.Name == "" {
		return errors.New("database name is required")
	}

	return nil
}
