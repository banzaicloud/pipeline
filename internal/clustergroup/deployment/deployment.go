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

package deployment

import (
	"encoding/json"
	"time"

	"github.com/ghodss/yaml"

	"github.com/banzaicloud/pipeline/helm"
)

// ClusterGroupDeployment describes a Helm deployment to a Cluster Group
type ClusterGroupDeployment struct {
	ReleaseName    string                            `json:"releaseName" yaml:"releaseName"`
	Name           string                            `json:"name" yaml:"name" binding:"required"`
	Version        string                            `json:"version,omitempty" yaml:"version,omitempty"`
	Package        []byte                            `json:"package,omitempty" yaml:"package,omitempty"`
	ReUseValues    bool                              `json:"reuseValues" yaml:"reuseValues"`
	Namespace      string                            `json:"namespace,omitempty" yaml:"namespace,omitempty"`
	DryRun         bool                              `json:"dryrun,omitempty" yaml:"dryrun,omitempty"`
	Values         map[string]interface{}            `json:"values,omitempty" yaml:"values,omitempty"`
	ValueOverrides map[string]map[string]interface{} `json:"valueOverrides,omitempty" yaml:"valueOverrides,omitempty"`
	RollingMode    bool                              `json:"rollingMode,omitempty" yaml:"rollingMode,omitempty"`
	Atomic         bool                              `json:"atomic,omitempty" yaml:"atomic,omitempty"`
}

// DeploymentInfo describes the details of a helm deployment
type DeploymentInfo struct {
	ReleaseName          string                            `json:"releaseName"`
	Chart                string                            `json:"chart"`
	ChartName            string                            `json:"chartName"`
	ChartVersion         string                            `json:"chartVersion"`
	Namespace            string                            `json:"namespace"`
	Version              int32                             `json:"version,omitempty"`
	Description          string                            `json:"description"`
	CreatedAt            time.Time                         `json:"createdAt,omitempty"`
	UpdatedAt            time.Time                         `json:"updatedAt,omitempty"`
	Values               map[string]interface{}            `json:"values"`
	ValueOverrides       map[string]map[string]interface{} `json:"valueOverrides,omitempty" yaml:"valueOverrides,omitempty"`
	TargetClusters       map[uint]bool                     `json:"-" yaml:"-"`
	TargetClustersStatus []TargetClusterStatus             `json:"targetClusters"`
}

func (c *DeploymentInfo) GetValuesForCluster(clusterName string) ([]byte, error) {
	// copy c.values into a new map before merging
	values := make(map[string]interface{})
	if c.Values != nil {
		m, err := json.Marshal(c.Values)
		if err != nil {
			return nil, err
		}
		err = json.Unmarshal(m, &values)
		if err != nil {
			return nil, err
		}
	}

	clusterSpecificOverrides, exists := c.ValueOverrides[clusterName]
	// merge values with overrides for cluster if any
	if exists {
		values = helm.MergeValues(values, clusterSpecificOverrides)
	}
	marshalledValues, err := yaml.Marshal(values)
	if err != nil {
		return nil, err
	}
	return marshalledValues, nil
}

// CreateUpdateDeploymentResponse describes a create/update deployment response
type CreateUpdateDeploymentResponse struct {
	ReleaseName    string                `json:"releaseName"`
	TargetClusters []TargetClusterStatus `json:"targetClusters"`
}

// TargetClusterStatus describes a status of a deployment on a target cluster
type TargetClusterStatus struct {
	ClusterId    uint   `json:"clusterId"`
	ClusterName  string `json:"clusterName"`
	Cloud        string `json:"cloud,omitempty"`
	Distribution string `json:"distribution,omitempty"`
	Status       string `json:"status"`
	Stale        bool   `json:"stale"`
	Version      string `json:"version,omitempty"`
	Error        string `json:"error,omitempty"`
}

// TargetOperationStatus describes a status of a deployment operation (install/upgrade/delete) on a target cluster
type TargetOperationStatus struct {
	ClusterId   uint   `json:"clusterId"`
	ClusterName string `json:"clusterName"`
	Status      string `json:"status"`
	Error       string `json:"error,omitempty"`
}

// ListDeploymentResponse describes a deployment list response
type ListDeploymentResponse struct {
	Name         string    `json:"releaseName"`
	Chart        string    `json:"chart"`
	ChartName    string    `json:"chartName"`
	ChartVersion string    `json:"chartVersion"`
	Version      int32     `json:"version,omitempty"`
	UpdatedAt    time.Time `json:"updatedAt"`
	Namespace    string    `json:"namespace"`
	CreatedAt    time.Time `json:"createdAt,omitempty"`
}

// DeleteResponse describes a deployment delete response
type DeleteResponse struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
	Name    string `json:"name"`
}
