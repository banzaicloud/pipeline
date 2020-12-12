// Copyright © 2020 Banzai Cloud
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
)

const (
	// supported DNS provider names
	dnsRoute53 = "route53"
	dnsAzure   = "azure"
	dnsGoogle  = "google"
	dnsBanzai  = "banzaicloud-dns"
)

// +kubebuilder:object:generate=true

type ServiceSpec struct {
	ClusterDomain ClusterDomainSpec `json:"clusterDomain" mapstructure:"clusterDomain"`
	ExternalDNS   ExternalDNSSpec   `json:"externalDns" mapstructure:"externalDns"`
	RBACEnabled   bool              `json:"rbacEnabled,omitempty"`
}

func (s ServiceSpec) Validate() error {
	return errors.Combine(s.ClusterDomain.Validate(), s.ExternalDNS.Validate())
}

type ClusterDomainSpec string

func (s ClusterDomainSpec) Validate() error {
	if s == "" {
		return requiredStringFieldError{fieldName: "clusterDomain"}
	}
	return nil
}

// +kubebuilder:object:generate=true

type ExternalDNSSpec struct {
	DomainFilters DomainFiltersSpec `json:"domainFilters,omitempty" mapstructure:"domainFilters"`
	Policy        PolicySpec        `json:"policy" mapstructure:"policy"`
	Provider      ProviderSpec      `json:"provider" mapstructure:"provider"`
	Sources       SourcesSpec       `json:"sources,omitempty" mapstructure:"sources"`
	TXTOwnerID    TxtOwnerIDSpec    `json:"txtOwnerId" mapstructure:"txtOwnerId"`
	TXTPrefix     TxtPrefixSpec     `json:"txtPrefix" mapstructure:"txtPrefix"`
}

func (s ExternalDNSSpec) Validate() error {
	return errors.Combine(s.DomainFilters.Validate(), s.Policy.Validate(), s.Provider.Validate(), s.Sources.Validate(), s.TXTOwnerID.Validate())
}

type DomainFiltersSpec []string

func (s DomainFiltersSpec) Validate() error {
	return nil
}

type PolicySpec string

func (PolicySpec) Validate() error {
	return nil
}

// +kubebuilder:object:generate=true

type ProviderSpec struct {
	Name     string           `json:"name" mapstructure:"name"`
	SecretID string           `json:"secretId" mapstructure:"secretId"`
	Options  *ProviderOptions `json:"options,omitempty" mapstructure:"options"`
}

func (s ProviderSpec) Validate() error {
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

// +kubebuilder:object:generate=true

type ProviderOptions struct {
	DNSMasked          bool   `json:"dnsMasked" mapstructure:"dnsMasked"`
	AzureResourceGroup string `json:"resourceGroup,omitempty" mapstructure:"resourceGroup"`
	GoogleProject      string `json:"project,omitempty" mapstructure:"project"`
	Region             string `json:"region,omitempty" mapstructure:"region"`
	BatchChangeSize    uint   `json:"batchSize,omitempty" mapstructure:"batchSize"`
}

func (o *ProviderOptions) Validate(provider string) error {
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

type SourcesSpec []string

func (SourcesSpec) Validate() error {
	return nil
}

type TxtOwnerIDSpec string

func (s TxtOwnerIDSpec) Validate() error {
	return nil
}

type TxtPrefixSpec string

func (TxtPrefixSpec) Validate() error {
	return nil
}

func BindIntegratedServiceSpec(spec map[string]interface{}) (ServiceSpec, error) {
	var boundSpec ServiceSpec
	if err := mapstructure.Decode(spec, &boundSpec); err != nil {
		return boundSpec, errors.WrapIf(err, "failed to bind integrated service spec")
	}
	return boundSpec, nil
}

type requiredStringFieldError struct {
	fieldName string
}

func (e requiredStringFieldError) Error() string {
	return fmt.Sprintf("%s must be specified and cannot be empty", e.fieldName)
}
