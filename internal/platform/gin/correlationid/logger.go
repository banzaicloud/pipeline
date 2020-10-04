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

package correlationid

import (
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"github.com/banzaicloud/pipeline/internal/common"
)

const correlationIdField = "correlation-id"

// Logger returns a new logger instance with a correlation ID in it.
func Logger(logger common.Logger, ctx *gin.Context) common.Logger {
	cid := ctx.GetString(ContextKey)

	if cid == "" {
		return logger
	}

	return logger.WithFields(map[string]interface{}{correlationIdField: cid})
}

// LogrusLogger returns a new logger instance with a correlation ID in it.
func LogrusLogger(logger logrus.FieldLogger, ctx *gin.Context) logrus.FieldLogger {
	cid := ctx.GetString(ContextKey)

	if cid == "" {
		return logger
	}

	return logger.WithField(correlationIdField, cid)
}
