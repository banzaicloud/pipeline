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
	"encoding/json"

	"emperror.dev/emperror"
	"github.com/goph/logur"
)

const (
	ExternalDnsChartVersion = "1.6.2"

	ExternalDnsImageVersion = "v0.5.11"

	ExternalDnsChartName = "stable/external-dns"

	ExternalDnsNamespace = "default"

	ExternalDnsRelease = "external-dns"
)

// ClusterService provides a thin access layer to clusters.
type ClusterService interface {
	// GetCluster retrieves the cluster representation based on the cluster identifier
	GetCluster(ctx context.Context, clusterID uint) (Cluster, error)

	// IsClusterReady checks whether the cluster is ready for features (eg.: exists and it's running).
	IsClusterReady(ctx context.Context, clusterID uint) (bool, error)
}

// Cluster represents a Kubernetes cluster.
type Cluster interface {
	GetID() uint
	GetOrganizationName() string
	GetKubeConfig() ([]byte, error)
}

// externalDnsFeatureManager synchronous feature manager
type externalDnsFeatureManager struct {
	logger            logur.Logger
	featureRepository FeatureRepository
	clusterService    ClusterService
	helmService       HelmService
}

// NewExternalDnsFeatureManager builds a new feature manager component
func NewExternalDnsFeatureManager(logger logur.Logger, featureRepository FeatureRepository, clusterService ClusterService) FeatureManager {
	hs := &featureHelmService{ // wired private component!
		logger: logur.WithFields(logger, map[string]interface{}{"helm-service": "comp"}),
	}
	return &externalDnsFeatureManager{
		logger:            logur.WithFields(logger, map[string]interface{}{"component": "feature-manager"}),
		featureRepository: featureRepository,
		clusterService:    clusterService,
		helmService:       hs,
	}
}

func (sfm *externalDnsFeatureManager) Activate(ctx context.Context, clusterID uint, feature Feature) error {
	fLog := logur.WithFields(sfm.logger, map[string]interface{}{"cluster": clusterID, "feature": feature.Name})
	fLog.Info("activating feature ...")

	if _, err := sfm.featureRepository.GetFeature(ctx, clusterID, feature.Name); err != nil {
		fLog.Debug("feature exists")

		return newFeatureExistsError(feature.Name)
	}

	cluster, err := sfm.clusterService.GetCluster(ctx, clusterID)
	if err != nil {
		fLog.Debug("failed to activate feature")
		// internal error at this point
		return emperror.WrapWith(err, "failed to activate feature")
	}

	kubeConfig, err := cluster.GetKubeConfig()
	if err != nil {
		return emperror.WrapWith(err, "failed to upgrade feature", "feature", feature.Name)
	}

	// todo implement the configuration fallback mechanism here
	externalDnsValues := map[string]interface{}{
		"rbac": map[string]bool{
			"create": false,
		},
		"image": map[string]string{
			"tag": "v0.5.11",
		},
		"aws": map[string]string{
			"secretKey": "",
			"accessKey": "",
			"region":    "",
		},
		"domainFilters": []string{"test-domain"},
		"policy":        "sync",
		"txtOwnerId":    "testing",
		"affinity":      "",
		"tolerations":   "",
	}

	externalDnsValuesJson, _ := yaml.Marshal(externalDnsValues)

	if _, err := sfm.featureRepository.SaveFeature(ctx, clusterID, feature.Name, feature.Spec); err != nil {
		fLog.Debug("failed to persist feature")

		return newDatabaseAccessError(feature.Name)
	}

	if err = sfm.helmService.InstallDeployment(ctx, cluster.GetOrganizationName(), kubeConfig, ExternalDnsNamespace,
		ExternalDnsChartName, ExternalDnsRelease, externalDnsValuesJson, ExternalDnsChartVersion, false); err != nil {
		// rollback
		fLog.Debug("failed to deploy feature  - rolling back ... ")
		if err = sfm.featureRepository.DeleteFeature(ctx, clusterID, feature.Name); err != nil {
			fLog.Debug("failed to deploy feature  - failed to roll back")

			return newDatabaseAccessError(feature.Name)
		}

		return emperror.Wrap(err, "failed to deploy feature")
	}

	if _, err := sfm.featureRepository.UpdateFeatureStatus(ctx, clusterID, feature.Name, FeatureStatusActive); err != nil {
		fLog.Debug("failed to persist feature")

		return newDatabaseAccessError(feature.Name)
	}

	fLog.Info("successfully activated feature")
	return nil

}

func (sfm *externalDnsFeatureManager) Deactivate(ctx context.Context, clusterID uint, featureName string) error {
	// method scoped logger
	mLog := logur.WithFields(sfm.logger, map[string]interface{}{"cluster": clusterID, "feature": featureName})
	mLog.Info("deactivating feature ...")

	var (
		mFeature *Feature
		err      error
	)

	if mFeature, err = sfm.featureRepository.GetFeature(ctx, clusterID, featureName); err != nil {
		mLog.Debug("feature could not be retrieved")

		return newDatabaseAccessError(featureName)
	}

	if mFeature == nil {
		mLog.Debug("feature could not found")

		return newFeatureNotFoundError(featureName)
	}

	cluster, err := sfm.clusterService.GetCluster(ctx, clusterID)
	if err != nil {
		// internal error at this point
		return emperror.WrapWith(err, "failed to deactivate feature")
	}

	kubeConfig, err := cluster.GetKubeConfig()
	if err != nil {
		return emperror.WrapWith(err, "failed to deactivate feature", "feature", featureName)
	}

	if err := sfm.helmService.DeleteDeployment(ctx, kubeConfig, ExternalDnsRelease); err != nil {
		mLog.Info("failed to delete feature deployment")

		return emperror.WrapWith(err, "failed to uninstall feature")
	}

	if err := sfm.featureRepository.DeleteFeature(ctx, clusterID, featureName); err != nil {
		mLog.Debug("feature could not be deleted")

		return newDatabaseAccessError(featureName)
	}

	mLog.Info("successfully deactivated feature")

	return nil
}

func (sfm *externalDnsFeatureManager) Update(ctx context.Context, clusterID uint, feature Feature) error {
	mLoger := logur.WithFields(sfm.logger, map[string]interface{}{"clusterId": clusterID, "feature": feature.Name})
	mLoger.Info("updating feature ...")

	var (
		persistedFeature *Feature
		err              error
	)

	if persistedFeature, err = sfm.featureRepository.GetFeature(ctx, clusterID, feature.Name); err != nil {
		mLoger.Debug("failed not retrieve feature")

		return newDatabaseAccessError(feature.Name)
	}

	if persistedFeature == nil {
		mLoger.Debug("feature not found")

		return newFeatureNotFoundError(feature.Name)
	}

	cluster, err := sfm.clusterService.GetCluster(ctx, clusterID)
	if err != nil {
		mLoger.Debug("failed to retrieve cluster")

		return emperror.WrapWith(err, "failed to retrieve cluster")
	}

	var valuesJson []byte
	if valuesJson, err = json.Marshal(feature.Spec); err != nil {
		return emperror.Wrap(err, "failed to update feature")
	}

	kubeConfig, err := cluster.GetKubeConfig()
	if err != nil {
		mLoger.Debug("failed to retrieve k8s configuration")

		return emperror.WrapWith(err, "failed to upgrade feature", "feature", feature.Name)
	}

	// "suspend" the feature till it gets updated
	if _, err = sfm.featureRepository.UpdateFeatureStatus(ctx, clusterID, feature.Name, FeatureStatusPending); err != nil {
		mLoger.Debug("failed to update feature status")

		return newDatabaseAccessError(feature.Name)
	}

	// todo revise this: we loose the "old" spec here
	if _, err = sfm.featureRepository.UpdateFeatureSpec(ctx, clusterID, feature.Name, feature.Spec); err != nil {
		mLoger.Debug("failed to update feature spec")

		return newDatabaseAccessError(feature.Name)
	}

	if err = sfm.helmService.UpdateDeployment(ctx, cluster.GetOrganizationName(), kubeConfig, ExternalDnsNamespace,
		ExternalDnsChartName, ExternalDnsRelease, valuesJson, ExternalDnsChartVersion); err != nil {
		mLoger.Debug("failed to deploy feature")

		// todo feature status in case the upgrade failed?!
		return emperror.Wrap(err, "failed to update feature")
	}

	// feature status set back to active
	if _, err = sfm.featureRepository.UpdateFeatureStatus(ctx, clusterID, feature.Name, FeatureStatusActive); err != nil {
		mLoger.Debug("failed to update feature status")

		return newDatabaseAccessError(feature.Name)
	}

	mLoger.Info("successfully updated feature")

	return nil
}

func (sfm *externalDnsFeatureManager) Validate(ctx context.Context, clusterID uint, featureName string, featureSpec map[string]interface{}) error {
	mLoger := logur.WithFields(sfm.logger, map[string]interface{}{"clusterId": clusterID, "feature": featureName})
	mLoger.Info("Validating feature")

	ready, err := sfm.clusterService.IsClusterReady(ctx, clusterID)
	if err != nil {

		return emperror.Wrap(err, "could not access cluster")
	}

	if !ready {
		mLoger.Debug("cluster not ready")

		return newClusterNotReadyError(featureName)
	}

	mLoger.Info("feature validation succeeded")
	return nil

}

func (sfm *externalDnsFeatureManager) Details(ctx context.Context, clusterID uint, featureName string) (*Feature, error) {
	mLoger := logur.WithFields(sfm.logger, map[string]interface{}{"clusterId": clusterID, "feature": featureName})
	mLoger.Debug("retrieving feature details ...")

	var (
		feature *Feature
		err     error
	)

	if feature, err = sfm.featureRepository.GetFeature(ctx, clusterID, featureName); err != nil {

		return nil, newDatabaseAccessError(featureName)
	}

	if feature == nil {

		return nil, newFeatureNotFoundError(featureName)
	}

	mLoger.Debug("successfully retrieved feature details")
	return feature, nil

}
