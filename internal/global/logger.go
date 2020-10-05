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

package global

import (
	"sync"

	"github.com/banzaicloud/pipeline/internal/common"
	"github.com/banzaicloud/pipeline/internal/global/commonadapter"
	"github.com/banzaicloud/pipeline/internal/platform/log"
)

// nolint: gochecknoglobals
var logger common.Logger

// nolint: gochecknoglobals
var loggerOnce sync.Once

// nolint: gochecknoglobals
var loggerMu sync.RWMutex

// nolint: gochecknoglobals
var loggerSubscribers []func(l common.Logger)

// Logger returns an logrus logger.
func Logger() common.Logger {
	loggerMu.RLock()
	defer loggerMu.RUnlock()

	loggerOnce.Do(func() {
		logger = newLogger()
	})

	return logger
}

func newLogger() common.Logger {
	logger := commonadapter.NewLogger(log.NewLogger(log.Config{
		Level:  "info",
		Format: "text",
	}))

	return logger
}

// SubscribeLogger subscribes a handler for global error handler changes.
func SubscribeLogger(s func(l common.Logger)) {
	loggerMu.Lock()
	defer loggerMu.Unlock()

	loggerSubscribers = append(loggerSubscribers, s)
}

// SetLogger sets a global error handler.
//
// Note: setting an error handler after the application bootstrap is not safe.
func SetLogger(l common.Logger) {
	loggerMu.Lock()
	defer loggerMu.Unlock()

	logger = l

	for _, s := range loggerSubscribers {
		s(l)
	}
}
