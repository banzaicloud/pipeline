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

package auditlogdriver

import (
	"github.com/banzaicloud/pipeline/internal/platform/gin/auditlog"
)

// LogDriverConfig configures a standard output audit log driver.
type LogDriverConfig struct {
	Verbosity int
}

// Logger is the fundamental interface for all log operations.
type Logger interface {
	// Info logs an info event.
	Info(msg string, fields ...map[string]interface{})
}

// NewLogDriver returns a standard output audit log driver.
func NewLogDriver(config LogDriverConfig, logger Logger) auditlog.Driver {
	return logDriver{
		config: config,
		logger: logger,
	}
}

type logDriver struct {
	config LogDriverConfig
	logger Logger
}

func (d logDriver) Store(entry auditlog.Entry) error {
	if d.config.Verbosity == 0 {
		return nil
	}

	data := make(map[string]interface{})

	if d.config.Verbosity >= 1 {
		data["time"] = entry.Time
		data["correlationID"] = entry.CorrelationID
		data["userID"] = entry.UserID
	}

	if d.config.Verbosity >= 2 {
		data["http.method"] = entry.HTTP.Method
		data["http.path"] = entry.HTTP.Path
		data["http.clientIP"] = entry.HTTP.ClientIP
	}

	if d.config.Verbosity >= 3 {
		data["http.userAgent"] = entry.HTTP.UserAgent
		data["http.statusCode"] = entry.HTTP.StatusCode
	}

	if d.config.Verbosity >= 4 {
		data["http.responseTime"] = entry.HTTP.ResponseTime
		data["http.responseSize"] = entry.HTTP.ResponseSize
		data["http.requestBody"] = entry.HTTP.RequestBody

		if len(entry.HTTP.Errors) > 0 {
			data["http.errors"] = entry.HTTP.Errors
		}
	}

	d.logger.Info("audit log event", data)

	return nil
}
