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
	noOrgID   = 0 // implicitly means platform helm env
	clusterId = uint(123)
)

func TestIntegration(t *testing.T) {
	if m := flag.Lookup("test.run").Value.String(); m == "" || !regexp.MustCompile(m).MatchString(t.Name()) {
		t.Skip("skipping as execution was not requested explicitly using go test -run")
	}

	// istio install/delete use cases, now also used by federation
	t.Run("helm", testIntegration)

	// cluster setup and posthook style use cases
	t.Run("helmInstall", testIntegrationInstall)

	// Env resolver
	t.Run("platformEnvResolver", testPlatformEnvResolver)
	t.Run("orgEnvResolver", testOrgEnvResolver)

	// Releaser
	t.Run("testReleaserHelm", testReleaserHelm)
}

func testIntegration(t *testing.T) {
	home := helmtesting.HelmHome(t)
	db := helmtesting.SetupDatabase(t)
	secretStore := helmtesting.SetupSecretStore()
	kubeConfig, clusterService := helmtesting.ClusterKubeConfig(t, clusterId)

	config := helm.Config{
		Home: home,
		Repositories: map[string]string{
			"stable": "https://charts.helm.sh/stable",
		},
	}

	const testNamespace = "istiofeature-helm"

	logger := common.NoopLogger{}
	helmService, _ := cmd.CreateUnifiedHelmReleaser(
		config,
		cmd.ClusterConfig{}, // Note: dummy cluster config value.
		db,
		secretStore,
		clusterService,
		helmadapter.NewOrgService(logger),
		logger,
	)

	t.Run("testDeleteChartBeforeSuite", testDeleteChart(helmService, kubeConfig, testNamespace))
	t.Run("testCreateChart", testCreateChart(helmService, kubeConfig, testNamespace))
	t.Run("testUpgradeChart", testUpgradeChart(helmService, kubeConfig, testNamespace))
	t.Run("testHandleFailedDeployment", testUpgradeFailedChart(helmService, kubeConfig, testNamespace))
	t.Run("testDeleteChartAfterSuite", testDeleteChart(helmService, kubeConfig, testNamespace))
}

func testIntegrationInstall(t *testing.T) {
	home := helmtesting.HelmHome(t)
	db := helmtesting.SetupDatabase(t)
	secretStore := helmtesting.SetupSecretStore()
	_, clusterService := helmtesting.ClusterKubeConfig(t, clusterId)

	config := helm.Config{
		Home: home,
		Repositories: map[string]string{
			"stable":               "https://charts.helm.sh/stable",
			"banzaicloud-stable":   "https://kubernetes-charts.banzaicloud.com",
			"prometheus-community": "https://prometheus-community.github.io/helm-charts",
		},
	}

	const testNamespace = "helm-install"

	logger := common.NoopLogger{}
	releaser, _ := cmd.CreateUnifiedHelmReleaser(
		config,
		cmd.ClusterConfig{}, // Note: dummy cluster config value.
		db,
		secretStore,
		clusterService,
		helmadapter.NewOrgService(logger),
		logger,
	)

	err := releaser.InstallDeployment(
		context.Background(),
		clusterId,
		testNamespace,
		"banzaicloud-stable/banzaicloud-docs",
		"helm-service-test",
		[]byte{},
		"0.1.2",
		true,
	)
	require.NoError(t, err)

	err = releaser.DeleteDeployment(
		context.Background(),
		clusterId,
		"helm-service-test",
		testNamespace,
	)
	require.NoError(t, err)
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
			noOrgID,
			&internaltesting.ClusterData{K8sConfig: kubeConfig, ID: clusterId},
			helm.Release{
				ReleaseName: "chartmuseum",
				ChartName:   "stable/chartmuseum",
				Namespace:   testNamespace,
				Values:      nil,
				Version:     "2.14.2",
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
			noOrgID,
			&internaltesting.ClusterData{K8sConfig: kubeConfig, ID: clusterId},
			helm.Release{
				ReleaseName: "chartmuseum",
				ChartName:   "stable/chartmuseum",
				Namespace:   testNamespace,
				Values:      serializedValues,
				Version:     "2.14.2",
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
			noOrgID,
			&internaltesting.ClusterData{K8sConfig: kubeConfig, ID: clusterId},
			helm.Release{
				ReleaseName: "chartmuseum",
				ChartName:   "stable/chartmuseum",
				Namespace:   testNamespace,
				Values:      serializedValues,
				Version:     "2.14.2",
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
			noOrgID,
			&internaltesting.ClusterData{K8sConfig: kubeConfig, ID: clusterId},
			helm.Release{
				ReleaseName: "chartmuseum",
				ChartName:   "stable/chartmuseum",
				Namespace:   testNamespace,
				Values:      nil,
				Version:     "2.14.2",
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

	ds, err := clientSet.AppsV1().Deployments(testNamespace).Get(context.Background(), "chartmuseum-chartmuseum", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("%+v", err)
	}

	if ds.Status.ReadyReplicas != ds.Status.Replicas || ds.Status.ReadyReplicas < 1 {
		t.Fatalf("chartmuseum is not running")
	}

	svc, err := clientSet.CoreV1().Services(testNamespace).Get(context.Background(), "chartmuseum-chartmuseum", metav1.GetOptions{})
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

	dsList, err := clientSet.AppsV1().Deployments(testNamespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		t.Fatalf("%+v", err)
	}

	if len(dsList.Items) > 0 {
		t.Fatalf("no deployments expected, chartmuseum should be removed")
	}
}
