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

const (
	// ProblemMediaType is the default media type for a DefaultProblem response
	ProblemMediaType = problems.ProblemMediaType

	// ProblemMediaTypeXML is the XML variant on the DefaultProblem Media type
	ProblemMediaTypeXML = problems.ProblemMediaTypeXML

	// DefaultURL is the default url to use for problem types
	DefaultURL = problems.DefaultURL
)

// Problem is the interface describing an HTTP API problem. These "problem
// details" are designed to encompass a way to carry machine- readable details
// of errors in a HTTP response to avoid the need to define new error response
// formats for HTTP APIs.
type Problem = problems.Problem

// StatusProblem is the interface describing a problem with an associated
// Status code.
type StatusProblem = problems.StatusProblem

// ValidateProblem ensures that the provided Problem implementation meets the
// Problem description requirements. Which means that the Type is a valid uri,
// and that the Title be a non-empty string. Should the provided Problem be in
// violation of either of these requirements, an error is returned.
func ValidateProblem(p Problem) error {
	return problems.ValidateProblem(p)
}

// DefaultProblem describes an RFC-7807 problem.
type DefaultProblem struct {
	*problems.DefaultProblem

	// Legacy banzai error response fields
	Code    int    `json:"code"`
	Message string `json:"message"`
	Error   string `json:"error"`
}

// NewDetailedProblem returns a problem with details and legacy banzai fields filled.
func NewDetailedProblem(status int, details string) *DefaultProblem {
	detailedProblem := problems.NewDetailedProblem(status, details)

	return &DefaultProblem{
		DefaultProblem: detailedProblem,
		Code:           detailedProblem.Status,
		Message:        detailedProblem.Detail,
		Error:          detailedProblem.Detail,
	}
}
