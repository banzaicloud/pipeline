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

package clusteradapter

import (
	"context"

	"emperror.dev/errors"

	"github.com/banzaicloud/pipeline/internal/cluster"
)

// PolyClusterDeleter combines many cluster specific deleters into one.
type PolyClusterDeleter struct {
	clusters cluster.Store
	deleters map[string]cluster.Deleter
}

// NewPolyClusterDeleter returns a new PolyClusterDeleter instance.
func NewPolyClusterDeleter(clusters cluster.Store, deleters ...ClusterDeleterEntry) PolyClusterDeleter {
	ds := make(map[string]cluster.Deleter, len(deleters))
	for _, d := range deleters {
		if _, exists := ds[d.Key.String()]; exists {
			panic(errors.Errorf("duplicate key: %v", d.Key))
		}

		ds[d.Key.String()] = d.Deleter
	}

	return PolyClusterDeleter{
		clusters: clusters,
		deleters: ds,
	}
}

// ClusterDeleterEntry is a ClusterDeleterKey - Deleter pair.
type ClusterDeleterEntry struct {
	Key     ClusterDeleterKey
	Deleter cluster.Deleter
}

// ClusterDeleterKey is used to select the cluster specific deleter implementation.
type ClusterDeleterKey struct {
	Provider     string
	Distribution string
}

// MakeClusterDeleterKey is a helper function that returns a ClusterDeleterKey.
func MakeClusterDeleterKey(provider, distribution string) ClusterDeleterKey {
	return ClusterDeleterKey{
		Provider:     provider,
		Distribution: distribution,
	}
}

// String returns a string representation of the ClusterDeleterKey.
// This string can be used as a key in a map.
func (k ClusterDeleterKey) String() string {
	const sep = "/"
	return k.Provider + sep + k.Distribution
}

// DeleteCluster selects the matching deleter for the cluster and delegates deletion to it.
func (cd PolyClusterDeleter) DeleteCluster(ctx context.Context, clusterID uint, options cluster.DeleteClusterOptions) error {
	c, err := cd.clusters.GetCluster(ctx, clusterID)
	if err != nil {
		return err
	}

	key := ClusterDeleterKey{
		Provider:     c.Cloud,
		Distribution: c.Distribution,
	}

	d := cd.deleters[key.String()]
	if d == nil {
		return errors.NewWithDetails("no cluster deleter for distribution on provider", "provider", key.Provider, "distribution", key.Distribution)
	}

	return d.DeleteCluster(ctx, clusterID, options)
}
