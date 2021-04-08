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

package deployment_test

import (
	"flag"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/banzaicloud/pipeline/internal/clustergroup/deployment"
	"github.com/banzaicloud/pipeline/internal/cmd"
	"github.com/banzaicloud/pipeline/internal/common"
	"github.com/banzaicloud/pipeline/internal/helm"
	helmtesting "github.com/banzaicloud/pipeline/internal/helm/testing"
)

var clusterId = uint(123) // nolint:gochecknoglobals

func TestIntegration(t *testing.T) {
	if m := flag.Lookup("test.run").Value.String(); m == "" || !regexp.MustCompile(m).MatchString(t.Name()) {
		t.Skip("skipping as execution was not requested explicitly using go test -run")
	}

	helmHome := helmtesting.HelmHome(t)
	t.Run("testGetChartDesc", testGetChartDesc(helmHome))
}

func testGetChartDesc(home string) func(*testing.T) {
	return func(t *testing.T) {
		db := helmtesting.SetupDatabase(t)
		secretStore := helmtesting.SetupSecretStore()
		_, clusterService := helmtesting.ClusterKubeConfig(t, clusterId)

		fakeOrgId := uint(123)
		fakeOrgName := "asd"

		config := helm.Config{
			Home: home,
			Repositories: map[string]string{
				"stable": "https://charts.helm.sh/stable",
			},
		}

		logger := common.NoopLogger{}
		releaser, facade := cmd.CreateUnifiedHelmReleaser(
			config,
			cmd.ClusterConfig{}, // Note: dummy cluster config value.
			db,
			secretStore,
			clusterService,
			helmtesting.FakeOrg{
				OrgId:   fakeOrgId,
				OrgName: fakeOrgName,
			},
			logger,
		)

		helmService := deployment.NewHelmService(facade, releaser)

		chartMeta, err := helmService.GetChartMeta(fakeOrgId, "stable/mysql", "1.6.3")
		if err != nil {
			t.Fatalf("%+v", err)
		}

		assert.Equal(t, "mysql", chartMeta.Name)
		assert.Equal(t, "1.6.3", chartMeta.Version)
		assert.Equal(t, "Fast, reliable, scalable, and easy to use open-source relational database system.", chartMeta.Description)
	}
}
