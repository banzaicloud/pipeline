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

package common

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/banzaicloud/pipeline/config"
	pkgCommon "github.com/banzaicloud/pipeline/pkg/common"
)

// Log is a logrus.FieldLogger
var Log logrus.FieldLogger

// init initializes the fieldlogger
func init() {
	Log = config.Logger()
}

// ErrorResponseWithStatus aborts the http request with a JSON error response with the given status code and error
func ErrorResponseWithStatus(c *gin.Context, status int, err error) {

	if c.Writer.Status() != http.StatusOK {
		return
	}

	c.AbortWithStatusJSON(status, pkgCommon.ErrorResponse{
		Code:    status,
		Message: err.Error(),
		Error:   errors.Cause(err).Error(),
	})
}

// ErrorResponse aborts the http request with a JSON error response with a status code and error
func ErrorResponse(c *gin.Context, err error) {

	status := http.StatusBadRequest

	if errors.Cause(err) == gorm.ErrRecordNotFound {
		status = http.StatusNotFound
	}

	ErrorResponseWithStatus(c, status, err)
}
