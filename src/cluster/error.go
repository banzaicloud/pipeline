// Copyright © 2018 Banzai Cloud
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

package cluster

import (
	"errors"
	"fmt"

	emperrors "emperror.dev/errors"
	"go.uber.org/cadence"

	eksWorkflow "github.com/banzaicloud/pipeline/internal/cluster/distribution/eks/eksprovider/workflow"
)

var ErrInvalidClusterInstance = errors.New("invalid cluster instance")

type invalidError struct {
	err error
}

func (e *invalidError) Error() string {
	return e.err.Error()
}

func (invalidError) IsInvalid() bool {
	return true
}

// decodeCloudFormationError decodes a Cloud Formation cadence.CustomError into
// a simplified object which returns a detailed error message on its Error()
// (message string) function implementation.
func decodeCloudFormationError(err error) error {
	if customError, ok := err.(*cadence.CustomError); ok && customError.Reason() == eksWorkflow.ErrReasonStackFailed {
		var details string
		_ = customError.Details(&details)

		return emperrors.NewPlain(fmt.Sprintf("%s: %s", customError.Reason(), details))
	}

	return err
}
