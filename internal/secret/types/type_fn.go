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
	"github.com/banzaicloud/pipeline/internal/secret"
)

const Fn = "fn"

const (
	FieldFnMasterToken = "master_token"
)

type FnType struct{}

func (FnType) Name() string {
	return Fn
}

func (FnType) Definition() secret.TypeDefinition {
	return secret.TypeDefinition{
		Fields: []secret.FieldDefinition{
			{Name: FieldFnMasterToken, Required: true},
		},
	}
}

func (t FnType) Validate(data map[string]string) error {
	return validateDefinition(data, t.Definition())
}
