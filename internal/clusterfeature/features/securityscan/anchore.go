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

package securityscan

import (
	"context"

	"emperror.dev/errors"

	"github.com/banzaicloud/pipeline/internal/common"
	anchore "github.com/banzaicloud/pipeline/internal/security"
)

// FeatureAnchoreService decouples anchore related operations
type FeatureAnchoreService interface {
	GenerateUser(ctx context.Context, orgID uint, clusterID uint) (string, error)

	// Deletes a previously generated user from the anchore
	DeleteUser(ctx context.Context, orgID uint, clusterID uint) error
}

// anchoreService basic implementer of the FeatureAnchoreService
type anchoreService struct {
	anchoreUserService anchore.AnchoreUserService
	logger             common.Logger
}

func NewFeatureAnchoreService(anchoreUserService anchore.AnchoreUserService, logger common.Logger) FeatureAnchoreService {
	return anchoreService{
		anchoreUserService: anchoreUserService,
		logger:             logger,
	}
}

func (a anchoreService) GenerateUser(ctx context.Context, orgID uint, clusterID uint) (string, error) {
	userName, err := a.anchoreUserService.EnsureUser(ctx, orgID, clusterID)
	if err != nil {

		a.logger.Debug("error creating anchore user", map[string]interface{}{"organization": orgID,
			"clusterGUID": clusterID})

		return "", errors.WrapWithDetails(err, "error creating anchore user", "organization", orgID,
			"clusterGUID", clusterID)
	}

	a.logger.Debug("anchore user ensured", map[string]interface{}{"organization": orgID,
		"clusterGUID": clusterID})

	return userName, nil
}

func (a anchoreService) DeleteUser(ctx context.Context, orgID uint, clusterID uint) error {

	if err := a.anchoreUserService.RemoveUser(ctx, orgID, clusterID); err != nil {

		a.logger.Debug("error deleting anchore user", map[string]interface{}{"organization": orgID,
			"clusterID": clusterID})

		return errors.WrapWithDetails(err, "error deleting anchore user", "organization", orgID,
			"clusterID", clusterID)
	}

	a.logger.Info("anchore user deleted", map[string]interface{}{"organization": orgID,
		"clusterID": clusterID})
	return nil
}
