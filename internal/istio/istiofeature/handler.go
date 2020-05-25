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
	"github.com/gofrs/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/validation"

	"github.com/banzaicloud/pipeline/internal/clustergroup/api"
)

type ServiceMeshFeatureHandler struct {
	clusterGetter api.ClusterGetter
	logger        logrus.FieldLogger
	errorHandler  emperror.Handler
	staticConfig  StaticConfig
	helmService   HelmService
}

// NewServiceMeshFeatureHandler returns a new ServiceMeshFeatureHandler instance.
func NewServiceMeshFeatureHandler(
	clusterGetter api.ClusterGetter,
	logger logrus.FieldLogger,
	errorHandler emperror.Handler,
	staticConfig StaticConfig,
	helmService HelmService,
) *ServiceMeshFeatureHandler {
	return &ServiceMeshFeatureHandler{
		clusterGetter: clusterGetter,
		logger:        logger,
		errorHandler:  errorHandler,
		staticConfig:  staticConfig,
		helmService:   helmService,
	}
}

// ReconcileState reconciles service mesh feature on a cluster group
func (h *ServiceMeshFeatureHandler) ReconcileState(featureState api.Feature) error {
	cid, err := uuid.NewV4()
	if err != nil {
		return errors.WrapIf(err, "could not generate uuid")
	}
	logger := h.logger.WithFields(logrus.Fields{
		"correlationID":    cid,
		"clusterGroupID":   featureState.ClusterGroup.Id,
		"clusterGroupName": featureState.ClusterGroup.Name,
	})

	logger.Info("reconciling service mesh feature")
	defer logger.Info("service mesh feature reconciled")

	config, err := h.getConfigFromState(featureState)
	if err != nil {
		return errors.WithStack(err)
	}

	mesh := NewMeshReconciler(*config, h.clusterGetter, logger, h.errorHandler, h.helmService)
	err = mesh.Reconcile()
	if err != nil {
		h.errorHandler.Handle(err)
		return errors.WrapIf(err, "could not reconcile service mesh")
	}

	return nil
}

// ValidateState validates feature state
func (h *ServiceMeshFeatureHandler) ValidateState(featureState api.Feature) error {
	var config Config
	err := mapstructure.Decode(featureState.Properties, &config)
	if err != nil {
		return errors.WrapIf(err, "could not decode properties into config")
	}

	if featureState.ClusterGroup.Clusters[config.MasterClusterID] == nil {
		return errors.New("cluster with master role cannot be removed from the group")
	}

	return nil
}

// ValidateProperties validates feature properties
func (h *ServiceMeshFeatureHandler) ValidateProperties(clusterGroup api.ClusterGroup, currentProperties, properties interface{}) error {
	var currentConfig Config
	err := mapstructure.Decode(currentProperties, &currentConfig)
	if err != nil {
		return errors.WrapIf(err, "could not decode current properties into config")
	}

	var config Config
	err = mapstructure.Decode(properties, &config)
	if err != nil {
		return errors.WrapIf(err, "could not decode new properties into config")
	}

	if config.MasterClusterID == 0 {
		return errors.New("master cluster ID is required")
	}

	if currentConfig.MasterClusterID > 0 && config.MasterClusterID != currentConfig.MasterClusterID {
		return errors.New("master cluster ID cannot be changed")
	}

	errs := validation.IsDNS1123Subdomain(clusterGroup.Name)
	if len(errs) > 0 {
		return errors.WithDetails(errors.Errorf("invalid mesh name: %s", errs[0]), "name", clusterGroup.Name)
	}

	masterClusterIsAMember := false
	for _, member := range clusterGroup.Members {
		if member.ID == config.MasterClusterID {
			masterClusterIsAMember = true
		}
	}

	if !masterClusterIsAMember {
		return errors.New("the specified master cluster is not a member of the cluster group")
	}

	return nil
}

// GetMembersStatus gets member clusters' status
func (h *ServiceMeshFeatureHandler) GetMembersStatus(featureState api.Feature) (map[uint]string, error) {
	var statusMap map[uint]string

	config, err := h.getConfigFromState(featureState)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	mesh := NewMeshReconciler(*config, h.clusterGetter, h.logger, h.errorHandler, h.helmService)
	statusMap, err = mesh.GetClusterStatus()
	if err != nil {
		return nil, errors.WrapIf(err, "could not get clusters status")
	}

	return statusMap, nil
}

func (h *ServiceMeshFeatureHandler) getConfigFromState(state api.Feature) (*Config, error) {
	var config Config
	err := mapstructure.Decode(state.Properties, &config)
	if err != nil {
		return nil, errors.WrapIf(err, "could not decode properties into config")
	}

	config.name = state.ClusterGroup.Name
	config.enabled = state.Enabled
	config.clusterGroup = state.ClusterGroup
	config.internalConfig = h.staticConfig

	return &config, nil
}
