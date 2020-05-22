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

package helm

import (
	"context"

	"emperror.dev/errors"
)

// +kit:endpoint:errorStrategy=service

// HelmAPI interface intended to gather all the helm operations exposed as rest services
// The abstraction layer is intended to aggregate information from different specialized components
// TODO move all exposed operations to this interface (currently the Service interface methods are exposed)
type RestAPI interface {
	// Gets releases and decorates the re
	GetReleases(ctx context.Context, organizationID uint, clusterID uint, filters ReleaseFilter, options Options) (releaseList []DetailedRelease, err error)
}

// DetailedRelease wraps a release and adds additional information to it
type DetailedRelease struct {
	Release
	Supported   bool `json:"supported"`
	Whitelisted bool `json:"whitelisted"`
	Rejected    bool `json:"rejected"`
}

type ReleaseSecurityInfo struct {
	Rejected    bool
	Whitelisted bool
}

// SecurityInfoService provides security resource information for releases
type SecurityInfoService interface {
	// GetSecurityInfo gets security information for the provided releases
	GetSecurityInfo(ctx context.Context, clusterId uint, releases []Release) (map[string]ReleaseSecurityInfo, error)
}

// restService component struct implementing the rest interface of Helm functionality
type restService struct {
	helmFacade          Service
	securityInfoService SecurityInfoService
}

func NewRestAPIService(helmService Service, securityInfoService SecurityInfoService) RestAPI {
	return restService{
		helmFacade:          helmService,
		securityInfoService: securityInfoService,
	}
}

func (r restService) GetReleases(ctx context.Context, organizationID uint, clusterID uint, filters ReleaseFilter, options Options) ([]DetailedRelease, error) {
	releases, err := r.helmFacade.ListReleases(ctx, organizationID, clusterID, filters, options)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to retrieve releases")
	}

	ret := make([]DetailedRelease, 0, len(releases))
	supportedChartMap, err := r.helmFacade.CheckReleases(ctx, organizationID, releases)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to retrieve charts")
	}

	securityInfoMap, err := r.securityInfoService.GetSecurityInfo(ctx, clusterID, releases)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to retrieve security information for releases")
	}

	for _, release := range releases {
		detailedRelease := DetailedRelease{Release: release}
		if supportedChartMap != nil {
			detailedRelease.Supported = supportedChartMap[release.ReleaseName]
		}

		if securityInfoMap != nil {
			if secInfo, ok := securityInfoMap[release.ReleaseName]; ok {
				detailedRelease.Rejected = secInfo.Rejected
				detailedRelease.Whitelisted = secInfo.Whitelisted
			}
		}
		ret = append(ret, detailedRelease)
	}

	return ret, nil
}
