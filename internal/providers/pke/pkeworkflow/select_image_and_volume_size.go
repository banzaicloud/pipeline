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
	"fmt"
	"math"

	"emperror.dev/errors"
	"go.uber.org/cadence/workflow"
)

const (
	// fallbackVolumeSize is the substituted value for 0/unspecified default
	// volume size.
	fallbackVolumeSize = 50
)

// SelectImageAndVolumeSize returns an image ID and a volume size for the
// specified arguments after determining an image to use and the required volume
// size for it.
func SelectImageAndVolumeSize(
	ctx workflow.Context,
	awsActivityInput AWSActivityInput,
	clusterID uint,
	instanceType string,
	optionalImageID string,
	optionalVolumeSize int,
	defaultVolumeSize int,
) (selectedImageID string, selectedVolumeSize int, err error) {
	selectedImageID = optionalImageID
	if selectedImageID == "" {
		activityInput := SelectImageActivityInput{
			ClusterID:    clusterID,
			InstanceType: instanceType,
		}
		var activityOutput SelectImageActivityOutput
		err := workflow.ExecuteActivity(ctx, SelectImageActivityName, activityInput).Get(ctx, &activityOutput)
		if err != nil {
			return "", 0, err
		}

		selectedImageID = activityOutput.ImageID
	}

	{
		activityInput := SelectVolumeSizeActivityInput{
			AWSActivityInput: awsActivityInput,
			ImageID:          selectedImageID,
		}
		var activityOutput SelectVolumeSizeActivityOutput
		err := workflow.ExecuteActivity(ctx, SelectVolumeSizeActivityName, activityInput).Get(ctx, &activityOutput)
		if err != nil {
			return "", 0, err
		}

		valueSource := ""
		if optionalVolumeSize > 0 {
			selectedVolumeSize = optionalVolumeSize
			valueSource = "explicitly set"
		} else if defaultVolumeSize > 0 {
			selectedVolumeSize = defaultVolumeSize
			valueSource = "default configured"
		} else {
			selectedVolumeSize = int(math.Max(float64(fallbackVolumeSize), float64(activityOutput.VolumeSize)))
			valueSource = "fallback value"
		}

		if selectedVolumeSize < activityOutput.VolumeSize {
			return "", 0, errors.New(fmt.Sprintf(
				"selected volume size of %d GiB (source: %s) for %q instance type using %s image"+
					" is less than the AMI size of %d GiB",
				selectedVolumeSize, valueSource, instanceType, selectedImageID, activityOutput.VolumeSize,
			))
		}
	}

	return selectedImageID, selectedVolumeSize, nil
}
