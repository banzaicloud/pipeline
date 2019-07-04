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

package clusterfeatureadapter

import (
	"context"

	"emperror.dev/emperror"
	"github.com/goph/logur"
	"github.com/goph/logur/adapters/logrusadapter"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	k8sHelm "k8s.io/helm/pkg/helm"
	"k8s.io/helm/pkg/proto/hapi/release"

	"github.com/banzaicloud/pipeline/helm"
	"github.com/banzaicloud/pipeline/internal/clusterfeature"
)

// SyncFeatureManager synchronous feature manager
type SyncFeatureManager struct {
	logger         logur.Logger
	clusterService clusterfeature.ClusterService
	helmService    helmService
}

// NewSyncFeatureManager builds a new feature manager component
func NewSyncFeatureManager(clusterService clusterfeature.ClusterService) *SyncFeatureManager {
	l := logur.WithFields(logrusadapter.New(logrus.New()), map[string]interface{}{"component": "feature-manager"})
	return &SyncFeatureManager{
		logger:         l,
		clusterService: clusterService,
		helmService: &featureHelmService{ // wired private component!
			logger: logur.WithFields(l, map[string]interface{}{"comp": "helm-installer"}),
		},
	}
}

func (sfm *SyncFeatureManager) Activate(ctx context.Context, clusterId uint, feature clusterfeature.Feature) (string, error) {

	cluster, err := sfm.clusterService.GetCluster(ctx, clusterId)
	if err != nil {
		// internal error at this point
		return "", emperror.WrapWith(err, "failed to activate feature")
	}

	if err := sfm.helmService.InstallFeature(ctx, cluster, feature); err != nil {
		return "", emperror.WrapWith(err, "failed to install feature")
	}

	return "", nil

}

func (sfm *SyncFeatureManager) Deactivate(ctx context.Context, clusterId uint, feature clusterfeature.Feature) error {
	cluster, err := sfm.clusterService.GetCluster(ctx, clusterId)
	if err != nil {
		// internal error at this point
		return emperror.WrapWith(err, "failed to deactivate feature")
	}

	if err := sfm.helmService.UninstallFeature(ctx, cluster, feature); err != nil {
		return emperror.WrapWith(err, "failed to uninstall feature")
	}

	return nil

}

func (sfm *SyncFeatureManager) Update(ctx context.Context, clusterId uint, feature clusterfeature.Feature) (error) {
	cluster, err := sfm.clusterService.GetCluster(ctx, clusterId)
	if err != nil {
		// internal error at this point
		return emperror.WrapWith(err, "failed to deactivate feature")
	}

	if err := sfm.helmService.UpdateFeature(ctx, cluster, feature); err != nil {
		return emperror.WrapWith(err, "failed to uninstall feature")
	}

	return nil
}

// helmService interface for helm operations
type helmService interface {
	// InstallFeature installs a feature to the given cluster
	InstallFeature(ctx context.Context, cluster clusterfeature.Cluster, feature clusterfeature.Feature) error

	// UninstallFeature removes a feature to the given cluster
	UninstallFeature(ctx context.Context, cluster clusterfeature.Cluster, feature clusterfeature.Feature) error

	// UpdateFeature updates / upgrades an already existing feature
	UpdateFeature(ctx context.Context, cluster clusterfeature.Cluster, feature clusterfeature.Feature) error
}

// component in chrge for installing features from helmcharts
type featureHelmService struct {
	logger logur.Logger
}

func (hs *featureHelmService) UpdateFeature(ctx context.Context, cluster clusterfeature.Cluster, feature clusterfeature.Feature) error {

	// processing feature specific configuration
	// todo factor the manager out to the Feature interface (possibly)

	ns, ok := feature.Spec["namespace"]
	if !ok {
		ns = helm.DefaultNamespace
	}

	deploymentName, ok := feature.Spec[clusterfeature.DNSExternalDnsChartName]
	if !ok {
		return errors.New("chart-name for feature not provided")
	}

	releaseName := "testing-externaldns"

	values, ok := feature.Spec[clusterfeature.DNSExternalDnsValues]
	if !ok {
		return errors.New("values for feature not available")
	}

	chartVersion, ok := feature.Spec[clusterfeature.DNSExternalDnsChartVersion]
	if !ok {
		return errors.New("values for feature not available")
	}

	kubeConfig, err := cluster.GetKubeConfig()
	if err != nil {
		return emperror.WrapWith(err, "failed to upgrade feature", "feature", feature.Name)
	}

	return hs.updateDeployment(cluster.GetOrganizationName(), kubeConfig, ns.(string), deploymentName.(string), releaseName, values.([]byte), chartVersion.(string))
}

func (hs *featureHelmService) InstallFeature(ctx context.Context, cluster clusterfeature.Cluster, feature clusterfeature.Feature) error {
	// processing feature specific configuration
	// todo factor the manager out to the Feature interface (possibly)

	ns, ok := feature.Spec["namespace"]
	if !ok {
		ns = helm.DefaultNamespace
	}

	deploymentName, ok := feature.Spec[clusterfeature.DNSExternalDnsChartName]
	if !ok {
		return errors.New("chart-name for feature not provided")
	}

	releaseName := "testing-externaldns"

	values, ok := feature.Spec[clusterfeature.DNSExternalDnsValues]
	if !ok {
		return errors.New("values for feature not available")
	}

	chartVersion, ok := feature.Spec[clusterfeature.DNSExternalDnsChartVersion]
	if !ok {
		return errors.New("values for feature not available")
	}

	kubeConfig, err := cluster.GetKubeConfig()
	if err != nil {
		return emperror.WrapWith(err, "failed to upgrade feature", "feature", feature.Name)
	}

	return hs.installDeployment(cluster.GetOrganizationName(), kubeConfig, ns.(string), deploymentName.(string), releaseName, values.([]byte), chartVersion.(string), false)
}

func (hs *featureHelmService) UninstallFeature(ctx context.Context, cluster clusterfeature.Cluster, feature clusterfeature.Feature) error {

	releaseName := "testing-externaldns"

	return hs.deleteDeployment(cluster, releaseName)

}

func (hs *featureHelmService) installDeployment(
	orgName string,
	kubeConfig []byte,
	namespace string,
	deploymentName string,
	releaseName string,
	values []byte,
	chartVersion string,
	wait bool,
) error {

	deployments, err := helm.ListDeployments(&releaseName, "", kubeConfig)
	if err != nil {
		hs.logger.Error("failed to fetch deployments", map[string]interface{}{"deployment": deploymentName})
		return err
	}

	var foundRelease *release.Release

	if deployments != nil {
		for _, rel := range deployments.Releases {
			if rel.Name == releaseName {
				foundRelease = rel
				break
			}
		}
	}

	if foundRelease != nil {
		switch foundRelease.GetInfo().GetStatus().GetCode() {
		case release.Status_DEPLOYED:
			hs.logger.Info("deployment is already installed", map[string]interface{}{"deployment": deploymentName})
			return nil
		case release.Status_FAILED:
			err = helm.DeleteDeployment(releaseName, kubeConfig)
			if err != nil {
				hs.logger.Error("failed to delete failed deployment", map[string]interface{}{"deployment": deploymentName})
				return err
			}
		}
	}

	options := []k8sHelm.InstallOption{
		k8sHelm.InstallWait(wait),
		k8sHelm.ValueOverrides(values),
	}
	_, err = helm.CreateDeployment(
		deploymentName,
		chartVersion,
		nil,
		namespace,
		releaseName,
		false,
		nil,
		kubeConfig,
		helm.GenerateHelmRepoEnv(orgName),
		options...,
	)

	if err != nil {
		hs.logger.Error("failed to create deployment", map[string]interface{}{"deployment": deploymentName})
		return err
	}

	hs.logger.Info("installed deployment", map[string]interface{}{"deployment": deploymentName})
	return nil
}

func (hs *featureHelmService) deleteDeployment(cluster clusterfeature.Cluster, releaseName string) error {

	kubeConfig, err := cluster.GetKubeConfig()
	if err != nil {
		hs.logger.Error("failed to get k8s config", map[string]interface{}{"clusterID": cluster.GetID()})
		return err
	}

	deployments, err := helm.ListDeployments(&releaseName, "", kubeConfig)
	if err != nil {
		hs.logger.Error("failed to fetch deployments", map[string]interface{}{"clusterid": cluster.GetID()})
		return err
	}

	var foundRelease *release.Release

	if deployments != nil {
		for _, rel := range deployments.Releases {
			if rel.Name == releaseName {
				foundRelease = rel
				break
			}
		}
	}

	if foundRelease != nil {
		err = helm.DeleteDeployment(releaseName, kubeConfig)
		if err != nil {
			hs.logger.Error("failed to delete deployment", map[string]interface{}{"deployment": releaseName})
			return err
		}
	}

	return nil

}

func (hs *featureHelmService) updateDeployment(
	orgName string,    // identifies the organization
	kubeConfig []byte, // identifies the cluster
	namespace string,
	deploymentName string,
	releaseName string,
	values []byte,
	chartVersion string,
) error {

	deployments, err := helm.ListDeployments(&releaseName, "", kubeConfig)
	if err != nil {
		return emperror.Wrap(err, "unable to fetch deployments")
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
			_, err = helm.UpgradeDeployment(
				releaseName,
				deploymentName,
				chartVersion,
				nil,
				values,
				false,
				kubeConfig,
				helm.GenerateHelmRepoEnv(orgName))
			if err != nil {
				return emperror.WrapWith(err, "could not upgrade deployment", "deploymentName", deploymentName)
			}
			return nil
		}
	}

	return nil
}
