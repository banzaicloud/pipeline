// Copyright © 2017 The Kubicorn Authors
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

package fileresource

import (
	"net/url"
	"strings"
)

// ReadFromResource reads a file from different sources
// at the moment suppoted resources are http, http, local file system(POSIX)
func ReadFromResource(r string) (string, error) {
	switch {
	case strings.HasPrefix(strings.ToLower(r), "http://") || strings.HasPrefix(strings.ToLower(r), "https://"):
		url, err := url.ParseRequestURI(r)
		if err != nil {
			return "", err
		}
		return readFromHTTP(url)

	default:
		return readFromFS(r)
	}
}
