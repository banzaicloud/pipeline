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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (op FeatureOperator) createClusterFlowResource(ctx context.Context, managers []outputDefinitionManager, clusterID uint) error {
	// delete old ClusterFlow resource
	if err := op.kubernetesService.DeleteObject(ctx, clusterID, &v1beta1.ClusterFlow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      flowResourceName,
			Namespace: op.config.Namespace,
		},
	}); err != nil {
		return errors.WrapIfWithDetails(err, "failed to delete flow resource")
	}

	// create new flow resource
	var flowResource = op.generateFlowResource(managers)
	return op.kubernetesService.EnsureObject(ctx, clusterID, flowResource)
}

func (op FeatureOperator) generateFlowResource(definitions []outputDefinitionManager) *v1beta1.ClusterFlow {
	var outputRefs []string
	for _, d := range definitions {
		outputRefs = append(outputRefs, d.getName())
	}

	return &v1beta1.ClusterFlow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      flowResourceName,
			Namespace: op.config.Namespace,
		},
		Spec: v1beta1.FlowSpec{
			Selectors:  map[string]string{},
			OutputRefs: outputRefs,
		},
	}
}
