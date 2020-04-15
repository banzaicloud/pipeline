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

package ingress

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"emperror.dev/errors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/mitchellh/mapstructure"

	"github.com/banzaicloud/pipeline/internal/integratedservices/services"
	"github.com/banzaicloud/pipeline/internal/providers/amazon"
	"github.com/banzaicloud/pipeline/pkg/any"
	pkgcluster "github.com/banzaicloud/pipeline/pkg/cluster"
	"github.com/banzaicloud/pipeline/pkg/jsonstructure"
)

type traefikManager struct {
	clusters         OperatorClusterStore
	config           Config
	helmService      services.HelmService
	orgDomainService OrgDomainService
}

func (m traefikManager) Deploy(ctx context.Context, clusterID uint, spec Spec) error {
	chartValues, err := m.compileChartValues(ctx, clusterID, spec)
	if err != nil {
		return errors.WrapIf(err, "failed to compile traefik chart values")
	}

	chartValuesBytes, err := json.Marshal(chartValues)
	if err != nil {
		return errors.WrapIf(err, "failed to marshal chart values to JSON")
	}

	if err := m.helmService.ApplyDeployment(
		ctx,
		clusterID,
		m.config.Namespace,
		m.config.Charts.Traefik.Chart,
		m.config.ReleaseName,
		chartValuesBytes,
		m.config.Charts.Traefik.Version,
	); err != nil {
		return errors.WrapIf(err, "failed to apply deployment")
	}

	return nil
}

func (m traefikManager) Remove(ctx context.Context, clusterID uint) error {
	return errors.WrapIf(m.helmService.DeleteDeployment(ctx, clusterID, m.config.ReleaseName, m.config.Namespace), "failed to delete deployment")
}

func (m traefikManager) compileChartValues(ctx context.Context, clusterID uint, spec Spec) (interface{}, error) {
	defaultValues, err := jsonstructure.CopyObject(m.config.Charts.Traefik.Values)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to copy default chart values from config")
	}

	type traefikKubernetesValues struct {
		IngressClass string `json:"ingressClass,omitempty" mapstructure:"ingressClass"`
	}

	type traefikServiceValues struct {
		Annotations map[string]string `json:"annotations,omitempty" mapstructure:"annotations"`
	}

	type traefikSSLValues struct {
		Enabled        bool     `json:"enabled" mapstructure:"enabled"`
		GenerateTLS    bool     `json:"generateTLS" mapstructure:"generateTLS"`
		DefaultCN      string   `json:"defaultCN,omitempty" mapstructure:"defaultCN"`
		DefaultIPList  []string `json:"defaultIPList,omitempty" mapstructure:"defaultIPList"`
		DefaultSANList []string `json:"defaultSANList,omitempty" mapstructure:"defaultSANList"`
	}

	type traefikValues struct {
		Kubernetes  traefikKubernetesValues `json:"kubernetes,omitempty" mapstructure:"kubernetes"`
		Service     traefikServiceValues    `json:"service,omitempty" mapstructure:"service"`
		ServiceType string                  `json:"serviceType,omitempty" mapstructure:"serviceType"`
		SSL         traefikSSLValues        `json:"ssl,omitempty" mapstructure:"ssl"`
	}

	var typedValues traefikValues
	if err := mapstructure.Decode(defaultValues, &typedValues); err != nil {
		return nil, errors.WrapIf(err, "failed to decode default chart values")
	}

	typedValues.Kubernetes.IngressClass = spec.IngressClass
	typedValues.ServiceType = spec.Service.Type
	typedValues.Service.Annotations = mergeServiceAnnotations(typedValues.Service.Annotations, spec.Service.Annotations)

	cluster, err := m.clusters.Get(ctx, clusterID)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to get cluster")
	}

	traefikConfig, err := spec.Controller.TraefikConfig()
	if err != nil {
		return nil, errors.WrapIf(err, "failed to get traefik config")
	}

	if defaultCN := traefikConfig.SSL.DefaultCN; defaultCN != "" {
		typedValues.SSL.DefaultCN = defaultCN
	}

	if defaultSANList := traefikConfig.SSL.DefaultSANList; len(defaultSANList) != 0 {
		typedValues.SSL.DefaultSANList = defaultSANList
	}

	if defaultIPList := traefikConfig.SSL.DefaultIPList; len(defaultIPList) != 0 {
		typedValues.SSL.DefaultIPList = defaultIPList
	}

	if typedValues.SSL.DefaultCN == "" && len(typedValues.SSL.DefaultSANList) == 0 {
		orgDomain, err := m.orgDomainService.GetOrgDomain(ctx, cluster.OrganizationID)
		if err != nil {
			return nil, errors.WrapIf(err, "failed to get org domain")
		}

		typedValues.SSL.DefaultCN = orgDomain.Name
		if orgDomain.Name != "" {
			typedValues.SSL.DefaultSANList = append(typedValues.SSL.DefaultSANList, orgDomain.Name)
		}
		if orgDomain.WildcardName != "" {
			typedValues.SSL.DefaultSANList = append(typedValues.SSL.DefaultSANList, orgDomain.WildcardName)
		}
	}

	if cluster.Cloud == pkgcluster.Amazon {
		const (
			tagsKey = "service.beta.kubernetes.io/aws-load-balancer-additional-resource-tags"
			sep     = ","
		)

		var tags []string

		if tagsVal := typedValues.Service.Annotations[tagsKey]; tagsVal != "" {
			tags = strings.Split(tagsVal, sep)
		}

		for _, tag := range amazon.PipelineTags() {
			tags = append(tags, fmt.Sprintf("%s=%s", aws.StringValue(tag.Key), aws.StringValue(tag.Value)))
		}

		if typedValues.Service.Annotations == nil {
			typedValues.Service.Annotations = make(map[string]string)
		}
		typedValues.Service.Annotations[tagsKey] = strings.Join(tags, sep)
	}

	untypedValues, err := jsonstructure.Encode(typedValues, jsonstructure.WithZeroStructsAsEmpty)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to encode chart values as JSON structure")
	}

	finalValues, err := any.Merge(defaultValues, untypedValues, jsonstructure.DefaultMergeOptions())
	if err != nil {
		return nil, errors.WrapIf(err, "failed to merge chart values")
	}

	return finalValues, nil
}

func mergeServiceAnnotations(dst map[string]string, src map[string]string) map[string]string {
	if len(src) == 0 {
		return dst
	}

	if dst == nil {
		dst = make(map[string]string)
	}

	for k, v := range src {
		dst[k] = v
	}

	return dst
}
