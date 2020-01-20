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

	"logur.dev/logur"

	"github.com/banzaicloud/pipeline/internal/common"
)

// Logger wraps a logur logger and exposes it under a custom interface.
type Logger struct {
	logur.LoggerFacade

	extractor ContextExtractor
}

// ContextExtractor extracts log fields from a context.
type ContextExtractor func(ctx context.Context) map[string]interface{}

// NewLogger returns a new Logger instance.
func NewLogger(logger logur.LoggerFacade) *Logger {
	return &Logger{
		LoggerFacade: logger,
	}
}

// NewContextAwareLogger returns a new Logger instance that can extract information from a context.
func NewContextAwareLogger(logger logur.LoggerFacade, extractor ContextExtractor) *Logger {
	return &Logger{
		LoggerFacade: logur.WithContextExtractor(logger, logur.ContextExtractor(extractor)),
		extractor:    extractor,
	}
}

// WithFields annotates a logger with key-value pairs.
func (l *Logger) WithFields(fields map[string]interface{}) common.Logger {
	return &Logger{
		LoggerFacade: logur.WithFields(l.LoggerFacade, fields),
		extractor:    l.extractor,
	}
}

// WithContext annotates a logger with a context.
func (l *Logger) WithContext(ctx context.Context) common.Logger {
	if l.extractor == nil {
		return l
	}

	return l.WithFields(l.extractor(ctx))
}
