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

package utils

import (
	"encoding/base64"
)

// Contains checks slice contains `s` string
func Contains(slice []string, s string) bool {
	for _, sl := range slice {
		if sl == s {
			return true
		}
	}
	return false
}

// EncodeStringToBase64 first checks if the string is encoded if yes returns it if no than encodes it.
func EncodeStringToBase64(s string) string {
	if _, err := base64.StdEncoding.DecodeString(s); err != nil {
		return base64.StdEncoding.EncodeToString([]byte(s))
	}
	return s
}
