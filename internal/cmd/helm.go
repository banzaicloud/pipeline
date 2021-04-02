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

package cmd

import (
	"context"
	"sort"
	"strings"

	"emperror.dev/emperror"
	"emperror.dev/errors"
	"github.com/jinzhu/gorm"
	"github.com/mitchellh/mapstructure"

	"github.com/banzaicloud/pipeline/internal/common"
	"github.com/banzaicloud/pipeline/internal/helm"
	"github.com/banzaicloud/pipeline/internal/helm/helmadapter"
	"github.com/banzaicloud/pipeline/src/cluster"
)

type ClusterGetter interface {
	GetClusterByIDOnly(ctx context.Context, clusterID uint) (cluster.CommonCluster, error)
}

// CreateUnifiedHelmReleaser utility function for assembling the helm releaser
func CreateUnifiedHelmReleaser(
	helmConfig helm.Config,
	clusterConfig ClusterConfig,
	db *gorm.DB,
	commonSecretStore common.SecretStore,
	clusterService helm.ClusterService,
	orgService helm.OrgService,
	logger helm.Logger,
) (helm.UnifiedReleaser, helm.Service) {
	repoStore := helmadapter.NewHelmRepoStore(db, logger)
	secretStore := helmadapter.NewSecretStore(commonSecretStore, logger)
	validator := helm.NewHelmRepoValidator()
	releaser := helmadapter.NewReleaser(logger)

	envResolver := helm.NewHelm3EnvResolver(helmConfig.Home, orgService, logger)
	envService := helmadapter.NewHelm3EnvService(secretStore, logger)
	// wrap the envresolver
	ensuringEnvResolver := helm.NewEnsuringEnvResolver(envResolver, envService, repoStore, helmConfig.Repositories, logger)

	// set up platform helm env
	platformHelmEnv, _ := envResolver.ResolvePlatformEnv(context.Background())
	reconciler := helm.NewBuiltinEnvReconciler(helmConfig.Repositories, envService, logger)
	if err := reconciler.Reconcile(context.Background(), platformHelmEnv); err != nil {
		emperror.Panic(errors.Wrap(err, "failed to set up platform helm environment"))
	}

	service := helm.NewService(
		helmConfig,
		newClusterChartsFromConfig(clusterConfig),
		repoStore,
		secretStore,
		validator,
		ensuringEnvResolver,
		envService,
		releaser,
		clusterService,
		logger)

	return helmadapter.NewUnifiedHelm3Releaser(service, logger), service
}

// CreateReleaseDeleter creates a new helm3 specific deleter instance based on the provided argunments
func CreateReleaseDeleter(helmConfig helm.Config, db *gorm.DB, secretStore helmadapter.SecretStore, logger helm.Logger) helm.ReleaseDeleter {
	logger.Debug("assembling helm release deleter...")

	repoStore := helmadapter.NewHelmRepoStore(db, logger)
	envService := helmadapter.NewHelm3EnvService(helmadapter.NewSecretStore(secretStore, logger), logger)
	orgService := helmadapter.NewOrgService(logger)
	releaser := helmadapter.NewReleaser(logger)
	helm3EnvResolver := helm.NewHelm3EnvResolver(helmConfig.Home, orgService, logger)
	ensuringEnvResolver := helm.NewEnsuringEnvResolver(helm3EnvResolver, envService, repoStore, helmConfig.Repositories, logger)

	logger.Debug("assembled helm 3 release deleter")
	return helmadapter.NewReleaseDeleter(ensuringEnvResolver, releaser, logger)
}

// isConfigEnabled determines whether the specified configuration is enabled.
// Configurations which cannot be disabled return true.
func isConfigEnabled(config interface{}) bool {
	if config == nil {
		return false
	}

	var enabledConfig struct {
		Enabled *bool // Note: pointer indicates field existence, missing field yields zero value instead of error.
	}
	cannotBeDisabledError := mapstructure.Decode(config, &enabledConfig)

	return cannotBeDisabledError != nil || // Note: not an associative type, no Enabled field, enabled by default.
		enabledConfig.Enabled == nil || // Note: associative type, but no Enabled field, enabled by default.
		*enabledConfig.Enabled // Note: associative type, has Enabled field, its value is used.
}

// newClusterChartsFromConfig creates the cluster chart collection from the
// cluster configuration.
func newClusterChartsFromConfig(clusterConfig ClusterConfig) (clusterChartConfigs []helm.ChartConfig) {
	clusterChartConfigs = parseClusterChartConfigsRecursively(nil, nil, clusterConfig)

	sort.Slice(clusterChartConfigs, func(firstIndex, secondIndex int) (isLessThan bool) {
		return clusterChartConfigs[firstIndex].IsLessThan(clusterChartConfigs[secondIndex])
	})

	return clusterChartConfigs
}

// parseClusterChartConfigsRecursively collects the availbale cluster chart
// configurations from the specified config map.
//
// For the initial call, decoder and decoderConfig MAY be nil, they are
// automatically initialized.
func parseClusterChartConfigsRecursively(
	decoder *mapstructure.Decoder,
	decoderConfig *mapstructure.DecoderConfig,
	config interface{},
) (clusterChartConfigs []helm.ChartConfig) {
	if config == nil ||
		!isConfigEnabled(config) { // Note: explicitly disabled configs are not parsed.
		return nil
	}

	if decoderConfig == nil {
		decoderConfig = &mapstructure.DecoderConfig{
			DecodeHook:       nil,
			ErrorUnused:      false,
			ZeroFields:       false,
			WeaklyTypedInput: false,
			Squash:           false,
			Metadata:         nil,
			Result:           new(string), // Note: dummy value, gonna be overwritten.
			TagName:          "chartConfig",
		}
	}

	if decoder == nil {
		decoder, _ = mapstructure.NewDecoder(decoderConfig) // Note: the hard coded configuration never panics.
	}

	var chart struct { // Note: pointers indicate field existence, missing field yields zero value instead of error.
		Chart   *string
		Version *string
		Values  map[string]interface{}
	}
	decoderConfig.Result = &chart
	err := decoder.Decode(config)
	if err == nil &&
		chart.Chart != nil &&
		chart.Version != nil {
		name := *chart.Chart
		repo := ""
		lastSeparatorIndex := strings.LastIndex(*chart.Chart, "/")
		if lastSeparatorIndex != -1 {
			repo = (*chart.Chart)[:lastSeparatorIndex]
			name = (*chart.Chart)[lastSeparatorIndex+1:]
		}

		return []helm.ChartConfig{
			{
				Name:       name,
				Repository: repo,
				Version:    *chart.Version,
				Values:     chart.Values,
			},
		}
	}

	var configMap map[string]interface{}
	decoderConfig.Result = &configMap
	err = decoder.Decode(config)
	if err == nil &&
		len(configMap) > 0 { // Note: map, struct or pointers to those.
		for _, subconfig := range configMap {
			clusterChartConfigs = append(
				clusterChartConfigs,
				parseClusterChartConfigsRecursively(decoder, decoderConfig, subconfig)...,
			)
		}

		return clusterChartConfigs
	}

	var configs []interface{}
	decoderConfig.Result = &configs
	err = decoder.Decode(config)
	if err == nil &&
		len(configs) > 0 { // Note: slice or pointer to that.
		for _, subconfig := range configs {
			clusterChartConfigs = append(
				clusterChartConfigs,
				parseClusterChartConfigsRecursively(decoder, decoderConfig, subconfig)...,
			)
		}

		return clusterChartConfigs
	}

	return nil // Note: else it is a basic type in which case there is nothing to do.
}
