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

package istiofeature

import (
	"flag"
	"io/ioutil"
	"os"
	"regexp"
	"testing"

	"github.com/banzaicloud/pipeline/internal/global"
	"github.com/banzaicloud/pipeline/pkg/k8sclient"
	"github.com/ghodss/yaml"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

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

	testNamespace := "istiofeature-helm"

	t.Run("testDeleteNodeExporter", testDeleteNodeExporter(kubeConfig, testNamespace))
	t.Run("testCreateNodeExporter", testCreateNodeExporter(kubeConfig, testNamespace))
	t.Run("testUpgradeNodeExporter", testUpgradeNodeExporter(kubeConfig, testNamespace))
	t.Run("testHandleFailedDeployment", testUpgradeFailedNodeExporter(kubeConfig, testNamespace))
	t.Run("testDeleteNodeExporterAfterSuite", testDeleteNodeExporter(kubeConfig, testNamespace))

}

func testDeleteNodeExporter(kubeConfig []byte, testNamespace string) func(*testing.T) {
	return func(t *testing.T) {
		err := deleteDeployment(
			&clusterProviderData{k8sConfig: kubeConfig},
			"node-exporter",
		)
		if err != nil {
			t.Fatalf("%+v", err)
		}

		assertNodeExporterRemoved(t, kubeConfig, testNamespace)
	}
}

func testCreateNodeExporter(kubeConfig []byte, testNamespace string) func(*testing.T) {
	return func(t *testing.T) {
		err := installOrUpgradeDeployment(
			&clusterProviderData{k8sConfig: kubeConfig},
			testNamespace,
			"stable/prometheus-node-exporter",
			"node-exporter",
			nil,
			"1.8.1",
			true, true)
		if err != nil {
			t.Fatalf("%+v", err)
		}

		assertNodeExporter(t, kubeConfig, testNamespace, 9100)
	}
}

func testUpgradeNodeExporter(kubeConfig []byte, testNamespace string) func(*testing.T) {
	return func(t *testing.T) {
		var expectPort int32 = 19191

		type Values struct {
			Service struct {
				Type        string            `json:"type,omitempty"`
				Port        int               `json:"port,omitempty"`
				TargetPort  int               `json:"targetPort,omitempty"`
				NodePort    int               `json:"nodePort,omitempty"`
				Annotations map[string]string `json:"annotations,omitempty"`
			} `json:"service,omitempty"`
		}

		values := Values{}
		values.Service.Port = int(expectPort)

		serializedValues, err := yaml.Marshal(values)
		if err != nil {
			t.Fatalf("%+v", serializedValues)
		}

		err = installOrUpgradeDeployment(
			&clusterProviderData{k8sConfig: kubeConfig},
			testNamespace,
			"stable/prometheus-node-exporter",
			"node-exporter",
			serializedValues,
			"1.8.1",
			true, true)
		if err != nil {
			t.Fatalf("%+v", err)
		}

		assertNodeExporter(t, kubeConfig, testNamespace, expectPort)
	}
}

func testUpgradeFailedNodeExporter(kubeConfig []byte, testNamespace string) func(*testing.T) {
	return func(t *testing.T) {
		// invalid port will fail the release
		var expectPort int32 = 1111111

		type Values struct {
			Service struct {
				Type        string            `json:"type,omitempty"`
				Port        int               `json:"port,omitempty"`
				TargetPort  int               `json:"targetPort,omitempty"`
				NodePort    int               `json:"nodePort,omitempty"`
				Annotations map[string]string `json:"annotations,omitempty"`
			} `json:"service,omitempty"`
		}

		values := Values{}
		values.Service.Port = int(expectPort)

		serializedValues, err := yaml.Marshal(values)
		if err != nil {
			t.Fatalf("%+v", serializedValues)
		}

		err = installOrUpgradeDeployment(
			&clusterProviderData{k8sConfig: kubeConfig},
			testNamespace,
			"stable/prometheus-node-exporter",
			"node-exporter",
			serializedValues,
			"1.8.1",
			true, true)
		if err == nil {
			t.Fatalf("this upgrade should fail because of the invalid port")
		}

		// restore with original values
		err = installOrUpgradeDeployment(
			&clusterProviderData{k8sConfig: kubeConfig},
			testNamespace,
			"stable/prometheus-node-exporter",
			"node-exporter",
			nil,
			"1.8.1",
			true, true)
		if err != nil {
			t.Fatalf("%+v", err)
		}

		assertNodeExporter(t, kubeConfig, testNamespace, 9100)
	}
}

func assertNodeExporter(t *testing.T, kubeConfig []byte, testNamespace string, expectedPort int32) {
	restConfig, err := k8sclient.NewClientConfig(kubeConfig)
	if err != nil {
		t.Fatalf("%+v", err)
	}

	clientSet, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		t.Fatalf("%+v", err)
	}

	ds, err := clientSet.AppsV1().DaemonSets(testNamespace).Get("node-exporter-prometheus-node-exporter", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("%+v", err)
	}

	if ds.Status.NumberReady != ds.Status.DesiredNumberScheduled {
		t.Fatalf("Node exporters are not running")
	}

	svc, err := clientSet.CoreV1().Services(testNamespace).Get("node-exporter-prometheus-node-exporter", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("%+v", err)
	}

	if len(svc.Spec.Ports) < 1 {
		t.Fatalf("Missing node exporter service ports")
	}

	if svc.Spec.Ports[0].Port != expectedPort {
		t.Fatalf("Node exporter service port mismatch, expected %d vs %d", expectedPort, svc.Spec.Ports[0].Port)
	}
}

func assertNodeExporterRemoved(t *testing.T, kubeConfig []byte, testNamespace string) {
	restConfig, err := k8sclient.NewClientConfig(kubeConfig)
	if err != nil {
		t.Fatalf("%+v", err)
	}

	clientSet, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		t.Fatalf("%+v", err)
	}

	dsList, err := clientSet.AppsV1().DaemonSets(testNamespace).List(metav1.ListOptions{})
	if err != nil {
		t.Fatalf("%+v", err)
	}

	if len(dsList.Items) > 0 {
		t.Fatalf("no daemonsets expected, node exporter should be removed")
	}
}
