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
	"regexp"

	"emperror.dev/errors"
	"github.com/mitchellh/mapstructure"

	"github.com/banzaicloud/pipeline/dns"
	"github.com/banzaicloud/pipeline/internal/clusterfeature"
)

type featureSpec struct {
	Prometheus   prometheusSpec   `json:"prometheus" mapstructure:"prometheus"`
	Grafana      grafanaSpec      `json:"grafana" mapstructure:"grafana"`
	Exporters    exportersSpec    `json:"exporters" mapstructure:"exporters"`
	Alertmanager alertmanagerSpec `json:"alertmanager" mapstructure:"alertmanager"`
	Pushgateway  pushgatewaySpec  `json:"pushgateway" mapstructure:"pushgateway"`
}

type prometheusSpec struct {
	Enabled bool                  `json:"enabled" mapstructure:"enabled"`
	Storage storageSpec           `json:"storage" mapstructure:"storage"`
	Ingress ingressSpecWithSecret `json:"ingress" mapstructure:"ingress"`
}

type grafanaSpec struct {
	Enabled    bool            `json:"enabled" mapstructure:"enabled"`
	SecretId   string          `json:"secretId" mapstructure:"secretId"`
	Dashboards bool            `json:"defaultDashboards" mapstructure:"defaultDashboards"`
	Ingress    baseIngressSpec `json:"ingress" mapstructure:"ingress"`
}

type storageSpec struct {
	Class     string `json:"class" mapstructure:"class"`
	Size      uint   `json:"size" mapstructure:"size"`
	Retention string `json:"retention" mapstructure:"retention"`
}

type ingressSpecWithSecret struct {
	baseIngressSpec `mapstructure:",squash"`
	SecretID        string `json:"secretId" mapstructure:"secretId"`
}

type baseIngressSpec struct {
	Enabled bool   `json:"enabled" mapstructure:"enabled"`
	Domain  string `json:"domain" mapstructure:"domain"`
	Path    string `json:"path" mapstructure:"path"`
}

type exportersSpec struct {
	Enabled          bool             `json:"enabled" mapstructure:"enabled"`
	NodeExporter     exporterBaseSpec `json:"nodeExporter" mapstructure:"nodeExporter"`
	KubeStateMetrics exporterBaseSpec `json:"kubeStateMetrics" mapstructure:"kubeStateMetrics"`
}

type exporterBaseSpec struct {
	Enabled bool `json:"enabled" mapstructure:"enabled"`
}

type alertmanagerSpec struct {
	Enabled  bool                   `json:"enabled" mapstructure:"enabled"`
	Provider map[string]interface{} `json:"provider" mapstructure:"provider"`
	Ingress  ingressSpecWithSecret  `json:"ingress" mapstructure:"ingress"`
}

type pushgatewaySpec struct {
	Enabled bool `json:"enabled" mapstructure:"enabled"`
}

type pagerDutySpec struct {
	Enabled         bool   `json:"enabled" mapstructure:"enabled"`
	URL             string `json:"url" mapstructure:"url"`
	SecretID        string `json:"secretId" mapstructure:"secretId"`
	IntegrationType string `json:"integrationType" mapstructure:"integrationType"`
	SendResolved    bool   `json:"sendResolved" mapstructure:"sendResolved"`
}

type slackSpec struct {
	Enabled      bool   `json:"enabled" mapstructure:"enabled"`
	SecretID     string `json:"secretId" mapstructure:"secretId"`
	Channel      string `json:"channel" mapstructure:"channel"`
	SendResolved bool   `json:"sendResolved" mapstructure:"sendResolved"`
}

func (s featureSpec) Validate() error {
	// Prometheus validation
	if err := s.Prometheus.Validate(); err != nil {
		return err
	}

	// Grafana validation
	if err := s.Grafana.Validate(); err != nil {
		return err
	}

	// Alertmanager validation
	if err := s.Alertmanager.Validate(); err != nil {
		return err
	}

	if !s.Exporters.Enabled {
		return canNotDisabledError{fieldName: "exporters"}
	}

	if !s.Exporters.KubeStateMetrics.Enabled {
		return canNotDisabledError{fieldName: "kubeStateMetrics"}
	}

	if !s.Exporters.NodeExporter.Enabled {
		return canNotDisabledError{fieldName: "nodeExporter"}
	}

	return nil
}

func (s prometheusSpec) Validate() error {
	if !s.Enabled {
		// Prometheus cannot be disabled
		return canNotDisabledError{fieldName: "prometheus"}
	}

	// ingress validation
	if err := s.Ingress.Validate(ingressTypePrometheus); err != nil {
		return errors.WrapIf(err, "error during validate Prometheus ingress")
	}

	// storage validation
	if err := s.Storage.Validate(); err != nil {
		return err
	}

	return nil
}

func (s ingressSpecWithSecret) Validate(ingressType string) error {
	return s.baseIngressSpec.Validate(ingressType)
}

func (s baseIngressSpec) Validate(ingressType string) error {
	if s.Enabled {
		if s.Path == "" {
			return requiredFieldError{fieldName: fmt.Sprintf("%s path", ingressType)}
		}

		if s.Domain != "" {
			err := dns.ValidateSubdomain(s.Domain)
			if err != nil {
				return errors.Append(err, invalidIngressHostError{hostType: ingressType})
			}
		}
	}

	return nil
}

func (s storageSpec) Validate() error {
	if s.Size < 0 {
		return errors.New("storage size must be a non-negative number")
	}

	if s.Retention == "" {
		return requiredFieldError{fieldName: "retention"}
	}

	match, err := regexp.MatchString("[0-9]+(ms|s|m|h|d|w|y)", s.Retention)
	if err != nil {
		return errors.WrapIf(err, "failed to check retention")
	}

	if !match {
		return errors.WrapIf(err, "invalid retention")
	}

	return nil
}

func (s grafanaSpec) Validate() error {
	if s.Enabled {
		if err := s.Ingress.Validate(ingressTypeGrafana); err != nil {
			return errors.WrapIf(err, "error during validate Grafana ingress")
		}
	}

	return nil
}

func (s alertmanagerSpec) Validate() error {
	if s.Enabled {
		// ingress validation
		if err := s.Ingress.Validate(ingressTypeAlertmanager); err != nil {
			return err
		}

		// validate Slack notification provider
		if slackProv, ok := s.Provider[alertmanagerProviderSlack]; ok {
			var slack slackSpec
			if err := mapstructure.Decode(slackProv, &slack); err != nil {
				return errors.WrapIf(err, "failed to bind Slack config")
			}
			if err := slack.Validate(); err != nil {
				return errors.WrapIf(err, "error during validating Slack")
			}
		}

		// validate PagerDuty notification provider
		if pagerDutyProv, ok := s.Provider[alertmanagerProviderPagerDuty]; ok {
			var pd pagerDutySpec
			if err := mapstructure.Decode(pagerDutyProv, &pd); err != nil {
				return errors.WrapIf(err, "failed to bind PagerDuty config")
			}
			if err := pd.Validate(); err != nil {
				return errors.WrapIf(err, "error during validating PagerDuty")
			}
		}
	}

	return nil
}

func (s slackSpec) Validate() error {
	if s.Enabled {
		if s.SecretID == "" {
			return requiredFieldError{fieldName: "secretId"}
		}

		if s.Channel == "" {
			return requiredFieldError{fieldName: "channel"}
		}
	}

	return nil
}

func (s pagerDutySpec) Validate() error {
	if s.Enabled {
		if s.SecretID == "" {
			return requiredFieldError{fieldName: "secretId"}
		}

		if s.URL == "" {
			return requiredFieldError{fieldName: "url"}
		}

		if s.IntegrationType == "" {
			return requiredFieldError{fieldName: "integrationType"}
		}

		if s.IntegrationType != pagerDutyIntegrationEventApiV2 && s.IntegrationType != pagerDutyIntegrationPrometheus {
			return errors.New(fmt.Sprintf("integration type should be only just: %s or %s", pagerDutyIntegrationEventApiV2, pagerDutyIntegrationPrometheus))
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
