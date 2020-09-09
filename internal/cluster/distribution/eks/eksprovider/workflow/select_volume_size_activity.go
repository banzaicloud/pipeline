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
	AMISize int
}

type SelectVolumeSizeActivityOutput struct {
	VolumeSize int
}

func NewSelectVolumeSizeActivity(defaultVolumeSize int) (activity *SelectVolumeSizeActivity) {
	return &SelectVolumeSizeActivity{
		defaultVolumeSize: defaultVolumeSize,
	}
}

func (activity *SelectVolumeSizeActivity) Execute(ctx context.Context, input SelectVolumeSizeActivityInput) (output *SelectVolumeSizeActivityOutput, err error) {
	output = &SelectVolumeSizeActivityOutput{}
	if activity.defaultVolumeSize > 0 {
		output.VolumeSize = activity.defaultVolumeSize
	} else {
		output.VolumeSize = int(math.Max(float64(fallbackVolumeSize), float64(input.AMISize)))
	}

	if output.VolumeSize < input.AMISize {
		return nil, errors.New(fmt.Sprintf(
			"selected volume size of %d GB (default configuration) is less than the AMI size of %d GB",
			output.VolumeSize, input.AMISize,
		))
	}

	return output, nil
}
