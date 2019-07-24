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

package clustergroup

import (
	"encoding/json"
	"fmt"

	"emperror.dev/emperror"

	"github.com/banzaicloud/pipeline/internal/clustergroup/api"
)

func (g *Manager) RegisterFeatureHandler(featureName string, handler api.FeatureHandler) {
	g.featureHandlerMap[featureName] = handler
}

func (g *Manager) GetFeatureHandler(featureName string) (api.FeatureHandler, error) {
	handler := g.featureHandlerMap[featureName]
	if handler == nil {
		return nil, &unknownFeature{
			name: featureName,
		}
	}

	return handler, nil
}

func (g *Manager) GetFeatureStatus(feature api.Feature) (map[uint]string, error) {
	handler, ok := g.featureHandlerMap[feature.Name]
	if !ok {
		return nil, nil
	}
	return handler.GetMembersStatus(feature)
}

func (g *Manager) GetEnabledFeatures(clusterGroup api.ClusterGroup) (map[string]api.Feature, error) {
	enabledFeatures := make(map[string]api.Feature, 0)

	features, err := g.GetFeatures(clusterGroup)
	if err != nil {
		return nil, err
	}

	for name, feature := range features {
		if feature.Enabled {
			enabledFeatures[name] = feature
		}
	}

	return enabledFeatures, nil
}

func (g *Manager) ReconcileFeatures(clusterGroup api.ClusterGroup, onlyEnabledHandlers bool) error {
	g.logger.Debugf("reconcile features for group: %s", clusterGroup.Name)

	features, err := g.cgRepo.GetAllFeatures(clusterGroup.Id)
	if err != nil {
		if IsRecordNotFoundError(err) {
			return nil
		}
		return emperror.With(err,
			"clusterGroupId", clusterGroup.Id,
		)
	}

	for _, featureModel := range features {
		g.reconcileFeature(clusterGroup, featureModel, onlyEnabledHandlers)
	}

	return nil
}

func (g *Manager) ReconcileFeature(clusterGroup api.ClusterGroup, featureName string) error {
	g.logger.Debugf("reconcile feature %s for group: %s", featureName, clusterGroup.Name)

	feature, err := g.cgRepo.GetFeature(clusterGroup.Id, featureName)
	if err != nil {
		if IsRecordNotFoundError(err) {
			return nil
		}
		return emperror.With(err,
			"clusterGroupId", clusterGroup.Id,
		)
	}

	err = g.reconcileFeature(clusterGroup, *feature, false)
	if err != nil {
		return emperror.Wrap(err, fmt.Sprintf("error during reconciling state of %s", featureName))
	}
	return nil
}

func (g *Manager) DisableFeatures(clusterGroup api.ClusterGroup) error {
	g.logger.WithField("clusterGroupName", clusterGroup.Name).Debug("disable all enabled features")

	features, err := g.GetFeatures(clusterGroup)
	if err != nil {
		return err
	}

	for name, feature := range features {
		if feature.Enabled {
			g.DisableFeature(name, &clusterGroup)
		}
	}

	// call feature handlers on members update
	err = g.ReconcileFeatures(clusterGroup, false)
	if err != nil {
		return err
	}

	return nil
}

func (g *Manager) GetFeatures(clusterGroup api.ClusterGroup) (map[string]api.Feature, error) {
	features := make(map[string]api.Feature, 0)

	results, err := g.cgRepo.GetAllFeatures(clusterGroup.Id)
	if err != nil {
		if IsRecordNotFoundError(err) {
			return features, nil
		}
		return nil, emperror.With(err,
			"clusterGroupId", clusterGroup.Id,
		)
	}

	for _, r := range results {
		feature, err := g.getFeatureFromModel(clusterGroup, &r)
		if err != nil {
			g.logger.Error(emperror.Wrap(err, "error reading cluster group feature model").Error())
			continue
		}
		features[r.Name] = *feature
	}

	return features, nil
}

func (g *Manager) getFeatureFromModel(clusterGroup api.ClusterGroup, model *ClusterGroupFeatureModel) (*api.Feature, error) {
	var featureProperties interface{}
	if model.Properties != nil {
		err := json.Unmarshal(model.Properties, &featureProperties)
		if err != nil {
			return nil, emperror.Wrap(err, "could not unmarshal feature properties")
		}
	}
	return &api.Feature{
		ClusterGroup:       clusterGroup,
		Properties:         featureProperties,
		Name:               model.Name,
		Enabled:            model.Enabled,
		ReconcileState:     model.ReconcileState,
		LastReconcileError: model.LastReconcileError,
	}, nil
}

// GetFeature returns params of a cluster group feature by clusterGroupId and feature name
func (g *Manager) GetFeature(clusterGroup api.ClusterGroup, featureName string) (*api.Feature, error) {
	result, err := g.cgRepo.GetFeature(clusterGroup.Id, featureName)
	if err != nil {
		return nil, emperror.With(err,
			"clusterGroupId", clusterGroup.Id,
			"featureName", featureName,
		)
	}
	feature, err := g.getFeatureFromModel(clusterGroup, result)
	if err != nil {
		return nil, err
	}
	return feature, nil
}

// DisableFeature disable a cluster group feature
func (g *Manager) DisableFeature(featureName string, clusterGroup *api.ClusterGroup) error {
	err := g.disableFeature(featureName, clusterGroup)
	if err != nil {
		return emperror.Wrap(err, "could not disable feature")
	}

	return nil
}

func (g *Manager) disableFeature(featureName string, clusterGroup *api.ClusterGroup) error {
	_, err := g.GetFeatureHandler(featureName)
	if err != nil {
		return err
	}

	result, err := g.cgRepo.GetFeature(clusterGroup.Id, featureName)
	if err != nil {
		return emperror.With(err,
			"clusterGroupId", clusterGroup.Id,
			"featureName", featureName,
		)
	}

	result.Enabled = false
	err = g.cgRepo.SaveFeature(result)
	if err != nil {
		return emperror.Wrap(err, "could not save feature")
	}

	return nil
}

func (g *Manager) EnableFeature(featureName string, clusterGroup *api.ClusterGroup, properties interface{}) error {
	err := g.setFeatureParams(featureName, clusterGroup, true, properties)
	if err != nil {
		return emperror.Wrap(err, "could not enable feature")
	}

	return nil
}

func (g *Manager) UpdateFeature(featureName string, clusterGroup *api.ClusterGroup, properties interface{}) error {
	err := g.setFeatureParams(featureName, clusterGroup, false, properties)
	if err != nil {
		return emperror.Wrap(err, "could not update feature")
	}

	return nil
}

// SetFeatureParams sets params of a cluster group feature
func (g *Manager) setFeatureParams(featureName string, clusterGroup *api.ClusterGroup, setEnableFlag bool, properties interface{}) error {
	handler, err := g.GetFeatureHandler(featureName)
	if err != nil {
		return emperror.Wrap(err, "could not get feature handler")
	}

	currentlyEnabled := true

	result, err := g.cgRepo.GetFeature(clusterGroup.Id, featureName)
	if IsFeatureRecordNotFoundError(err) {
		result = &ClusterGroupFeatureModel{
			Name:           featureName,
			ClusterGroupID: clusterGroup.Id,
		}
	} else {
		if err != nil {
			return emperror.With(err,
				"clusterGroupId", clusterGroup.Id,
				"featureName", featureName,
			)
		}
		currentlyEnabled = result.Enabled
	}

	var currentProperties interface{}
	if currentlyEnabled {
		if result.Properties != nil {
			err = json.Unmarshal(result.Properties, &currentProperties)
			if err != nil {
				return emperror.Wrap(err, "could not marshal current feature properties")
			}
		}
	}

	if setEnableFlag {
		result.Enabled = true
	}

	result.Properties, err = json.Marshal(properties)
	if err != nil {
		return emperror.Wrap(err, "could not marshal new feature properties")
	}

	err = handler.ValidateProperties(*clusterGroup, currentProperties, properties)
	if err != nil {
		return emperror.Wrap(err, "invalid properties")
	}

	err = g.cgRepo.SaveFeature(result)
	if err != nil {
		return emperror.Wrap(err, "could not save feature")
	}

	return nil
}
