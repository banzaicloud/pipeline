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
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/banzaicloud/pipeline/pkg/problems"
)

type notFoundStub struct{}

func (notFoundStub) Error() string {
	return "not found"
}

func (notFoundStub) NotFound() bool {
	return true
}

type validationStub struct{}

func (validationStub) Error() string {
	return "validation"
}

func (validationStub) Validation() bool {
	return true
}

type badRequestStub struct{}

func (badRequestStub) Error() string {
	return "bad request"
}

func (badRequestStub) BadRequest() bool {
	return true
}

type conflictStub struct{}

func (conflictStub) Error() string {
	return "conflict"
}

func (conflictStub) Conflict() bool {
	return true
}

type internalErrorStub struct{}

func (internalErrorStub) Error() string {
	return "something went wrong"
}

func TestDefaultProblemMatchers(t *testing.T) {
	tests := []struct {
		err            error
		expectedStatus int
	}{
		{
			err:            notFoundStub{},
			expectedStatus: http.StatusNotFound,
		},
		{
			err:            validationStub{},
			expectedStatus: http.StatusUnprocessableEntity,
		},
		{
			err:            badRequestStub{},
			expectedStatus: http.StatusBadRequest,
		},
		{
			err:            conflictStub{},
			expectedStatus: http.StatusConflict,
		},
		{
			err:            internalErrorStub{},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	converter := NewDefaultProblemConverter()

	for _, test := range tests {
		test := test

		t.Run(fmt.Sprintf("%d", test.expectedStatus), func(t *testing.T) {
			problem := converter.NewProblem(context.Background(), test.err).(*problems.DefaultProblem)

			if want, have := test.expectedStatus, problem.Status; want != have {
				t.Errorf("unexpected status\nexpected: %d\nactual:   %d", want, have)
			}

			if want, have := test.expectedStatus, problem.Code; want != have {
				t.Errorf("unexpected code\nexpected: %d\nactual:   %d", want, have)
			}

			if want, have := test.err.Error(), problem.Detail; want != have {
				t.Errorf("unexpected detail\nexpected: %s\nactual:   %s", want, have)
			}

			if want, have := test.err.Error(), problem.Message; want != have {
				t.Errorf("unexpected message\nexpected: %s\nactual:   %s", want, have)
			}

			if want, have := test.err.Error(), problem.Error; want != have {
				t.Errorf("unexpected error\nexpected: %s\nactual:   %s", want, have)
			}
		})
	}
}

type validationWithViolationsStub struct{}

func (validationWithViolationsStub) Error() string {
	return "validation"
}

func (validationWithViolationsStub) Validation() bool {
	return true
}

func (validationWithViolationsStub) Violations() []string {
	return []string{
		"violation",
	}
}

func TestDefaultProblemMatchers_ValidationWithViolations(t *testing.T) {
	converter := NewDefaultProblemConverter()

	err := validationWithViolationsStub{}

	problem := converter.NewProblem(context.Background(), err).(*problems.ValidationProblem)

	if want, have := http.StatusUnprocessableEntity, problem.Status; want != have {
		t.Errorf("unexpected status\nexpected: %d\nactual:   %d", want, have)
	}

	if want, have := http.StatusUnprocessableEntity, problem.Code; want != have {
		t.Errorf("unexpected code\nexpected: %d\nactual:   %d", want, have)
	}

	if want, have := err.Error(), problem.Detail; want != have {
		t.Errorf("unexpected detail\nexpected: %s\nactual:   %s", want, have)
	}

	if want, have := err.Error(), problem.Message; want != have {
		t.Errorf("unexpected message\nexpected: %s\nactual:   %s", want, have)
	}

	if want, have := err.Error(), problem.Error; want != have {
		t.Errorf("unexpected error\nexpected: %s\nactual:   %s", want, have)
	}

	assert.ElementsMatch(t, err.Violations(), problem.Violations)
}

type serviceErrorStub struct{}

func (serviceErrorStub) Error() string {
	return "service error"
}

func (serviceErrorStub) ServiceError() bool {
	return true
}

func TestDefaultProblemMatchers_Service(t *testing.T) {
	converter := NewDefaultProblemConverter()

	err := serviceErrorStub{}

	problem := converter.NewProblem(context.Background(), err).(*problems.DefaultProblem)

	if want, have := http.StatusInternalServerError, problem.Status; want != have {
		t.Errorf("unexpected status\nexpected: %d\nactual:   %d", want, have)
	}

	if want, have := http.StatusInternalServerError, problem.Code; want != have {
		t.Errorf("unexpected code\nexpected: %d\nactual:   %d", want, have)
	}

	if want, have := err.Error(), problem.Detail; want != have {
		t.Errorf("unexpected detail\nexpected: %s\nactual:   %s", want, have)
	}

	if want, have := err.Error(), problem.Message; want != have {
		t.Errorf("unexpected message\nexpected: %s\nactual:   %s", want, have)
	}

	if want, have := err.Error(), problem.Error; want != have {
		t.Errorf("unexpected error\nexpected: %s\nactual:   %s", want, have)
	}
}
