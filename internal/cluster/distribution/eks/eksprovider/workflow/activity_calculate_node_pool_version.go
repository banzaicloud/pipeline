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

package workflow

import (
	"context"
	"crypto/sha1"
	"fmt"
	"strings"

	"go.uber.org/cadence/activity"
	"go.uber.org/cadence/workflow"

	"github.com/banzaicloud/pipeline/internal/cluster/distribution/eks"
	"github.com/banzaicloud/pipeline/pkg/cadence/worker"
)

const CalculateNodePoolVersionActivityName = "eks-calculate-node-pool-version"

// CalculateNodePoolVersionActivity calculates the node pool version.
type CalculateNodePoolVersionActivity struct{}

type CalculateNodePoolVersionActivityInput struct {
	Image                string
	Volumes              eks.NodePoolVolumes
	CustomSecurityGroups []string
}

type CalculateNodePoolVersionActivityOutput struct {
	Version string
}

// NewCalculateNodePoolVersionActivity creates a new CalculateNodePoolVersionActivity instance.
func NewCalculateNodePoolVersionActivity() CalculateNodePoolVersionActivity {
	return CalculateNodePoolVersionActivity{}
}

// Register registers the activity in the worker.
func (a CalculateNodePoolVersionActivity) Register(worker worker.ActivityRegistry) {
	worker.RegisterActivityWithOptions(a.Execute, activity.RegisterOptions{Name: CalculateNodePoolVersionActivityName})
}

// Execute is the main body of the activity.
func (a CalculateNodePoolVersionActivity) Execute(
	_ context.Context,
	input CalculateNodePoolVersionActivityInput,
) (CalculateNodePoolVersionActivityOutput, error) {
	instanceRootVolumeStorage := "<nil>"
	instanceRootVolumeEncryption := "<nil>"
	instanceRootVolumeSize := 0

	if input.Volumes.InstanceRoot != nil {
		instanceRootVolumeStorage = input.Volumes.InstanceRoot.Storage
		instanceRootVolumeSize = input.Volumes.InstanceRoot.Size
		if input.Volumes.InstanceRoot.Encryption != nil {
			instanceRootVolumeEncryption = fmt.Sprintf("%v", *input.Volumes.InstanceRoot.Encryption)
		}
	}

	kubeletRootVolumeStorage := "<nil>"
	kubeletRootVolumeEncryption := "<nil>"
	kubeletRootVolumeSize := 0

	if input.Volumes.KubeletRoot != nil {
		kubeletRootVolumeStorage = input.Volumes.KubeletRoot.Storage
		kubeletRootVolumeSize = input.Volumes.KubeletRoot.Size
		if input.Volumes.KubeletRoot.Encryption != nil {
			kubeletRootVolumeEncryption = fmt.Sprintf("%v", *input.Volumes.KubeletRoot.Encryption)
		}
	}

	calculationParams := []string{
		input.Image,
		instanceRootVolumeStorage,
		instanceRootVolumeEncryption,
		fmt.Sprintf("%d", instanceRootVolumeSize),
		kubeletRootVolumeStorage,
		kubeletRootVolumeEncryption,
		fmt.Sprintf("%d", kubeletRootVolumeSize),
		strings.Join(input.CustomSecurityGroups, ","),
	}

	h := sha1.New() // #nosec

	for _, i := range calculationParams {
		_, _ = h.Write([]byte(i))
	}

	return CalculateNodePoolVersionActivityOutput{Version: fmt.Sprintf("%x", h.Sum(nil))}, nil
}

// CalculateNodePoolVersion retrieves the calculated nodePoolVersion
//
// This is a convenience wrapper around the corresponding activity.
func calculateNodePoolVersion(
	ctx workflow.Context,
	image string,
	volumes eks.NodePoolVolumes,
	customSecurityGroups []string,
) (string, error) {
	var activityOutput CalculateNodePoolVersionActivityOutput
	err := calculateNodePoolVersionAsync(
		ctx, image, volumes, customSecurityGroups).Get(ctx, &activityOutput)
	if err != nil {
		return "", err
	}

	return activityOutput.Version, nil
}

// calculateNodePoolVersion retrieves a future object for calucating the  nodePoolVersion
//
// This is a convenience wrapper around the corresponding activity.
func calculateNodePoolVersionAsync(
	ctx workflow.Context,
	image string,
	volumes eks.NodePoolVolumes,
	customSecurityGroups []string,
) workflow.Future {
	return workflow.ExecuteActivity(ctx, CalculateNodePoolVersionActivityName, CalculateNodePoolVersionActivityInput{
		Image:                image,
		Volumes:              volumes,
		CustomSecurityGroups: customSecurityGroups,
	})
}
