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
	v1 "k8s.io/api/core/v1"
)

// ExternalDnsChartValues describes external-dns helm chart values (https://hub.helm.sh/charts/stable/external-dns)
type ExternalDnsChartValues struct {
	Sources       []string                  `json:"sources,omitempty"`
	Rbac          *ExternalDnsRbacSettings  `json:"rbac,omitempty"`
	Image         *ExternalDnsImageSettings `json:"image,omitempty"`
	DomainFilters []string                  `json:"domainFilters,omitempty"`
	Policy        string                    `json:"policy,omitempty"`
	TxtOwnerId    string                    `json:"txtOwnerId,omitempty"`
	Affinity      *v1.Affinity              `json:"affinity,omitempty"`
	Tolerations   []v1.Toleration           `json:"tolerations,omitempty"`
	ExtraArgs     map[string]string         `json:"extraArgs,omitempty"`
	TxtPrefix     string                    `json:"txtPrefix,omitempty"`
	Azure         ProviderSettings          `json:"azure,omitempty"`
	Aws           ProviderSettings          `json:"aws,omitempty"`
	Google        ProviderSettings          `json:"google,omitempty"`
	Provider      string                    `json:"provider"`
}

type ExternalDnsRbacSettings struct {
	Create             bool   `json:"create,omitempty"`
	ServiceAccountName string `json:"serviceAccountName,omitempty"`
	ApiVersion         string `json:"apiVersion,omitempty"`
	PspEnabled         bool   `json:"pspEnabled,omitempty"`
}

type ExternalDnsImageSettings struct {
	Registry   string `json:"registry,omitempty"`
	Repository string `json:"repository,omitempty"`
	Tag        string `json:"tag,omitempty"`
}

type ExternalDnsCrdSourceSettings struct {
	Create     bool   `json:"create,omitempty"`
	ApiVersion string `json:"apiVersion,omitempty"`
	Kind       string `json:"kind,omitempty"`
}

type ExternalDnsAwsSettings struct {
	genericProviderSettings
	Credentials     *ExternalDnsAwsCredentials `json:"credentials,omitempty"`
	Region          string                     `json:"region,omitempty"`
	ZoneType        string                     `json:"zoneType,omitempty"`
	AssumeRoleArn   string                     `json:"assumeRoleArn,omitempty"`
	BatchChangeSize uint                       `json:"batchChangeSize,omitempty"`
}

type ExternalDnsAwsCredentials struct {
	AccessKey string `json:"accessKey,omitempty"`
	SecretKey string `json:"secretKey,omitempty"`
	MountPath string `json:"mountPath,omitempty"`
}

type ExternalDnsAzureSettings struct {
	genericProviderSettings
	SecretName    string `json:"secretName,omitempty"`
	ResourceGroup string `json:"resourceGroup,omitempty"`
}

type ExternalDnsGoogleSettings struct {
	genericProviderSettings
	Project              string `json:"project"`
	ServiceAccountSecret string `json:"serviceAccountSecret"`
	ServiceAccountKey    string `json:"serviceAccountKey"`
}

// ProviderSettings marks a struct holding DNS provider specific values
type ProviderSettings interface {
	Provider() string
}

// genericProviderSettings struct to be embedded in all provider specific settings type
// marks a struct as being a provider settings type
type genericProviderSettings struct {
	Name string `json:"name"`
}

func (p *genericProviderSettings) Provider() string {
	return p.Name
}
