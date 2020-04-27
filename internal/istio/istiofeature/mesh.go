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

package istiofeature

import (
	"emperror.dev/emperror"
	"emperror.dev/errors"
	"github.com/banzaicloud/istio-operator/pkg/apis/istio/v1beta1"
	"github.com/sirupsen/logrus"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/banzaicloud/pipeline/internal/clustergroup/api"
	"github.com/banzaicloud/pipeline/pkg/k8sclient"
	"github.com/banzaicloud/pipeline/src/cluster"
)

func init() {
	_ = v1beta1.AddToScheme(scheme.Scheme)
	_ = apiextensionsv1beta1.AddToScheme(scheme.Scheme)
}

// NewMeshReconciler crates a new mesh feature reconciler
func NewMeshReconciler(
	config Config,
	clusterGetter api.ClusterGetter,
	logger logrus.FieldLogger,
	errorHandler emperror.Handler,
	helmService HelmService) *MeshReconciler {
	reconciler := &MeshReconciler{
		Configuration: config,

		helmService:   helmService,
		clusterGetter: clusterGetter,
		logger:        logger,
		errorHandler:  errorHandler,
	}

	reconciler.init() // nolint: errcheck

	reconciler.logger = reconciler.logger.WithFields(logrus.Fields{
		"clusterID":   reconciler.Master.GetID(),
		"clusterName": reconciler.Master.GetName(),
	})

	return reconciler
}

func (m *MeshReconciler) init() error {
	m.Master = m.getMasterCluster()
	m.Remotes = m.getRemoteClusters()

	return nil
}

func (m *MeshReconciler) getMasterCluster() cluster.CommonCluster {
	for _, c := range m.Configuration.clusterGroup.Clusters {
		if m.Configuration.MasterClusterID == c.GetID() {
			return c.(cluster.CommonCluster)
		}
	}

	return nil
}

func (m *MeshReconciler) getRemoteClusters() []cluster.CommonCluster {
	remotes := make([]cluster.CommonCluster, 0)

	for _, c := range m.Configuration.clusterGroup.Clusters {
		if m.Configuration.MasterClusterID != c.GetID() {
			remotes = append(remotes, c.(cluster.CommonCluster))
		}
	}

	return remotes
}

func (m *MeshReconciler) getK8sClient(c cluster.CommonCluster) (*kubernetes.Clientset, error) {
	kubeConfig, err := c.GetK8sConfig()
	if err != nil {
		return nil, errors.WrapIf(err, "could not get k8s config")
	}

	client, err := k8sclient.NewClientFromKubeConfig(kubeConfig)
	if err != nil {
		return nil, errors.WrapIf(err, "could not create client from kubeconfig")
	}

	return client, nil
}

func (m *MeshReconciler) getRuntimeK8sClient(c cluster.CommonCluster) (runtimeclient.Client, error) {
	kubeConfig, err := c.GetK8sConfig()
	if err != nil {
		return nil, errors.WrapIf(err, "could not get k8s config")
	}

	config, err := k8sclient.NewClientConfig(kubeConfig)
	if err != nil {
		return nil, errors.WrapIf(err, "could not create rest config from kubeconfig")
	}

	client, err := runtimeclient.New(config, runtimeclient.Options{})
	if err != nil {
		return nil, errors.WrapIf(err, "could not create runtime client")
	}

	return client, nil
}
