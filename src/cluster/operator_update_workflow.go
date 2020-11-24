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
	"context"
	"time"

	"emperror.dev/errors"
	"go.uber.org/cadence/workflow"
	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"

	"github.com/banzaicloud/pipeline/pkg/cluster"
	"github.com/banzaicloud/pipeline/pkg/k8sclient"
)

const (
	IntServiceOperatorUpdaterActivityName = "integrated-service-operator-updater"
	IntServiceCRDName                     = "serviceinstances.integrated-service.banzaicloud.io"
)

type IntServiceOperatorUpdaterWorkflow struct {
	manager *Manager
}

func NewIntServiceOperatorUpdaterWorkflow(manager *Manager) *IntServiceOperatorUpdaterWorkflow {
	return &IntServiceOperatorUpdaterWorkflow{
		manager: manager,
	}
}

type IntServiceOperatorUpdaterActivityInput struct {
	ClusterID uint
}

func (w IntServiceOperatorUpdaterWorkflow) Execute(ctx workflow.Context, input IntServiceOperatorUpdaterActivityInput) error {
	logger := workflow.GetLogger(ctx).Sugar().With(
		"clusterID", input.ClusterID,
	)

	commonCluster, err := w.manager.GetClusterByIDOnly(context.Background(), input.ClusterID)
	if err != nil {
		logger.Errorf("failed to get cluster from database: %s", err.Error())
		return err
	}

	status, err := commonCluster.GetStatus()
	if err != nil {
		logger.Errorf("failed to get cluster status: %s", err.Error())
	} else {
		if status.Status == cluster.Deleting {
			// stop workflow
			return nil
		}

		err = checkIntegratedServiceOperator(commonCluster)
		if err != nil {
			logger.Errorf("failed to check integrated service operator: %s", err.Error())
		} else {
			logger.Info("integrated service operator CRD installed")
		}
	}

	err = workflow.Sleep(ctx, 1*time.Minute)
	if err != nil {
		return errors.WrapIf(err, "sleep cancelled")
	}

	return workflow.NewContinueAsNewError(ctx, w.Execute, input)
}

type OperatorUpdater struct {
	commonCluster CommonCluster
}

func (ou *OperatorUpdater) getClientConfig() (*rest.Config, error) {
	kubeConfig, err := ou.commonCluster.GetK8sConfig()
	if err != nil {
		return nil, errors.WrapIf(err, "could not get k8s config")
	}

	clientConfig, err := k8sclient.NewClientConfig(kubeConfig)
	if err != nil {
		return nil, errors.WrapIf(err, "cloud not create client config from kubeconfig")
	}

	return clientConfig, nil
}

func checkIntegratedServiceOperator(commonCluster CommonCluster) error {
	ou := OperatorUpdater{
		commonCluster: commonCluster,
	}
	clientConfig, err := ou.getClientConfig()
	if err != nil {
		return errors.WrapIf(err, "failed to get client config")
	}

	return ou.getCRD(clientConfig)
}

func (ou *OperatorUpdater) getCRD(clientConfig *rest.Config) error {
	cl, err := v1beta1.NewForConfig(clientConfig)
	if err != nil {
		return errors.WrapIf(err, "failed to get client for config")
	}

	_, err = cl.CustomResourceDefinitions().Get(context.Background(), IntServiceCRDName, metav1.GetOptions{})
	if err != nil {
		return errors.WrapIf(err, "failed to get CRD")
	}

	return nil
}
