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

package istio

import (
	"github.com/goph/emperror"
	"github.com/sirupsen/logrus"

	"github.com/banzaicloud/pipeline/internal/clustergroup/api"
)

type ServiceMeshFeatureHandler struct {
	logger       logrus.FieldLogger
	errorHandler emperror.Handler
}

const FeatureName = "servicemesh"

// NewServiceMeshFeatureHandler returns a new ServiceMeshFeatureHandler instance.
func NewServiceMeshFeatureHandler(
	logger logrus.FieldLogger,
	errorHandler emperror.Handler,
) *ServiceMeshFeatureHandler {
	return &ServiceMeshFeatureHandler{
		logger:       logger,
		errorHandler: errorHandler,
	}
}

func (h *ServiceMeshFeatureHandler) ReconcileState(featureState api.Feature) error {
	h.logger.Infof("federation enabled %v on group: %v", featureState.Enabled, featureState.ClusterGroup.Name)
	return nil
}
