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

package federation_test

import (
	"testing"
	"time"

	"github.com/banzaicloud/pipeline/internal/cmd"
	"github.com/banzaicloud/pipeline/internal/federation"
	"github.com/banzaicloud/pipeline/internal/helm"
	helmtesting "github.com/banzaicloud/pipeline/internal/helm/testing"
	internaltesting "github.com/banzaicloud/pipeline/internal/testing"
	"github.com/banzaicloud/pipeline/pkg/k8sclient"
)

func testEnsureCRDSourceForExtDNS(v3 bool, testNamespace string) func(t *testing.T) {
	return func(t *testing.T) {
		chartName := "stable/external-dns"
		releaseName := "fed-ext-dns"
		chartVersion := "2.15.2"

		clusterId := uint(1)

		kubeConfig, clusterService := helmtesting.ClusterKubeConfig(t, clusterId)

		client, err := k8sclient.NewClientFromKubeConfig(kubeConfig)
		if err != nil {
			t.Fatalf("%+v", err)
		}

		err = internaltesting.EnsureNamespaceRemoved(client, testNamespace, time.Second*20)
		if err != nil {
			t.Fatalf("%+v", err)
		}

		clusterConfig := internaltesting.ClusterData{
			K8sConfig: kubeConfig,
			ID:        clusterId,
		}

		db := helmtesting.SetupDatabase(t)
		secretStore := helmtesting.SetupSecretStore()
		helmLogger, logrusLogger := helmtesting.Loggers()

		home := helmtesting.HelmHome(t)

		config := helm.Config{
			Home: home,
			V3:   v3,
			Repositories: map[string]string{
				"stable": "https://kubernetes-charts.storage.googleapis.com",
			},
		}

		orgService := helmtesting.FakeOrg{
			OrgId:   0,
			OrgName: "",
		}

		unifiedReleaser, _ := cmd.CreateUnifiedHelmReleaser(config, db, secretStore, clusterService, orgService, helmLogger)

		if err := unifiedReleaser.Delete(&clusterConfig, releaseName, testNamespace); err != nil {
			t.Fatalf("%+v", err)
		}
		err = unifiedReleaser.InstallOrUpgrade(&clusterConfig, helm.Release{
			ReleaseName: releaseName,
			ChartName:   chartName,
			Namespace:   testNamespace,
			Values:      nil,
			Version:     chartVersion,
		}, helm.Options{
			Namespace: testNamespace,
			Wait:      true,
		})
		if err != nil {
			t.Fatalf("%+v", err)
		}

		var desiredState federation.DesiredState

		desiredState = federation.DesiredStateAbsent
		upgraded, err := federation.EnsureCRDSourceForExtDNS(&clusterConfig, unifiedReleaser, testNamespace, chartName, releaseName, desiredState, logrusLogger)
		if err != nil {
			t.Fatalf("%+v", err)
		}
		if upgraded {
			t.Fatal("expected the chart not to be upgraded")
		}

		desiredState = federation.DesiredStatePresent
		upgraded, err = federation.EnsureCRDSourceForExtDNS(&clusterConfig, unifiedReleaser, testNamespace, chartName, releaseName, desiredState, logrusLogger)
		if err != nil {
			t.Fatalf("%+v", err)
		}
		if !upgraded {
			t.Fatal("expected the chart has been upgraded")
		}

		desiredState = federation.DesiredStatePresent
		upgraded, err = federation.EnsureCRDSourceForExtDNS(&clusterConfig, unifiedReleaser, testNamespace, chartName, releaseName, desiredState, logrusLogger)
		if err != nil {
			t.Fatalf("%+v", err)
		}
		if upgraded {
			t.Fatal("expected the chart not to be upgraded the second time")
		}

		desiredState = federation.DesiredStateAbsent
		upgraded, err = federation.EnsureCRDSourceForExtDNS(&clusterConfig, unifiedReleaser, testNamespace, chartName, releaseName, desiredState, logrusLogger)
		if err != nil {
			t.Fatalf("%+v", err)
		}
		if !upgraded {
			t.Fatal("expected the chart not to be upgraded")
		}

		if err := unifiedReleaser.Delete(&clusterConfig, releaseName, testNamespace); err != nil {
			t.Fatalf("%+v", err)
		}
	}
}
