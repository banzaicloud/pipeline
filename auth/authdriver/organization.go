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

package authdriver

import (
	"context"

	"emperror.dev/errors"
	"github.com/jinzhu/gorm"

	"github.com/banzaicloud/pipeline/auth"
)

// NewOrganizationGetter creates and OrganizationGetter
func NewOrganizationGetter(db *gorm.DB) OrganizationGetter {
	return OrganizationGetter{
		db: db,
	}
}

// OrganizationGetter implements organization retrieval
type OrganizationGetter struct {
	db *gorm.DB
}

// Get retrieves an organization by its ID
func (g OrganizationGetter) Get(_ context.Context, id uint) (auth.Organization, error) {
	org := auth.Organization{
		ID: id,
	}
	if err := g.db.Where(&org).Find(&org).Error; err != nil {
		return org, errors.WrapIf(err, "failed to load organization from database")
	}
	return org, nil
}
