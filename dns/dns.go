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
	"errors"
	"strings"

	"emperror.dev/emperror"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/validation"

	"github.com/banzaicloud/pipeline/internal/global"
)

// ExternalDnsChartValues describes external-dns helm chart values (https://hub.helm.sh/charts/stable/external-dns)
type ExternalDnsChartValues struct {
	Sources       []string                      `json:"sources,omitempty" yaml:"sources,omitempty"`
	Rbac          *ExternalDnsRbacSettings      `json:"rbac,omitempty" yaml:"rbac,omitempty"`
	Image         *ExternalDnsImageSettings     `json:"image,omitempty" yaml:"image,omitempty"`
	DomainFilters []string                      `json:"domainFilters,omitempty" yaml:"domainFilters,omitempty"`
	Policy        string                        `json:"policy,omitempty" yaml:"policy,omitempty"`
	TxtOwnerId    string                        `json:"txtOwnerId,omitempty" yaml:"txtOwnerId,omitempty"`
	Affinity      *v1.Affinity                  `json:"affinity,omitempty" yaml:"affinity,omitempty"`
	Tolerations   []v1.Toleration               `json:"tolerations,omitempty" yaml:"tolerations,omitempty"`
	ExtraArgs     map[string]string             `json:"extraArgs,omitempty" yaml:"extraArgs,omitempty"`
	TxtPrefix     string                        `json:"txtPrefix,omitempty" yaml:"txtPrefix,omitempty"`
	Crd           *ExternalDnsCrdSourceSettings `json:"crd,omitempty" yaml:"crd,omitempty"`
	Aws           *ExternalDnsAwsSettings       `json:"aws,omitempty" yaml:"aws,omitempty"`
}

type ExternalDnsRbacSettings struct {
	Create             bool   `json:"create,omitempty" yaml:"create,omitempty"`
	ServiceAccountName string `json:"serviceAccountName,omitempty" yaml:"serviceAccountName,omitempty"`
	ApiVersion         string `json:"apiVersion,omitempty" yaml:"apiVersion,omitempty"`
	PspEnabled         bool   `json:"pspEnabled,omitempty" yaml:"pspEnabled,omitempty"`
}

type ExternalDnsImageSettings struct {
	Registry   string `json:"registry,omitempty" yaml:"registry,omitempty"`
	Repository string `json:"repository,omitempty" yaml:"repository,omitempty"`
	Tag        string `json:"tag,omitempty" yaml:"tag,omitempty"`
}

type ExternalDnsCrdSourceSettings struct {
	Create     bool   `json:"create,omitempty" yaml:"create,omitempty"`
	Apiversion string `json:"apiversion,omitempty" yaml:"apiversion,omitempty"`
	Kind       string `json:"kind,omitempty" yaml:"kind,omitempty"`
}

type ExternalDnsAwsSettings struct {
	Credentials     *ExternalDnsAwsCredentials `json:"credentials,omitempty" yaml:"credentials,omitempty"`
	Region          string                     `json:"region,omitempty" yaml:"region,omitempty"`
	ZoneType        string                     `json:"zoneType,omitempty" yaml:"zoneType,omitempty"`
	AssumeRoleArn   string                     `json:"assumeRoleArn,omitempty" yaml:"assumeRoleArn,omitempty"`
	BatchChangeSize uint                       `json:"batchChangeSize,omitempty" yaml:"batchChangeSize,omitempty"`
}

type ExternalDnsAwsCredentials struct {
	AccessKey string `json:"accessKey,omitempty" yaml:"accessKey,omitempty"`
	SecretKey string `json:"secretKey,omitempty" yaml:"secretKey,omitempty"`
	MountPath string `json:"mountPath,omitempty" yaml:"mountPath,omitempty"`
}

// GetBaseDomain returns the DNS base domain from [dns.domain] config. Changes the read domain to lowercase
// to ensure it's DNS-1123 compliant
func GetBaseDomain() (string, error) {
	baseDomain := strings.ToLower(global.Config.Cluster.DNS.BaseDomain)

	err := ValidateSubdomain(baseDomain)
	if err != nil {
		return "", emperror.WrapWith(err, "invalid base domain")
	}

	return baseDomain, nil
}

// ValidateSubdomain verifies if the provided subdomain string complies with DNS-1123
func ValidateSubdomain(subdomain string) error {
	errs := validation.IsDNS1123Subdomain(subdomain)
	if len(errs) > 0 {
		return emperror.With(errors.New(strings.Join(errs, "\n")), "subdomain", subdomain)
	}

	return nil
}

// ValidateWildcardSubdomain verifies if the provided subdomain string complies with wildcard DNS-1123
func ValidateWildcardSubdomain(subdomain string) error {
	errs := validation.IsWildcardDNS1123Subdomain(subdomain)
	if len(errs) > 0 {
		return emperror.With(errors.New(strings.Join(errs, "\n")), "subdomain", subdomain)
	}

	return nil
}
