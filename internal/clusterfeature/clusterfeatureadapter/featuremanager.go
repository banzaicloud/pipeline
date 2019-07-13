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

	"github.com/goph/emperror"
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
	helmInstaller  helmInstaller
}

// NewSyncFeatureManager builds a new feature manager component
func NewSyncFeatureManager(clusterService clusterfeature.ClusterService) *SyncFeatureManager {
	l := logur.WithFields(logrusadapter.New(logrus.New()), map[string]interface{}{"component": "feature-manager"})
	return &SyncFeatureManager{
		logger:         l,
		clusterService: clusterService,
		helmInstaller: &featureHelmInstaller{ // wired private component!
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

	if err := sfm.helmInstaller.InstallFeature(ctx, cluster, feature); err != nil {
		return "", emperror.WrapWith(err, "failed to install feature")
	}

	return "", nil

}

func (sfm *SyncFeatureManager) Update(ctx context.Context, clusterId uint, feature clusterfeature.Feature) (string, error) {
	panic("implement me")
}

// helmInstaller interface for helm operations
type helmInstaller interface {
	// InstallFeature installs a feature to the given cluster
	InstallFeature(ctx context.Context, cluster clusterfeature.Cluster, feature clusterfeature.Feature) error
}

// component in chrge for installing features from helmcharts
type featureHelmInstaller struct {
	logger logur.Logger
}

func (fhi *featureHelmInstaller) InstallFeature(ctx context.Context, cluster clusterfeature.Cluster, feature clusterfeature.Feature) error {
	ns, ok := feature.Spec["namespace"]
	if !ok {
		return errors.New("namespace for feature not provided")
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

	chartVersion := feature.Spec[clusterfeature.DNSExternalDnsChartVersion]

	return fhi.installDeployment(cluster, ns.(string), deploymentName.(string), releaseName, values.([]byte), chartVersion.(string), false)
}

func (fhi *featureHelmInstaller) installDeployment(
	cluster clusterfeature.Cluster,
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
		cluster.GetOrganizationName(),
		deploymentName,
		chartVersion,
		nil,
		namespace,
		releaseName,
		false,
		nil,
		kubeConfig,
		fhi.logger,
		options...,
	)
	if err != nil {
		fhi.logger.Error("failed to create deployment", map[string]interface{}{"deployment": deploymentName})
		return err
	}
	fhi.logger.Info("installed deployment", map[string]interface{}{"deployment": deploymentName})
	return nil
}
