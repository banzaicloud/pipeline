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

package helm3

import (
	"context"
	"fmt"
	"net/url"

	"emperror.dev/errors"

	"github.com/banzaicloud/pipeline/internal/cluster"
)

type RepoValidator interface {
	Validate(ctx context.Context, repository Repository) error
}

func NewHelmRepoValidator() RepoValidator {
	// todo refine validator implementation: are more validators needed, add external / internal validators
	return repoValidator{}
}

type RepoValidators []RepoValidator

func (r RepoValidators) Validate(ctx context.Context, repository Repository) error {
	var violations []string

	for _, validator := range r {
		err := validator.Validate(ctx, repository)
		if err != nil {
			violations = append(violations, unwrapViolations(err)...)
		}
	}

	if len(violations) > 0 {
		return errors.WithStack(
			cluster.NewValidationError("invalid helm repository", violations))
	}

	return nil
}

type repoValidator struct {
}

func (r repoValidator) Validate(ctx context.Context, repository Repository) error {
	var violations []string

	if repository.Name == "" {
		violations = append(violations, "name cannot be empty")
	}

	// name matches a regex

	_, err := url.Parse(repository.URL)
	if err != nil {
		violations = append(violations, fmt.Sprintf("invalid repository URL: %s", err.Error()))
	}

	if repository.PasswordSecretID == "" && repository.TlsSecretID == "" {
		violations = append(violations, fmt.Sprintf("a secret ID must be provided"))
	}

	if len(violations) > 0 {
		return errors.WithStack(NewValidationError("invalid chart repository", violations))
	}

	return nil
}

// unwrapViolations is a helper func to unwrap violations from a validation error
func unwrapViolations(err error) []string {
	var verr interface {
		Violations() []string
	}

	if errors.As(err, &verr) {
		return verr.Violations()
	}

	return []string{err.Error()}
}
