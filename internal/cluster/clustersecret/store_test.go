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

package clustersecret

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type ensureSecretExistsStub struct {
	secretID       string
	organizationID uint
	secret         NewSecret
}

func (s *ensureSecretExistsStub) EnsureSecretExists(organizationID uint, secret NewSecret) (string, error) {
	s.organizationID = organizationID
	s.secret = secret

	return s.secretID, nil
}

type clustersStub struct {
	cluster Cluster
}

func (s *clustersStub) GetCluster(ctx context.Context, id uint) (Cluster, error) {
	return s.cluster, nil
}

type clusterStub struct {
	organizationID uint
	uid            string
}

func (s *clusterStub) GetOrganizationID() uint {
	return s.organizationID
}

func (s *clusterStub) GetUID() string {
	return s.uid
}

func TestStore_EnsureSecretExists(t *testing.T) {
	clusters := &clustersStub{
		cluster: &clusterStub{
			organizationID: 1,
			uid:            "abcd",
		},
	}
	secretStore := &ensureSecretExistsStub{
		secretID: "secret",
	}
	store := NewStore(clusters, secretStore)

	secret := NewSecret{
		Name: "name",
		Type: "secret",
		Values: map[string]string{
			"key": "value",
		},
		Tags: []string{"key:value"},
	}

	clusterID := uint(1)
	secretID, err := store.EnsureSecretExists(context.Background(), clusterID, secret)
	require.NoError(t, err)

	expectedSecret := NewSecret{
		Name: "cluster-1-name",
		Type: "secret",
		Values: map[string]string{
			"key": "value",
		},
		Tags: []string{"key:value", "clusterUID:abcd", "clusterID:1"},
	}

	assert.Equal(t, "secret", secretID)
	assert.Equal(t, clusters.cluster.GetOrganizationID(), secretStore.organizationID)
	assert.Equal(t, expectedSecret, secretStore.secret)
}
