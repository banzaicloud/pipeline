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
	"strings"

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

// ImageSelectors select an image using a number of selectors.
// When one fails, it moves onto the next.
type ImageSelectors []ImageSelector

func (s ImageSelectors) SelectImage(ctx context.Context, criteria ImageSelectionCriteria) (string, error) {
	for _, selector := range s {
		image, err := selector.SelectImage(ctx, criteria)
		if err != nil {
			continue
		}

		return image, nil
	}

	return "", errors.WithStack(ImageNotFoundError)
}

// DefaultImageSelector selects an image from the local fallback images based on the instance type (ie. GPU or not).
//
// Note: this process should be refactored once image sourcing is improved (eg. move to cloudinfo).
type DefaultImageSelector struct {
	DefaultImages            ImageSelector
	DefaultAcceleratedImages ImageSelector
	DefaultARMImages         ImageSelector
}

// NewDefaultImageSelector returns a new DefaultImageSelector.
func NewDefaultImageSelector() DefaultImageSelector {
	return DefaultImageSelector{
		DefaultImages:            defaultImages,
		DefaultAcceleratedImages: defaultAcceleratedImages,
		DefaultARMImages:         defaultARMImages,
	}
}

func (s DefaultImageSelector) SelectImage(ctx context.Context, criteria ImageSelectionCriteria) (string, error) {
	var image string

	if isGPUInstance(criteria.InstanceType) {
		image, _ = s.DefaultAcceleratedImages.SelectImage(ctx, criteria)
	} else if isARMInstance(criteria.InstanceType) {
		image, _ = s.DefaultARMImages.SelectImage(ctx, criteria)
	}

	if image == "" {
		return s.DefaultImages.SelectImage(ctx, criteria)
	}

	return image, nil
}

func isGPUInstance(instanceType string) bool {
	return strings.HasPrefix(instanceType, "p2.") || strings.HasPrefix(instanceType, "p3.") ||
		strings.HasPrefix(instanceType, "g3.") || strings.HasPrefix(instanceType, "g4.")
}

func isARMInstance(instanceType string) bool {
	return strings.HasPrefix(instanceType, "a1.") || strings.HasPrefix(instanceType, "t4g.") ||
		strings.HasPrefix(instanceType, "m6g.") || strings.HasPrefix(instanceType, "c6g.") ||
		strings.HasPrefix(instanceType, "r6g.")
}
