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

package helm_test

import (
	"context"
	"flag"
	"io/ioutil"
	"os"
	"regexp"
	"testing"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"github.com/stretchr/testify/require"

	"github.com/banzaicloud/pipeline/internal/cmd"
	"github.com/banzaicloud/pipeline/internal/common"
	"github.com/banzaicloud/pipeline/internal/common/commonadapter"
	"github.com/banzaicloud/pipeline/internal/global"
	"github.com/banzaicloud/pipeline/internal/helm"
	"github.com/banzaicloud/pipeline/internal/helm/helmadapter"
	helm2 "github.com/banzaicloud/pipeline/src/helm"
	"github.com/banzaicloud/pipeline/src/secret"
)

func setupDatabase(t *testing.T) *gorm.DB {
	db, err := gorm.Open("sqlite3", "file::memory:")
	require.NoError(t, err)

	err = helmadapter.Migrate(db, common.NoopLogger{})
	require.NoError(t, err)

	return db
}

func setupSecretStore(t *testing.T) common.SecretStore {
	return commonadapter.NewSecretStore(secret.Store, commonadapter.OrgIDContextExtractorFunc(func(ctx context.Context) (uint, bool) {
		return 0, false
	}))
}

func clusterKubeConfig(t *testing.T) helm.ClusterService {
	kubeConfigFile := os.Getenv("KUBECONFIG")
	if kubeConfigFile == "" {
		t.Skip("skipping as Kubernetes config was not provided")
	}

	kubeConfigBytes, err := ioutil.ReadFile(kubeConfigFile)
	require.NoError(t, err)

	return helm.ClusterKubeConfigFunc(func(ctx context.Context, clusterID uint) ([]byte, error) {
		return kubeConfigBytes, nil
	})
}

func TestIntegration(t *testing.T) {
	if m := flag.Lookup("test.run").Value.String(); m == "" || !regexp.MustCompile(m).MatchString(t.Name()) {
		t.Skip("skipping as execution was not requested explicitly using go test -run")
	}

	db := setupDatabase(t)
	secretStore := setupSecretStore(t)
	kubeConfig := clusterKubeConfig(t)

	t.Run("helmv3install", func(t *testing.T) {
		home, err := ioutil.TempDir("", "")
		if err != nil {
			t.Fatalf("%+v", err)
		}

		global.Config.Helm.Home = home

		config := helm.Config{
			Home: home,
			V3:   true,
		}

		helm2.GeneratePlatformHelmRepoEnv()

		releaser, _ := cmd.CreateUnifiedHelmReleaser(config, db, secretStore, kubeConfig, common.NoopLogger{})

		err = releaser.InstallDeployment(
			context.Background(),
			1,
			"default",
			"banzaicloud-stable/banzaicloud-docs",
			"helm-service-test-v3",
			[]byte{},
			"0.1.2",
			true,
		)
		require.NoError(t, err)

		err = releaser.DeleteDeployment(
			context.Background(),
			1,
			"helm-service-test-v3",
			"",
		)
		require.NoError(t, err)
	})
}
