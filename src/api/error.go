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

package api

import (
	"github.com/pkg/errors"

	pkgErrors "github.com/banzaicloud/pipeline/pkg/errors"
	"github.com/banzaicloud/pipeline/src/secret"
)

// isInvalid checks whether an error is about a resource not being found.
func isInvalid(err error) bool {
	// Check the root cause error.
	err = errors.Cause(err)

	if e, ok := err.(interface {
		IsInvalid() bool
	}); ok {
		return e.IsInvalid()
	}

	switch err {
	case secret.ErrSecretNotExists:
		return true
	case pkgErrors.ErrorNotSupportedDistributionType:
		return true
	case pkgErrors.ErrorNotSupportedKubernetesVersion:
		return true
	}

	switch err.(type) {
	case secret.MismatchError:
		return true
	}

	return false
}

// isPreconditionFailed checks whether an error is about a resource not being found.
func isPreconditionFailed(err error) bool {
	// Check the root cause error.
	err = errors.Cause(err)

	if e, ok := err.(interface {
		PreconditionFailed() bool
	}); ok {
		return e.PreconditionFailed()
	}

	return false
}

type notFoundError interface {
	NotFound() bool
}

// isNotFoundError returns true if the error implements the NotFound behavior and it returns true.
func isNotFoundError(err error) bool {
	var nfe notFoundError

	return errors.As(err, &nfe) &&
		nfe.NotFound()
}
