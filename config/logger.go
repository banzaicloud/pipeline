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

package config

import (
	"sync"

	runtime "github.com/banzaicloud/logrus-runtime-formatter"
	"github.com/banzaicloud/pipeline/internal/platform/log"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// nolint: gochecknoglobals
var logger *logrus.Logger

// nolint: gochecknoglobals
var loggerOnce sync.Once

// Logger is a configured Logrus logger.
func Logger() *logrus.Logger {
	loggerOnce.Do(func() { logger = newLogger() })

	return logger
}

func newLogger() *logrus.Logger {
	logger := log.NewLogrusLogger(log.Config{
		Level:  viper.GetString("logging.loglevel"),
		Format: viper.GetString("logging.logformat"),
	})

	logger.Formatter = &runtime.Formatter{ChildFormatter: logger.Formatter}

	return logger
}
