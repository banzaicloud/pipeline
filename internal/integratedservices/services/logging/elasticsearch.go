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
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
)

const (
	elasticObjectNames     = "banzai-elastic"
	elasticsearchNamespace = "elastic-system"
)

type elasticSearchInstaller struct {
	clusterID         uint
	config            ElasticConfig
	kubernetesService KubernetesService
}

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

func (esi elasticSearchInstaller) removeElasticsearchCluster(ctx context.Context) error {
	return esi.kubernetesService.DeleteObject(ctx, esi.clusterID, &esType.Elasticsearch{
		ObjectMeta: v1.ObjectMeta{
			Name:      elasticObjectNames,
			Namespace: elasticsearchNamespace,
		},
	})
}

func (esi elasticSearchInstaller) installElasticsearchCluster(ctx context.Context) error {
	return esi.kubernetesService.EnsureObject(ctx, esi.clusterID, &esType.Elasticsearch{
		ObjectMeta: v1.ObjectMeta{
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

func (esi elasticSearchInstaller) removeKibana(
	ctx context.Context,
) error {
	return esi.kubernetesService.EnsureObject(ctx, esi.clusterID, &kibanaType.Kibana{
		ObjectMeta: v1.ObjectMeta{
			Name:      elasticObjectNames,
			Namespace: elasticsearchNamespace,
		},
	})
}

func (esi elasticSearchInstaller) installKibana(ctx context.Context) error {
	return esi.kubernetesService.EnsureObject(ctx, esi.clusterID, &kibanaType.Kibana{
		ObjectMeta: v1.ObjectMeta{
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
