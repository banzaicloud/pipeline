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
	"fmt"
	"sync"
)

// NewInMemoryFeatureRepository returns a new in-memory feature repository.
func NewInMemoryFeatureRepository(features map[uint][]Feature) *InMemoryFeatureRepository {
	lookup := make(map[uint]map[string]Feature, len(features))
	for clID, fs := range features {
		m := make(map[string]Feature, len(fs))
		lookup[clID] = m
		for _, f := range fs {
			m[f.Name] = f
		}
	}
	return &InMemoryFeatureRepository{
		features: lookup,
	}
}

// InMemoryFeatureRepository keeps features in the memory.
// Use it in tests or for development/demo purposes.
type InMemoryFeatureRepository struct {
	features map[uint]map[string]Feature

	mu sync.RWMutex
}

// GetFeatures returns a list of all the features stored in the repository for the specified cluster
func (r *InMemoryFeatureRepository) GetFeatures(ctx context.Context, clusterID uint) ([]Feature, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	features, ok := r.features[clusterID]
	if !ok {
		return nil, nil
	}

	fs := make([]Feature, 0, len(features))

	for _, feature := range features {
		fs = append(fs, feature)
	}

	return fs, nil
}

// GetFeature returns the feature identified by the parameters if it is in the repository, otherwise an error is returned
func (r *InMemoryFeatureRepository) GetFeature(ctx context.Context, clusterID uint, featureName string) (Feature, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if clusterFeatures, ok := r.features[clusterID]; ok {
		if feature, ok := clusterFeatures[featureName]; ok {
			return feature, nil
		}
	}

	return Feature{}, featureNotFoundError{
		clusterID:   clusterID,
		featureName: featureName,
	}
}

// SaveFeature persists the feature to the repository
func (r *InMemoryFeatureRepository) SaveFeature(ctx context.Context, clusterID uint, featureName string, spec FeatureSpec, status FeatureStatus) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	clusterFeatures, ok := r.features[clusterID]
	if !ok {
		clusterFeatures = make(map[string]Feature)
		r.features[clusterID] = clusterFeatures
	}

	clusterFeatures[featureName] = Feature{
		Name:   featureName,
		Spec:   spec,
		Status: status,
	}

	return nil
}

// UpdateFeatureStatus sets the feature's status
func (r *InMemoryFeatureRepository) UpdateFeatureStatus(ctx context.Context, clusterID uint, featureName string, status string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if clusterFeatures, ok := r.features[clusterID]; ok {
		if feature, ok := clusterFeatures[featureName]; ok {
			feature.Status = status
			clusterFeatures[featureName] = feature
			return nil
		}
	}

	return featureNotFoundError{
		clusterID:   clusterID,
		featureName: featureName,
	}
}

// UpdateFeatureSpec sets the feature's specification
func (r *InMemoryFeatureRepository) UpdateFeatureSpec(ctx context.Context, clusterID uint, featureName string, spec FeatureSpec) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if clusterFeatures, ok := r.features[clusterID]; ok {
		if feature, ok := clusterFeatures[featureName]; ok {
			feature.Spec = spec
			clusterFeatures[featureName] = feature
			return nil
		}
	}

	return featureNotFoundError{
		clusterID:   clusterID,
		featureName: featureName,
	}
}

// DeleteFeature removes the feature from the repository.
// It is an idempotent operation.
func (r *InMemoryFeatureRepository) DeleteFeature(ctx context.Context, clusterID uint, featureName string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if clusterFeatures, ok := r.features[clusterID]; ok {
		delete(clusterFeatures, featureName)
	}

	return nil
}

// Clear removes every entry from the repository
func (r *InMemoryFeatureRepository) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.features = make(map[uint]map[string]Feature)
}

// Snapshot returns a snapshot of the repository's state that can be restored later
func (r *InMemoryFeatureRepository) Snapshot() map[uint]map[string]Feature {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return copyClusterLookup(r.features)
}

// Restore sets the repository's state from a snapshot
func (r *InMemoryFeatureRepository) Restore(snapshot map[uint]map[string]Feature) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.features = copyClusterLookup(snapshot)
}

func copyFeatureLookup(lookup map[string]Feature) map[string]Feature {
	if lookup == nil {
		return nil
	}
	result := make(map[string]Feature, len(lookup))
	for n, f := range lookup {
		result[n] = f
	}
	return result
}

func copyClusterLookup(lookup map[uint]map[string]Feature) map[uint]map[string]Feature {
	if lookup == nil {
		return nil
	}
	result := make(map[uint]map[string]Feature, len(lookup))
	for c, fs := range lookup {
		result[c] = copyFeatureLookup(fs)
	}
	return result
}

type featureNotFoundError struct {
	clusterID   uint
	featureName string
}

func (e featureNotFoundError) Error() string {
	return fmt.Sprintf("Feature %q not found for cluster %d.", e.featureName, e.clusterID)
}

func (e featureNotFoundError) Details() []interface{} {
	return []interface{}{
		"clusterId", e.clusterID,
		"feature", e.featureName,
	}
}

func (featureNotFoundError) FeatureNotFound() bool {
	return true
}
