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

package pkeworkflow

import (
	"context"
	"strings"

	"emperror.dev/errors"

	"github.com/banzaicloud/pipeline/internal/cluster/distribution/pke/pkeaws"
)

const SelectImagesActivityName = "pke-select-images-activity"

type SelectImagesActivity struct {
	clusters      Clusters
	imageSelector pkeaws.ImageSelector
}

func NewSelectImagesActivity(clusters Clusters, imageSelector pkeaws.ImageSelector) *SelectImagesActivity {
	return &SelectImagesActivity{
		clusters:      clusters,
		imageSelector: imageSelector,
	}
}

type SelectImagesActivityInput struct {
	ClusterID uint
	NodePools []NodePool
}

func (a *SelectImagesActivity) Execute(ctx context.Context, input SelectImagesActivityInput) ([]NodePool, error) {
	c, err := a.clusters.GetCluster(ctx, input.ClusterID)
	if err != nil {
		return nil, err
	}

	pools := input.NodePools

	awsCluster, ok := c.(AWSCluster)
	if !ok {
		return nil, errors.Errorf("unexpected type of cluster: %t", c)
	}

	ver, err := awsCluster.GetKubernetesVersion()
	if err != nil {
		return nil, errors.WrapIf(err, "can't get Kubernetes version")
	}

	cri, err := awsCluster.GetKubernetesContainerRuntime()
	if err != nil {
		return nil, errors.WrapIf(err, "can't get Kubernetes container runtime")
	}

	isGPUInstance := func(instanceType string) bool {
		return strings.HasPrefix(instanceType, "p2.") || strings.HasPrefix(instanceType, "p3.") ||
			strings.HasPrefix(instanceType, "g3.") || strings.HasPrefix(instanceType, "g4.")
	}

	for i, pool := range pools {
		if pool.ImageID == "" {
			// Special logic if the instance type is a GPU instance
			if isGPUInstance(pool.InstanceType) {
				cri = "docker"
			}

			criteria := pkeaws.ImageSelectionCriteria{
				Region:            c.GetLocation(),
				InstanceType:      pool.InstanceType,
				PKEVersion:        pkeaws.Version,
				KubernetesVersion: ver,
				OperatingSystem:   "ubuntu",
				ContainerRuntime:  cri,
			}

			image, err := a.imageSelector.SelectImage(ctx, criteria)
			if err != nil {
				return nil, errors.WrapIff(err, "failed to get default image for Kubernetes version %s", ver)
			}

			pools[i].ImageID = image
		}
	}

	return pools, nil
}
