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
)

const (
	externalDns = "external-dns"
	// DNSExternalDnsChartVersion set the external-dns chart version default value: "1.6.2"
	DNSExternalDnsChartVersion = "dns.externalDnsChartVersion"

	// DNSExternalDnsImageVersion set the external-dns image version
	DNSExternalDnsImageVersion = "dns.externalDnsImageVersion"

	// Status signaling a feature being activated or inactive
	STATUS_PENDING = "PENDING"

	// Status signaling a feature being active
	STATUS_ACTIVE = "ACTIVE"
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
	case externalDns:
		// todo add other internals here
		feature.Spec[DNSExternalDnsChartVersion] = "1.6.2"
		feature.Spec[DNSExternalDnsImageVersion] = "v0.5.11"

		// TODO assemble values and add the byte array to the feature
		// TODO DISCUSS IT FIRST

		return &feature, nil
	}

	return nil, errors.New("unsupported feature")
}

func NewFeatureSelector(logger logur.Logger) FeatureSelector {
	return &featureSelector{
		logger: logger,
	}
}
