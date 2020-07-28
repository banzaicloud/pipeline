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
	"sync"

	"emperror.dev/errors"
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
	PKEVersion        string
	KubernetesVersion string
	OperatingSystem   string
	ContainerRuntime  string
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

// ImageSelectorChain tries to select an image using a number of selectors.
// When one fails, it moves onto the next in the chain.
type ImageSelectorChain struct {
	imageSelectors []ImageSelector
	names          []string

	mu sync.RWMutex

	logger     Logger
	loggerOnce sync.Once

	errorHandler     ErrorHandler
	errorHandlerOnce sync.Once
}

// NewImageSelectorChain returns a new ImageSelectorChain.
func NewImageSelectorChain(logger Logger, errorHandler ErrorHandler) *ImageSelectorChain {
	return &ImageSelectorChain{
		logger:       logger,
		errorHandler: errorHandler,
	}
}

func (s *ImageSelectorChain) getLogger() Logger {
	s.loggerOnce.Do(func() {
		if s.logger == nil {
			s.logger = NoopLogger{}
		}
	})

	return s.logger
}

func (s *ImageSelectorChain) getErrorHandler() ErrorHandler {
	s.errorHandlerOnce.Do(func() {
		if s.errorHandler == nil {
			s.errorHandler = NoopErrorHandler{}
		}
	})

	return s.errorHandler
}

// AddSelector registers a new ImageSelector in the chain.
func (s *ImageSelectorChain) AddSelector(name string, selector ImageSelector) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.imageSelectors = append(s.imageSelectors, selector)
	s.names = append(s.names, name)
}

func (s *ImageSelectorChain) SelectImage(ctx context.Context, criteria ImageSelectionCriteria) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for i, imageSelector := range s.imageSelectors {
		image, err := imageSelector.SelectImage(ctx, criteria)
		if errors.Is(err, ImageNotFoundError) {
			s.getLogger().InfoContext(ctx, "image selector could not find a matching image", map[string]interface{}{
				"imageSelector": s.names[i],
				"criteria":      criteria,
			})

			continue
		} else if err != nil {
			s.getErrorHandler().HandleContext(ctx, errors.WrapIf(err, "pke image selector"))

			// The original behavior is to move onto the next image selector
			continue
		}

		return image, nil
	}

	return "", errors.Wrap(ImageNotFoundError, "pke image selector")
}
