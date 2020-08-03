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

package logging

import (
	"context"

	"emperror.dev/errors"
	"github.com/banzaicloud/logging-operator/pkg/sdk/api/v1beta1"
	corev1 "k8s.io/api/core/v1"
	k8sapierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (op IntegratedServiceOperator) createClusterFlowResource(ctx context.Context, managers []outputDefinitionManager, clusterID uint) error {
	if len(managers) == 0 {
		// create flow only in case of non empty output list
		return nil
	}

	flowResource := op.generateFlowResource(managers)

	var oldFlow v1beta1.ClusterFlow
	if err := op.kubernetesService.GetObject(ctx, clusterID, corev1.ObjectReference{
		Namespace: op.config.Namespace,
		Name:      flowResourceName,
	}, &oldFlow); err != nil {
		if k8sapierrors.IsNotFound(err) {
			// ClusterFlow resource is not found, create it
			return op.kubernetesService.EnsureObject(ctx, clusterID, flowResource)
		}

		return errors.WrapIf(err, "failed to get ClusterFlow resource")
	}

	flowResource.ResourceVersion = oldFlow.ResourceVersion
	return op.kubernetesService.Update(ctx, clusterID, flowResource)
}

func (op IntegratedServiceOperator) generateFlowResource(definitions []outputDefinitionManager) *v1beta1.ClusterFlow {
	var outputRefs []string
	for _, d := range definitions {
		outputRefs = append(outputRefs, d.getName())
	}

	return &v1beta1.ClusterFlow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      flowResourceName,
			Namespace: op.config.Namespace,
			Labels:    map[string]string{resourceLabelKey: integratedServiceName},
		},
		Spec: v1beta1.ClusterFlowSpec{
			Match: []v1beta1.ClusterMatch{
				{
					ClusterSelect: &v1beta1.ClusterSelect{},
				},
			},
			OutputRefs: outputRefs,
		},
	}
}
