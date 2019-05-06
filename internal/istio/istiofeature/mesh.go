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
	"github.com/goph/emperror"
	"github.com/sirupsen/logrus"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/client-go/kubernetes"

	istiooperatorclientset "github.com/banzaicloud/istio-operator/pkg/client/clientset/versioned"
	"github.com/banzaicloud/pipeline/cluster"
	"github.com/banzaicloud/pipeline/internal/clustergroup/api"
	"github.com/banzaicloud/pipeline/pkg/k8sclient"
)

func NewMeshReconciler(config Config, clusterGetter api.ClusterGetter, logger logrus.FieldLogger, errorHandler emperror.Handler) *MeshReconciler {
	reconciler := &MeshReconciler{
		Configuration: config,

		clusterGetter: clusterGetter,
		logger:        logger,
		errorHandler:  errorHandler,
	}

	reconciler.init()

	reconciler.logger = reconciler.logger.WithFields(logrus.Fields{
		"clusterID":   reconciler.Master.GetID(),
		"clusterName": reconciler.Master.GetName(),
	})

	return reconciler
}

func (m *MeshReconciler) GetMasterK8sClient() (*kubernetes.Clientset, error) {
	return m.GetK8sClient(m.Master)
}

func (m *MeshReconciler) GetK8sClient(c cluster.CommonCluster) (*kubernetes.Clientset, error) {
	kubeConfig, err := c.GetK8sConfig()
	if err != nil {
		return nil, emperror.Wrap(err, "could not get k8s config")
	}

	client, err := k8sclient.NewClientFromKubeConfig(kubeConfig)
	if err != nil {
		return nil, emperror.Wrap(err, "cloud not create client from kubeconfig")
	}

	return client, nil
}

func (m *MeshReconciler) GetMasterIstioOperatorK8sClient() (*istiooperatorclientset.Clientset, error) {
	return m.GetIstioOperatorK8sClient(m.Master)
}

func (m *MeshReconciler) GetIstioOperatorK8sClient(c cluster.CommonCluster) (*istiooperatorclientset.Clientset, error) {
	kubeConfig, err := m.Master.GetK8sConfig()
	if err != nil {
		return nil, emperror.Wrap(err, "could not get k8s config")
	}

	config, err := k8sclient.NewClientConfig(kubeConfig)
	if err != nil {
		return nil, emperror.Wrap(err, "could not create rest config from kubeconfig")
	}

	client, err := istiooperatorclientset.NewForConfig(config)
	if err != nil {
		return nil, emperror.Wrap(err, "could not create istio operator client")
	}

	return client, nil
}

func (m *MeshReconciler) GetApiExtensionK8sClient(c cluster.CommonCluster) (*apiextensionsclient.Clientset, error) {
	kubeConfig, err := m.Master.GetK8sConfig()
	if err != nil {
		return nil, emperror.Wrap(err, "could not get k8s config")
	}

	config, err := k8sclient.NewClientConfig(kubeConfig)
	if err != nil {
		return nil, emperror.Wrap(err, "could not create rest config from kubeconfig")
	}

	client, err := apiextensionsclient.NewForConfig(config)
	if err != nil {
		return nil, emperror.Wrap(err, "could not create istio operator client")
	}

	return client, nil
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
