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

package helm_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/banzaicloud/pipeline/internal/common"
	"github.com/banzaicloud/pipeline/internal/helm"
	"github.com/banzaicloud/pipeline/internal/helm/helmadapter"
	testing2 "github.com/banzaicloud/pipeline/internal/helm/testing"
)

func testPlatformEnvResolver(t *testing.T) {
	logger := common.NoopLogger{}
	testCtx := context.Background()
	db := testing2.SetupDatabase(t)
	helmConfig := setUpHelmConfig(t)
	secretStore := helmadapter.NewSecretStore(testing2.SetupSecretStore(), logger)
	repoStore := helmadapter.NewHelmRepoStore(db, logger)
	envResolver := helm.NewHelm3EnvResolver(helmConfig.Home, setupOrgService(testCtx, t), logger)
	envService := helmadapter.NewHelm3EnvService(secretStore, logger)
	ensuringEnvResolver := helm.NewEnsuringEnvResolver(envResolver, envService, repoStore, helmConfig.Repositories, logger)

	assert.NoFileExists(t, helmConfig.Home, "the helm home must not exist")
	helmEnv, err := ensuringEnvResolver.ResolvePlatformEnv(testCtx)

	assert.Nil(t, err)
	assert.FileExists(t, helmEnv.GetHome(), "the platform helm home must have been created")

	platformRepos, err := envService.ListRepositories(testCtx, helmEnv)
	assert.Nil(t, err, "the list repositories call must succeed")

	assert.Equal(t, len(helmConfig.Repositories), len(platformRepos), "default repositories must be installed")
}

func testOrgEnvResolver(t *testing.T) {
	logger := common.NoopLogger{}
	testCtx := context.Background()
	db := testing2.SetupDatabase(t)
	helmConfig := setUpHelmConfig(t)
	secretStore := helmadapter.NewSecretStore(testing2.SetupSecretStore(), logger)
	repoStore := helmadapter.NewHelmRepoStore(db, logger)
	envResolver := helm.NewHelm3EnvResolver(helmConfig.Home, setupOrgService(testCtx, t), logger)
	envService := helmadapter.NewHelm3EnvService(secretStore, logger)
	ensuringEnvResolver := helm.NewEnsuringEnvResolver(envResolver, envService, repoStore, helmConfig.Repositories, logger)

	// add (one) user-defined repository to the database
	if err := repoStore.Create(testCtx, 1, helm.Repository{
		Name: "banzai-stable",
		URL:  "https://kubernetes-charts.banzaicloud.com",
	}); err != nil {
		t.Fatal("failed to setup database for testing")
	}

	assert.NoFileExists(t, helmConfig.Home, "the helm home must not exist")
	helmEnv, err := ensuringEnvResolver.ResolveHelmEnv(testCtx, 1)

	assert.Nil(t, err)
	assert.FileExists(t, helmEnv.GetHome(), "the platform helm home must have been created")

	platformRepos, err := envService.ListRepositories(testCtx, helmEnv)
	assert.Nil(t, err, "the list repositories call must succeed")

	assert.Equal(t, len(helmConfig.Repositories)+1, len(platformRepos), "default repositories must be installed along with the user registered ones")
}
