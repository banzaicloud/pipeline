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
	"context"
	"regexp"
	"strconv"
	"strings"

	"emperror.dev/emperror"
	"emperror.dev/errors"
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

// RbacEnforcer makes authorization decisions based on user roles.
type RbacEnforcer struct {
	roleSource RoleSource
	logger     Logger
}

// RoleSource returns the user's role in a given organization.
type RoleSource interface {
	// FindUserRole returns the user's role in a given organization.
	// Returns false as the second parameter if the user is not a member of the organization.
	FindUserRole(ctx context.Context, organizationID uint, userID uint) (string, bool, error)
}

// NewRbacEnforcer returns a new RbacEnforcer.
func NewRbacEnforcer(roleSource RoleSource, logger Logger) RbacEnforcer {
	return RbacEnforcer{
		roleSource: roleSource,

		logger: logger,
	}
}

// Enforce makes authorization decisions.
func (e RbacEnforcer) Enforce(org *Organization, user *User, path, method string) (bool, error) {
	// Non-organizational resources are always allowed.
	// TODO: this shouldn't be decided here, remove it!
	if org == nil {
		return true, nil
	}

	// Unauthenticated users are never allowed.
	// TODO: this shouldn't be decided here, remove it!
	if user == nil {
		return false, nil
	}

	// This is a virtual user
	if user.ID == 0 {
		e.logger.Debug("authorizing virtual user", map[string]interface{}{
			"organizationId": org.ID,
			"virtualUser":    user.Login,
		})

		if strings.HasPrefix(user.Login, "clusters/") {
			segments := strings.Split(user.Login, "/")
			if len(segments) < 2 {
				return false, nil
			}

			orgID, err := strconv.Atoi(segments[1])
			if err != nil {
				return false, errors.WrapIf(err, "failed to parse user token")
			}

			return org.ID == uint(orgID), nil
		}

		orgName := GetOrgNameFromVirtualUser(user.Login)

		return org.Name == orgName, nil
	}

	role, member, err := e.roleSource.FindUserRole(context.Background(), org.ID, user.ID)
	if err != nil {
		return false, errors.WrapIfWithDetails(
			err, "failed to check user organization membership",
			"method", method,
			"path", path,
		)
	}

	if !member {
		e.logger.Debug("user is not a member of the organization", map[string]interface{}{
			"organizationId": org.ID,
			"userId":         user.ID,
		})

		return false, nil
	}

	switch role {
	case RoleAdmin:
		return true, nil
	case RoleMember:
		// Members can only read organization resources
		if ok, err := regexp.MatchString(`^/api/v1/orgs(?:/.*)?$`, path); err != nil || (ok && method != "GET") {
			return false, nil
		}

		// Members cannot access secrets at all
		if ok, err := regexp.MatchString(`^/api/v1/orgs/.+/secrets(?:/.*)?$`, path); err != nil || ok {
			return false, errors.WithStackIf(err)
		}

		return true, nil
	default:
		return false, errors.NewWithDetails(
			"unknown membership role",
			"userId", user.ID,
			"organizationId", org.ID,
			"role", role,
			"method", method,
			"path", path,
		)
	}
}

// Authorizer checks if a context has permission to execute an action.
type Authorizer struct {
	db         *gorm.DB
	roleSource RoleSource
}

// NewAuthorizer returns a new Authorizer.
func NewAuthorizer(db *gorm.DB, roleSource RoleSource) Authorizer {
	return Authorizer{
		db:         db,
		roleSource: roleSource,
	}
}

// Authorize authorizes a context to execute an action on an object.
func (a Authorizer) Authorize(ctx context.Context, action string, object interface{}) (bool, error) {
	if action == "virtualUser.create" {
		orgName, ok := object.(string)
		if !ok {
			return false, errors.NewWithDetails("invalid object for action", "action", action, "object", object)
		}

		organization := Organization{Name: orgName}
		err := a.db.
			Where(organization).
			First(&organization).Error
		if err != nil {
			return false, errors.Wrap(err, "failed to query organization name for virtual user")
		}

		userID, ok := UserExtractor{}.GetUserID(ctx)
		if !ok {
			return false, errors.New("user not found in the context")
		}

		role, member, err := a.roleSource.FindUserRole(ctx, organization.ID, userID)
		if err != nil {
			return false, errors.WithMessage(err, "failed to query organization membership for virtual user")
		}

		// TODO: implement better authorization here
		if !member || role != RoleAdmin {
			return false, nil
		}
	}

	return true, nil
}
