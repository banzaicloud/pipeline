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

package helmadapter

import (
	"context"

	"emperror.dev/errors"

	"github.com/banzaicloud/pipeline/src/auth"
)

// orgService component implementing organization related operations
type orgService struct {
	logger Logger
}

// NewOrgService constructs a new organization service instance
func NewOrgService(logger Logger) OrgService {
	return orgService{logger: logger}
}

// GetOrgNameByOrgID gets the organization name for the provided organization ID
func (o orgService) GetOrgNameByOrgID(ctx context.Context, orgID uint) (string, error) {
	org, err := auth.GetOrganizationById(orgID)
	if err != nil {
		return "", errors.WrapIf(err, "failed to get organization by ID")
	}

	o.logger.Debug("found organization name for organization ID", map[string]interface{}{
		"org ID": orgID, "orgName": org.Name,
	})
	return org.Name, nil
}
