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

	"github.com/goph/logur"

	"github.com/banzaicloud/pipeline/internal/common"
)

// Logger wraps a logur logger and exposes it under a custom interface.
type Logger struct {
	logger       logur.Logger
	ctxExtractor ContextExtractor
}

// ContextExtractor extracts log fields from a context.
type ContextExtractor interface {
	// Extract extracts log fields from a context.
	Extract(ctx context.Context) map[string]interface{}
}

// NewLogger returns a new Logger instance.
func NewLogger(logger logur.Logger) *Logger {
	return &Logger{
		logger: logger,
	}
}

// NewContextAwareLogger returns a new Logger instance that can extract information from a context.
func NewContextAwareLogger(logger logur.Logger, ctxExtractor ContextExtractor) *Logger {
	return &Logger{
		logger:       logger,
		ctxExtractor: ctxExtractor,
	}
}

// Trace logs a trace event.
func (l *Logger) Trace(msg string, fields ...map[string]interface{}) {
	l.logger.Trace(msg, fields...)
}

// Debug logs a debug event.
func (l *Logger) Debug(msg string, fields ...map[string]interface{}) {
	l.logger.Debug(msg, fields...)
}

// Info logs an info event.
func (l *Logger) Info(msg string, fields ...map[string]interface{}) {
	l.logger.Info(msg, fields...)
}

// Warn logs a warning event.
func (l *Logger) Warn(msg string, fields ...map[string]interface{}) {
	l.logger.Warn(msg, fields...)
}

// Error logs an error event.
func (l *Logger) Error(msg string, fields ...map[string]interface{}) {
	l.logger.Error(msg, fields...)
}

// WithFields annotates a logger with key-value pairs.
func (l *Logger) WithFields(fields map[string]interface{}) common.Logger {
	return &Logger{
		logger:       logur.WithFields(l.logger, fields),
		ctxExtractor: l.ctxExtractor,
	}
}

// WithContext annotates a logger with a context.
func (l *Logger) WithContext(ctx context.Context) common.Logger {
	if l.ctxExtractor == nil {
		return l
	}

	return l.WithFields(l.ctxExtractor.Extract(ctx))
}

// NewNoopLogger returns a logger that discards all received log events.
func NewNoopLogger() *Logger {
	return NewLogger(logur.NewNoopLogger())
}
