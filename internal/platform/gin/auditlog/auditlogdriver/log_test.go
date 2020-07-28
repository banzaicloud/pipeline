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
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"logur.dev/logur"
	"logur.dev/logur/logtesting"

	"github.com/banzaicloud/pipeline/internal/platform/gin/auditlog"
)

func TestLogDriver(t *testing.T) {
	entry := auditlog.Entry{
		Time:          time.Date(1984, time.April, 4, 0, 0, 0, 0, time.UTC),
		CorrelationID: "cid",
		UserID:        1,
		HTTP: auditlog.HTTPEntry{
			ClientIP:     "127.0.0.1",
			UserAgent:    "go-test",
			Method:       "POST",
			Path:         "/",
			RequestBody:  "",
			StatusCode:   200,
			ResponseTime: 1000,
			ResponseSize: 10,
			Errors:       nil,
		},
	}

	t.Run("Verbosity", func(t *testing.T) {
		config := LogDriverConfig{Verbosity: 4}
		logger := &logur.TestLogger{}

		driver := NewLogDriver(config, logger)

		err := driver.Store(entry)
		require.NoError(t, err)

		event := logur.LogEvent{
			Line:  "audit log event",
			Level: logur.Info,
			Fields: map[string]interface{}{
				"timestamp":         entry.Time,
				"correlationID":     entry.CorrelationID,
				"userID":            entry.UserID,
				"http.method":       entry.HTTP.Method,
				"http.path":         entry.HTTP.Path,
				"http.clientIP":     entry.HTTP.ClientIP,
				"http.userAgent":    entry.HTTP.UserAgent,
				"http.statusCode":   entry.HTTP.StatusCode,
				"http.responseTime": entry.HTTP.ResponseTime,
				"http.responseSize": entry.HTTP.ResponseSize,
				"http.requestBody":  "",
			},
		}

		logtesting.AssertLogEventsEqual(t, event, *(logger.LastEvent()))
	})

	t.Run("FieldList", func(t *testing.T) {
		config := LogDriverConfig{Fields: []string{"userID", "http.method", "http.path"}}
		logger := &logur.TestLogger{}

		driver := NewLogDriver(config, logger)

		err := driver.Store(entry)
		require.NoError(t, err)

		event := logur.LogEvent{
			Line:  "audit log event",
			Level: logur.Info,
			Fields: map[string]interface{}{
				"userID":      entry.UserID,
				"http.method": entry.HTTP.Method,
				"http.path":   entry.HTTP.Path,
			},
		}

		logtesting.AssertLogEventsEqual(t, event, *(logger.LastEvent()))
	})
}
