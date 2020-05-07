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
	"regexp"
	"testing"

	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/banzaicloud/pipeline/internal/cmd"
	"github.com/banzaicloud/pipeline/internal/common"
	"github.com/banzaicloud/pipeline/internal/helm"
	"github.com/banzaicloud/pipeline/internal/helm/helmadapter"
	helmtesting "github.com/banzaicloud/pipeline/internal/helm/testing"
	internaltesting "github.com/banzaicloud/pipeline/internal/testing"
	"github.com/banzaicloud/pipeline/pkg/k8sclient"
)

type Values struct {
	Service struct {
		ExternalPort int `json:"externalPort,omitempty"`
	} `json:"service,omitempty"`
}

const (
	v2 = false
	v3 = true
)

var clusterId = uint(123) //nolint:gochecknoglobals

func TestIntegration(t *testing.T) {
	if m := flag.Lookup("test.run").Value.String(); m == "" || !regexp.MustCompile(m).MatchString(t.Name()) {
		t.Skip("skipping as execution was not requested explicitly using go test -run")
	}

	helmHome := helmtesting.HelmHome(t)

	// istio install/delete use cases, now also used by federation
	t.Run("helmV2", testIntegration(v2, helmHome, "istiofeature-helm-v2"))
	t.Run("helmV3", testIntegration(v3, helmHome, "istiofeature-helm-v3"))

	// cluster setup and posthook style use cases
	t.Run("helmInstallV2", testIntegrationInstall(v2, helmHome, "helm-v2-install"))
	t.Run("helmInstallV3", testIntegrationInstall(v3, helmHome, "helm-v3-install"))

	// covers the federation use case for adding a custom platform repository on the fly
	t.Run("addPlatformRepositoryV3", testAddPlatformRepository(helmHome, v3))
	t.Run("addPlatformRepositoryV2", testAddPlatformRepository(helmHome, v2))
}

func testAddPlatformRepository(home string, v3 bool) func(t *testing.T) {
	return func(t *testing.T) {
		db := helmtesting.SetupDatabase(t)
		secretStore := helmtesting.SetupSecretStore()
		_, clusterService := helmtesting.ClusterKubeConfig(t, clusterId)
		config := helm.Config{
			Home: home,
			V3:   v3,
			Repositories: map[string]string{
				"stable": "https://kubernetes-charts.storage.googleapis.com",
			},
		}

		logger := common.NoopLogger{}
		helmService, _ := cmd.CreateUnifiedHelmReleaser(config, db, secretStore, clusterService, helmadapter.NewOrgService(logger), logger)

		for i := 0; i < 2; i++ {
			err := helmService.AddRepositoryIfNotExists(helm.Repository{
				Name: "kubefed",
				URL:  "https://raw.githubusercontent.com/banzaicloud/kubefed/helm_chart/charts",
			})
			if err != nil {
				t.Fatalf("%+v", err)
			}
		}
	}
}

func testIntegration(v3 bool, home, testNamespace string) func(t *testing.T) {
	return func(t *testing.T) {
		db := helmtesting.SetupDatabase(t)
		secretStore := helmtesting.SetupSecretStore()
		kubeConfig, clusterService := helmtesting.ClusterKubeConfig(t, clusterId)

		config := helm.Config{
			Home: home,
			V3:   v3,
			Repositories: map[string]string{
				"stable": "https://kubernetes-charts.storage.googleapis.com",
			},
		}

		logger := common.NoopLogger{}
		helmService, _ := cmd.CreateUnifiedHelmReleaser(config, db, secretStore, clusterService, helmadapter.NewOrgService(logger), logger)

		t.Run("testDeleteChartBeforeSuite", testDeleteChart(helmService, kubeConfig, testNamespace))
		t.Run("testCreateChart", testCreateChart(helmService, kubeConfig, testNamespace))
		t.Run("testUpgradeChart", testUpgradeChart(helmService, kubeConfig, testNamespace))
		t.Run("testHandleFailedDeployment", testUpgradeFailedChart(helmService, kubeConfig, testNamespace))
		t.Run("testDeleteChartAfterSuite", testDeleteChart(helmService, kubeConfig, testNamespace))
	}
}

func testIntegrationInstall(v3 bool, home, testNamespace string) func(t *testing.T) {
	return func(t *testing.T) {
		db := helmtesting.SetupDatabase(t)
		secretStore := helmtesting.SetupSecretStore()
		_, clusterService := helmtesting.ClusterKubeConfig(t, clusterId)

		config := helm.Config{
			Home: home,
			V3:   v3,
			Repositories: map[string]string{
				"stable":             "https://kubernetes-charts.storage.googleapis.com",
				"banzaicloud-stable": "https://kubernetes-charts.banzaicloud.com",
			},
		}

		t.Run("helmv3install", func(t *testing.T) {
			logger := common.NoopLogger{}
			releaser, _ := cmd.CreateUnifiedHelmReleaser(config, db, secretStore, clusterService, helmadapter.NewOrgService(logger), logger)

			err := releaser.InstallDeployment(
				context.Background(),
				clusterId,
				testNamespace,
				"banzaicloud-stable/banzaicloud-docs",
				"helm-service-test-v3",
				[]byte{},
				"0.1.2",
				true,
			)
			require.NoError(t, err)

			err = releaser.DeleteDeployment(
				context.Background(),
				clusterId,
				"helm-service-test-v3",
				testNamespace,
			)
			require.NoError(t, err)
		})
	}
}

func testDeleteChart(helmService helm.UnifiedReleaser, kubeConfig []byte, testNamespace string) func(*testing.T) {
	return func(t *testing.T) {
		err := helmService.Delete(
			&internaltesting.ClusterData{K8sConfig: kubeConfig, ID: clusterId},
			"chartmuseum",
			testNamespace,
		)
		if err != nil {
			t.Fatalf("%+v", err)
		}

		assertChartmuseumRemoved(t, kubeConfig, testNamespace)
	}
}

func testCreateChart(helmService helm.UnifiedReleaser, kubeConfig []byte, testNamespace string) func(*testing.T) {
	return func(t *testing.T) {
		err := helmService.InstallOrUpgrade(
			&internaltesting.ClusterData{K8sConfig: kubeConfig, ID: clusterId},
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

func testUpgradeChart(helmService helm.UnifiedReleaser, kubeConfig []byte, testNamespace string) func(*testing.T) {
	return func(t *testing.T) {
		var expectPort int32 = 19191

		values := Values{}
		values.Service.ExternalPort = int(expectPort)

		serializedValues, err := helm.ConvertStructure(values)
		if err != nil {
			t.Fatalf("%+v", serializedValues)
		}

		err = helmService.InstallOrUpgrade(
			&internaltesting.ClusterData{K8sConfig: kubeConfig, ID: clusterId},
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

func testUpgradeFailedChart(helmService helm.UnifiedReleaser, kubeConfig []byte, testNamespace string) func(*testing.T) {
	return func(t *testing.T) {
		// invalid port will fail the release
		var expectPort int32 = 1111111

		values := Values{}
		values.Service.ExternalPort = int(expectPort)

		serializedValues, err := helm.ConvertStructure(values)
		if err != nil {
			t.Fatalf("%+v", serializedValues)
		}

		err = helmService.InstallOrUpgrade(
			&internaltesting.ClusterData{K8sConfig: kubeConfig, ID: clusterId},
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
			&internaltesting.ClusterData{K8sConfig: kubeConfig, ID: clusterId},
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
