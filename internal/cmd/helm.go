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

	"emperror.dev/emperror"
	"emperror.dev/errors"
	"github.com/jinzhu/gorm"

	"github.com/banzaicloud/pipeline/internal/common"
	"github.com/banzaicloud/pipeline/internal/common/commonadapter"
	"github.com/banzaicloud/pipeline/internal/helm"
	"github.com/banzaicloud/pipeline/internal/helm/helmadapter"
	"github.com/banzaicloud/pipeline/internal/helm2"
	"github.com/banzaicloud/pipeline/src/cluster"
)

type ClusterGetter interface {
	GetClusterByIDOnly(ctx context.Context, clusterID uint) (cluster.CommonCluster, error)
}

// CreateUnifiedHelmReleaser utility function for assembling the helm releaser
func CreateUnifiedHelmReleaser(
	helmConfig helm.Config,
	db *gorm.DB,
	commonSecretStore common.SecretStore,
	clusterService helm.ClusterService,
	logger helm.Logger,
) (helm.UnifiedReleaser, helm.Service) {
	repoStore := helmadapter.NewHelmRepoStore(db, logger)
	secretStore := helmadapter.NewSecretStore(commonSecretStore, logger)
	orgService := helmadapter.NewOrgService(logger)
	validator := helm.NewHelmRepoValidator()
	releaser := helmadapter.NewReleaser(logger)
	helm2EnvResolver := helm.NewHelm2EnvResolver(helmConfig.Home, orgService, logger)

	if !helmConfig.V3 {
		envService := helmadapter.NewHelmEnvService(helmadapter.NewConfig(helmConfig.Repositories), secretStore, logger)
		service := helm.NewService(
			helmConfig,
			repoStore,
			secretStore,
			validator,
			helm.NewEnsuringEnvResolver(helm2EnvResolver, envService, repoStore, helmConfig.Repositories, logger),
			envService,
			releaser,
			clusterService,
			logger)
		return helm2.NewLegacyHelmService(clusterService, service, commonadapter.NewLogger(logger)), service
	}

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

// CreateReleaseDeleter creates a new (helm2 or helm3 specific) deleter instance based on the provided argunments
func CreateReleaseDeleter(helmConfig helm.Config, db *gorm.DB, secretStore helmadapter.SecretStore, logger helm.Logger) helm.ReleaseDeleter {
	logger.Debug("assembling helm release deleter...")
	if helmConfig.V3 {
		repoStore := helmadapter.NewHelmRepoStore(db, logger)
		secretStore := helmadapter.NewSecretStore(secretStore, logger)
		envService := helmadapter.NewHelm3EnvService(secretStore, logger)
		orgService := helmadapter.NewOrgService(logger)
		releaser := helmadapter.NewReleaser(logger)
		helm3EnvResolver := helm.NewHelm3EnvResolver(helmConfig.Home, orgService, logger)
		ensuringEnvResolver := helm.NewEnsuringEnvResolver(helm3EnvResolver, envService, repoStore, helmConfig.Repositories, logger)

		logger.Debug("assembled helm 3 release deleter")
		return helmadapter.NewReleaseDeleter(ensuringEnvResolver, releaser, logger)
	}

	logger.Debug("assembled helm 2 release deleter")
	return helmadapter.NewHelm2ReleaseDeleter()
}
