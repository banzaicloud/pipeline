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

package zaplog

import (
	"os"
	"strings"

	"github.com/jsternberg/zap-logfmt"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// NewLogger creates a new zap logger instance.
func NewLogger(config Config) *zap.Logger {
	level := parseLevel(config.Level)

	switch config.Format {
	case "logfmt", "text":
		encoderConfig := zap.NewProductionEncoderConfig()
		encoderConfig.TimeKey = "time"
		encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

		return zap.New(zapcore.NewCore(
			zaplogfmt.NewEncoder(encoderConfig),
			os.Stdout,
			level,
		))

	default:
		if level == zapcore.DebugLevel {
			l, _ := zap.NewDevelopment()

			return l
		}

		l, _ := zap.NewProduction()

		return l
	}
}

func parseLevel(l string) zapcore.Level {
	switch strings.ToLower(l) {
	case "debug":
		return zapcore.DebugLevel

	case "info":
		return zapcore.InfoLevel

	case "warn", "warning":
		return zapcore.WarnLevel

	case "error":
		return zapcore.ErrorLevel

	case "panic":
		return zapcore.PanicLevel

	case "fatal":
		return zapcore.FatalLevel

	default:
		return zapcore.InfoLevel
	}
}
