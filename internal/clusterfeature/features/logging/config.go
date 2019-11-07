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

package logging

import (
	"emperror.dev/errors"
)

// Config contains configuration for the logging feature.
type Config struct {
	Namespace string
	Charts    ChartsConfig
}

func (c Config) Validate() error {
	if c.Namespace == "" {
		return errors.New("logging namespace is required")
	}

	if err := c.Charts.Operator.Validate(); err != nil {
		return errors.WrapIf(err, "error during validation logging operator config")
	}

	if err := c.Charts.Logging.Validate(); err != nil {
		return errors.WrapIf(err, "error during validation logging-operator-logging config")
	}

	if err := c.Charts.Loki.Validate(); err != nil {
		return errors.WrapIf(err, "error during validation loki chart config")
	}

	return nil
}

type ChartsConfig struct {
	Operator ChartConfig
	Logging  ChartConfig
	Loki     ChartConfig
}

type ChartConfig struct {
	Chart   string
	Version string
	Values  map[string]interface{}
}

func (c ChartConfig) Validate() error {
	if c.Chart == "" {
		return errors.New("chart is required")
	}

	if c.Version == "" {
		return errors.New("chart version is required")
	}

	return nil
}
