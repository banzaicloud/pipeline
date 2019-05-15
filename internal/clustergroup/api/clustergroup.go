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

import (
	"github.com/pkg/errors"
)

// CreateRequest describes fields of a create cluster group request
type CreateRequest struct {
	Name    string `json:"name" yaml:"name" example:"cluster_group_name"`
	Members []uint `json:"members" yaml:"members"`
}

// Validate validates CreateRequest
func (g *CreateRequest) Validate() error {
	if len(g.Name) == 0 {
		return errors.New("cluster group name is empty")
	}

	if len(g.Members) == 0 {
		return errors.New("there should be at least one cluster member")
	}
	return nil
}

// CreateResponse describes fields of a create cluster group response
type CreateResponse struct {
	Name       string `json:"name" example:"cluster_group_name"`
	ResourceID uint   `json:"id"`
}

// UpdateRequest describes fields of a update cluster group request
type UpdateRequest struct {
	Name    string `json:"name" yaml:"name" example:"cluster_group_name"`
	Members []uint `json:"members,omitempty" yaml:"members"`
}

// Validate validates UpdateRequest
func (g *UpdateRequest) Validate() error {
	if len(g.Name) == 0 {
		return errors.New("cluster group name is empty")
	}

	if len(g.Members) == 0 {
		return errors.New("there should be at least one cluster member")
	}
	return nil
}

// UpdateResponse describes fields of a update cluster group response
type UpdateResponse struct {
	Name       string `json:"name" example:"cluster_group_name"`
	ResourceID uint   `json:"id"`
}

// Member
type Member struct {
	ID           uint   `json:"id" yaml:"id" example:"1001"`
	Cloud        string `json:"cloud,omitempty" yaml:"cloud" example:"google"`
	Distribution string `json:"distribution,omitempty" yaml:"distribution" example:"gke"`
	Name         string `json:"name" yaml:"name" example:"clusterName"`
	Status       string `json:"status,omitempty" yaml:"status,omitempty"`
}

// ClusterGroup
type ClusterGroup struct {
	Id              uint             `json:"id" yaml:"id" example:"10"`
	UID             string           `json:"uid" yaml:"uid"`
	Name            string           `json:"name" yaml:"name"`
	OrganizationID  uint             `json:"organizationId" yaml:"organizationId"`
	Members         []Member         `json:"members,omitempty" yaml:"members"`
	EnabledFeatures []string         `json:"enabledFeatures,omitempty" yaml:"enabledFeatures"`
	Clusters        map[uint]Cluster `json:"-" yaml:"-"`
}

func (g *ClusterGroup) IsMember(clusterName string) bool {
	for _, m := range g.Members {
		if clusterName == m.Name {
			return true
		}
	}
	return false
}
