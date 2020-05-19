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

package logging

import (
	"context"
	"io/ioutil"
	"net/http"
	"strings"

	"emperror.dev/errors"
	esCommon "github.com/elastic/cloud-on-k8s/pkg/apis/common/v1"
	esType "github.com/elastic/cloud-on-k8s/pkg/apis/elasticsearch/v1"
	kibanaType "github.com/elastic/cloud-on-k8s/pkg/apis/kibana/v1"
	"github.com/mitchellh/copystructure"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"

	"github.com/banzaicloud/pipeline/internal/integratedservices/services"
)

const (
	elasticObjectNames     = "banzai-elastic"
	elasticsearchNamespace = "elastic-system"
)

type elasticSearchInstaller struct {
	clusterID         uint
	config            ElasticConfig
	demoConfig        ChartConfig
	kubernetesService KubernetesService
	helmService       services.HelmService
}

func makeElasticSearchInstaller(
	clusterID uint,
	config ElasticConfig,
	demoConfig ChartConfig,
	kubernetesService KubernetesService,
	helmService services.HelmService,
) elasticSearchInstaller {
	return elasticSearchInstaller{
		clusterID:         clusterID,
		config:            config,
		demoConfig:        demoConfig,
		kubernetesService: kubernetesService,
		helmService:       helmService,
	}
}

// installElasticsearchOperator installs custom resource definitions and the operator with its RBAC rules
// default all-in-one YAML: https://download.elastic.co/downloads/eck/1.1.1/all-in-one.yaml
func (esi elasticSearchInstaller) installElasticsearchOperator(ctx context.Context) error {
	// Install custom resource definitions and the operator with its RBAC rules
	objectYamls, err := esi.getECKResourceYaml()
	if err != nil {
		return errors.WrapIf(err, "failed to get ECK resources from URL")
	}

	for _, yaml := range objectYamls {
		// decode YAML
		obj, _, err := scheme.Codecs.UniversalDeserializer().Decode([]byte(yaml), nil, nil)
		if err != nil {
			return errors.WrapIf(err, "failed to decode all-in-one yaml")
		}

		// create object
		if err := esi.kubernetesService.EnsureObject(ctx, esi.clusterID, obj); err != nil {
			return errors.WrapIf(err, "failed to ensure object")
		}
	}

	return nil
}

// removeElasticsearchOperator removes all CRDs and and the operator which installed with
// all-in-one YAML
func (esi elasticSearchInstaller) removeElasticsearchOperator(ctx context.Context) error {
	// remove CRDs
	objectYamls, err := esi.getECKResourceYaml()
	if err != nil {
		return errors.WrapIf(err, "failed to get ECK resources from URL")
	}

	for _, yaml := range objectYamls {
		// decode YAML
		obj, _, err := scheme.Codecs.UniversalDeserializer().Decode([]byte(yaml), nil, nil)
		if err != nil {
			return errors.WrapIf(err, "failed to decode all-in-one yaml")
		}

		if err := esi.kubernetesService.DeleteObject(ctx, esi.clusterID, obj); err != nil {
			return errors.Wrap(err, "failed to delete object")
		}
	}

	return nil
}

// getECKResourceYaml returns with the custom resource definitions and the operator with its RBAC rules,
// separated with ---
func (esi elasticSearchInstaller) getECKResourceYaml() ([]string, error) {
	response, err := http.Get(esi.config.AllInOneYAML)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to get CRDs")
	}

	defer response.Body.Close()

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to read response body")
	}

	return strings.Split(string(body), "---"), nil
}

// removeElasticsearchCluster removes Elasticsearch cluster specification
func (esi elasticSearchInstaller) removeElasticsearchCluster(ctx context.Context) error {
	return esi.kubernetesService.DeleteObject(ctx, esi.clusterID, &esType.Elasticsearch{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      elasticObjectNames,
			Namespace: elasticsearchNamespace,
		},
	})
}

// installElasticsearchCluster applies a simple Elasticsearch cluster specification, with one Elasticsearch node
func (esi elasticSearchInstaller) installElasticsearchCluster(ctx context.Context) error {
	return esi.kubernetesService.EnsureObject(ctx, esi.clusterID, &esType.Elasticsearch{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      elasticObjectNames,
			Namespace: elasticsearchNamespace,
		},
		Spec: esType.ElasticsearchSpec{
			Version: esi.config.Version,
			NodeSets: []esType.NodeSet{
				{
					Name:  "default",
					Count: 1,
					Config: &esCommon.Config{
						Data: map[string]interface{}{
							"node.master":           true,
							"node.data":             true,
							"node.ingest":           true,
							"node.store.allow_mmap": false,
						},
					},
				},
			},
		},
	})
}

// removeKibana deletes Kibana instance
func (esi elasticSearchInstaller) removeKibana(
	ctx context.Context,
) error {
	return esi.kubernetesService.EnsureObject(ctx, esi.clusterID, &kibanaType.Kibana{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      elasticObjectNames,
			Namespace: elasticsearchNamespace,
		},
	})
}

// installKibana specifies a Kibana instance and associate it with Elasticsearch cluster
func (esi elasticSearchInstaller) installKibana(ctx context.Context) error {
	return esi.kubernetesService.EnsureObject(ctx, esi.clusterID, &kibanaType.Kibana{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      elasticObjectNames,
			Namespace: elasticsearchNamespace,
		},
		Spec: kibanaType.KibanaSpec{
			Version: esi.config.Kibana.Version,
			Count:   1,
			ElasticsearchRef: esCommon.ObjectSelector{
				Name: elasticObjectNames,
			},
		},
	})
}

func (esi elasticSearchInstaller) installLoggingDemo(ctx context.Context) error {
	var chartValues = &loggingDemoValues{
		Elasticsearch: elasticValues{
			Enabled: true,
		},
	}

	demoConfigValues, err := copystructure.Copy(esi.demoConfig.Values)
	if err != nil {
		return errors.WrapIf(err, "failed to copy logging-demo values")
	}
	valuesBytes, err := mergeValuesWithConfig(chartValues, demoConfigValues)
	if err != nil {
		return errors.WrapIf(err, "failed to merge logging-demo values with config")
	}

	return esi.helmService.ApplyDeployment(
		ctx,
		esi.clusterID,
		elasticsearchNamespace,
		esi.demoConfig.Chart,
		loggingDemoReleaseName,
		valuesBytes,
		esi.demoConfig.Version,
	)
}

func (esi elasticSearchInstaller) removeLoggingDemo(ctx context.Context) error {
	return esi.helmService.DeleteDeployment(ctx, esi.clusterID, loggingDemoReleaseName, elasticsearchNamespace)
}
