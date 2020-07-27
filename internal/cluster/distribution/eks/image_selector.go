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

	"emperror.dev/errors"
	"github.com/Masterminds/semver/v3"
)

// +testify:mock:testOnly=true

// ImageSelector chooses an image based on the selection criteria.
// It returns an ImageNotFoundError when no images can be found matching the provided criteria.
type ImageSelector interface {
	SelectImage(ctx context.Context, criteria ImageSelectionCriteria) (string, error)
}

// ImageSelectionCriteria contains all parameters for selecting an image.
type ImageSelectionCriteria struct {
	Region            string
	InstanceType      string
	KubernetesVersion string
}

// ImageNotFoundError is returned by an ImageSelector when it cannot find an image matching the provided criteria.
const ImageNotFoundError = errors.Sentinel("no images found matching the selection criteria")

// RegionMapImageSelector selects an image based on the region in the selection criteria.
type RegionMapImageSelector map[string]string

func (r RegionMapImageSelector) SelectImage(_ context.Context, criteria ImageSelectionCriteria) (string, error) {
	image, ok := r[criteria.Region]
	if !ok {
		return "", ImageNotFoundError
	}

	return image, nil
}

// KubernetesVersionImageSelector selects an image from the delegated selector if the kubernetes version criteria
// matches the constraint.
type KubernetesVersionImageSelector struct {
	Constraint    *semver.Constraints
	ImageSelector ImageSelector
}

func (s KubernetesVersionImageSelector) SelectImage(ctx context.Context, criteria ImageSelectionCriteria) (string, error) {
	kubeVersion, err := semver.NewVersion(criteria.KubernetesVersion)
	if err != nil {
		return "", errors.WrapWithDetails(
			err, "parse kubernetes version",
			"kubernetesVersion", criteria.KubernetesVersion,
		)
	}

	if !s.Constraint.Check(kubeVersion) {
		return "", errors.WithStack(ImageNotFoundError)
	}

	return s.ImageSelector.SelectImage(ctx, criteria)
}
