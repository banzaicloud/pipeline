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

package log

import "github.com/sirupsen/logrus"

// NewLogger creates a new logrus logger instance.
// TODO: rename to NewLogrusLogger
func NewLogger(config Config) *logrus.Logger {
	logger := logrus.New()

	level, err := logrus.ParseLevel(config.Level)
	if err != nil {
		level = logrus.InfoLevel
	}

	logger.Level = level

	switch config.Format {
	case "json":
		logger.Formatter = new(logrus.JSONFormatter)

	default:
		logger.Formatter = &logrus.TextFormatter{
			DisableColors:             config.NoColor,
			EnvironmentOverrideColors: true,
			FullTimestamp:             true,
		}
	}

	return logger
}
