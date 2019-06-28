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

package clusterfeature

import (
	"context"
	"errors"

	"github.com/goph/logur"
	"gopkg.in/yaml.v2"
)

const (
	ExternalDns = "external-dns"
	// DNSExternalDnsChartVersion set the external-dns chart version default value: "1.6.2"
	DNSExternalDnsChartVersion = "dns.externalDnsChartVersion"

	DNSExternalDnsChartName = "dns.externalDnsChartName"

	// DNSExternalDnsImageVersion set the external-dns image version
	DNSExternalDnsImageVersion = "dns.externalDnsImageVersion"

	DNSExternalDnsValues = "dns.externalDnsValues"
)

type ExternalDnsFeature struct {
	// chart details : name, version?
	DomainFilters []string
	Provider      string
	Credentials   interface{}
	Values        map[string]interface{}
}

// FeatureSelector operations for identifying supported features.
type FeatureSelector interface {
	// SelectFeature selects the feature to be worked with, eventually decorates it with internal information
	SelectFeature(ctx context.Context, feature Feature) (*Feature, error)
}

type featureSelector struct {
	logger logur.Logger
}

func (fs *featureSelector) SelectFeature(ctx context.Context, feature Feature) (*Feature, error) {
	switch feature.Name {
	case ExternalDns:

		// todo this is for testing purposes only
		externalDnsValues := map[string]interface{}{
			"rbac": map[string]bool{
				"create": false,
			},
			"image": map[string]string{
				"tag": "v0.5.11",
			},
			"aws": map[string]string{
				"secretKey": "",
				"accessKey": "",
				"region":    "",
			},
			"domainFilters": []string{"test-domain"},
			"policy":        "sync",
			"txtOwnerId":    "testing",
			"affinity":      "",
			"tolerations":   "",
		}

		externalDnsValuesJson, _ := yaml.Marshal(externalDnsValues)

		feature.Spec[DNSExternalDnsChartVersion] = "1.6.2"
		feature.Spec[DNSExternalDnsImageVersion] = "v0.5.11"
		feature.Spec[DNSExternalDnsValues] = externalDnsValuesJson
		feature.Spec[DNSExternalDnsChartName] = "stable/external-dns"

		return &feature, nil

	}

	return nil, errors.New("unsupported feature")
}

func NewFeatureSelector(logger logur.Logger) FeatureSelector {
	return &featureSelector{
		logger: logger,
	}
}
