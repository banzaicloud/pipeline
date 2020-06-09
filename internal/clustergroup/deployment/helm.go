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

package deployment

import (
	"context"
	"strings"

	"emperror.dev/errors"

	"github.com/banzaicloud/pipeline/internal/helm"
)

type ChartMeta struct {
	Name        string
	Version     string
	Description string
}

type HelmService interface {
	GetChartMeta(orgId uint, name, version string) (ChartMeta, error)
	InstallOrUpgrade(
		orgID uint,
		c helm.ClusterDataProvider,
		release helm.Release,
		opts helm.Options,
	) error
	GetRelease(c helm.ClusterDataProvider, releaseName, namespace string) (helm.Release, error)
	DeleteRelease(c helm.ClusterDataProvider, releaseName, namespace string) error
}

type helm3Service struct {
	facade   helm.Service
	releaser helm.UnifiedReleaser
}

func (h *helm3Service) GetRelease(c helm.ClusterDataProvider, releaseName, namespace string) (helm.Release, error) {
	return h.releaser.GetRelease(c, releaseName, namespace)
}

func NewHelmService(facade helm.Service, releaser helm.UnifiedReleaser) HelmService {
	return &helm3Service{
		facade:   facade,
		releaser: releaser,
	}
}

func (h *helm3Service) DeleteRelease(c helm.ClusterDataProvider, releaseName, namespace string) error {
	return h.releaser.Delete(c, releaseName, namespace)
}

func (h *helm3Service) InstallOrUpgrade(orgID uint, c helm.ClusterDataProvider, release helm.Release, opts helm.Options) error {
	return h.releaser.InstallOrUpgrade(orgID, c, release, opts)
}

func (h *helm3Service) GetChartMeta(orgId uint, name, version string) (ChartMeta, error) {
	repoAndChart := strings.Split(name, "/")
	if len(repoAndChart) != 2 {
		return ChartMeta{}, errors.Errorf("missing repo ref from chart name %s", name)
	}
	chart, err := h.facade.GetChart(context.TODO(), orgId, helm.ChartFilter{
		Repo:    []string{repoAndChart[0]},
		Name:    []string{repoAndChart[1]},
		Version: []string{version},
	}, helm.Options{})
	if err != nil {
		return ChartMeta{}, err
	}
	desc, err := chart.GetDescription(version)
	if err != nil {
		return ChartMeta{}, err
	}
	return ChartMeta{
		Name:        repoAndChart[1],
		Version:     version,
		Description: desc,
	}, nil
}
