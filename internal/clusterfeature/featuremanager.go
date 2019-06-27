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

	"github.com/goph/emperror"
	"github.com/goph/logur"
	"github.com/goph/logur/adapters/logrusadapter"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	k8sHelm "k8s.io/helm/pkg/helm"
	"k8s.io/helm/pkg/proto/hapi/release"

	"github.com/banzaicloud/pipeline/helm"
)

// FeatureManager operations in charge for applying features to the cluster
type FeatureManager interface {
	// Deploys and activates a feature on the given cluster
	Activate(ctx context.Context, clusterId uint, feature Feature) (string, error)

	// Updates a feature on the given cluster
	Update(ctx context.Context, clusterId uint, feature Feature) (string, error)
}

// syncFeatureManager synchronous feature manager
type syncFeatureManager struct {
	logger          logur.Logger
	clusterService  ClusterService
	featureSelector FeatureSelector
	helmInstaller   helmInstaller
}

// NewSyncFeatureManager builds a new feature manager component
func NewSyncFeatureManager(clusterService ClusterService) FeatureManager {
	l := logur.WithFields(logrusadapter.New(logrus.New()), map[string]interface{}{"component": "feature-manager"})
	return &syncFeatureManager{
		logger:          l,
		clusterService:  clusterService,
		featureSelector: NewFeatureSelector(l),
		helmInstaller: &featureHelmInstaller{ // wired private component!
			logger: logur.WithFields(l, map[string]interface{}{"comp": "helm-installer"}),
		},
	}
}

func (sfm *syncFeatureManager) Activate(ctx context.Context, clusterId uint, feature Feature) (string, error) {

	cluster, err := sfm.clusterService.GetCluster(ctx, clusterId)
	if err != nil {
		return "", emperror.WrapWith(err, "failed to activate feature")
	}

	// todo move this out to the service /return early in case the feature is not supported
	selectedFeature, err := sfm.featureSelector.SelectFeature(ctx, feature)
	if err != nil {
		return "", emperror.WrapWith(err, "failed to select feature")
	}

	if err := sfm.helmInstaller.InstallFeature(ctx, cluster, *selectedFeature); err != nil {
		return "", emperror.WrapWith(err, "failed to install feature")
	}

	return "", nil

}

func (sfm *syncFeatureManager) Update(ctx context.Context, clusterId uint, feature Feature) (string, error) {
	panic("implement me")
}

// helmInstaller interface for helm operations
type helmInstaller interface {
	// InstallFeature installs a feature to the given cluster
	InstallFeature(ctx context.Context, cluster Cluster, feature Feature) error
}

// component in chrge for installing features from helmcharts
type featureHelmInstaller struct {
	logger logur.Logger
}

func (fhi *featureHelmInstaller) InstallFeature(ctx context.Context, cluster Cluster, feature Feature) error {
	ns, ok := feature.Spec["namespace"]
	if !ok {
		return errors.New("namespace for feature not provided")
	}

	deploymentName, ok := feature.Spec[DNSExternalDnsChartName]
	if !ok {
		return errors.New("chart-name for feature not provided")
	}

	releaseName := "testing-externaldns"

	values, ok := feature.Spec[DNSExternalDnsValues]
	if !ok {
		return errors.New("values for feature not available")
	}

	chartVersion := feature.Spec[DNSExternalDnsChartVersion]

	return fhi.installDeployment(cluster, ns.(string), deploymentName.(string), releaseName, values.([]byte), chartVersion.(string), false)
}

func (fhi *featureHelmInstaller) installDeployment(
	cluster Cluster,
	namespace string,
	deploymentName string,
	releaseName string,
	values []byte,
	chartVersion string,
	wait bool,
) error {
	// --- [ Get K8S Config ] --- //
	kubeConfig, err := cluster.GetKubeConfig()
	if err != nil {
		fhi.logger.Error("failed to get k8s config", map[string]interface{}{"clusterid": cluster.GetID()})
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
	_, err = helm.CreateDeployment(
		deploymentName,
		chartVersion,
		nil,
		namespace,
		releaseName,
		false,
		nil,
		kubeConfig,
		helm.GenerateHelmRepoEnv(cluster.GetOrganizationName()),
		options...,
	)
	if err != nil {
		fhi.logger.Error("failed to create deployment", map[string]interface{}{"deployment": deploymentName})
		return err
	}
	fhi.logger.Info("installed deployment", map[string]interface{}{"deployment": deploymentName})
	return nil
}
