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

package cluster

import (
	"github.com/banzaicloud/pipeline/pkg/cluster"
	pkgHelm "github.com/banzaicloud/pipeline/pkg/helm"
	"github.com/ghodss/yaml"
	"github.com/goph/emperror"
)

// InstallServiceMeshParams describes InstallServiceMesh posthook params
type InstallServiceMeshParams struct {
	EnableMtls bool `json:"mtls,omitempty"`
}

// InstallServiceMesh is a posthook for installing Istio on a cluster
func InstallServiceMesh(cluster CommonCluster, param cluster.PostHookParam) error {
	var params InstallServiceMeshParams
	err := castToPostHookParam(&param, &params)
	if err != nil {
		return err
	}

	log.Infof("istio params: %v", params)

	values := map[string]interface{}{}

	marshalledValues, err := yaml.Marshal(values)
	if err != nil {
		return emperror.Wrap(err, "failed to marshal yaml values")
	}

	err = installDeployment(
		cluster,
		"istio-system",
		pkgHelm.BanzaiRepository+"/istio",
		"istio",
		marshalledValues,
		"",
		false,
	)
	if err != nil {
		return emperror.Wrap(err, "installing Istio failed")
	}

	cluster.SetServiceMesh(true)

	return nil
}
