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

package capdriver

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/banzaicloud/pipeline/internal/app/pipeline/cap"
)

// NewHTTPHandler creates a new HTTP handler.
func NewHTTPHandler(c cap.Capabilities, errorHandler cap.ErrorHandler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := json.NewEncoder(w).Encode(c)
		if err != nil {
			errorHandler.Handle(r.Context(), err)
		}
	})
}

// RegisterHTTPHandler registers an HTTP handler in a Gin router.
func RegisterHTTPHandler(c cap.Capabilities, errorHandler cap.ErrorHandler, r gin.IRoutes) {
	r.GET("/capabilities", gin.WrapH(NewHTTPHandler(c, errorHandler)))
}
