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

package monitoring

import (
	"emperror.dev/errors"
)

// Config contains configuration for the monitoring feature.
type Config struct {
	Namespace string
	Grafana   GrafanaConfig
	Charts    ChartsConfig
	Images    ImagesConfig
}

func (c Config) Validate() error {
	if c.Namespace == "" {
		return errors.New("monitoring namespace is required")
	}

	if err := c.Grafana.Validate(); err != nil {
		return err
	}

	if err := c.Charts.Operator.Validate(); err != nil {
		return errors.WrapIf(err, "error during validation Prometheus operator config")
	}

	if err := c.Charts.Pushgateway.Validate(); err != nil {
		return errors.WrapIf(err, "error during validation Pushgateway config")
	}

	if err := c.Images.Prometheus.Validate(); err != nil {
		return errors.WrapIf(err, "error during validate Prometheus images config")
	}

	if err := c.Images.Alertmanager.Validate(); err != nil {
		return errors.WrapIf(err, "error during validate Alertmanager images config")
	}

	if err := c.Images.Grafana.Validate(); err != nil {
		return errors.WrapIf(err, "error during validate Grafana images config")
	}

	if err := c.Images.Nodeexporter.Validate(); err != nil {
		return errors.WrapIf(err, "error during validate NodeExporter images config")
	}

	if err := c.Images.Kubestatemetrics.Validate(); err != nil {
		return errors.WrapIf(err, "error during validate KubeStateMetrics images config")
	}

	if err := c.Images.Operator.Validate(); err != nil {
		return errors.WrapIf(err, "error during validate Operator images config")
	}

	if err := c.Images.Pushgateway.Validate(); err != nil {
		return errors.WrapIf(err, "error during validate Pushgateway images config")
	}

	return nil
}

type GrafanaConfig struct {
	AdminUser string
}

func (c GrafanaConfig) Validate() error {
	if c.AdminUser == "" {
		return errors.New("monitoring grafana username is required")
	}

	return nil
}

type ChartsConfig struct {
	Operator    ChartConfig
	Pushgateway ChartConfig
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
	Operator         ImageConfig
	Prometheus       ImageConfig
	Alertmanager     ImageConfig
	Grafana          ImageConfig
	Kubestatemetrics ImageConfig
	Nodeexporter     ImageConfig
	Pushgateway      ImageConfig
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
