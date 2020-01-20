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

package http

import (
	"context"
	"errors"
	"net/http"

	appkiterrors "github.com/sagikazarmark/appkit/errors"
	appkithttp "github.com/sagikazarmark/appkit/transport/http"

	"github.com/banzaicloud/pipeline/pkg/problems"
)

// NewValidationWithViolationsProblemMatcher returns a problem matcher for validation errors that contain violations.
// If the returned error matches the following interface, a special validation problem is returned by NewProblem:
// 	type violationError interface {
// 		Violations() map[string][]string
// 	}
func NewValidationWithViolationsProblemMatcher() appkithttp.ProblemMatcher {
	return validationWithViolationsProblemMatcher{}
}

type violationError interface {
	Violations() []string
}

type validationWithViolationsProblemMatcher struct{}

func (v validationWithViolationsProblemMatcher) MatchError(err error) bool {
	var verr violationError

	return appkiterrors.IsValidationError(err) && errors.As(err, &verr)
}

func (v validationWithViolationsProblemMatcher) NewProblem(_ context.Context, err error) interface{} {
	var verr violationError

	if errors.As(err, &verr) {
		return problems.NewValidationProblem(err.Error(), verr.Violations())
	}

	return problems.NewDetailedProblem(http.StatusUnprocessableEntity, err.Error())
}
