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
	"github.com/banzaicloud/pipeline/internal/clusterfeature/features"
	"github.com/banzaicloud/pipeline/internal/common"
)

const (
	featureName = "dns"

	// hardcoded values for externalDns feature
	externalDnsChartVersion = "1.6.2"

	// externalDnsImageVersion = "v0.5.11"

	externalDnsChartName = "stable/external-dns"

	externalDnsNamespace = "default"

	externalDnsRelease = "external-dns"
)

// dnsFeatureManager synchronous feature manager
type dnsFeatureManager struct {
	featureRepository    clusterfeature.FeatureRepository
	helmService          features.HelmService
	featureSpecProcessor features.FeatureSpecProcessor

	logger common.Logger
}

// NewDnsFeatureManager builds a new feature manager component
func NewDnsFeatureManager(
	featureRepository clusterfeature.FeatureRepository,
	helmService features.HelmService,
	processor features.FeatureSpecProcessor,
	logger common.Logger,
) clusterfeature.FeatureManager {
	return &dnsFeatureManager{
		featureRepository:    featureRepository,
		helmService:          helmService,
		featureSpecProcessor: processor,

		logger: logger,
	}
}

func (m *dnsFeatureManager) Details(ctx context.Context, clusterID uint) (*clusterfeature.Feature, error) {
	panic("implement me")
}

func (m *dnsFeatureManager) Name() string {
	return "dns"
}

func (m *dnsFeatureManager) Activate(ctx context.Context, clusterID uint, spec clusterfeature.FeatureSpec) error {
	logger := m.logger.WithContext(ctx).WithFields(map[string]interface{}{"cluster": clusterID, "feature": featureName})

	values, err := m.featureSpecProcessor.Process(ctx, clusterID, spec)
	if err != nil {
		logger.Debug("failed to process feature spec")

		return errors.WrapIf(err, "failed to process feature spec")
	}

	if err = m.helmService.InstallDeployment(
		ctx,
		clusterID,
		externalDnsNamespace,
		externalDnsChartName,
		externalDnsRelease,
		values.([]byte),
		externalDnsChartVersion,
		false,
	); err != nil {
		return errors.WrapIf(err, "failed to deploy feature")
	}

	if _, err := m.featureRepository.UpdateFeatureStatus(ctx, clusterID, featureName, clusterfeature.FeatureStatusActive); err != nil {
		return err
	}

	return nil
}

func (m *dnsFeatureManager) ValidateSpec(ctx context.Context, spec clusterfeature.FeatureSpec) error {
	// TODO(laszlop): implement validation
	return nil
}

func (m *dnsFeatureManager) Deactivate(ctx context.Context, clusterID uint) error {
	logger := m.logger.WithContext(ctx).WithFields(map[string]interface{}{"cluster": clusterID, "feature": featureName})

	if err := m.helmService.DeleteDeployment(ctx, clusterID, externalDnsRelease); err != nil {
		logger.Info("failed to delete feature deployment")

		return errors.WrapIf(err, "failed to uninstall feature")
	}

	return nil
}

func (m *dnsFeatureManager) Update(ctx context.Context, clusterID uint, spec clusterfeature.FeatureSpec) error {
	logger := m.logger.WithContext(ctx).WithFields(map[string]interface{}{"clusterId": clusterID, "feature": featureName})

	var valuesJson []byte
	var err error
	if valuesJson, err = json.Marshal(spec); err != nil {
		return errors.WrapIf(err, "failed to update feature")
	}

	// "suspend" the feature till it gets updated
	if _, err = m.featureRepository.UpdateFeatureStatus(ctx, clusterID, featureName, clusterfeature.FeatureStatusPending); err != nil {
		logger.Debug("failed to update feature status")

		return err
	}

	// todo revise this: we loose the "old" spec here
	if _, err = m.featureRepository.UpdateFeatureSpec(ctx, clusterID, featureName, spec); err != nil {
		logger.Debug("failed to update feature spec")

		return err
	}

	if err = m.helmService.UpdateDeployment(ctx, clusterID, externalDnsNamespace,
		externalDnsChartName, externalDnsRelease, valuesJson, externalDnsChartVersion); err != nil {
		logger.Debug("failed to deploy feature")

		// todo feature status in case the upgrade failed?!
		return errors.WrapIf(err, "failed to update feature")
	}

	// feature status set back to active
	if _, err = m.featureRepository.UpdateFeatureStatus(ctx, clusterID, featureName, clusterfeature.FeatureStatusActive); err != nil {
		logger.Debug("failed to update feature status")

		return err
	}

	logger.Info("successfully updated feature")

	return nil
}
