// Copyright © 2021 Banzai Cloud
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

package operator

import (
	"context"
	"strings"

	"emperror.dev/errors"

	"github.com/banzaicloud/pipeline/internal/cluster"
	"github.com/banzaicloud/pipeline/internal/helm"
	"github.com/banzaicloud/pipeline/internal/integratedservices"
)

const (
	IntegratedServiceOperatorInstallerActivityName = "integrated-service-operator-installer-activity"
	GetNextClusterRefActivityName                  = "get-next-cluster-id-activity"
)

type IntegratedServicesOperatorInstallerActivityInput struct {
	OrgID     uint
	ClusterID uint
}

func NewInstallerActivityInput(orgID uint, clusterID uint) IntegratedServicesOperatorInstallerActivityInput {
	return IntegratedServicesOperatorInstallerActivityInput{
		OrgID:     orgID,
		ClusterID: clusterID,
	}
}

type IntegratedServicesOperatorInstallerActivity struct {
	config        Config
	repoUpdater   helm.Service
	chartReleaser helm.UnifiedReleaser
}

func NewInstallerActivity(repoUpdater helm.Service, chartReleaser helm.UnifiedReleaser, config Config) IntegratedServicesOperatorInstallerActivity {
	return IntegratedServicesOperatorInstallerActivity{
		repoUpdater:   repoUpdater,
		chartReleaser: chartReleaser,
		config:        config,
	}
}

func (r IntegratedServicesOperatorInstallerActivity) Execute(ctx context.Context, input IntegratedServicesOperatorInstallerActivityInput) error {
	repoName := strings.Split(r.config.Chart, "/")[0]

	if err := r.repoUpdater.UpdateRepository(ctx,
		input.OrgID,
		helm.Repository{
			Name: repoName,
		}); err != nil {
		return errors.WrapIf(err, "failed to update helm repository")
	}

	if err := r.chartReleaser.InstallOrUpgrade(
		input.OrgID,
		clusterDataProvider{
			clusterID: input.ClusterID,
		},
		helm.Release{
			ChartName:   r.config.Chart,
			Namespace:   r.config.Namespace,
			Version:     r.config.Version,
			ReleaseName: r.config.Release,
		},
		helm.Options{
			Namespace: r.config.Namespace,
		},
	); err != nil {
		return errors.WrapIf(err, "failed to install or upgrade the chart")
	}
	return nil
}

type NextClusterIDActivity struct {
	NextIDProvider NextIDProvider
	ClusterChecker integratedservices.ClusterService
}

func NewNextClusterIDActivity(clusterService integratedservices.ClusterService, NextidProvider NextIDProvider) NextClusterIDActivity {
	return NextClusterIDActivity{
		ClusterChecker: clusterService,
		NextIDProvider: NextidProvider,
	}
}

// ClusterRef encapsulates information about the next cluster to be processed
type ClusterRef struct {
	ID       uint
	OrgID    uint
	NotFound bool
}

func (n NextClusterIDActivity) Execute(ctx context.Context, lastClusterID uint) (ClusterRef, error) {
	orgID, clusterID, err := n.NextIDProvider(lastClusterID)
	if err != nil {
		if cluster.IsNotFoundError(err) {
			return ClusterRef{NotFound: true}, nil
		}
		return ClusterRef{}, errors.WrapIfWithDetails(err, "failed to retrieve the next cluster reference",
			"lastClusterID", lastClusterID)
	}

	if err := n.ClusterChecker.CheckClusterReady(ctx, clusterID); err != nil {
		return ClusterRef{}, errors.WrapIfWithDetails(err, "cluster is not running",
			"lastClusterID", lastClusterID)
	}

	return ClusterRef{ID: clusterID, OrgID: orgID}, nil
}

type clusterDataProvider struct {
	clusterID uint
}

func (c clusterDataProvider) GetK8sConfig() ([]byte, error) {
	// no op here!
	return nil, nil
}

func (c clusterDataProvider) GetID() uint {
	return c.clusterID
}
