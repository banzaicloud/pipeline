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
	"fmt"

	"emperror.dev/errors"
	"github.com/banzaicloud/pipeline/dns"
	"github.com/banzaicloud/pipeline/internal/clusterfeature"
	"github.com/mitchellh/mapstructure"
)

type featureSpec struct {
	Grafana      grafanaSpec      `json:"grafana" mapstructure:"grafana"`
	Alertmanager alertManagerSpec `json:"alertmanager" mapstructure:"alertmanager"`
	Prometheus   prometheusSpec   `json:"prometheus" mapstructure:"prometheus"`
}

type baseSpec struct {
	Enabled bool        `json:"enabled" mapstructure:"enabled"`
	Public  ingressSpec `json:"public" mapstructure:"public"`
}

type grafanaSpec struct {
	baseSpec `mapstructure:",squash"`
	SecretId string `json:"secretId" mapstructure:"secretId"`
}

type alertManagerSpec struct {
	baseSpec `mapstructure:",squash"`
	Provider providerSpec `json:"provider" mapstructure:"provider"`
}

type prometheusSpec struct {
	baseSpec `mapstructure:",squash"`
	SecretId string `json:"secretId" mapstructure:"secretId"`
}

type providerSpec struct {
	Slack     slackPropertiesSpec     `json:"slack" mapstructure:"slack"`
	Pagerduty pagerdutyPropertiesSpec `json:"pagerduty" mapstructure:"pagerduty"`
	Email     emailPropertiesSpec     `json:"email" mapstructure:"email"`
}

type slackPropertiesSpec struct {
	Enabled      bool `json:"enabled" mapstructure:"enabled"`
	ApiUrl       string `json:"apiUrl" mapstructure:"apiUrl"`
	Channel      string `json:"channel" mapstructure:"channel"`
	SendResolved bool   `json:"sendResolved" mapstructure:"sendResolved"`
}

type emailPropertiesSpec struct {
	Enabled      bool `json:"enabled" mapstructure:"enabled"`
	To           string `json:"to" mapstructure:"to"`
	From         string `json:"from" mapstructure:"from"`
	SendResolved bool   `json:"sendResolved" mapstructure:"sendResolved"`
}

type pagerdutyPropertiesSpec struct {
	Enabled      bool `json:"enabled" mapstructure:"enabled"`
	RoutingKey   string `json:"routingKey" mapstructure:"routingKey"`
	ServiceKey   string `json:"serviceKey" mapstructure:"serviceKey"`
	Url          string `json:"url" mapstructure:"url"`
	SendResolved bool   `json:"sendResolved" mapstructure:"sendResolved"`
}

type ingressSpec struct {
	Enabled bool   `json:"enabled" mapstructure:"enabled"`
	Domain  string `json:"domain" mapstructure:"domain"`
	Path    string `json:"path" mapstructure:"path"`
}

type requiredFieldError struct {
	fieldName string
}

type invalidIngressHost struct {
	hostType string
}

func (e invalidIngressHost) Error() string {
	return fmt.Sprintf("invalid %s ingress host", e.hostType)
}

func (e requiredFieldError) Error() string {
	return fmt.Sprintf("%q cannot be empty", e.fieldName)
}

func (s ingressSpec) Validate(ingressType string) error {
	if len(s.Path) == 0 {
		return requiredFieldError{fieldName: fmt.Sprintf("%s path", ingressType)}
	}

	if len(s.Domain) != 0 {
		err := dns.ValidateSubdomain(s.Domain)
		if err != nil {
			return errors.Append(err, invalidIngressHost{hostType: ingressType})
		}
	}

	return nil
}

func (s featureSpec) Validate() error {
	// Grafana spec validation
	if s.Grafana.Enabled && s.Grafana.Public.Enabled {
		if err := s.Grafana.Public.Validate(ingressTypeGrafana); err != nil {
			return err
		}
	}

	// Prometheus spec validation
	if s.Prometheus.Enabled && s.Prometheus.Public.Enabled {
		if err := s.Prometheus.Public.Validate(ingressTypePrometheus); err != nil {
			return err
		}
	}

	// Alertmanager spec validation
	if s.Alertmanager.Enabled && s.Alertmanager.Public.Enabled {
		if err := s.Alertmanager.Public.Validate(ingressTypeAlertmanager); err != nil {
			return err
		}
	}

	return nil
}

func bindFeatureSpec(spec clusterfeature.FeatureSpec) (featureSpec, error) {
	var boundSpec featureSpec
	if err := mapstructure.Decode(spec, &boundSpec); err != nil {
		return boundSpec, clusterfeature.InvalidFeatureSpecError{
			FeatureName: featureName,
			Problem:     errors.WrapIf(err, "failed to bind feature spec").Error(),
		}
	}
	return boundSpec, nil
}
