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
	"fmt"

	"github.com/banzaicloud/pipeline/auth"
	pipConfig "github.com/banzaicloud/pipeline/config"
	pkgHelm "github.com/banzaicloud/pipeline/pkg/helm"
	"github.com/ghodss/yaml"
	"github.com/go-errors/errors"
	"github.com/goph/emperror"
	"github.com/spf13/viper"
)

type treafikSslConfig struct {
	Enabled        bool     `json:"enabled"`
	GenerateTLS    bool     `json:"generateTLS"`
	DefaultCN      string   `json:"defaultCN"`
	DefaultSANList []string `json:"defaultSANList"`
}

//InstallIngressControllerPostHook post hooks can't return value, they can log error and/or update state?
func InstallIngressControllerPostHook(input interface{}) error {
	cluster, ok := input.(CommonCluster)
	if !ok {
		return errors.Errorf("Wrong parameter type: %T", cluster)
	}

	orgID := cluster.GetOrganizationId()
	organization, err := auth.GetOrganizationById(orgID)
	if err != nil {
		return emperror.WrapWith(err, "failed to get organization", "orgID", orgID)
	}

	domainWithOrgName := fmt.Sprintf("%s.%s", organization.Name, viper.GetString(pipConfig.DNSBaseDomain))

	ssl := treafikSslConfig{
		Enabled:     true,
		GenerateTLS: true,
		DefaultCN:   fmt.Sprintf("*.%s", domainWithOrgName),
		DefaultSANList: []string{
			domainWithOrgName,
			fmt.Sprintf("%s.%s", cluster.GetName(), domainWithOrgName),
		},
	}

	ingressValues := map[string]interface{}{
		"traefik": map[string]interface{}{
			"ssl":         ssl,
			"affinity":    getHeadNodeAffinity(cluster),
			"tolerations": getHeadNodeTolerations(),
		},
	}

	ingressValuesJson, err := yaml.Marshal(ingressValues)
	if err != nil {
		return emperror.Wrap(err, "converting ingress config to json failed")
	}

	namespace := viper.GetString(pipConfig.PipelineSystemNamespace)

	return installDeployment(cluster, namespace, pkgHelm.BanzaiRepository+"/pipeline-cluster-ingress", "ingress", ingressValuesJson, "", false)
}
