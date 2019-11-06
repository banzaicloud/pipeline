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
	ClusterDomain clusterDomainSpec `json:"clusterDomain" mapstructure:"clusterDomain"`
	ExternalDNS   externalDNSSpec   `json:"externalDns" mapstructure:"externalDns"`
}

func (s dnsFeatureSpec) Validate() error {
	return errors.Combine(s.ClusterDomain.Validate(), s.ExternalDNS.Validate())
}

type clusterDomainSpec string

func (s clusterDomainSpec) Validate() error {
	if s == "" {
		return requiredStringFieldError{fieldName: "clusterDomain"}
	}
	return nil
}

type externalDNSSpec struct {
	DomainFilters domainFiltersSpec `json:"domainFilters" mapstructure:"domainFilters"`
	Policy        policySpec        `json:"policy" mapstructure:"policy"`
	Provider      providerSpec      `json:"provider" mapstructure:"provider"`
	Sources       sourcesSpec       `json:"sources" mapstructure:"sources"`
	TXTOwnerID    txtOwnerIDSpec    `json:"txtOwnerId" mapstructure:"txtOwnerId"`
	TXTPrefix     txtPrefixSpec     `json:"txtPrefix" mapstructure:"txtPrefix"`
}

func (s externalDNSSpec) Validate() error {
	return errors.Combine(s.DomainFilters.Validate(), s.Policy.Validate(), s.Provider.Validate(), s.Sources.Validate(), s.TXTOwnerID.Validate())
}

type domainFiltersSpec []string

func (s domainFiltersSpec) Validate() error {
	return nil
}

type policySpec string

func (policySpec) Validate() error {
	return nil
}

type providerSpec struct {
	Name     string           `json:"name" mapstructure:"name"`
	SecretID string           `json:"secretId" mapstructure:"secretId"`
	Options  *providerOptions `json:"options,omitempty" mapstructure:"options"`
}

func (s providerSpec) Validate() error {
	var errs error

	if s.Name == "" {
		errs = errors.Append(errs, requiredStringFieldError{fieldName: "name"})
	}

	if s.Name == dnsBanzai {
		if s.SecretID != "" {
			errs = errors.Append(errs, errors.Errorf("secret ID cannot be specified for provider %q", dnsBanzai))
		}
	} else {
		if s.SecretID == "" {
			errs = errors.Append(errs, errors.New("secret ID with DNS provider credentials must be provided"))
		}
	}

	return errors.Combine(errs, s.Options.Validate(s.Name))
}

type providerOptions struct {
	DNSMasked          bool   `json:"dnsMasked" mapstructure:"dnsMasked"`
	AzureResourceGroup string `json:"resourceGroup,omitempty" mapstructure:"resourceGroup"`
	GoogleProject      string `json:"project,omitempty" mapstructure:"project"`
	Region             string `json:"region,omitempty" mapstructure:"region"`
	BatchChangeSize    uint   `json:"batchSize,omitempty" mapstructure:"batchSize"`
}

func (o *providerOptions) Validate(provider string) error {
	switch provider {
	case dnsAzure:
		if o == nil || o.AzureResourceGroup == "" {
			return requiredStringFieldError{
				fieldName: "resourceGroup",
			}
		}
	case dnsGoogle:
		if o == nil || o.GoogleProject == "" {
			return requiredStringFieldError{
				fieldName: "project",
			}
		}
	}

	return nil
}

type sourcesSpec []string

func (sourcesSpec) Validate() error {
	return nil
}

type txtOwnerIDSpec string

func (s txtOwnerIDSpec) Validate() error {
	return nil
}

type txtPrefixSpec string

func (txtPrefixSpec) Validate() error {
	return nil
}

func bindFeatureSpec(spec clusterfeature.FeatureSpec) (dnsFeatureSpec, error) {
	var boundSpec dnsFeatureSpec
	if err := mapstructure.Decode(spec, &boundSpec); err != nil {
		return boundSpec, errors.WrapIf(err, "failed to bind feature spec")
	}
	return boundSpec, nil
}

type requiredStringFieldError struct {
	fieldName string
}

func (e requiredStringFieldError) Error() string {
	return fmt.Sprintf("%s must be specified and cannot be empty", e.fieldName)
}
