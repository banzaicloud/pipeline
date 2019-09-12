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

package auth

import (
	"strconv"
	"strings"

	"emperror.dev/emperror"
	"github.com/jinzhu/gorm"
)

// BasicEnforcer is the default enforcer implementation for authorization.
type BasicEnforcer struct {
	db *gorm.DB
}

// NewBasicEnforcer returns a new enforcer.
func NewBasicEnforcer(db *gorm.DB) *BasicEnforcer {
	return &BasicEnforcer{db: db}
}

// Enforce makes authorization decisions.
func (e *BasicEnforcer) Enforce(org *Organization, user *User, path, method string) (bool, error) {
	if user == nil {
		return false, nil
	}

	if org == nil {
		return true, nil
	}

	if user.ID == 0 {
		if strings.HasPrefix(user.Login, "clusters/") {
			segments := strings.Split(user.Login, "/")
			if len(segments) < 2 {
				return false, nil
			}

			orgID, err := strconv.Atoi(segments[1])
			if err != nil {
				return false, emperror.Wrap(err, "failed to parse user token")
			}

			return org.ID == uint(orgID), nil
		}

		orgName := GetOrgNameFromVirtualUser(user.Login)
		return org.Name == orgName, nil
	}

	err := e.db.Model(user).Where(org).Related(org, "Organizations").Error

	if err != nil {
		if gorm.IsRecordNotFoundError(err) {
			return false, nil
		}
		return false, emperror.Wrap(err, "failed to query user's organizations from db")
	}

	return true, nil
}
