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
	Fields    []string
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
	data := make(map[string]interface{})

	if len(d.config.Fields) > 0 {
		appendFields(data, entry, d.config.Fields)
	} else {
		if d.config.Verbosity == 0 {
			return nil
		}

		if d.config.Verbosity >= 1 {
			appendFields(data, entry, []string{"timestamp", "correlationID", "userID"})
		}

		if d.config.Verbosity >= 2 {
			appendFields(data, entry, []string{"http.method", "http.path", "http.clientIP"})
		}

		if d.config.Verbosity >= 3 {
			appendFields(data, entry, []string{"http.userAgent", "http.statusCode", "http.responseTime", "http.responseSize"})
		}

		if d.config.Verbosity >= 4 {
			appendFields(data, entry, []string{"http.requestBody", "http.errors"})
		}
	}

	d.logger.Info("audit log event", data)

	return nil
}

func appendFields(data map[string]interface{}, entry auditlog.Entry, fields []string) {
	for _, field := range fields {
		switch field {
		case "timestamp":
			data[field] = entry.Time
		case "correlationID":
			data[field] = entry.CorrelationID
		case "userID":
			data[field] = entry.UserID
		case "http.method":
			data[field] = entry.HTTP.Method
		case "http.path":
			data[field] = entry.HTTP.Path
		case "http.clientIP":
			data[field] = entry.HTTP.ClientIP
		case "http.userAgent":
			data[field] = entry.HTTP.UserAgent
		case "http.statusCode":
			data[field] = entry.HTTP.StatusCode
		case "http.responseTime":
			data[field] = entry.HTTP.ResponseTime
		case "http.responseSize":
			data[field] = entry.HTTP.ResponseSize
		case "http.requestBody":
			data[field] = entry.HTTP.RequestBody
		case "http.errors":
			if len(entry.HTTP.Errors) > 0 {
				data[field] = entry.HTTP.Errors
			}
		}
	}
}
