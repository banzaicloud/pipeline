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

package clusterfeature

import (
	"context"
	"sync"

	"emperror.dev/errors"
)

// InMemoryFeatureRepository keeps features in the memory.
// Use it in tests or for development/demo purposes.
type InMemoryFeatureRepository struct {
	features map[uint]map[string]Feature

	mu sync.RWMutex
}

// NewInMemoryFeatureRepository returns a new inmemory feature repository.
func NewInMemoryFeatureRepository() *InMemoryFeatureRepository {
	return &InMemoryFeatureRepository{
		features: make(map[uint]map[string]Feature),
	}
}

func (r *InMemoryFeatureRepository) GetFeatures(ctx context.Context, clusterID uint) ([]Feature, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	features, ok := r.features[clusterID]
	if !ok {
		return nil, nil
	}

	f := make([]Feature, len(features))
	i := 0

	for _, feature := range features {
		f[i] = feature
		i++
	}

	return f, nil
}

func (r *InMemoryFeatureRepository) GetFeature(ctx context.Context, clusterID uint, featureName string) (*Feature, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	features, ok := r.features[clusterID]
	if !ok {
		return nil, nil
	}

	feature, ok := features[featureName]
	if ok {
		feature := feature

		return &feature, nil
	}

	return nil, nil
}

func (r *InMemoryFeatureRepository) CreateFeature(ctx context.Context, clusterID uint, featureName string, spec FeatureSpec, status string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.features[clusterID]; !ok {
		r.features[clusterID] = make(map[string]Feature)
	}

	if _, ok := r.features[clusterID][featureName]; ok {
		return errors.New("feature already exists")
	}

	r.features[clusterID][featureName] = Feature{
		Name:   featureName,
		Spec:   spec,
		Status: status,
	}

	return nil
}

func (r *InMemoryFeatureRepository) CreateOrUpdateFeature(ctx context.Context, clusterID uint, featureName string, spec FeatureSpec, status string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.features[clusterID]; !ok {
		r.features[clusterID] = make(map[string]Feature)
	}

	r.features[clusterID][featureName] = Feature{
		Name:   featureName,
		Spec:   spec,
		Status: status,
	}

	return nil
}

func (r *InMemoryFeatureRepository) UpdateFeatureStatus(ctx context.Context, clusterID uint, featureName string, status string) (*Feature, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	features, ok := r.features[clusterID]
	if !ok {
		r.features[clusterID] = make(map[string]Feature)
	}

	f, ok := features[featureName]
	if !ok {
		return nil, errors.NewWithDetails("feature not found", "feature", featureName)
	}

	f.Status = status

	r.features[clusterID][featureName] = f

	return &f, nil

}

func (r *InMemoryFeatureRepository) UpdateFeatureSpec(ctx context.Context, clusterID uint, featureName string, spec FeatureSpec) (*Feature, error) {
	features, ok := r.features[clusterID]
	if !ok {
		r.features[clusterID] = make(map[string]Feature)
	}

	f, ok := features[featureName]
	if !ok {
		return nil, errors.NewWithDetails("feature not found", "feature", featureName)
	}

	f.Spec = spec

	r.features[clusterID][featureName] = f

	return &f, nil
}

func (r *InMemoryFeatureRepository) DeleteFeature(ctx context.Context, clusterID uint, featureName string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	_, ok := r.features[clusterID]
	if !ok {
		return nil
	}

	delete(r.features[clusterID], featureName)

	return nil
}
