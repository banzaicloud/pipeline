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

package clustergroup

import (
	"fmt"

	"github.com/pkg/errors"
)

type unknownFeature struct {
	name string
}

func (e *unknownFeature) Error() string {
	return "unknown feature"
}

func (e *unknownFeature) Context() []interface{} {
	return []interface{}{
		"name", e.name,
	}
}

// IsUnknownFeatureError returns true if the passed in error designates an unknown feature (no registered handler) error
func IsUnknownFeatureError(err error) bool {
	_, ok := errors.Cause(err).(*unknownFeature)

	return ok
}

type clusterGroupNotFoundError struct {
	clusterGroup ClusterGroupModel
}

func (e *clusterGroupNotFoundError) Error() string {
	return "cluster group not found"
}

func (e *clusterGroupNotFoundError) Context() []interface{} {
	return []interface{}{
		"clusterGroupID", e.clusterGroup.ID,
		"organizationID", e.clusterGroup.OrganizationID,
	}
}

// IsClusterGroupNotFoundError returns true if the passed in error designates a cluster group not found error
func IsClusterGroupNotFoundError(err error) bool {
	_, ok := errors.Cause(err).(*clusterGroupNotFoundError)

	return ok
}

type clusterGroupAlreadyExistsError struct {
	clusterGroup ClusterGroupModel
}

func (e *clusterGroupAlreadyExistsError) Error() string {
	return "cluster group already exists with this name"
}

func (e *clusterGroupAlreadyExistsError) Context() []interface{} {
	return []interface{}{
		"clusterGroupName", e.clusterGroup.Name,
		"organizationID", e.clusterGroup.OrganizationID,
	}
}

// IsClusterGroupAlreadyExistsError returns true if the passed in error designates a cluster group already exists error
func IsClusterGroupAlreadyExistsError(err error) bool {
	_, ok := errors.Cause(err).(*clusterGroupAlreadyExistsError)

	return ok
}

type memberClusterNotFoundError struct {
	orgID     uint
	clusterID uint
}

func (e *memberClusterNotFoundError) Error() string {
	return "member cluster not found"
}

func (e *memberClusterNotFoundError) Message() string {
	return fmt.Sprintf("%s: %d", e.Error(), e.clusterID)
}

func (e *memberClusterNotFoundError) Context() []interface{} {
	return []interface{}{
		"clusterID", e.clusterID,
		"organizationID", e.orgID,
	}
}

// IsMemberClusterNotFoundError returns true if the passed in error designates a cluster group member is not found
func IsMemberClusterNotFoundError(err error) (*memberClusterNotFoundError, bool) {
	e, ok := errors.Cause(err).(*memberClusterNotFoundError)

	return e, ok
}

type recordNotFoundError struct{}

func (e *recordNotFoundError) Error() string {
	return "record not found"
}

// IsRecordNotFoundError returns true if the passed in error designates that a DB record not found
func IsRecordNotFoundError(err error) bool {
	_, ok := errors.Cause(err).(*recordNotFoundError)

	return ok
}

type featureRecordNotFoundError struct{}

func (e *featureRecordNotFoundError) Error() string {
	return "feature not found"
}

// IsFeatureRecordNotFoundError returns true if the passed in error designates that a feature DB record not found
func IsFeatureRecordNotFoundError(err error) bool {
	_, ok := errors.Cause(err).(*featureRecordNotFoundError)

	return ok
}

type clusterGroupUpdateRejectedError struct {
	err error
}

func (e *clusterGroupUpdateRejectedError) Error() string {
	if e.err != nil {
		return e.err.Error()
	}

	return "update rejected"
}

// IsClusterGroupUpdateRejectedError returns true if the passed in error designates that a cluster group update is denied
func IsClusterGroupUpdateRejectedError(err error) bool {
	_, ok := errors.Cause(err).(*clusterGroupUpdateRejectedError)

	return ok
}

type unableToJoinMemberClusterError struct {
	clusterID     uint
	clusterName   string
	clusterStatus string
}

func (e *unableToJoinMemberClusterError) Error() string {
	return "unable to join member cluster with status"
}

func (e *unableToJoinMemberClusterError) Context() []interface{} {
	return []interface{}{
		"clusterName", e.clusterName,
		"clusterID", e.clusterID,
		"status", e.clusterStatus,
	}
}

// IsUnableToJoinMemberClusterError returns true if the passed in error is IsUnableToJoinMemberClusterError
func IsUnableToJoinMemberClusterError(err error) bool {
	_, ok := errors.Cause(err).(*unableToJoinMemberClusterError)

	return ok
}

type memberClusterPartOfAClusterGroupError struct {
	orgID     uint
	clusterID uint
}

func (e *memberClusterPartOfAClusterGroupError) Error() string {
	return "member cluster is already part of a cluster group"
}

func (e *memberClusterPartOfAClusterGroupError) Message() string {
	return fmt.Sprintf("%s: %d", e.Error(), e.clusterID)
}

func (e *memberClusterPartOfAClusterGroupError) Context() []interface{} {
	return []interface{}{
		"clusterID", e.clusterID,
		"organizationID", e.orgID,
	}
}

// IsMemberClusterPartOfAClusterGroupError returns true if the passed in error designates a cluster group member is already part of a cluster group
func IsMemberClusterPartOfAClusterGroupError(err error) (*memberClusterPartOfAClusterGroupError, bool) {
	e, ok := errors.Cause(err).(*memberClusterPartOfAClusterGroupError)

	return e, ok
}

type invalidClusterGroupCreateRequestError struct {
	message string
}

func (e *invalidClusterGroupCreateRequestError) Error() string {
	if e.message == "" {
		e.message = "invalid cluster create request"
	}
	return e.message
}

// IsInvalidClusterGroupCreateRequestError returns true if the passed in error designates invalid cluster group create request
func IsInvalidClusterGroupCreateRequestError(err error) bool {
	_, ok := errors.Cause(err).(*invalidClusterGroupCreateRequestError)

	return ok
}

type featureReconcileError struct {
	OriginalError error
}

func (e *featureReconcileError) Error() string {
	return "failed to reconcile feature: " + e.OriginalError.Error()
}

// IsFeatureReconcileError returns true if the passed in error designates a feature reconciliation error
func IsFeatureReconcileError(err error) bool {
	_, ok := errors.Cause(err).(*featureReconcileError)

	return ok
}
