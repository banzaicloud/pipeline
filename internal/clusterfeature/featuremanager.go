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
	"strconv"

	"github.com/banzaicloud/pipeline/auth"
	"github.com/banzaicloud/pipeline/cluster"
	"github.com/banzaicloud/pipeline/helm"
	"github.com/goph/emperror"
	"github.com/goph/logur"
	"github.com/goph/logur/adapters/logrusadapter"
	"github.com/sirupsen/logrus"
	k8sHelm "k8s.io/helm/pkg/helm"
	"k8s.io/helm/pkg/proto/hapi/release"
)

// FeatureManager operations in charge for applying features to the cluster
type FeatureManager interface {
	// Deploys and activates a feature on the given cluster
	Activate(ctx context.Context, clusterId string, feature Feature) (string, error)

	// Updates a feature on the given cluster
	Update(ctx context.Context, clusterId string, feature Feature) (string, error)
}

// syncFeatureManager synchronous feature manager
type syncFeatureManager struct {
	logger            logur.Logger
	clusterRepository ClusterRepository
	featureRepository FeatureRepository
	helmInstaller     helmInstaller
}

// clusterGetter restricts the external dependencies for the repository
type clusterGetter interface {
	GetClusterByIDOnly(ctx context.Context, clusterID uint) (cluster.CommonCluster, error)
}

//
type featureClusterRepository struct {
	clusterGetter clusterGetter
}


func (fcs *featureClusterRepository) GetCluster(ctx context.Context, clusterId string) (cluster.CommonCluster, error) {
	// todo use uint everywhere
	cid, err := strconv.ParseUint(clusterId, 0, 64)
	if err != nil {
		return nil, emperror.WrapWith(err, "failed to parse clusterid", "clusterid", clusterId)
	}

	cluster, err := fcs.clusterGetter.GetClusterByIDOnly(ctx, uint(cid))
	if err != nil {
		return nil, emperror.WrapWith(err, "failed to retrieve cluster", "clusterid", clusterId)
	}

	return cluster, nil
}



func NewClusterRepository(getter clusterGetter) ClusterRepository {
	return &featureClusterRepository{
		clusterGetter: getter,
	}
}

func (sfm *syncFeatureManager) Activate(ctx context.Context, clusterId string, feature Feature) (string, error) {

	cluster, err := sfm.clusterRepository.GetCluster(ctx, clusterId)
	if err != nil {
		return "", emperror.WrapWith(err, "failed to activate feature")
	}

	if err := sfm.helmInstaller.InstallFeature(ctx, cluster, feature); err != nil {
		return "", emperror.WrapWith(err, "failed to install feature")
	}

	if _, err := sfm.featureRepository.UpdateFeatureStatus(ctx, clusterId, feature, "ACTIVE"); err != nil {
		return "", emperror.WrapWith(err, "failed to update feature status")
	}

	return "", nil

}

func (sfm *syncFeatureManager) Update(ctx context.Context, clusterId string, feature Feature) (string, error) {
	panic("implement me")
}

// helmInstaller interface for helm operations
type helmInstaller interface {
	// InstallFeature installs a feature to the given cluster
	InstallFeature(ctx context.Context, cluster cluster.CommonCluster, feature Feature) error
}

// component in chrge for installing features from helmcharts
type featureHelmInstaller struct {
	logger logur.Logger
}

func (fhi *featureHelmInstaller) InstallFeature(ctx context.Context, cluster cluster.CommonCluster, feature Feature) error {
	// todo process / get information from the feature
	return fhi.installDeployment(cluster, "default", "test_dep", "test_release_name", nil, "chart_version", false)
}

func (fhi *featureHelmInstaller) installDeployment(cluster cluster.CommonCluster, namespace string, deploymentName string, releaseName string, values []byte, chartVersion string, wait bool) error {
	// --- [ Get K8S Config ] --- //
	kubeConfig, err := cluster.GetK8sConfig()
	if err != nil {
		fhi.logger.Error("failed to get k8s config", map[string]interface{}{"clusterid": cluster.GetID()})
		return err
	}

	org, err := auth.GetOrganizationById(cluster.GetOrganizationId())
	if err != nil {
		fhi.logger.Error("failed to get organization", map[string]interface{}{"clusterid": cluster.GetID()})
		return err
	}

	deployments, err := helm.ListDeployments(&releaseName, "", kubeConfig)
	if err != nil {
		fhi.logger.Error("failed to fetch deployments", map[string]interface{}{"clusterid": cluster.GetID()})
		return err
	}

	var foundRelease *release.Release

	if deployments != nil {
		for _, release := range deployments.Releases {
			if release.Name == releaseName {
				foundRelease = release
				break
			}
		}
	}

	if foundRelease != nil {
		switch foundRelease.GetInfo().GetStatus().GetCode() {
		case release.Status_DEPLOYED:
			fhi.logger.Info("deployment is already installed", map[string]interface{}{"deployment": deploymentName})
			return nil
		case release.Status_FAILED:
			err = helm.DeleteDeployment(releaseName, kubeConfig)
			if err != nil {
				fhi.logger.Error("failed to delete failed deployment", map[string]interface{}{"deployment": deploymentName})
				return err
			}
		}
	}

	options := []k8sHelm.InstallOption{
		k8sHelm.InstallWait(wait),
		k8sHelm.ValueOverrides(values),
	}
	_, err = helm.CreateDeployment(deploymentName, chartVersion, nil, namespace, releaseName, false, nil, kubeConfig, helm.GenerateHelmRepoEnv(org.Name), options...)
	if err != nil {
		fhi.logger.Error("failed to create deployment", map[string]interface{}{"deployment": deploymentName})
		return err
	}
	fhi.logger.Info("installed deployment", map[string]interface{}{"deployment": deploymentName})
	return nil
}

// NewSyncFeatureManager builds a new feature manager component
func NewSyncFeatureManager(clusterRepository ClusterRepository, featureRepository FeatureRepository) FeatureManager {
	l := logur.WithFields(logrusadapter.New(logrus.New()), map[string]interface{}{"component": "feature-manager"})
	return &syncFeatureManager{
		logger:            l,
		clusterRepository: clusterRepository,
		featureRepository: featureRepository,
		helmInstaller: &featureHelmInstaller{
			logger: logur.WithFields(l, map[string]interface{}{"comp": "helm-installer"}),
		}, // wired private component!
	}
}
