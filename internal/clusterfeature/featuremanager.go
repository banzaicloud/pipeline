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
	"github.com/goph/logur/adapters/logrusadapter"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// syncFeatureManager synchronous feature manager
type syncFeatureManager struct {
	logger         logur.Logger
	clusterService ClusterService
	helmService    HelmService
}

// NewSyncFeatureManager builds a new feature manager component
func NewSyncFeatureManager(clusterService ClusterService) FeatureManager {
	l := logur.WithFields(logrusadapter.New(logrus.New()), map[string]interface{}{"component": "feature-manager"})
	return &syncFeatureManager{
		logger:         l,
		clusterService: clusterService,
		helmService: &featureHelmService{ // wired private component!
			logger: logur.WithFields(l, map[string]interface{}{"comp": "helm-installer"}),
		},
	}
}

func (sfm *syncFeatureManager) Activate(ctx context.Context, clusterId uint, feature Feature) error {

	cluster, err := sfm.clusterService.GetCluster(ctx, clusterId)
	if err != nil {
		// internal error at this point
		return emperror.WrapWith(err, "failed to activate feature")
	}

	ns, ok := feature.Spec[DNSExternalDnsNamespace]
	if !ok {
		return errors.New("namespace for feature not provided")
	}

	deploymentName, ok := feature.Spec[DNSExternalDnsChartName]
	if !ok {
		return errors.New("chart-name for feature not provided")
	}

	releaseName := "testing-externaldns"

	values, ok := feature.Spec[DNSExternalDnsValues]
	if !ok {
		return errors.New("values for feature not available")
	}

	chartVersion, ok := feature.Spec[DNSExternalDnsChartVersion]
	if !ok {
		return errors.New("values for feature not available")
	}

	kubeConfig, err := cluster.GetKubeConfig()
	if err != nil {
		return emperror.WrapWith(err, "failed to upgrade feature", "feature", feature.Name)
	}

	return sfm.helmService.InstallDeployment(ctx, cluster.GetOrganizationName(), kubeConfig, ns.(string), deploymentName.(string), releaseName, values.([]byte), chartVersion.(string), false)

}

func (sfm *syncFeatureManager) Deactivate(ctx context.Context, clusterId uint, feature Feature) error {
	cluster, err := sfm.clusterService.GetCluster(ctx, clusterId)
	if err != nil {
		// internal error at this point
		return emperror.WrapWith(err, "failed to deactivate feature")
	}

	kubeConfig, err := cluster.GetKubeConfig()
	if err != nil {
		return emperror.WrapWith(err, "failed to deactivate feature", "feature", feature.Name)
	}

	releaseName := "testing-externaldns"

	if err := sfm.helmService.DeleteDeployment(ctx, kubeConfig, releaseName); err != nil {
		return emperror.WrapWith(err, "failed to uninstall feature")
	}

	return nil

}

func (sfm *syncFeatureManager) Update(ctx context.Context, clusterId uint, feature Feature) error {

	cluster, err := sfm.clusterService.GetCluster(ctx, clusterId)
	if err != nil {
		// internal error at this point
		return emperror.WrapWith(err, "failed to deactivate feature")
	}

	ns, ok := feature.Spec[DNSExternalDnsNamespace]
	if !ok {
		return errors.New("namespace for feature not provided")
	}

	deploymentName, ok := feature.Spec[DNSExternalDnsChartName]
	if !ok {
		return errors.New("chart-name for feature not provided")
	}

	releaseName := "testing-externaldns"

	values, ok := feature.Spec[DNSExternalDnsValues]
	if !ok {
		return errors.New("values for feature not available")
	}

	var valuesJson []byte
	if valuesJson, err = json.Marshal(values); err != nil {
		return emperror.Wrap(err, "failed to update feature")
	}

	chartVersion, ok := feature.Spec[DNSExternalDnsChartVersion]
	if !ok {
		return errors.New("values for feature not available")
	}

	kubeConfig, err := cluster.GetKubeConfig()
	if err != nil {
		return emperror.WrapWith(err, "failed to upgrade feature", "feature", feature.Name)
	}

	return sfm.helmService.UpdateDeployment(ctx, cluster.GetOrganizationName(), kubeConfig, ns.(string), deploymentName.(string), releaseName, valuesJson, chartVersion.(string))

}
