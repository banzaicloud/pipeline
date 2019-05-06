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
	"net/url"

	"github.com/goph/emperror"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/banzaicloud/pipeline/config"
	pkgCommon "github.com/banzaicloud/pipeline/pkg/common"
)

const (
	getJsonFailed      = "failed to get Istio Grafana dashboard.json"
	addDashboardFailed = "couldn't add Istio Grafana dashboard"

	createdByLabel = pkgCommon.PipelineSpecificLabelsCommonPart + "/created-by"
	appLabel       = pkgCommon.PipelineSpecificLabelsCommonPart + "/app"
)

func DeleteGrafanaDashboards(log logrus.FieldLogger, client kubernetes.Interface) error {
	pipelineSystemNamespace := viper.GetString(config.PipelineSystemNamespace)

	cms, err := client.CoreV1().ConfigMaps(pipelineSystemNamespace).List(metav1.ListOptions{
		LabelSelector: createdByLabel + "=pipeline," + appLabel + "=grafana",
	})
	if err != nil {
		return emperror.Wrap(err, "could not list configmaps")
	}

	caughtErrors := emperror.NewMultiErrorBuilder()
	for _, cm := range cms.Items {
		err := client.CoreV1().ConfigMaps(pipelineSystemNamespace).Delete(cm.Name, &metav1.DeleteOptions{})
		if err != nil {
			caughtErrors.Add(emperror.Wrap(err, "could not delete configmap"))
		}
	}

	return caughtErrors.ErrOrNil()
}

func AddGrafanaDashboards(log logrus.FieldLogger, client kubernetes.Interface) error {
	pipelineSystemNamespace := viper.GetString(config.PipelineSystemNamespace)

	for _, dashboard := range []string{"galley", "istio-mesh", "istio-performance", "istio-service", "istio-workload", "mixer", "pilot"} {
		dashboardJson, err := getDashboardJson(log, dashboard)
		if err != nil {
			return emperror.WrapWith(err, addDashboardFailed, "dashboard", dashboard)
		}
		_, err = client.CoreV1().ConfigMaps(pipelineSystemNamespace).Create(&v1.ConfigMap{
			Data: map[string]string{
				fmt.Sprintf("%s.json", dashboard): dashboardJson,
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: fmt.Sprintf("%s-grafana-dashboard", dashboard),
				Labels: map[string]string{
					"pipeline_grafana_dashboard": "1",
					createdByLabel:               "pipeline",
					appLabel:                     "grafana",
				},
			},
		})
		if err != nil {
			if errors.IsAlreadyExists(err) {
				log.Warnf("Istio Grafana dashboard %s already exists", dashboard)
				continue
			} else {
				return emperror.WrapWith(err, addDashboardFailed, "dashboard", dashboard)
			}
		}
		log.Debugf("created Istio Grafana dashboard %s", dashboard)
	}
	return nil
}

func getDashboardJson(log logrus.FieldLogger, name string) (string, error) {
	templatePath := viper.GetString(config.IstioGrafanaDashboardLocation) + "/" + name + "-dashboard.json"
	log.Infof("Getting Istio dashboard from %s", templatePath)
	u, err := url.Parse(templatePath)
	if err != nil {
		return "", emperror.WrapWith(err, getJsonFailed, "url", templatePath)
	}
	var content []byte
	switch u.Scheme {
	case "file", "":
		content, err = ioutil.ReadFile(u.String())
		if err != nil {
			return "", emperror.WrapWith(err, getJsonFailed, "url", u.String())
		}
	case "http", "https":
		var client http.Client
		resp, err := client.Get(u.String())
		if err != nil {
			return "", emperror.WrapWith(err, getJsonFailed, "url", u.String())
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return "", emperror.WrapWith(err, getJsonFailed, "url", u.String(), "statusCode", resp.StatusCode)
		}
		content, err = ioutil.ReadAll(resp.Body)
		if err != nil {
			return "", emperror.WrapWith(err, getJsonFailed, "url", u.String())
		}
	default:
		return "", fmt.Errorf("unsupported scheme: %s", u.Scheme)
	}
	return string(content), nil
}
