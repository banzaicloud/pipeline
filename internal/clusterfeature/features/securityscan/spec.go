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

package securityscan

import (
	"emperror.dev/errors"
	"github.com/mitchellh/mapstructure"

	"github.com/banzaicloud/pipeline/internal/clusterfeature"
	"github.com/banzaicloud/pipeline/internal/global"
)

//featureSpec security scan cluster feature specific specification
type featureSpec struct {
	CustomAnchore    anchoreSpec       `json:"customAnchore" mapstructure:"customAnchore"`
	Policy           policySpec        `json:"policy" mapstructure:"policy"`
	ReleaseWhiteList []releaseSpec     `json:"releaseWhiteList,omitempty" mapstructure:"releaseWhiteList"`
	WebhookConfig    webHookConfigSpec `json:"webhookConfig" mapstructure:"webhookConfig"`
}

// Validate validates the input security scan specification.
func (s featureSpec) Validate() error {

	var validationErrors error

	if s.CustomAnchore.Enabled {
		validationErrors = s.CustomAnchore.Validate()
	}

	if !s.Policy.CustomPolicy.Enabled && s.Policy.PolicyID == "" {
		validationErrors = errors.Combine(validationErrors, errors.New("policyId is required"))
	}

	for _, releaseSpec := range s.ReleaseWhiteList {
		validationErrors = errors.Combine(validationErrors, releaseSpec.Validate())
	}

	validationErrors = errors.Combine(validationErrors, s.WebhookConfig.Validate())

	return validationErrors
}

type anchoreSpec struct {
	Enabled  bool   `json:"enabled" mapstructure:"enabled"`
	Url      string `json:"url" mapstructure:"url"`
	SecretID string `json:"secretId" mapstructure:"secretId"`
}

func (a anchoreSpec) Validate() error {

	if a.Enabled {
		if a.Url == "" || a.SecretID == "" {
			return errors.New("both anchore url and secretId are required")
		}
	}

	return nil
}

type policySpec struct {
	PolicyID     string           `json:"policyId,omitempty" mapstructure:"policyId"`
	CustomPolicy customPolicySpec `json:"customPolicy,omitempty" mapstructure:"customPolicy"`
}

type customPolicySpec struct {
	Enabled bool                   `json:"enabled" mapstructure:"enabled"`
	Policy  map[string]interface{} `json:"policy" mapstructure:"policy"`
}

type releaseSpec struct {
	Name   string `json:"name" mapstructure:"name"`
	Reason string `json:"reason" mapstructure:"reason"`
	Regexp string `json:"regexp,omitempty" mapstructure:"regexp"`
}

func (r releaseSpec) Validate() error {
	if r.Name == "" || r.Reason == "" {
		return errors.NewPlain("both name and reason must be specified")
	}

	return nil
}

type webHookConfigSpec struct {
	Enabled    bool     `json:"enabled" mapstructure:"enabled"`
	Selector   string   `json:"selector" mapstructure:"selector"`
	Namespaces []string `json:"namespaces" mapstructure:"namespaces"`
}

func (w webHookConfigSpec) Validate() error {
	if w.Enabled {
		if w.Selector == "" || len(w.Namespaces) == 0 {
			return errors.NewPlain("selector and namespaces must be filled")
		}

		for _, ns := range w.Namespaces {
			if ns == global.Config.Cluster.Namespace || ns == "kube-system" {
				return errors.Errorf("the following namespaces may not be modified: %v",
					[]string{global.Config.Cluster.Namespace, "kube-system"})
			}
		}
	}

	return nil
}

func bindFeatureSpec(spec clusterfeature.FeatureSpec) (featureSpec, error) {
	var boundSpec featureSpec
	if err := mapstructure.Decode(spec, &boundSpec); err != nil {
		return boundSpec, clusterfeature.InvalidFeatureSpecError{
			FeatureName: FeatureName,
			Problem:     errors.WrapIf(err, "failed to bind feature spec").Error(),
		}
	}
	return boundSpec, nil
}
