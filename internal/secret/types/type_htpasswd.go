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

package types

import (
	"fmt"

	"emperror.dev/errors"
	"golang.org/x/crypto/bcrypt"

	"github.com/banzaicloud/pipeline/internal/secret"
)

const Htpasswd = "htpasswd"

const (
	FieldHtpasswdUsername = "username"
	FieldHtpasswdPassword = "password"
	FieldHtpasswdFile     = "htpasswd"
)

type HtpasswdType struct{}

func (HtpasswdType) Name() string {
	return Htpasswd
}

func (HtpasswdType) Definition() secret.TypeDefinition {
	return secret.TypeDefinition{
		Fields: []secret.FieldDefinition{
			{Name: FieldHtpasswdUsername, Required: true, IsSafeToDisplay: true, Opaque: true, Description: "Your username"},
			{Name: FieldHtpasswdPassword, Required: false, Opaque: true, Description: "Your password"},
			{Name: FieldHtpasswdFile, Required: false},
		},
	}
}

// Note: this will only require the username field.
func (t HtpasswdType) Validate(data map[string]string) error {
	var violations []string

	if _, ok := data[FieldHtpasswdUsername]; !ok {
		violations = append(violations, fmt.Sprintf("missing key: %s", FieldHtpasswdUsername))
	}

	if _, ok := data[FieldHtpasswdPassword]; !ok {
		violations = append(violations, fmt.Sprintf("missing key: %s", FieldHtpasswdPassword))
	}

	if len(violations) > 0 {
		// For backward compatibility reasons, return the first violation as message
		return secret.NewValidationError(violations[0], violations)
	}

	return nil
}

func (t HtpasswdType) ValidateNew(data map[string]string) (bool, error) {
	if _, ok := data[FieldHtpasswdFile]; ok {
		return true, nil
	}

	if _, ok := data[FieldHtpasswdUsername]; !ok {
		violation := fmt.Sprintf("missing key: %s", FieldHtpasswdUsername)

		return false, secret.NewValidationError(violation, []string{violation})
	}

	if password, ok := data[FieldHtpasswdPassword]; !ok || password == "" {
		return false, nil
	}

	return true, nil
}

func (t HtpasswdType) Generate(_ uint, _ string, data map[string]string, _ []string) (map[string]string, error) {
	password, err := passwordRandomString("randAlphaNum", 12)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to generate password")
	}

	data[FieldHtpasswdPassword] = password

	return data, nil
}

func (t HtpasswdType) Process(data map[string]string) (map[string]string, error) {
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(data[FieldHtpasswdPassword]), bcrypt.DefaultCost)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to generate password hash")
	}

	data[FieldHtpasswdFile] = fmt.Sprintf("%s:%s", data[FieldHtpasswdUsername], string(passwordHash))

	return data, nil
}
