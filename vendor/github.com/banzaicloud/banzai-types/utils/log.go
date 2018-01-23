package utils

import (
	"github.com/sirupsen/logrus"
	"bytes"
	"fmt"
	"github.com/banzaicloud/banzai-types/configuration"
)

var log *logrus.Logger

func init() {
	log = configuration.Logger()
}

func LogInfo(tag string, args ... interface{}) {
	log.Info(getTag(tag), getMessage(args))
}

func LogInfof(tag string, format string, args ... interface{}) {
	log.Infof(prepareFormat(tag, format), getMessage(args))
}

func LogError(tag string, args ... interface{}) {
	log.Error(getTag(tag), getMessage(args))
}

func LogErrorf(tag string, format string, args ... interface{}) {
	log.Errorf(prepareFormat(tag, format), getMessage(args))
}

func LogWarn(tag string, args ... interface{}) {
	log.Warn(getTag(tag), getMessage(args))
}

func LogWarnf(tag string, format string, args ... interface{}) {
	log.Warnf(prepareFormat(tag, format), getMessage(args))
}

func LogDebug(tag string, args ... interface{}) {
	log.Debug(getTag(tag), getMessage(args))
}

func LogDebugf(tag string, format string, args ... interface{}) {
	log.Debugf(prepareFormat(tag, format), getMessage(args))
}

func LogFatal(tag string, args ... interface{}) {
	log.Fatal(getTag(tag), getMessage(args))
}

func LogFatalf(tag string, format string, args ... interface{}) {
	log.Fatalf(prepareFormat(tag, format), getMessage(args))
}

func SetLogLevel(level string) {
	l, _ := logrus.ParseLevel(level)
	log.SetLevel(l)
}

func getTag(tag string) string {
	return " ### [" + tag + "] ### "
}

func prepareFormat(tag string, format string) string {
	buffer:= bytes.NewBufferString(getTag(tag))
	buffer.WriteString(format)
	return buffer.String()
}

func getMessage(args []interface{}) string {
	var buffer bytes.Buffer
	for i, a := range args {
		switch a.(type) {
		case string:
			buffer.WriteString(fmt.Sprintf("%s", a))
			break
		case int:
			buffer.WriteString(fmt.Sprintf("%d", a))
			break
		default:
			buffer.WriteString(fmt.Sprintf("%v", a))
			break
		}
		if i+1 < len(args) {
			buffer.WriteString(" ")
		}
	}
	return buffer.String()
}
