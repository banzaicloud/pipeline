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
	"io/ioutil"
	"strings"
	"testing"

	"emperror.dev/errors"
	"github.com/stretchr/testify/assert"

	"github.com/banzaicloud/pipeline/internal/common"
	"github.com/banzaicloud/pipeline/internal/helm"
	"github.com/banzaicloud/pipeline/internal/helm/helmadapter"
	helmtesting "github.com/banzaicloud/pipeline/internal/helm/testing"
)

func testReleaserHelm(t *testing.T) {
	helmFacade := getHelmFacade(t)
	hasRun := t.Run("deleteReleaseBefore", testDeleteRelease(context.Background(), helmFacade, getTestReleases()[0].ReleaseName, helm.Options{}))
	if !hasRun {
		t.Fatal("failed to delete release")
	}

	hasRun = t.Run("installRelease", testInstallRelease(context.Background(), helmFacade, getTestReleases()[0], helm.Options{}))
	if !hasRun {
		t.Fatal("failed to install release")
	}

	hasRun = t.Run("getRelease", testGetRelease(context.Background(), helmFacade, getTestReleases()[0].ReleaseName, helm.Options{}))
	if !hasRun {
		t.Fatal("failed to get release")
	}

	filter := "release"
	hasRun = t.Run("listReleasesWithFilter", testListReleaseWithFilter(context.Background(), helmFacade, helm.ReleaseFilter{Filter: &filter}, helm.Options{}))
	if !hasRun {
		t.Fatal("failed list release with filter")
	}

	hasRun = t.Run("deleteReleaseAfter", testDeleteRelease(context.Background(), helmFacade, getTestReleases()[0].ReleaseName, helm.Options{}))
	if !hasRun {
		t.Fatal("failed to delete release")
	}
}

func setUpHelmConfig(t *testing.T) helm.Config {
	home, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatalf("failed to set up helm config for testing: #%v", err)
	}

	config := helm.Config{
		Home:         home,
		Repositories: map[string]string{"stable": "https://charts.helm.sh/stable"},
	}
	return config
}

// setupOrgService mocks the org service
func setupOrgService(ctx context.Context, t *testing.T) helm.OrgService {
	t.Log("set up org service mock...")
	orgService := &MockOrgService{}
	orgService.On("GetOrgNameByOrgID", ctx, uint(1)).Return("test_org", nil)
	return orgService
}

func getTestReleases() []helm.Release {
	return []helm.Release{
		{ReleaseName: "release-abc", ChartName: "stable/mysql", Namespace: "default", Values: nil, Version: "1.6.6"},
		{ReleaseName: "release-efg", ChartName: "stable/mysql", Namespace: "default", Values: nil, Version: "1.6.6"},
	}
}

// getHelmFacade sets up the service being tested
func getHelmFacade(t *testing.T) helm.Service {
	logger := common.NoopLogger{}
	testCtx := context.Background()
	db := helmtesting.SetupDatabase(t)
	helmConfig := setUpHelmConfig(t)
	secretStore := helmadapter.NewSecretStore(helmtesting.SetupSecretStore(), logger)
	repoStore := helmadapter.NewHelmRepoStore(db, logger)
	envResolver := helm.NewHelm3EnvResolver(helmConfig.Home, setupOrgService(testCtx, t), logger)
	envService := helmadapter.NewHelm3EnvService(secretStore, logger)
	ensuringEnvResolver := helm.NewEnsuringEnvResolver(envResolver, envService, repoStore, helmConfig.Repositories, logger)

	_, clusterConfigProvider := helmtesting.ClusterKubeConfig(t, clusterId)

	return helm.NewService(
		helmConfig,
		[]helm.ChartConfig{},
		repoStore,
		secretStore,
		helm.NewHelmRepoValidator(),
		ensuringEnvResolver,
		envService,
		helmadapter.NewReleaser(logger),
		clusterConfigProvider,
		logger)
}

func testInstallRelease(ctx context.Context, helmFacade helm.Service, releaseInput helm.Release, options helm.Options) func(t *testing.T) {
	return func(t *testing.T) {
		t.Logf("installing release %#v", releaseInput)
		if _, err := helmFacade.InstallRelease(ctx, 1, 123, releaseInput, options); err != nil {
			t.Fatalf("failed to install release %#v", releaseInput)
		}

		// assertions
		rel, err := helmFacade.GetRelease(ctx, 1, 123, releaseInput.ReleaseName, options)
		assert.Nil(t, err)
		assert.Equal(t, "deployed", rel.ReleaseInfo.Status)
	}
}

func testListReleaseWithFilter(ctx context.Context, helmFacade helm.Service, filter helm.ReleaseFilter, options helm.Options) func(t *testing.T) {
	return func(t *testing.T) {
		t.Logf("listing releases; filter: %#v", filter)

		releases, err := helmFacade.ListReleases(ctx, 1, 123, filter, options)
		if err != nil {
			t.Fatalf("failed to list releases; filter %#v", filter)
		}

		// we assume there is a single result
		assert.Equal(t, 1, len(releases), "found more then one release")

		// fake filter
		inexistingFilter := "InexistingReleaseName"
		filter.Filter = &inexistingFilter
		releases, notFound := helmFacade.ListReleases(ctx, 1, 123, filter, options)

		assert.Nil(t, notFound)
		assert.Equal(t, 0, len(releases), "no release should match the filter")
	}
}

func testDeleteRelease(ctx context.Context, helmFacade helm.Service, releaseName string, options helm.Options) func(t *testing.T) {
	return func(t *testing.T) {
		t.Logf("deleting release %#v", releaseName)
		if err := helmFacade.DeleteRelease(ctx, 1, 123, releaseName, options); err != nil {
			if !errReleaseNotFound(err) {
				t.Fatalf("failed to delete release %q: %s", releaseName, err)
			}
		}

		// the release can't be found
		_, err := helmFacade.GetRelease(ctx, 1, 123, releaseName, options)
		if err == nil {
			t.Fatalf("release %q is not deleted", releaseName)
		} else if !errReleaseNotFound(err) {
			t.Fatalf("failed to check if release %q is deleted: %s", releaseName, err)
		}
	}
}

func testGetRelease(ctx context.Context, helmFacade helm.Service, releaseName string, options helm.Options) func(t *testing.T) {
	return func(t *testing.T) {
		t.Logf("deleting release %#v", releaseName)
		release, err := helmFacade.GetRelease(ctx, 1, 123, releaseName, options)
		if err != nil {
			t.Fatalf("failed to retrieve release %#v", releaseName)
		}

		assert.Equal(t, getTestReleases()[0].ReleaseName, release.ReleaseName)
	}
}

func errReleaseNotFound(err error) bool {
	return strings.Contains(errors.Cause(err).Error(), "not found")
}
