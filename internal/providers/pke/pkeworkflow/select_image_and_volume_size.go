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

		if optionalVolumeSize == 0 {
			selectedVolumeSize = int(math.Max(float64(MinimalVolumeSize), float64(activityOutput.VolumeSize)))
		} else if optionalVolumeSize < activityOutput.VolumeSize { // && optionalVolumeSize != 0 {
			return "", 0, errors.New(fmt.Sprintf(
				"specified volume size of %dGB for %q instance type using %s image is less than the AMI image size of %dGB",
				optionalVolumeSize, instanceType, selectedImageID, activityOutput.VolumeSize,
			))
		} else { // if optionalVolumeSize != 0 && optionalVolumeSize >= activityOutput.VolumeSize {
			selectedVolumeSize = optionalVolumeSize
		}
	}

	return selectedImageID, selectedVolumeSize, nil
}
