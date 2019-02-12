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
			os.Stderr,
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
