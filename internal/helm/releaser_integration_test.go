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
	"flag"
	"io/ioutil"
	"regexp"
	"strings"
	"testing"

	"emperror.dev/errors"
	"github.com/stretchr/testify/assert"

	"github.com/banzaicloud/pipeline/internal/common"
	"github.com/banzaicloud/pipeline/internal/helm"
	"github.com/banzaicloud/pipeline/internal/helm/helmadapter"
)

// TestReleaser integration test for releaser operations
func TestReleaser(t *testing.T) {
	if m := flag.Lookup("test.run").Value.String(); m == "" || !regexp.MustCompile(m).MatchString(t.Name()) {
		t.Skip("skipping as execution was not requested explicitly using go test -run")
	}

	// TODO try to test the helm 2 operations as well
	t.Run("testReleaserHelmV3", testReleaserHelmV3())

}

func testReleaserHelmV3() func(t *testing.T) {

	return func(t *testing.T) {
		helmFacade := getHelmFacade(t)
		t.Run("deleteRelease", testDeleteRelease(context.Background(), helmFacade, getTestReleases()[0].ReleaseName, helm.Options{}))
		t.Run("installRelease", testInstallRelease(context.Background(), helmFacade, getTestReleases()[0], helm.Options{}))
		t.Run("getRelease", testGetRelease(context.Background(), helmFacade, getTestReleases()[0].ReleaseName, helm.Options{}))
		filter := "a"
		t.Run("listReleasesWithFilter", testListReleaseWithFilter(context.Background(), helmFacade, helm.ReleaseFilter{Filter: &filter}, helm.Options{}))
	}
}
func setUpHelmConfig(t *testing.T) helm.Config {
	home, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatalf("failed to set up helm config for testing: #%v", err)
	}

	config := helm.Config{
		Home:         home,
		Repositories: map[string]string{"stable": "https://kubernetes-charts.storage.googleapis.com"},
		V3:           true,
	}
	return config
}

// setupOrgService mocks the org service
func setupOrgService(ctx context.Context, t *testing.T) helm.OrgService {
	t.Log("set up org service mock...")
	orgService := &helm.MockOrgService{}
	orgService.On("GetOrgNameByOrgID", ctx, uint(1)).Return("test_org", nil)
	return orgService
}

func getTestReleases() []helm.Release {
	return []helm.Release{
		{ReleaseName: "release-abc", ChartName: "stable/mysql", Namespace: "default", Values: nil, Version: "5.7.28"},
		{ReleaseName: "release-efg", ChartName: "stable/mysql", Namespace: "default", Values: nil, Version: "5.7.28"},
	}
}

// getHelmFacade sets up the service being tested
func getHelmFacade(t *testing.T) helm.Service {
	logger := common.NoopLogger{}
	testCtx := context.Background()
	db := setupDatabase(t)
	helmConfig := setUpHelmConfig(t)
	secretStore := helmadapter.NewSecretStore(setupSecretStore(t), logger)
	repoStore := helmadapter.NewHelmRepoStore(db, logger)
	envResolver := helm.NewHelm3EnvResolver(helmConfig.Home, setupOrgService(testCtx, t), logger)
	envService := helmadapter.NewHelm3EnvService(secretStore, logger)
	ensuringEnvResolver := helm.NewEnsuringEnvResolver(envResolver, envService, repoStore, helmConfig.Repositories, logger)

	return helm.NewService(
		helmConfig,
		repoStore,
		secretStore,
		helm.NewHelmRepoValidator(),
		ensuringEnvResolver,
		envService,
		helmadapter.NewReleaser(logger),
		clusterKubeConfig(t),
		logger)
}

func testInstallRelease(ctx context.Context, helmFacade helm.Service, releaseInput helm.Release, options helm.Options) func(t *testing.T) {
	return func(t *testing.T) {
		t.Logf("installing release %#v", releaseInput)
		if err := helmFacade.InstallRelease(ctx, 1, 1, releaseInput, options); err != nil {
			t.Fatalf("failed to install release %#v", releaseInput)
		}
		//TODO assertions on successful installation
	}
}

func testListReleaseWithFilter(ctx context.Context, helmFacade helm.Service, filter helm.ReleaseFilter, options helm.Options) func(t *testing.T) {
	return func(t *testing.T) {
		t.Logf("listing releases; filter: %#v", filter)

		releases, err := helmFacade.ListReleases(ctx, 1, 1, filter, options)
		if err != nil {
			t.Fatalf("failed to list releases; filter %#v", filter)
		}

		assert.Equal(t, 1, len(releases), "found more then one release")
	}
}

func testDeleteRelease(ctx context.Context, helmFacade helm.Service, releaseName string, options helm.Options) func(t *testing.T) {
	return func(t *testing.T) {
		t.Logf("deleting release %#v", releaseName)
		if err := helmFacade.DeleteRelease(ctx, 1, 1, releaseName, options); err != nil {
			if !ErrReleaseNotFound(err) {
				t.Fatalf("failed to delete release %#v", releaseName)
			}
		}
		//TODO assertions successful deletion
	}
}

func testGetRelease(ctx context.Context, helmFacade helm.Service, releaseName string, options helm.Options) func(t *testing.T) {
	return func(t *testing.T) {
		t.Logf("deleting release %#v", releaseName)
		if _, err := helmFacade.GetRelease(ctx, 1, 1, releaseName, options); err != nil {
			t.Fatalf("failed to delete release %#v", releaseName)
		}
		//TODO assertions successful deletion
	}
}

// TODO make this more robust
func ErrReleaseNotFound(err error) bool {
	return strings.Contains(errors.Cause(err).Error(), "not found")
}
