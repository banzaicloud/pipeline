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

package integratedserviceadapter

import (
	"context"

	"emperror.dev/errors"
	"sigs.k8s.io/yaml"

	"github.com/banzaicloud/pipeline/internal/cluster/clustersetup"
	"github.com/banzaicloud/pipeline/internal/common"
	"github.com/banzaicloud/pipeline/internal/helm"
	"github.com/banzaicloud/pipeline/internal/integratedservices/services"
	helm2 "github.com/banzaicloud/pipeline/pkg/helm"
)

// helper interface for integrating helm services
// TODO revise and refactor these interfaces not to differ
type AdaptedHelmService interface {
	services.HelmService
	clustersetup.HelmService
}

// helmServiceAdapter component providing helm3 implementation for integrated services
type helmServiceAdapter struct {
	systemNamespace string
	helmService     helm.Service

	logger common.Logger
}

func NewHelmService(service helm.Service, systemNamespace string, logger common.Logger) AdaptedHelmService {
	return helmServiceAdapter{
		systemNamespace: systemNamespace,
		helmService:     service,
		logger:          logger,
	}
}

func (h helmServiceAdapter) ApplyDeployment(
	ctx context.Context,
	clusterID uint,
	namespace string,
	deploymentName string,
	releaseName string,
	values []byte,
	chartVersion string,
) error {
	var valuesMap map[string]interface{}
	if err := yaml.Unmarshal(values, &valuesMap); err != nil {
		return errors.WrapIf(err, "failed to unmarshal values")
	}

	options := helm.Options{
		Namespace:    namespace,
		DryRun:       false,
		GenerateName: false,
		Wait:         false,
		ReuseValues:  false,
	}
	release := helm.Release{
		ReleaseName: releaseName,
		ChartName:   deploymentName,
		Namespace:   namespace,
		Version:     chartVersion,
		Values:      valuesMap,
	}

	return h.helmService.InstallRelease(ctx, 0, clusterID, release, options)
}

func (h helmServiceAdapter) DeleteDeployment(ctx context.Context, clusterID uint, releaseName string) error {
	return h.helmService.DeleteRelease(ctx, 0, clusterID, releaseName, helm.Options{
		Namespace: h.systemNamespace,
	})
}

func (h helmServiceAdapter) GetDeployment(ctx context.Context, clusterID uint, releaseName string) (*helm2.GetDeploymentResponse, error) {
	release, err := h.helmService.GetRelease(ctx, 0, clusterID, releaseName, helm.Options{
		Namespace: h.systemNamespace,
	})
	if err != nil {
		return nil, errors.WrapIf(err, "failed to retrieve release")
	}

	// TODO identify the minimum set of required fields, map only those
	return &helm2.GetDeploymentResponse{
		ReleaseName:  release.ReleaseName,
		Chart:        release.ChartName,
		ChartName:    release.ChartName,
		ChartVersion: release.Version,
		Namespace:    release.Namespace,
		Version:      0,
		Status:       release.ReleaseInfo.Status,
	}, nil
}

// for clustersetup!
func (h helmServiceAdapter) InstallDeployment(
	ctx context.Context,
	clusterID uint,
	namespace string,
	chartName string,
	releaseName string,
	values []byte,
	chartVersion string,
	_ bool,
) error {
	return h.ApplyDeployment(ctx, clusterID, namespace, chartName, releaseName, values, chartVersion)
}
