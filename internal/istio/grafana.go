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

package istio

import (
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/banzaicloud/pipeline/config"
	"github.com/goph/emperror"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func AddGrafanaDashboards(log logrus.FieldLogger, client kubernetes.Interface) error {
	pipelineSystemNamespace := viper.GetString(config.PipelineSystemNamespace)

	for _, dashboard := range []string{"galley", "istio-mesh", "istio-performance", "istio-service", "istio-workload", "mixer", "pilot"} {
		dashboardJson, err := getDashboardJsonFromURL(fmt.Sprintf("https://raw.githubusercontent.com/banzaicloud/banzai-charts/master/istio/deps/grafana/dashboards/%s-dashboard.json", dashboard))
		if err != nil {
			return emperror.Wrapf(err, "couldn't add Istio grafana dashboard: %s", dashboard)
		}

		_, err = client.CoreV1().ConfigMaps(pipelineSystemNamespace).Create(&v1.ConfigMap{
			Data: map[string]string{
				fmt.Sprintf("%s.json", dashboard): dashboardJson,
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: fmt.Sprintf("%s-grafana-dashboard", dashboard),
				Labels: map[string]string{
					"pipeline_grafana_dashboard": "1",
				},
			},
		})
		if err != nil {
			return emperror.Wrapf(err, "couldn't add Istio grafana dashboard: %s", dashboard)
		}
		log.Debugf("created Istio grafana dashboard %s", dashboard)
	}
	return nil
}

func getDashboardJsonFromURL(url string) (string, error) {
	var client http.Client
	resp, err := client.Get(url)
	if err != nil {
		return "", emperror.Wrapf(err, "Failed to get dashboard.json from url %s", url)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", emperror.Wrapf(err, "Failed to get dashboard.json from url %s, status code: %v", url, resp.StatusCode)
	}
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", emperror.Wrapf(err, "Failed to get dashboard.json from url %s", url)
	}
	return string(bodyBytes), nil

}
