package conf

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

var logger *logrus.Logger

func Logger() *logrus.Logger {
	if logger == nil {
		logger = logrus.New()
		switch viper.GetString("dev.loglevel") {
		case "debug":
			logrus.SetLevel(logrus.DebugLevel)
		case "info":
			logrus.SetLevel(logrus.InfoLevel)
		case "warn":
			logrus.SetLevel(logrus.WarnLevel)
		case "error":
			logrus.SetLevel(logrus.ErrorLevel)
		case "fatal":
			logrus.SetLevel(logrus.FatalLevel)
		default:
			logrus.WithField("dev.loglevel", viper.GetString("dev.loglevel")).Warning("Invalid log level. Defaulting to info.")
			logrus.SetLevel(logrus.InfoLevel)
		}

		switch viper.GetString("dev.logformat") {
		case "text":
			logrus.SetFormatter(new(logrus.TextFormatter))
		case "json":
			logrus.SetFormatter(new(logrus.JSONFormatter))
		default:
			logrus.WithField("dev.logformat", viper.GetString("dev.logformat")).Warning("Invalid log format. Defaulting to text.")
			logrus.SetFormatter(new(logrus.TextFormatter))
		}

		logger.SetLevel(logrus.DebugLevel)
	}
	return logger
}
