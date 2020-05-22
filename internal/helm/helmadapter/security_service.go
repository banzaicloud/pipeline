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

package helmadapter

import (
	"context"

	"emperror.dev/errors"
	securityV1Alpha "github.com/banzaicloud/anchore-image-validator/pkg/apis/security/v1alpha1"

	"github.com/banzaicloud/pipeline/internal/helm"
	anchore "github.com/banzaicloud/pipeline/internal/security"
)

// SecurityResourcer adapter interface for the security resource service
type SecurityResourcer interface {
	ListScanLogs(ctx context.Context, cluster anchore.Cluster) (interface{}, error)

	GetWhitelists(ctx context.Context, cluster anchore.Cluster) ([]securityV1Alpha.WhiteListItem, error)
}

// component struct to provide security information about helm releases
type securityService struct {
	resourcer      SecurityResourcer
	clusterService helm.ClusterService

	logger Logger
}

func NewSecurityService(clusterService helm.ClusterService, resourcer SecurityResourcer, logger Logger) securityService {
	return securityService{
		resourcer:      resourcer,
		clusterService: clusterService,
		logger:         logger,
	}
}

func (s securityService) GetSecurityInfo(ctx context.Context, clusterID uint, releases []helm.Release) (map[string]helm.ReleaseSecurityInfo, error) {
	kubeConfig, err := s.clusterService.GetKubeConfig(ctx, clusterID)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to get kubeConfig for cluster")
	}
	clusterData := NewClusterData(clusterID, kubeConfig)

	whiteListItems, err := s.resourcer.GetWhitelists(ctx, clusterData)
	if err != nil {
		// swallow the error deliberately here
		s.logger.Warn("no security scan whitelist information available...")
	}

	releaseToWhitelistMap := make(map[string]bool, len(whiteListItems))
	for _, whiteListItem := range whiteListItems {
		releaseToWhitelistMap[whiteListItem.ObjectMeta.Name] = true
	}

	scanLogs, err := s.resourcer.ListScanLogs(ctx, clusterData)
	if err != nil {
		// swallow the error deliberately here
		s.logger.Warn("no security scan log information available...")
	}

	releaseToScanLogsMap := make(map[string]bool)
	if scanLogs != nil {
		castScanLogs := scanLogs.([]securityV1Alpha.AuditSpec)

		for _, audit := range castScanLogs {
			if audit.Action == "reject" {
				releaseToScanLogsMap[audit.ReleaseName] = true
			}
		}
	}

	secInfoMap := make(map[string]helm.ReleaseSecurityInfo, len(releases))
	for _, release := range releases {
		secInfo := helm.ReleaseSecurityInfo{}

		if rejected, ok := releaseToScanLogsMap[release.ReleaseName]; ok {
			secInfo.Rejected = rejected
		}

		if whitelisted, ok := releaseToWhitelistMap[release.ReleaseName]; ok {
			secInfo.Whitelisted = whitelisted
		}

		secInfoMap[release.ReleaseName] = secInfo
	}

	return secInfoMap, nil
}

func NewClusterData(clusterID uint, kubeConfig []byte) clusterData {
	return clusterData{
		clusterID:  clusterID,
		kubeConfig: kubeConfig,
	}
}

type clusterData struct {
	clusterID  uint
	kubeConfig []byte
}

func (c clusterData) GetK8sConfig() ([]byte, error) {
	return c.kubeConfig, nil
}

func (c clusterData) GetID() uint {
	return c.clusterID
}
