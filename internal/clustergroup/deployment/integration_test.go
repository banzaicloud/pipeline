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
	"github.com/banzaicloud/pipeline/internal/global"
	"github.com/banzaicloud/pipeline/internal/helm"
	"github.com/banzaicloud/pipeline/internal/helm/helmadapter"
	helmtesting "github.com/banzaicloud/pipeline/internal/helm/testing"
)

var clusterId = uint(123) // nolint:gochecknoglobals

const (
	v3 = true
	v2 = false
)

func TestIntegration(t *testing.T) {
	if m := flag.Lookup("test.run").Value.String(); m == "" || !regexp.MustCompile(m).MatchString(t.Name()) {
		t.Skip("skipping as execution was not requested explicitly using go test -run")
	}

	helmHome := helmtesting.HelmHome(t)
	t.Run("testGetChartDescV3", testGetChartDesc(helmHome, v3))

	helmHomeV2 := helmtesting.HelmHome(t)
	t.Run("testGetChartDescV2", testGetChartDesc(helmHomeV2, v2))
}

func testGetChartDesc(home string, v3 bool) func(*testing.T) {
	return func(t *testing.T) {
		db := helmtesting.SetupDatabase(t)
		secretStore := helmtesting.SetupSecretStore()
		_, clusterService := helmtesting.ClusterKubeConfig(t, clusterId)

		global.Config.Helm.Home = home
		config := helm.Config{
			Home: home,
			V3:   v3,
			Repositories: map[string]string{
				"stable": "https://kubernetes-charts.storage.googleapis.com",
			},
		}

		logger := common.NoopLogger{}
		_, facade := cmd.CreateUnifiedHelmReleaser(config, db, secretStore, clusterService, helmadapter.NewOrgService(logger), logger)

		helmService := deployment.NewHelmService(facade)

		description, err := helmService.GetChartDescription("stable/mysql", "1.6.3")
		if err != nil {
			t.Fatalf("%+v", err)
		}

		assert.Equal(t, "Fast, reliable, scalable, and easy to use open-source relational database system.", description)
	}
}
