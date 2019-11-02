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

package global

import (
	"sync"

	runtime "github.com/banzaicloud/logrus-runtime-formatter"
	"github.com/sirupsen/logrus"

	"github.com/banzaicloud/pipeline/internal/platform/log"
)

// nolint: gochecknoglobals
var logrusLogger *logrus.Logger

// nolint: gochecknoglobals
var logrusLoggerOnce sync.Once

// nolint: gochecknoglobals
var logrusLoggerMu sync.RWMutex

// nolint: gochecknoglobals
var logrusLoggerSubscribers []func(l *logrus.Logger)

// LogrusLogger returns an logrus logger.
func LogrusLogger() *logrus.Logger {
	logrusLoggerMu.RLock()
	defer logrusLoggerMu.RUnlock()

	logrusLoggerOnce.Do(func() {
		logrusLogger = newLogrusLogger()
	})

	return logrusLogger
}

func newLogrusLogger() *logrus.Logger {
	logger := log.NewLogrusLogger(log.Config{
		Level:  "info",
		Format: "text",
	})

	logger.Formatter = &runtime.Formatter{ChildFormatter: logger.Formatter}

	return logger
}

// SubscribeLogrusLogger subscribes a handler for global error handler changes.
func SubscribeLogrusLogger(s func(l *logrus.Logger)) {
	logrusLoggerMu.Lock()
	defer logrusLoggerMu.Unlock()

	logrusLoggerSubscribers = append(logrusLoggerSubscribers, s)
}

// SetLogrusLogger sets a global error handler.
//
// Note: setting an error handler after the application bootstrap is not safe.
func SetLogrusLogger(l *logrus.Logger) {
	logrusLoggerMu.Lock()
	defer logrusLoggerMu.Unlock()

	logrusLogger = l

	for _, s := range logrusLoggerSubscribers {
		s(l)
	}
}
