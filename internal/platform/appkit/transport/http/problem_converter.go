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
	"net/http"

	appkithttp "github.com/sagikazarmark/appkit/transport/http"

	"github.com/banzaicloud/pipeline/pkg/problems"
)

type defaultProblemConverter struct{}

func (defaultProblemConverter) NewProblem(_ context.Context, err error) interface{} {
	return problems.NewDetailedProblem(http.StatusInternalServerError, err.Error())
}

func (defaultProblemConverter) NewStatusProblem(_ context.Context, status int, err error) appkithttp.StatusProblem {
	return problems.NewDetailedProblem(status, err.Error())
}

func NewProblemConverter(opts ...appkithttp.ProblemConverterOption) appkithttp.ProblemConverter {
	opts = append(
		[]appkithttp.ProblemConverterOption{
			appkithttp.WithProblemConverter(defaultProblemConverter{}),
			appkithttp.WithStatusProblemConverter(defaultProblemConverter{}),
		},
		opts...,
	)
	return appkithttp.NewProblemConverter(opts...)
}

func NewDefaultProblemConverter(opts ...appkithttp.ProblemConverterOption) appkithttp.ProblemConverter {
	opts = append(opts, appkithttp.WithProblemMatchers(DefaultProblemMatchers...))

	return NewProblemConverter(opts...)
}
