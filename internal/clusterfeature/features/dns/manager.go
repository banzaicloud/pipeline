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

package dns

import (
	"context"
	"encoding/json"

	"emperror.dev/errors"
	"github.com/banzaicloud/pipeline/internal/clusterfeature"
	"github.com/goph/logur"
)

const (
	// hardcoded values for externalDns feature
	externalDnsChartVersion = "1.6.2"

	//externalDnsImageVersion = "v0.5.11"

	externalDnsChartName = "stable/external-dns"

	externalDnsNamespace = "default"

	externalDnsRelease = "external-dns"
)

// dnsFeatureManager synchronous feature manager
type dnsFeatureManager struct {
	logger               logur.Logger
	featureRepository    clusterfeature.FeatureRepository
	clusterService       clusterfeature.ClusterService
	helmService          clusterfeature.HelmService
	featureSpecProcessor clusterfeature.FeatureSpecProcessor
}

// NewDnsFeatureManager builds a new feature manager component
func NewDnsFeatureManager(logger logur.Logger,
	featureRepository clusterfeature.FeatureRepository,
	clusterService clusterfeature.ClusterService,
	helmService clusterfeature.HelmService,
	processor clusterfeature.FeatureSpecProcessor,
) clusterfeature.FeatureManager {
	return &dnsFeatureManager{
		logger:               logur.WithFields(logger, map[string]interface{}{"feature-manager": "comp"}),
		featureRepository:    featureRepository,
		clusterService:       clusterService,
		helmService:          helmService,
		featureSpecProcessor: processor,
	}
}

func (sfm *dnsFeatureManager) Activate(ctx context.Context, clusterID uint, feature clusterfeature.Feature) error {
	mLogger := logur.WithFields(sfm.logger, map[string]interface{}{"cluster": clusterID, "feature": feature.Name})
	mLogger.Info("activating feature ...")

	f, err := sfm.featureRepository.GetFeature(ctx, clusterID, feature.Name)
	if err != nil {

		return clusterfeature.NewDatabaseAccessError(feature.Name)
	}

	if f != nil {
		mLogger.Debug("feature exists")

		return clusterfeature.NewFeatureExistsError(feature.Name)
	}

	cluster, err := sfm.clusterService.GetCluster(ctx, clusterID)
	if err != nil {
		mLogger.Debug("failed to activate feature")

		return errors.WrapIf(err, "failed to activate feature")
	}

	kubeConfig, err := cluster.GetKubeConfig()
	if err != nil {

		return errors.WrapIfWithDetails(err, "failed to upgrade feature", "feature", feature.Name)
	}

	// todo rollback this if the install failed?
	if _, err := sfm.featureRepository.SaveFeature(ctx, clusterID, feature.Name, feature.Spec); err != nil {
		mLogger.Debug("failed to persist feature")

		return clusterfeature.NewDatabaseAccessError(feature.Name)
	}

	values, err := sfm.featureSpecProcessor.Process(nil, cluster.GetOrganizationID(), feature.Spec)
	if err != nil {
		mLogger.Debug("failed to process feature spec")

		return errors.WrapIf(err, "failed to process feature spec")
	}

	if err = sfm.helmService.InstallDeployment(ctx, cluster.GetOrganizationName(), kubeConfig, externalDnsNamespace,
		externalDnsChartName, externalDnsRelease, values.([]byte), externalDnsChartVersion, false); err != nil {
		// rollback
		mLogger.Debug("failed to deploy feature  - rolling back ... ")
		if err = sfm.featureRepository.DeleteFeature(ctx, clusterID, feature.Name); err != nil {
			mLogger.Debug("failed to deploy feature  - failed to roll back")

			return clusterfeature.NewDatabaseAccessError(feature.Name)
		}

		return errors.WrapIf(err, "failed to deploy feature")
	}

	if _, err := sfm.featureRepository.UpdateFeatureStatus(ctx, clusterID, feature.Name, clusterfeature.FeatureStatusActive); err != nil {
		mLogger.Debug("failed to persist feature")

		return clusterfeature.NewDatabaseAccessError(feature.Name)
	}

	mLogger.Info("successfully activated feature")
	return nil

}

func (sfm *dnsFeatureManager) Deactivate(ctx context.Context, clusterID uint, featureName string) error {
	// method scoped logger
	mLog := logur.WithFields(sfm.logger, map[string]interface{}{"cluster": clusterID, "feature": featureName})
	mLog.Info("deactivating feature ...")

	mFeature, err := sfm.featureRepository.GetFeature(ctx, clusterID, featureName)
	if err != nil {
		mLog.Debug("feature could not be retrieved")

		return clusterfeature.NewDatabaseAccessError(featureName)
	}

	if mFeature == nil {
		mLog.Debug("feature could not found")

		return clusterfeature.NewFeatureNotFoundError(featureName)
	}

	cluster, err := sfm.clusterService.GetCluster(ctx, clusterID)
	if err != nil {
		// internal error at this point
		return errors.WrapIf(err, "failed to deactivate feature")
	}

	kubeConfig, err := cluster.GetKubeConfig()
	if err != nil {
		return errors.WrapIfWithDetails(err, "failed to deactivate feature", "feature", featureName)
	}

	if err := sfm.helmService.DeleteDeployment(ctx, kubeConfig, externalDnsRelease); err != nil {
		mLog.Info("failed to delete feature deployment")

		return errors.WrapIf(err, "failed to uninstall feature")
	}

	if err := sfm.featureRepository.DeleteFeature(ctx, clusterID, featureName); err != nil {
		mLog.Debug("feature could not be deleted")

		return clusterfeature.NewDatabaseAccessError(featureName)
	}

	mLog.Info("successfully deactivated feature")

	return nil
}

func (sfm *dnsFeatureManager) Update(ctx context.Context, clusterID uint, feature clusterfeature.Feature) error {
	mLoger := logur.WithFields(sfm.logger, map[string]interface{}{"clusterId": clusterID, "feature": feature.Name})
	mLoger.Info("updating feature ...")

	persistedFeature, err := sfm.featureRepository.GetFeature(ctx, clusterID, feature.Name)
	if err != nil {
		mLoger.Debug("failed not retrieve feature")

		return clusterfeature.NewDatabaseAccessError(feature.Name)
	}

	if persistedFeature == nil {
		mLoger.Debug("feature not found")

		return clusterfeature.NewFeatureNotFoundError(feature.Name)
	}

	cluster, err := sfm.clusterService.GetCluster(ctx, clusterID)
	if err != nil {
		mLoger.Debug("failed to retrieve cluster")

		return errors.WrapIf(err, "failed to retrieve cluster")
	}

	var valuesJson []byte
	if valuesJson, err = json.Marshal(feature.Spec); err != nil {
		return errors.WrapIf(err, "failed to update feature")
	}

	kubeConfig, err := cluster.GetKubeConfig()
	if err != nil {
		mLoger.Debug("failed to retrieve k8s configuration")

		return errors.WrapIfWithDetails(err, "failed to upgrade feature", "feature", feature.Name)
	}

	// "suspend" the feature till it gets updated
	if _, err = sfm.featureRepository.UpdateFeatureStatus(ctx, clusterID, feature.Name, clusterfeature.FeatureStatusPending); err != nil {
		mLoger.Debug("failed to update feature status")

		return clusterfeature.NewDatabaseAccessError(feature.Name)
	}

	// todo revise this: we loose the "old" spec here
	if _, err = sfm.featureRepository.UpdateFeatureSpec(ctx, clusterID, feature.Name, feature.Spec); err != nil {
		mLoger.Debug("failed to update feature spec")

		return clusterfeature.NewDatabaseAccessError(feature.Name)
	}

	if err = sfm.helmService.UpdateDeployment(ctx, cluster.GetOrganizationName(), kubeConfig, externalDnsNamespace,
		externalDnsChartName, externalDnsRelease, valuesJson, externalDnsChartVersion); err != nil {
		mLoger.Debug("failed to deploy feature")

		// todo feature status in case the upgrade failed?!
		return errors.WrapIf(err, "failed to update feature")
	}

	// feature status set back to active
	if _, err = sfm.featureRepository.UpdateFeatureStatus(ctx, clusterID, feature.Name, clusterfeature.FeatureStatusActive); err != nil {
		mLoger.Debug("failed to update feature status")

		return clusterfeature.NewDatabaseAccessError(feature.Name)
	}

	mLoger.Info("successfully updated feature")

	return nil
}

func (sfm *dnsFeatureManager) CheckPrerequisites(ctx context.Context, clusterID uint, featureName string, featureSpec clusterfeature.FeatureSpec) error {
	mLoger := logur.WithFields(sfm.logger, map[string]interface{}{"clusterId": clusterID, "feature": featureName})
	mLoger.Info("checking prerequisites for feature")

	ready, err := sfm.clusterService.IsClusterReady(ctx, clusterID)
	if err != nil {

		//return errors.WrapIf(err, "could not access cluster")
		// todo refine further the error handling in the underlying call stack
		return clusterfeature.NewClusterNotReadyError(featureName)
	}

	if !ready {
		mLoger.Debug("cluster not ready")

		return clusterfeature.NewClusterNotReadyError(featureName)
	}

	mLoger.Info("prerequisites satisfied")
	return nil

}

func (sfm *dnsFeatureManager) Details(ctx context.Context, clusterID uint, featureName string) (*clusterfeature.Feature, error) {
	mLoger := logur.WithFields(sfm.logger, map[string]interface{}{"clusterId": clusterID, "feature": featureName})
	mLoger.Debug("retrieving feature details ...")

	feature, err := sfm.featureRepository.GetFeature(ctx, clusterID, featureName)
	if err != nil {

		return nil, clusterfeature.NewDatabaseAccessError(featureName)
	}

	if feature == nil {

		return nil, clusterfeature.NewFeatureNotFoundError(featureName)
	}

	mLoger.Debug("successfully retrieved feature details")
	return feature, nil

}
