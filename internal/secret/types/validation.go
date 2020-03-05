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

	"github.com/banzaicloud/pipeline/internal/secret"
)

func validateDefinition(secretData map[string]string, def secret.TypeDefinition) error {
	var violations []string

	for _, field := range def.Fields {
		if _, ok := secretData[field.Name]; field.Required && !ok {
			violations = append(violations, fmt.Sprintf("missing key: %s", field.Name))
		}
	}

	if len(violations) > 0 {
		// For backward compatibility reasons, return the first violation as message
		return secret.NewValidationError(violations[0], violations)
	}

	return nil
}
