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
	"github.com/moogar0880/problems"
)

// Problem describes an RFC-7807 problem.
type Problem struct {
	problems.DefaultProblem

	// Legacy banzai error response fields
	Code    int    `json:"code"`
	Message string `json:"message"`
	Error   string `json:"error"`
}

// NewDetailedProblem returns a problem with details and legacy banzai fields filled.
func NewDetailedProblem(status int, details string) Problem {
	detailedProblem := problems.NewDetailedProblem(status, details)

	return Problem{
		DefaultProblem: *detailedProblem,
		Code:           detailedProblem.Status,
		Message:        detailedProblem.Detail,
		Error:          detailedProblem.Detail,
	}
}
