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
	"crypto/sha1"
	"fmt"

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
	VolumeSize   int    `mapstructure:"volumeSize"`
	InstanceType string `mapstructure:"instanceType"`
	Image        string `mapstructure:"image"`
	SpotPrice    string `mapstructure:"spotPrice"`
	SubnetID     string `mapstructure:"subnetId"`
}

// Validate semantically validates the new node pool.
//
// Some cluster specific compatibility information (eg. subnet settings) should be validated by an external validator.
func (n NewNodePool) Validate() error {
	var violations []string

	if n.Autoscaling.Enabled {
		if n.Autoscaling.MinSize < 0 {
			violations = append(violations, "minimum autoscaling size cannot be lower than zero")
		}

		if n.Autoscaling.MaxSize <= n.Autoscaling.MinSize {
			violations = append(violations, "maximum autoscaling size cannot be lower than the minimum")
		}

		if n.Size < n.Autoscaling.MinSize {
			violations = append(violations, "node pool size cannot be lower than the autoscaling minimum size")
		}

		if n.Size > n.Autoscaling.MaxSize {
			violations = append(violations, "node pool size cannot be higher than the autoscaling maximum size")
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

type ExistingNodePool struct {
	Name          string
	StackID       string
	Status        NodePoolStatus
	StatusMessage string
}

// +testify:mock

// NodePoolStore provides an interface for EKS node pool persistence.
type NodePoolStore interface {
	// CreateNodePool saves a new node pool.
	CreateNodePool(ctx context.Context, clusterID uint, createdBy uint, nodePool NewNodePool) error

	// DeleteNodePool deletes an existing node pool from the storage.
	DeleteNodePool(ctx context.Context, organizationID, clusterID uint, clusterName string, nodePoolName string) error

	// ListNodePools retrieves the node pools for the cluster specified by its
	// cluster ID.
	ListNodePools(
		ctx context.Context,
		organizationID uint,
		clusterID uint,
		clusterName string,
	) (existingNodePools map[string]ExistingNodePool, err error)

	// UpdateNodePoolStackID sets the stack ID in the node pool storage to the
	// specified value.
	UpdateNodePoolStackID(
		ctx context.Context,
		organizationID uint,
		clusterID uint,
		clusterName string,
		nodePoolName string,
		nodePoolStackID string,
	) (err error)

	// UpdateNodePoolStackID sets the status and status message in the node pool
	// storage to the specified value.
	UpdateNodePoolStatus(
		ctx context.Context,
		organizationID uint,
		clusterID uint,
		clusterName string,
		nodePoolName string,
		nodePoolStatus NodePoolStatus,
		nodePoolStatusMessage string,
	) (err error)
}

func CalculateNodePoolVersion(input ...string) string {
	h := sha1.New() // #nosec

	for _, i := range input {
		_, _ = h.Write([]byte(i))
	}

	return fmt.Sprintf("%x", h.Sum(nil))
}
