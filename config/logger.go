package config

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

var logger *logrus.Logger

//Logger is a configured Logrus logger
func Logger() *logrus.Logger {
	if logger == nil {
		logger = logrus.New()
		switch viper.GetString("logging.loglevel") {
		case "debug":
			logger.Level = logrus.DebugLevel
		case "info":
			logger.Level = logrus.InfoLevel
		case "warn":
			logger.Level = logrus.WarnLevel
		case "error":
			logger.Level = logrus.ErrorLevel
		case "fatal":
			logger.Level = logrus.FatalLevel
		default:
			//logrus.WithField("dev.loglevel", viper.GetString("dev.loglevel")).Warning("Invalid log level. Defaulting to info.")
			logger.Level = logrus.InfoLevel
		}

		switch viper.GetString("log.logformat") {
		case "text":
			logger.Formatter = new(logrus.TextFormatter)
		case "json":
			logger.Formatter = new(logrus.JSONFormatter)
		default:
			logger.Formatter = new(logrus.TextFormatter)
		}

	}
	return logger
}
