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

package istiofeature_test

import (
	"context"
	"flag"
	"io/ioutil"
	"os"
	"regexp"
	"testing"

	"github.com/banzaicloud/pipeline/internal/cmd"
	"github.com/banzaicloud/pipeline/internal/common"
	"github.com/banzaicloud/pipeline/internal/common/commonadapter"
	"github.com/banzaicloud/pipeline/internal/global"
	"github.com/banzaicloud/pipeline/internal/helm"
	"github.com/banzaicloud/pipeline/internal/helm/helmadapter"
	"github.com/banzaicloud/pipeline/internal/istio/istiofeature"
	"github.com/banzaicloud/pipeline/pkg/k8sclient"
	"github.com/banzaicloud/pipeline/src/secret"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type Values struct {
	Service struct {
		ExternalPort int `json:"externalPort,omitempty"`
	} `json:"service,omitempty"`
	UpdateStrategy struct {
		RollingUpdate struct {
			MaxUnavailable int `json:"maxUnavailable,omitempty"`
		} `json:"rollingUpdate,omitempty"`
	} `json:"updateStrategy,omitempty"`
}

func TestIntegration(t *testing.T) {
	if m := flag.Lookup("test.run").Value.String(); m == "" || !regexp.MustCompile(m).MatchString(t.Name()) {
		t.Skip("skipping as execution was not requested explicitly using go test -run")
	}

	var err error
	global.Config.Helm.Home, err = ioutil.TempDir("", "")
	if err != nil {
		t.Fatalf("%+v", err)
	}

	kubeConfigFile := os.Getenv("KUBECONFIG")
	if kubeConfigFile == "" {
		t.Skip("skipping as Kubernetes config was not provided")
	}

	kubeConfig, err := ioutil.ReadFile(kubeConfigFile)
	if err != nil {
		t.Fatalf("%+v", err)
	}

	t.Run("helmV2", testIntegrationV2(kubeConfig, "istiofeature-helm"))
	t.Run("helmV3", testIntegrationV3(kubeConfig, global.Config.Helm.Home, "istiofeature-helm-v3"))
}

func testIntegrationV2(kubeConfig []byte, testNamespace string) func(t *testing.T) {
	return func(t *testing.T) {
		helmService := &istiofeature.LegacyV2HelmService{}
		t.Run("testDeleteChartmuseumBeforeSuite", testDeleteChartmuseum(helmService, kubeConfig, testNamespace))
		t.Run("testCreateChartmuseum", testCreateChartmuseum(helmService, kubeConfig, testNamespace))
		t.Run("testUpgradeChartmuseum", testUpgradeChartmuseum(helmService, kubeConfig, testNamespace))
		t.Run("testHandleFailedDeployment", testUpgradeFailedChartmuseum(helmService, kubeConfig, testNamespace))
		t.Run("testDeleteChartmuseumAfterSuite", testDeleteChartmuseum(helmService, kubeConfig, testNamespace))
	}
}

func testIntegrationV3(kubeConfig []byte, home, testNamespace string) func(t *testing.T) {
	return func(t *testing.T) {
		db := setupDatabase(t)
		secretStore := setupSecretStore()
		clusterService := clusterKubeConfig(t, kubeConfig)

		config := helm.Config{
			Home: home,
			V3:   true,
			Repositories: map[string]string{
				"stable": "https://kubernetes-charts.storage.googleapis.com",
			},
		}

		_, helmFacade := cmd.CreateUnifiedHelmReleaser(config, db, secretStore, clusterService, common.NoopLogger{})

		helmService := istiofeature.NewHelmV3Service(helmFacade)

		t.Run("testDeleteChartmuseumBeforeSuite", testDeleteChartmuseum(helmService, kubeConfig, testNamespace))
		t.Run("testCreateChartmuseum", testCreateChartmuseum(helmService, kubeConfig, testNamespace))
		t.Run("testUpgradeChartmuseum", testUpgradeChartmuseum(helmService, kubeConfig, testNamespace))
		t.Run("testHandleFailedDeployment", testUpgradeFailedChartmuseum(helmService, kubeConfig, testNamespace))
		t.Run("testDeleteChartmuseumAfterSuite", testDeleteChartmuseum(helmService, kubeConfig, testNamespace))
	}
}

func testDeleteChartmuseum(helmService istiofeature.HelmService, kubeConfig []byte, testNamespace string) func(*testing.T) {
	return func(t *testing.T) {
		err := helmService.Delete(
			&istiofeature.ClusterProviderData{K8sConfig: kubeConfig, ID: 1},
			"chartmuseum",
			testNamespace,
		)
		if err != nil {
			t.Fatalf("%+v", err)
		}

		assertChartmuseumRemoved(t, kubeConfig, testNamespace)
	}
}

func testCreateChartmuseum(helmService istiofeature.HelmService, kubeConfig []byte, testNamespace string) func(*testing.T) {
	return func(t *testing.T) {
		err := helmService.InstallOrUpgrade(
			&istiofeature.ClusterProviderData{K8sConfig: kubeConfig, ID: 1},
			helm.Release{
				ReleaseName: "chartmuseum",
				ChartName:   "stable/chartmuseum",
				Namespace:   testNamespace,
				Values:      nil,
				Version:     "2.12.0",
			},
			helm.Options{
				Namespace: testNamespace,
				Wait:      true,
				Install:   true,
			},
		)
		if err != nil {
			t.Fatalf("%+v", err)
		}

		assertChartmuseum(t, kubeConfig, testNamespace, 8080)
	}
}

func testUpgradeChartmuseum(helmService istiofeature.HelmService, kubeConfig []byte, testNamespace string) func(*testing.T) {
	return func(t *testing.T) {
		var expectPort int32 = 19191

		values := Values{}
		values.Service.ExternalPort = int(expectPort)

		serializedValues, err := istiofeature.ConvertStructure(values)
		if err != nil {
			t.Fatalf("%+v", serializedValues)
		}

		err = helmService.InstallOrUpgrade(
			&istiofeature.ClusterProviderData{K8sConfig: kubeConfig, ID: 1},
			helm.Release{
				ReleaseName: "chartmuseum",
				ChartName:   "stable/chartmuseum",
				Namespace:   testNamespace,
				Values:      serializedValues,
				Version:     "2.12.0",
			},
			helm.Options{
				Namespace: testNamespace,
				Wait:      true,
				Install:   true,
			},
		)
		if err != nil {
			t.Fatalf("%+v", err)
		}

		assertChartmuseum(t, kubeConfig, testNamespace, expectPort)
	}
}

func testUpgradeFailedChartmuseum(helmService istiofeature.HelmService, kubeConfig []byte, testNamespace string) func(*testing.T) {
	return func(t *testing.T) {
		// invalid port will fail the release
		var expectPort int32 = 1111111

		values := Values{}
		values.Service.ExternalPort = int(expectPort)

		serializedValues, err := istiofeature.ConvertStructure(values)
		if err != nil {
			t.Fatalf("%+v", serializedValues)
		}

		err = helmService.InstallOrUpgrade(
			&istiofeature.ClusterProviderData{K8sConfig: kubeConfig, ID: 1},
			helm.Release{
				ReleaseName: "chartmuseum",
				ChartName:   "stable/chartmuseum",
				Namespace:   testNamespace,
				Values:      serializedValues,
				Version:     "2.12.0",
			},
			helm.Options{
				Namespace: testNamespace,
				Wait:      true,
				Install:   true,
			},
		)
		if err == nil {
			t.Fatalf("this upgrade should fail because of the invalid port")
		}

		// restore with original values
		err = helmService.InstallOrUpgrade(
			&istiofeature.ClusterProviderData{K8sConfig: kubeConfig, ID: 1},
			helm.Release{
				ReleaseName: "chartmuseum",
				ChartName:   "stable/chartmuseum",
				Namespace:   testNamespace,
				Values:      nil,
				Version:     "2.12.0",
			},
			helm.Options{
				Namespace: testNamespace,
				Wait:      true,
				Install:   true,
			},
		)
		if err != nil {
			t.Fatalf("%+v", err)
		}

		assertChartmuseum(t, kubeConfig, testNamespace, 8080)
	}
}

func assertChartmuseum(t *testing.T, kubeConfig []byte, testNamespace string, expectedPort int32) {
	restConfig, err := k8sclient.NewClientConfig(kubeConfig)
	if err != nil {
		t.Fatalf("%+v", err)
	}

	clientSet, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		t.Fatalf("%+v", err)
	}

	ds, err := clientSet.AppsV1().Deployments(testNamespace).Get("chartmuseum-chartmuseum", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("%+v", err)
	}

	if ds.Status.ReadyReplicas != ds.Status.Replicas || ds.Status.ReadyReplicas < 1 {
		t.Fatalf("chartmuseum is not running")
	}

	svc, err := clientSet.CoreV1().Services(testNamespace).Get("chartmuseum-chartmuseum", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("%+v", err)
	}

	if len(svc.Spec.Ports) < 1 {
		t.Fatalf("Missing chartmuseum service ports")
	}

	if svc.Spec.Ports[0].Port != expectedPort {
		t.Fatalf("chartmuseum service port mismatch, expected %d vs %d", expectedPort, svc.Spec.Ports[0].Port)
	}
}

func assertChartmuseumRemoved(t *testing.T, kubeConfig []byte, testNamespace string) {
	restConfig, err := k8sclient.NewClientConfig(kubeConfig)
	if err != nil {
		t.Fatalf("%+v", err)
	}

	clientSet, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		t.Fatalf("%+v", err)
	}

	dsList, err := clientSet.AppsV1().Deployments(testNamespace).List(metav1.ListOptions{})
	if err != nil {
		t.Fatalf("%+v", err)
	}

	if len(dsList.Items) > 0 {
		t.Fatalf("no deployments expected, chartmuseum should be removed")
	}
}

func setupDatabase(t *testing.T) *gorm.DB {
	db, err := gorm.Open("sqlite3", "file::memory:")
	require.NoError(t, err)

	err = helmadapter.Migrate(db, common.NoopLogger{})
	require.NoError(t, err)

	return db
}

func setupSecretStore() common.SecretStore {
	return commonadapter.NewSecretStore(secret.Store, commonadapter.OrgIDContextExtractorFunc(func(ctx context.Context) (uint, bool) {
		return 0, false
	}))
}

func clusterKubeConfig(t *testing.T, kubeConfigBytes []byte) helm.ClusterService {
	return helm.ClusterKubeConfigFunc(func(ctx context.Context, clusterID uint) ([]byte, error) {
		return kubeConfigBytes, nil
	})
}
