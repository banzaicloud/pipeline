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

package common

import (
	"context"
)

// Logger is the fundamental interface for all log operations.
type Logger interface {
	// Trace logs a debug event.
	Trace(msg string, fields ...map[string]interface{})

	// Debug logs a debug event.
	Debug(msg string, fields ...map[string]interface{})

	// Info logs an info event.
	Info(msg string, fields ...map[string]interface{})

	// Warn logs a warning event.
	Warn(msg string, fields ...map[string]interface{})

	// Error logs an error event.
	Error(msg string, fields ...map[string]interface{})

	// WithFields annotates a logger with key-value pairs.
	WithFields(fields map[string]interface{}) Logger

	// WithContext annotates a logger with a context.
	WithContext(ctx context.Context) Logger
}
