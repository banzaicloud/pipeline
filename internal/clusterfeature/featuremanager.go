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

package clusterfeature

import (
	"context"
	"encoding/json"

	"emperror.dev/emperror"
	"github.com/goph/logur"
)

const (
	ExternalDnsChartVersion = "1.6.2"

	ExternalDnsImageVersion = "v0.5.11"

	ExternalDnsChartName = "stable/external-dns"

	ExternalDnsNamespace = "default"

	ExternalDnsRelease = "external-dns"
)

// externalDnsFeatureManager synchronous feature manager
type externalDnsFeatureManager struct {
	logger         logur.Logger
	clusterService ClusterService
	helmService    HelmService
}

// NewExternalDnsFeatureManager builds a new feature manager component
func NewExternalDnsFeatureManager(logger logur.Logger, clusterService ClusterService) FeatureManager {
	hs := &featureHelmService{ // wired private component!
		logger: logur.WithFields(logger, map[string]interface{}{"comp": "helm-installer"}),
	}
	return &externalDnsFeatureManager{
		logger:         logur.WithFields(logger, map[string]interface{}{"component": "feature-manager"}),
		clusterService: clusterService,
		helmService:    hs,
	}
}

func (sfm *externalDnsFeatureManager) Activate(ctx context.Context, clusterId uint, feature Feature) error {

	cluster, err := sfm.clusterService.GetCluster(ctx, clusterId)
	if err != nil {
		// internal error at this point
		return emperror.WrapWith(err, "failed to activate feature")
	}

	kubeConfig, err := cluster.GetKubeConfig()
	if err != nil {
		return emperror.WrapWith(err, "failed to upgrade feature", "feature", feature.Name)
	}

	// todo merge the spec into a template!!!
	externalDnsValues := map[string]interface{}{
		"rbac": map[string]bool{
			"create": false,
		},
		"image": map[string]string{
			"tag": "v0.5.11",
		},
		"aws": map[string]string{
			"secretKey": "",
			"accessKey": "",
			"region":    "",
		},
		"domainFilters": []string{"test-domain"},
		"policy":        "sync",
		"txtOwnerId":    "testing",
		"affinity":      "",
		"tolerations":   "",
	}

	externalDnsValuesJson, _ := yaml.Marshal(externalDnsValues)

	return sfm.helmService.InstallDeployment(ctx, cluster.GetOrganizationName(), kubeConfig, ExternalDnsNamespace, ExternalDnsChartName, ExternalDnsRelease, externalDnsValuesJson, ExternalDnsChartVersion, false)

}

func (sfm *externalDnsFeatureManager) Deactivate(ctx context.Context, clusterId uint, feature Feature) error {
	cluster, err := sfm.clusterService.GetCluster(ctx, clusterId)
	if err != nil {
		// internal error at this point
		return emperror.WrapWith(err, "failed to deactivate feature")
	}

	kubeConfig, err := cluster.GetKubeConfig()
	if err != nil {
		return emperror.WrapWith(err, "failed to deactivate feature", "feature", feature.Name)
	}

	if err := sfm.helmService.DeleteDeployment(ctx, kubeConfig, ExternalDnsRelease); err != nil {
		return emperror.WrapWith(err, "failed to uninstall feature")
	}

	return nil
}

func (sfm *externalDnsFeatureManager) Update(ctx context.Context, clusterId uint, feature Feature) error {

	cluster, err := sfm.clusterService.GetCluster(ctx, clusterId)
	if err != nil {
		// internal error at this point
		return emperror.WrapWith(err, "failed to deactivate feature")
	}

	var valuesJson []byte
	if valuesJson, err = json.Marshal(feature.Spec); err != nil {
		return emperror.Wrap(err, "failed to update feature")
	}

	kubeConfig, err := cluster.GetKubeConfig()
	if err != nil {
		return emperror.WrapWith(err, "failed to upgrade feature", "feature", feature.Name)
	}

	return sfm.helmService.UpdateDeployment(ctx, cluster.GetOrganizationName(), kubeConfig, ExternalDnsNamespace, ExternalDnsChartName, ExternalDnsRelease, valuesJson, ExternalDnsChartVersion)

}
