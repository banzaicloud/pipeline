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

package commonadapter

import (
	"context"
	"testing"

	"logur.dev/logur"
	"logur.dev/logur/conformance"
	"logur.dev/logur/logtesting"
)

func TestLogger(t *testing.T) {
	t.Run("WithFields", testLoggerWithFields)
	t.Run("WithContext", testLoggerWithContext)

	suite := conformance.TestSuite{
		LoggerFactory: func(level logur.Level) (logur.Logger, conformance.TestLogger) {
			testLogger := &logur.TestLoggerFacade{}

			return NewLogger(testLogger), testLogger
		},
	}
	t.Run("Conformance", suite.Run)
}

func TestContextAwareLogger(t *testing.T) {
	t.Run("WithContext", testContextAwareLoggerWithContext)

	suite := conformance.TestSuite{
		LoggerFactory: func(level logur.Level) (logur.Logger, conformance.TestLogger) {
			testLogger := &logur.TestLoggerFacade{}

			return NewContextAwareLogger(
				testLogger,
				func(ctx context.Context) map[string]interface{} {
					return nil
				},
			), testLogger
		},
	}
	t.Run("Conformance", suite.Run)
}

func testLoggerWithFields(t *testing.T) {
	testLogger := &logur.TestLoggerFacade{}

	fields := map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
	}

	logger := NewLogger(testLogger).WithFields(fields)

	logger.Debug("message", nil)

	event := logur.LogEvent{
		Level:  logur.Debug,
		Line:   "message",
		Fields: fields,
	}

	logtesting.AssertLogEventsEqual(t, event, *(testLogger.LastEvent()))
}

func testLoggerWithContext(t *testing.T) {
	testLogger := &logur.TestLoggerFacade{}

	logger := NewLogger(testLogger).WithContext(context.Background())

	logger.Debug("message", nil)

	event := logur.LogEvent{
		Level: logur.Debug,
		Line:  "message",
	}

	logtesting.AssertLogEventsEqual(t, event, *(testLogger.LastEvent()))
}

func testContextAwareLoggerWithContext(t *testing.T) {
	testLogger := &logur.TestLoggerFacade{}

	logger := NewContextAwareLogger(
		testLogger,
		func(_ context.Context) map[string]interface{} {
			return map[string]interface{}{
				"key1": "value1",
				"key2": "value2",
			}
		},
	).WithContext(context.Background())

	logger.Debug("message", nil)

	event := logur.LogEvent{
		Level: logur.Debug,
		Line:  "message",
		Fields: map[string]interface{}{
			"key1": "value1",
			"key2": "value2",
		},
	}

	logtesting.AssertLogEventsEqual(t, event, *(testLogger.LastEvent()))
}
