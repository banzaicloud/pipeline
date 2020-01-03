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

package problems

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewDetailedProblem(t *testing.T) {
	const status = http.StatusBadRequest
	const detail = "invalid request"

	problem := NewDetailedProblem(status, detail)

	assert.Equal(t, status, problem.Code)
	assert.Equal(t, detail, problem.Message)
	assert.Equal(t, detail, problem.Error, detail)
}

func TestNewStatusProblem(t *testing.T) {
	const status = http.StatusBadRequest

	problem := NewStatusProblem(status)

	assert.Equal(t, status, problem.Code)
	assert.Equal(t, http.StatusText(status), problem.Message)
}

func TestNewValidationProblem(t *testing.T) {
	const detail = "invalid request"
	violations := []string{"error"}

	problem := NewValidationProblem(detail, violations)

	assert.Equal(t, detail, problem.Detail)
	assert.Equal(t, http.StatusUnprocessableEntity, problem.Status)
	assert.Equal(t, violations, problem.Violations)
	assert.Equal(t, http.StatusUnprocessableEntity, problem.Code)
	assert.Equal(t, detail, problem.Message)
	assert.Equal(t, detail, problem.Error, detail)
}
