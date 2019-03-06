// Copyright © 2019 Banzai Cloud
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

package cadence

import (
	"fmt"

	"github.com/pkg/errors"
)

// Config holds information necessary for connecting to Cadence.
type Config struct {
	// Cadence connection details
	Host string
	Port int

	Domain                                 string
	WorkflowExecutionRetentionPeriodInDays int32
	CreateNonexistentDomain                bool

	// Client identity (not used for now; falls back to machine host name)
	Identity string
}

// Validate checks that the configuration is valid.
func (c Config) Validate() error {
	if c.Host == "" {
		return errors.New("cadence host is required")
	}

	if c.Port == 0 {
		return errors.New("cadence port is required")
	}

	if c.Domain == "" {
		return errors.New("cadence domain is required")
	}

	return nil
}

// Addr returns the Cadence connection address.
func (c Config) Addr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}
