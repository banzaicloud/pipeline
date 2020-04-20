// Copyright © 2018 Banzai Cloud
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

package cluster

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"emperror.dev/errors"
	"github.com/aws/aws-sdk-go/aws"

	"github.com/banzaicloud/pipeline/internal/global"
	"github.com/banzaicloud/pipeline/internal/providers/amazon"
	"github.com/banzaicloud/pipeline/pkg/any"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	"github.com/banzaicloud/pipeline/pkg/jsonstructure"
	"github.com/banzaicloud/pipeline/src/auth"
	"github.com/banzaicloud/pipeline/src/dns"
)

type ingressControllerValues struct {
	Traefik traefikValues `json:"traefik"`
}

type traefikValues struct {
	SSL     sslTraefikValues     `json:"ssl"`
	Service serviceTraefikValues `json:"service,omitempty"`
}

type sslTraefikValues struct {
	Enabled        bool     `json:"enabled"`
	GenerateTLS    bool     `json:"generateTLS"`
	DefaultCN      string   `json:"defaultCN,omitempty"`
	DefaultSANList []string `json:"defaultSANList,omitempty"`
	DefaultCert    string   `json:"defaultCert,omitempty"`
	DefaultKey     string   `json:"defaultKey,omitempty"`
}

type serviceTraefikValues struct {
	Annotations map[string]string `json:"annotations,omitempty"`
}

type IngressControllerPostHook struct {
	helmServiceInjector
	Priority
	ErrorHandler
}

func (i *IngressControllerPostHook) Do(cluster CommonCluster) error {
	if i.helmService == nil {
		return errors.New("missing helm service dependency")
	}
	var config = global.Config.Cluster.PostHook
	if !config.Ingress.Enabled {
		return nil
	}

	orgID := cluster.GetOrganizationId()
	organization, err := auth.GetOrganizationById(orgID)
	if err != nil {
		return errors.WrapIfWithDetails(err, "failed to get organization", "organizationId", orgID)
	}

	var orgDomainName string
	var wildcardOrgDomainName string
	var baseDomain = strings.ToLower(global.Config.Cluster.DNS.BaseDomain)
	if baseDomain != "" {
		orgDomainName = strings.ToLower(fmt.Sprintf("%s.%s", organization.NormalizedName, baseDomain))
		err = dns.ValidateSubdomain(orgDomainName)
		if err != nil {
			return errors.WrapIf(err, "invalid domain for TLS cert")
		}

		wildcardOrgDomainName = fmt.Sprintf("*.%s", orgDomainName)
		err = dns.ValidateWildcardSubdomain(wildcardOrgDomainName)
		if err != nil {
			return errors.WrapIf(err, "invalid wildcard domain for TLS cert")
		}
	}

	var defaultCN = orgDomainName
	var defaultSANList []string
	if orgDomainName != "" {
		defaultSANList = append(defaultSANList, orgDomainName)
	}

	if wildcardOrgDomainName != "" {
		defaultSANList = append(defaultSANList, wildcardOrgDomainName)
	}

	if values, ok := config.Ingress.Values["traefik"].(map[string]interface{}); ok {
		if sslV, ok := values["ssl"].(map[string]interface{}); ok {
			if sanList, ok := sslV["defaultSANList"].([]interface{}); ok {
				for _, san := range sanList {
					if s, ok := san.(string); ok {
						defaultSANList = append(defaultSANList, s)
					}
				}
			}
		}
	}

	var ingressValues = ingressControllerValues{
		Traefik: traefikValues{
			SSL: sslTraefikValues{
				Enabled:        true,
				GenerateTLS:    true,
				DefaultCN:      defaultCN,
				DefaultSANList: defaultSANList,
			},
		},
	}

	// TODO: once we move this to an integrated service we must find a way to append tags to user configured annotations
	if cluster.GetCloud() == pkgCluster.Amazon {
		var tags []string

		for _, tag := range amazon.PipelineTags() {
			tags = append(tags, fmt.Sprintf("%s=%s", aws.StringValue(tag.Key), aws.StringValue(tag.Value)))
		}

		ingressValues.Traefik.Service.Annotations = map[string]string{
			"service.beta.kubernetes.io/aws-load-balancer-additional-resource-tags": strings.Join(tags, ","),
		}
	}

	valuesBytes, err := mergeValues(ingressValues, config.Ingress.Values)
	if err != nil {
		return errors.WrapIf(err, "failed to merge treafik values with config")
	}

	namespace := global.Config.Cluster.Namespace

	return i.helmService.ApplyDeployment(context.Background(), cluster.GetID(), namespace, config.Ingress.Chart, "ingress", valuesBytes, config.Ingress.Version)
}

func mergeValues(chartValues interface{}, configValues interface{}) ([]byte, error) {
	out, err := jsonstructure.Encode(chartValues)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to encode chart values")
	}

	result, err := any.Merge(configValues, out, jsonstructure.DefaultMergeOptions())
	if err != nil {
		return nil, errors.WrapIf(err, "failed to merge values")
	}

	return json.Marshal(result)
}
