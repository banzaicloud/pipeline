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
	"fmt"
	"math"

	"emperror.dev/errors"
	"go.uber.org/cadence/activity"
	"go.uber.org/cadence/workflow"

	"github.com/banzaicloud/pipeline/pkg/cadence/worker"
)

const (
	// SelectVolumeSizeActivityName is the unique name of the activity.
	SelectVolumeSizeActivityName = "eks-select-volume-size-activity"

	// fallbackVolumeSize is the substituted value for 0/unspecified default
	// volume size.
	fallbackVolumeSize = 50
)

type SelectVolumeSizeActivity struct {
	defaultVolumeSize int
}

type SelectVolumeSizeActivityInput struct {
	AMISize            int
	OptionalVolumeSize int
}

type SelectVolumeSizeActivityOutput struct {
	VolumeSize int
}

func NewSelectVolumeSizeActivity(defaultVolumeSize int) (activity *SelectVolumeSizeActivity) {
	return &SelectVolumeSizeActivity{
		defaultVolumeSize: defaultVolumeSize,
	}
}

func (a *SelectVolumeSizeActivity) Execute(ctx context.Context, input SelectVolumeSizeActivityInput) (output *SelectVolumeSizeActivityOutput, err error) {
	output = &SelectVolumeSizeActivityOutput{}
	valueSource := ""
	if input.OptionalVolumeSize > 0 {
		output.VolumeSize = input.OptionalVolumeSize
		valueSource = "explicitly set"
	} else if a.defaultVolumeSize > 0 {
		output.VolumeSize = a.defaultVolumeSize
		valueSource = "default configured"
	} else {
		output.VolumeSize = int(math.Max(float64(fallbackVolumeSize), float64(input.AMISize)))
		valueSource = "fallback value"
	}

	if output.VolumeSize < input.AMISize {
		return nil, errors.New(fmt.Sprintf(
			"selected volume size of %d GB (source: %s) is less than the AMI size of %d GB",
			output.VolumeSize, valueSource, input.AMISize,
		))
	}

	return output, nil
}

// Register registers the activity.
func (a SelectVolumeSizeActivity) Register(worker worker.Registry) {
	worker.RegisterActivityWithOptions(a.Execute, activity.RegisterOptions{Name: SelectVolumeSizeActivityName})
}

// selectVolumeSize selects a node volume size from the available values,
// including (in the order of precedence) the explicitly chosen value, the
// default configured value or the bigger value of the AMI size and a global
// minimum.
//
// This is a convenience wrapper around the corresponding activity.
func selectVolumeSize(ctx workflow.Context, amiSize, optionalVolumeSize int) (int, error) {
	var activityOutput SelectVolumeSizeActivityOutput
	err := selectVolumeSizeAsync(ctx, amiSize, optionalVolumeSize).Get(ctx, &activityOutput)
	if err != nil {
		return 0, err
	}

	return activityOutput.VolumeSize, nil
}

// selectVolumeSize returns a future selecting a node volume size from the
// available values, including (in the order of precedence) the explicitly
// chosen value, the default configured value or the bigger value of the AMI
// size and a global minimum.
//
// This is a convenience wrapper around the corresponding activity.
func selectVolumeSizeAsync(ctx workflow.Context, amiSize, optionalVolumeSize int) workflow.Future {
	return workflow.ExecuteActivity(ctx, SelectVolumeSizeActivityName, SelectVolumeSizeActivityInput{
		AMISize:            amiSize,
		OptionalVolumeSize: optionalVolumeSize,
	})
}
