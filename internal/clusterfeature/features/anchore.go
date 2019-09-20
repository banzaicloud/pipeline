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

package features

import (
	"context"

	"emperror.dev/errors"
	anchore "github.com/banzaicloud/pipeline/internal/security"
)

type AnchoreConfig struct {
	AnchoreEndpoint string
	AnchoreEnabled  bool
}

// AnchoreService decouples anchor related operations
type AnchoreService interface {
	// GenerateUser generates an anchore user and stores it in the secret store
	GenerateUser(ctx context.Context, orgID uint, clusterGUID string) (string, error)

	// Deletes a previously generated user from the anchore
	DeleteUser(ctx context.Context, orgID uint, clusterGUID string) error

	AnchoreConfig() AnchoreConfig
}

// anchoreService basic implementer of the AnchoreService
type anchoreService struct {
}

func (a *anchoreService) AnchoreConfig() AnchoreConfig {
	return AnchoreConfig{
		AnchoreEndpoint: anchore.AnchoreEndpoint,
		AnchoreEnabled:  anchore.AnchoreEnabled,
	}
}

func NewAnchoreService() AnchoreService {
	return new(anchoreService)
}

func (a *anchoreService) GenerateUser(ctx context.Context, orgID uint, clusterGUID string) (string, error) {

	usr, err := anchore.SetupAnchoreUser(orgID, clusterGUID)
	if err != nil {
		return "", errors.WrapWithDetails(err, "error creating anchore user", "organization", orgID,
			"clusterGUID", clusterGUID)
	}

	return usr.UserId, nil
}

func (a *anchoreService) DeleteUser(ctx context.Context, orgID uint, clusterGUID string) error {
	// todo refactor the original implementation to handle errors?
	// todo the secret only needs to be removed upon successful account deletion! (original implementation is wrong)

	anchore.RemoveAnchoreUser(orgID, clusterGUID)

	return nil
}
