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

const SelectImageActivityName = "pke-select-image-activity"

type SelectImageActivity struct {
	clusters      Clusters
	imageSelector pkeaws.ImageSelector
}

func NewSelectImageActivity(clusters Clusters, imageSelector pkeaws.ImageSelector) *SelectImageActivity {
	return &SelectImageActivity{
		clusters:      clusters,
		imageSelector: imageSelector,
	}
}

type SelectImageActivityInput struct {
	ClusterID    uint
	InstanceType string
}

type SelectImageActivityOutput struct {
	ImageID string
}

func (a *SelectImageActivity) Execute(ctx context.Context, input SelectImageActivityInput) (*SelectImageActivityOutput, error) {
	c, err := a.clusters.GetCluster(ctx, input.ClusterID)
	if err != nil {
		return nil, err
	}

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

	// Special logic if the instance type is a GPU instance
	if isGPUInstance(input.InstanceType) {
		cri = "docker"
	}

	criteria := pkeaws.ImageSelectionCriteria{
		Region:            c.GetLocation(),
		InstanceType:      input.InstanceType,
		PKEVersion:        pkeaws.Version,
		KubernetesVersion: ver,
		OperatingSystem:   "ubuntu",
		ContainerRuntime:  cri,
	}

	image, err := a.imageSelector.SelectImage(ctx, criteria)
	if err != nil {
		return nil, errors.WrapIff(err, "failed to get default image for Kubernetes version %s", ver)
	}

	return &SelectImageActivityOutput{
		ImageID: image,
	}, nil
}
