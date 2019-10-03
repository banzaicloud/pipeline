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

package anchore

import (
	"context"

	"emperror.dev/errors"
	"github.com/mitchellh/mapstructure"

	"github.com/banzaicloud/pipeline/internal/clusterfeature"
	"github.com/banzaicloud/pipeline/internal/common"
)

const securityScanFeatureName = "securityscan"

// ConfigurationService service in charge to gather anchore related configuration
type ConfigurationService interface {
	// GetConfiguration checks if the related cluster feature is activated,
	// gets the configuration from there, otherwise falls back for checking the env for the relevant entries
	GetConfiguration(ctx context.Context, clusterID uint) (Config, error)
}

// configurationService component struct
type configurationService struct {
	defaultConfig  Config
	featureAdapter FeatureAdapter
	logger         common.Logger
}

// MakeConfigurationService create a new configuration service instance
func MakeConfigurationService(defaultCfg Config, featureAdapter FeatureAdapter, log common.Logger) ConfigurationService {
	return configurationService{
		defaultConfig:  defaultCfg,
		featureAdapter: featureAdapter,
		logger:         log,
	}
}

func (c configurationService) GetConfiguration(ctx context.Context, clusterID uint) (Config, error) {
	fnLog := c.logger.WithFields(map[string]interface{}{"clusterID": clusterID, "feature": securityScanFeatureName})

	featureEnabled, err := c.featureAdapter.Enabled(ctx, clusterID, securityScanFeatureName)
	if err != nil {
		fnLog.Debug("failed to check feature")

		return Config{}, errors.WrapIf(err, "failed to check whether feature is enable")
	}

	if !featureEnabled {
		fnLog.Info("feature not enabled, falling back to the default config")

		return c.defaultConfig, nil
	}

	featureConfig, err := c.featureAdapter.GetFeatureConfig(ctx, clusterID, securityScanFeatureName)
	if err != nil {
		fnLog.Debug("failed to retrieve feature config")

		return Config{}, errors.WrapIf(err, "failed to retrieve feature config")
	}

	fnLog.Info("feature enabled, return config from feature")
	return featureConfig, nil
}

// FeatureAdapter decouples feature specifics from the configuration service
type FeatureAdapter interface {
	Enabled(ctx context.Context, clusterID uint, featureName string) (bool, error)
	GetFeatureConfig(ctx context.Context, clusterID uint, featureName string) (Config, error)
}

func MakeFeatureAdapter(featureRepo clusterfeature.FeatureRepository, logger common.Logger) FeatureAdapter {
	return featureAdapter{
		featureRepository: featureRepo,
		logger:            logger,
	}
}

type featureAdapter struct {
	logger            common.Logger
	featureRepository clusterfeature.FeatureRepository
}

func (f featureAdapter) Enabled(ctx context.Context, clusterID uint, featureName string) (bool, error) {
	feature, err := f.featureRepository.GetFeature(ctx, clusterID, featureName)
	if err != nil {
		return false, errors.WrapIf(err, "failed to retrieve feature")
	}

	return feature.Status == clusterfeature.FeatureStatusActive, nil
}

func (f featureAdapter) GetFeatureConfig(ctx context.Context, clusterID uint, featureName string) (Config, error) {

	fnCtx := map[string]interface{}{"clusterID": clusterID, "feature": featureName}
	// add method context to the logger
	f.logger.Info("looking up feature config", fnCtx)

	feature, err := f.featureRepository.GetFeature(ctx, clusterID, featureName)
	if err != nil {
		f.logger.Debug("failed to retrieve feature config", fnCtx)

		return Config{}, errors.WrapIf(err, "failed to retrieve feature")
	}

	customAnchore, ok := feature.Spec["customAnchore"]
	if !ok {
		f.logger.Debug("the feature has no custom anchore config", fnCtx)

		return Config{}, errors.WrapIf(err, "the feature has no custom anchore config")
	}

	var retConfig Config
	if err := mapstructure.Decode(&customAnchore, &retConfig); err != nil {
		f.logger.Debug("failed to decode custom anchore config", fnCtx)

		return Config{}, errors.WrapIf(err, "failed to decode custom anchore config")
	}

	f.logger.Info("feature config retrieved", fnCtx)
	return retConfig, nil
}
