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

package federation

import (
	"github.com/goph/emperror"
	"github.com/sirupsen/logrus"

	"github.com/banzaicloud/pipeline/internal/clustergroup/api"
)

type Handler struct {
	logger       logrus.FieldLogger
	errorHandler emperror.Handler
}

const FeatureName = "federation"

// NewFederationHandler returns a new Handler instance.
func NewFederationHandler(
	logger logrus.FieldLogger,
	errorHandler emperror.Handler,
) *Handler {
	return &Handler{
		logger:       logger.WithField("feature", FeatureName),
		errorHandler: errorHandler,
	}
}

func (f *Handler) ReconcileState(featureState api.Feature) error {
	logger := f.logger.WithField("clusterGroupName", featureState.ClusterGroup.Name)
	logger.Infof("reconcile federation state, enabled: %v", featureState.Enabled)
	//time.Sleep(20 * time.Second)
	return nil
}

func (f *Handler) ValidateState(featureState api.Feature) error {
	logger := f.logger.WithField("clusterGroupName", featureState.ClusterGroup.Name)
	logger.Info("validate update state")

	return nil
}

func (f *Handler) ValidateProperties(clusterGroup api.ClusterGroup, currentProperties, properties interface{}) error {
	return nil
}

func (f *Handler) GetMembersStatus(featureState api.Feature) (map[uint]string, error) {
	statusMap := make(map[uint]string, 0)
	for _, memberCluster := range featureState.ClusterGroup.Clusters {
		statusMap[memberCluster.GetID()] = "ready"
	}
	return statusMap, nil
}
