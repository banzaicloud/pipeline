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

package pkeaws

import (
	"context"

	"emperror.dev/errors"
)

// ImageSelector chooses an image based on the selection criteria.
// It returns an ImageNotFoundError when no images can be found matching the provided criteria.
type ImageSelector interface {
	SelectImage(ctx context.Context, criteria ImageSelectionCriteria) (string, error)
}

// ImageSelectionCriteria contains all parameters for selecting an image.
type ImageSelectionCriteria struct {
	Region            string
	InstanceType      string
	PKEVersion        string
	KubernetesVersion string
	OperatingSystem   string
	ContainerRuntime  string
}

// ImageNotFoundError is returned by an ImageSelector when it cannot find an image matching the provided criteria.
const ImageNotFoundError = errors.Sentinel("no images found matching the selection criteria")
