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
	"emperror.dev/emperror"
	"github.com/gofrs/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/banzaicloud/pipeline/internal/clustergroup/api"
)

type Handler struct {
	clusterGetter  api.ClusterGetter
	infraNamespace string
	logger         logrus.FieldLogger
	errorHandler   emperror.Handler
}

const FeatureName = "federation"

// NewFederationHandler returns a new Handler instance.
func NewFederationHandler(
	clusterGetter api.ClusterGetter,
	infraNamespace string,
	logger logrus.FieldLogger,
	errorHandler emperror.Handler,
) *Handler {
	return &Handler{
		clusterGetter:  clusterGetter,
		infraNamespace: infraNamespace,
		logger:         logger.WithField("feature", FeatureName),
		errorHandler:   errorHandler,
	}
}

func (f *Handler) ReconcileState(featureState api.Feature) error {

	cid, err := uuid.NewV4()
	if err != nil {
		return emperror.Wrap(err, "could not generate uuid")
	}
	logger := f.logger.WithFields(logrus.Fields{
		"correlationID":    cid,
		"clusterGroupID":   featureState.ClusterGroup.Id,
		"clusterGroupName": featureState.ClusterGroup.Name,
		"enabled":          featureState.Enabled,
	})

	logger.Infof("start reconciling federation state")
	defer logger.Infof("finished reconciling federation state")

	config, err := f.getConfigFromState(featureState)
	if err != nil {
		return errors.WithStack(err)
	}

	fedv2 := NewFederationReconciler(featureState.ClusterGroup.Name, *config, f.clusterGetter, f.infraNamespace, logger, f.errorHandler)
	err = fedv2.Reconcile()
	if err != nil {
		f.errorHandler.Handle(err)
		return emperror.Wrap(err, "could not reconcile federation")
	}

	return nil
}

func (f *Handler) ValidateState(featureState api.Feature) error {
	var config Config
	err := mapstructure.Decode(featureState.Properties, &config)
	if err != nil {
		return emperror.Wrap(err, "could not decode properties into config")
	}

	if featureState.ClusterGroup.Clusters[config.HostClusterID] == nil {
		return errors.New("host cluster cannot be removed from the group")
	}

	return nil
}

func (f *Handler) ValidateProperties(clusterGroup api.ClusterGroup, currentProperties, properties interface{}) error {
	var currentConfig Config
	err := mapstructure.Decode(currentProperties, &currentConfig)
	if err != nil {
		return emperror.Wrap(err, "could not decode current properties into config")
	}

	var config Config
	err = mapstructure.Decode(properties, &config)
	if err != nil {
		return emperror.Wrap(err, "could not decode new properties into config")
	}

	if config.HostClusterID == 0 {
		return errors.New("host cluster ID is required")
	}

	if currentConfig.HostClusterID > 0 && config.HostClusterID != currentConfig.HostClusterID {
		return errors.New("host cluster ID cannot be changed")
	}

	hostClusterIsAMember := false
	for _, member := range clusterGroup.Members {
		if member.ID == config.HostClusterID {
			hostClusterIsAMember = true
		}
	}

	if !hostClusterIsAMember {
		return errors.New("the specified host cluster is not a member of the cluster group")
	}

	return nil
}

func (f *Handler) GetMembersStatus(featureState api.Feature) (map[uint]string, error) {
	var statusMap map[uint]string

	config, err := f.getConfigFromState(featureState)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	fedv2 := NewFederationReconciler(featureState.ClusterGroup.Name, *config, f.clusterGetter, f.infraNamespace, f.logger, f.errorHandler)
	statusMap, err = fedv2.GetStatus()
	if err != nil {
		f.errorHandler.Handle(err)
		return nil, emperror.Wrap(err, "could not reconcile federation")
	}

	return statusMap, nil
}

func (f *Handler) getConfigFromState(state api.Feature) (*Config, error) {
	var config Config
	err := mapstructure.Decode(state.Properties, &config)
	if err != nil {
		return nil, emperror.Wrap(err, "could not decode properties into config")
	}

	config.name = state.ClusterGroup.Name
	config.enabled = state.Enabled
	config.clusterGroup = state.ClusterGroup

	if len(config.TargetNamespace) == 0 {
		config.TargetNamespace = "federation-system"
	}

	return &config, nil
}
