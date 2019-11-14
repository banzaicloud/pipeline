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
	Images    ImagesConfig
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

	if err := c.Images.Operator.Validate(); err != nil {
		return errors.WrapIf(err, "error during validation operator image config")
	}

	if err := c.Images.Loki.Validate(); err != nil {
		return errors.WrapIf(err, "error during validation loki image config")
	}

	if err := c.Images.Fluentbit.Validate(); err != nil {
		return errors.WrapIf(err, "error during validation fluentbit image config")
	}

	if err := c.Images.Fluentd.Validate(); err != nil {
		return errors.WrapIf(err, "error during validation fluentd image config")
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

type ImagesConfig struct {
	Operator  ImageConfig
	Loki      ImageConfig
	Fluentbit ImageConfig
	Fluentd   ImageConfig
}

type ImageConfig struct {
	Repository string
	Tag        string
}

func (c ImageConfig) Validate() error {
	if c.Repository == "" {
		return errors.New("repository is required")
	}

	if c.Tag == "" {
		return errors.New("tag is required")
	}

	return nil
}
