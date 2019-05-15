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

package deployment

import (
	"fmt"

	"github.com/pkg/errors"
)

type memberClusterNotFoundError struct {
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
	}
}

// IsMemberClusterNotFoundError returns true if the passed in error designates a cluster group member is not found
func IsMemberClusterNotFoundError(err error) (*memberClusterNotFoundError, bool) {
	e, ok := errors.Cause(err).(*memberClusterNotFoundError)

	return e, ok
}

type deploymentNotFoundError struct {
	clusterGroupID uint
	deploymentName string
}

func (e *deploymentNotFoundError) Error() string {
	return "deployment not found"
}

func (e *deploymentNotFoundError) Context() []interface{} {
	return []interface{}{
		"clusterGroupID", e.clusterGroupID,
		"deploymentName", e.deploymentName,
	}
}

// IsDeploymentNotFoundError returns true if the passed in error designates a deployment not found error
func IsDeploymentNotFoundError(err error) bool {
	_, ok := errors.Cause(err).(*deploymentNotFoundError)

	return ok
}

type deploymentAlreadyExistsError struct {
	clusterGroupID uint
	releaseName    string
}

func (e *deploymentAlreadyExistsError) Error() string {
	return "deployment already exists with this release name"
}

func (e *deploymentAlreadyExistsError) Context() []interface{} {
	return []interface{}{
		"clusterGroupID", e.clusterGroupID,
		"releaseName", e.releaseName,
	}
}

// IsDeploymentAlreadyExistsError returns true if the passed in error designates a deployment already exists error
func IsDeploymentAlreadyExistsError(err error) bool {
	_, ok := errors.Cause(err).(*deploymentAlreadyExistsError)

	return ok
}
