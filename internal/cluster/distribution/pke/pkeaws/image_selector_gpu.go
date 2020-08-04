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
	"strings"

	"emperror.dev/errors"
)

type gpuImageSelector struct {
	imageSelector ImageSelector
}

// NewGPUImageSelector returns a new GPU image selector.
func NewGPUImageSelector(imageSelector ImageSelector) ImageSelector {
	return gpuImageSelector{imageSelector: imageSelector}
}

func (s gpuImageSelector) SelectImage(ctx context.Context, criteria ImageSelectionCriteria) (string, error) {
	if !isGPUInstance(criteria.InstanceType) || criteria.ContainerRuntime != "docker" {
		return "", errors.WithStack(ImageNotFoundError)
	}

	return s.imageSelector.SelectImage(ctx, criteria)
}

func isGPUInstance(instanceType string) bool {
	return strings.HasPrefix(instanceType, "p2.") || strings.HasPrefix(instanceType, "p3.") ||
		strings.HasPrefix(instanceType, "g3.") || strings.HasPrefix(instanceType, "g4.")
}
