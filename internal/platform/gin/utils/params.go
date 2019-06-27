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

package ginutils

import (
	"github.com/gin-gonic/gin"
)

// ParamsToMap converts a Gin parameter list to a simple string map.
func ParamsToMap(params gin.Params) map[string]string {
	paramsMap := make(map[string]string, len(params))

	for _, param := range params {
		paramsMap[param.Key] = param.Value
	}

	return paramsMap
}
