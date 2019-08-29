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

package clusterfeaturedriver

import (
	"context"

	"github.com/banzaicloud/pipeline/internal/clusterfeature"
)

// dummyFeatureService is used for testing purposes.
type dummyFeatureService struct {
	FeatureList    []clusterfeature.Feature
	FeatureDetails clusterfeature.Feature
	Err            error
}

func (s *dummyFeatureService) List(ctx context.Context, clusterID uint) ([]clusterfeature.Feature, error) {
	if s.Err != nil {
		return nil, s.Err
	}

	return s.FeatureList, nil
}

func (s *dummyFeatureService) Details(ctx context.Context, clusterID uint, featureName string) (clusterfeature.Feature, error) {
	if s.Err != nil {
		return clusterfeature.Feature{}, s.Err
	}

	return s.FeatureDetails, nil
}

func (s *dummyFeatureService) Activate(ctx context.Context, clusterID uint, featureName string, spec map[string]interface{}) error {
	return s.Err
}

func (s *dummyFeatureService) Deactivate(ctx context.Context, clusterID uint, featureName string) error {
	return s.Err
}

func (s *dummyFeatureService) Update(ctx context.Context, clusterID uint, featureName string, spec map[string]interface{}) error {
	return s.Err
}
