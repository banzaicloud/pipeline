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

package dns

import (
	"fmt"

	"emperror.dev/errors"
	"github.com/mitchellh/mapstructure"

	"github.com/banzaicloud/pipeline/internal/clusterfeature"
)

type dnsFeatureSpec struct {
	AutoDNS   autoDNSSpec   `json:"autoDns" mapstructure:"autoDns"`
	CustomDNS customDNSSpec `json:"customDns" mapstructure:"customDns"`
}

func (s dnsFeatureSpec) Validate() error {
	if s.AutoDNS.Enabled == s.CustomDNS.Enabled {
		return errors.New("exactly one of autoDns and customDns components must be enabled")
	}

	return errors.Combine(s.AutoDNS.Validate(), s.CustomDNS.Validate())
}

type autoDNSSpec struct {
	Enabled bool `json:"enabled"`
}

func (autoDNSSpec) Validate() error {
	return nil
}

type customDNSSpec struct {
	Enabled       bool         `json:"enabled" mapstructure:"enabled"`
	DomainFilters []string     `json:"domainFilters" mapstructure:"domainFilters"`
	ClusterDomain string       `json:"clusterDomain" mapstructure:"clusterDomain"`
	Provider      providerSpec `json:"provider" mapstructure:"provider"`
}

func (s customDNSSpec) Validate() error {
	if !s.Enabled {
		return nil
	}

	var errs error

	if len(s.DomainFilters) < 1 {
		errs = errors.Append(errs, errors.New("domain filters must be provided"))
	}

	return errors.Combine(errs, s.Provider.Validate())
}

type providerSpec struct {
	Name     string           `json:"name" mapstructure:"name"`
	SecretID string           `json:"secretId" mapstructure:"secretId"`
	Options  *providerOptions `json:"options,omitempty" mapstructure:"options"`
}

func (s providerSpec) Validate() error {
	var errs error

	if s.Name == "" {
		errs = errors.Append(errs, errors.New("DNS provider name must be provided"))
	}

	if s.SecretID == "" {
		errs = errors.Append(errs, errors.New("secret ID with DNS provider credentials must be provided"))
	}

	return errors.Combine(errs, s.Options.Validate(s.Name))
}

// providerOptions placeholder struct for additional provider specific configuration
// extrend it with the required fields here as appropriate
type providerOptions struct {
	DNSMasked          bool   `json:"dnsMasked" mapstructure:"dnsMasked"`
	AzureResourceGroup string `json:"resourceGroup,omitempty" mapstructure:"resourceGroup"`
	GoogleProject      string `json:"project,omitempty" mapstructure:"project"`
}

func (o *providerOptions) Validate(provider string) error {
	switch provider {
	case dnsAzure:
		if o == nil || len(o.AzureResourceGroup) == 0 {
			return &EmptyOptionFieldError{
				fieldName: "resourceGroup",
			}
		}
	case dnsGoogle:
		if o == nil || len(o.GoogleProject) == 0 {
			return &EmptyOptionFieldError{
				fieldName: "project",
			}
		}
	}

	return nil
}

func bindFeatureSpec(spec clusterfeature.FeatureSpec) (dnsFeatureSpec, error) {
	var boundSpec dnsFeatureSpec
	if err := mapstructure.Decode(spec, &boundSpec); err != nil {
		return boundSpec, clusterfeature.InvalidFeatureSpecError{
			FeatureName: FeatureName,
			Problem:     errors.WrapIf(err, "failed to bind feature spec").Error(),
		}
	}
	return boundSpec, nil
}

// EmptyOptionFieldError is returned when resource group field is empty in case of Azure provider.
type EmptyOptionFieldError struct {
	fieldName string
}

func (e *EmptyOptionFieldError) Error() string {
	return fmt.Sprintf("%s cannot be empty", e.fieldName)
}
