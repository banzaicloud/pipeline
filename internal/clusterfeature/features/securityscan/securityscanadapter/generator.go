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

package securityscanadapter

import (
	"context"
	"fmt"

	"emperror.dev/errors"
)

const (
	anchoreUserUIDNameTpl = "%v-anchore-user"
)

// ClusterService provides access to clusters.
type ClusterService interface {
	// GetClusterUID returns the unique ID of a cluster.
	GetClusterUID(ctx context.Context, clusterID uint) (string, error)
}

// UserNameGenerator generates an Anchore username for a cluster.
type UserNameGenerator struct {
	clusterService ClusterService
}

// NewUserNameGenerator returns a new UserNameGenerator.
func NewUserNameGenerator(clusterService ClusterService) UserNameGenerator {
	return UserNameGenerator{
		clusterService: clusterService,
	}
}

// GenerateUsername generates a unique username using the cluster's UUID
func (g UserNameGenerator) GenerateUsername(ctx context.Context, clusterID uint) (string, error) {
	uid, err := g.clusterService.GetClusterUID(ctx, clusterID)
	if err != nil {
		return "", errors.WrapIf(err, "failed to generate username")
	}

	return fmt.Sprintf(anchoreUserUIDNameTpl, uid), nil
}
