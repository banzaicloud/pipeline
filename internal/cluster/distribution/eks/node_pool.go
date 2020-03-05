// Copyright © 2020 Banzai Cloud
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

package eks

import (
	"context"

	"github.com/banzaicloud/pipeline/internal/cluster"
)

// NewNodePool describes new a Kubernetes node pool in an Amazon EKS cluster.
type NewNodePool struct {
	Name        string            `mapstructure:"name"`
	Labels      map[string]string `mapstructure:"labels"`
	Size        int               `mapstructure:"size"`
	Autoscaling struct {
		Enabled bool `mapstructure:"enabled"`
		MinSize int  `mapstructure:"minSize"`
		MaxSize int  `mapstructure:"maxSize"`
	} `mapstructure:"autoscaling"`
	InstanceType string `mapstructure:"instanceType"`
	Image        string `mapstructure:"image"`
	SpotPrice    string `mapstructure:"spotPrice"`
	Subnet       struct {
		SubnetId         string `mapstructure:"subnetId"`
		Cidr             string `mapstructure:"cidr"`
		AvailabilityZone string `mapstructure:"availabilityZone"`
	} `mapstructure:"subnet"`
}

// Validate semantically validates the new node pool.
//
// Some cluster specific compatibility information (eg. subnet settings) should be validated by an external validator.
func (n NewNodePool) Validate() error {
	var violations []string

	if n.Autoscaling.Enabled {
		if n.Autoscaling.MinSize < 1 {
			violations = append(violations, "minimum autoscaling size cannot be lower than one")
		}

		if n.Autoscaling.MaxSize <= n.Autoscaling.MinSize {
			violations = append(violations, "maximum autoscaling size cannot be lower than the minimum")
		}
	} else if n.Size < 1 {
		violations = append(violations, "size cannot be lower than one")
	}

	if n.InstanceType == "" {
		violations = append(violations, "instance type cannot be empty")
	}

	if len(violations) > 0 {
		return cluster.NewValidationError("invalid node pool creation request", violations)
	}

	return nil
}

// NodePoolStore provides an interface for EKS node pool persistence.
type NodePoolStore interface {
	// CreateNodePool saves a new node pool.
	CreateNodePool(ctx context.Context, clusterID uint, createdBy uint, nodePool NewNodePool) error
}
