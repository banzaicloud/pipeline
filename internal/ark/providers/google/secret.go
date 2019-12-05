// Copyright © 2018 Banzai Cloud
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

package google

import (
	"encoding/json"

	"github.com/banzaicloud/pipeline/src/secret"
)

// GetSecret gets formatted secret for ARK
func GetSecret(secret *secret.SecretItemResponse) (string, error) {

	values, err := json.Marshal(secret.Values)
	if err != nil {
		return "", err
	}

	return string(values), nil
}
