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

package api

// FeatureRequest
type FeatureRequest interface{}

// FeatureResponse
type FeatureResponse struct {
	Name               string          `json:"name"`
	ClusterGroup       ClusterGroup    `json:"clusterGroup"`
	Enabled            bool            `json:"enabled"`
	Properties         FeatureRequest  `json:"properties,omitempty" yaml:"properties"`
	Status             map[uint]string `json:"status,omitempty" yaml:"status"`
	ReconcileState     string          `json:"reconcileState,omitempty" yaml:"reconcileState"`
	LastReconcileError string          `json:"lastReconcileError,omitempty" yaml:"lastReconcileError"`
}

const ReconcileInProgress = "IN_PROGRESS"

const ReconcileSucceded = "SUCCESS"

const ReconcileFailed = "FAILED"

// Feature
type Feature struct {
	Name               string       `json:"name"`
	ClusterGroup       ClusterGroup `json:"clusterGroup"`
	Enabled            bool         `json:"enabled"`
	Properties         interface{}  `json:"properties,omitempty"`
	ReconcileState     string       `json:"reconcileState,omitempty"`
	LastReconcileError string       `json:"lastReconcileError,omitempty"`
}

type FeatureHandler interface {
	ReconcileState(featureState Feature) error
	ValidateState(featureState Feature) error
	ValidateProperties(clusterGroup ClusterGroup, currentProperties, properties interface{}) error
	GetMembersStatus(featureState Feature) (map[uint]string, error)
}
