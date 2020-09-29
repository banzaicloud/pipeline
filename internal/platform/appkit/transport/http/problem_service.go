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

// ServiceProblemMatcher is a problem matcher for service errors.
// If the returned error matches the following interface, a special service problem is returned by NewProblem:
// 	type ServiceError interface {
// 		ServiceError() bool
// 	}
type serviceProblemMatcher struct{}

// NewServiceProblemMatcher returns a problem matcher for service errors.
// If the returned error matches the following interface, a special service problem is returned by NewProblem:
// 	type ServiceError interface {
// 		ServiceError() bool
// 	}
func NewServiceProblemMatcher() appkithttp.ProblemMatcher {
	return serviceProblemMatcher{}
}

func (matcher serviceProblemMatcher) MatchError(err error) bool {
	return appkiterrors.IsServiceError(err)
}

func (matcher serviceProblemMatcher) NewProblem(_ context.Context, err error) interface{} {
	if appkiterrors.IsServiceError(err) {
		var serviceError interface {
			Error() string
			ServiceError() bool
		}
		errors.As(err, &serviceError)

		return problems.NewDetailedProblem(http.StatusInternalServerError, serviceError.Error())
	}

	return problems.NewDetailedProblem(http.StatusUnprocessableEntity, err.Error())
}
