package utils

import (
	"bytes"
	"fmt"
	"github.com/banzaicloud/banzai-types/configuration"
	"github.com/sirupsen/logrus"
)

var log *logrus.Logger

func init() {
	log = configuration.Logger()
}

func LogInfo(tag string, args ...interface{}) {
	log.Info(appendTagAndArgs(tag, args...)...)
}

func LogInfof(tag string, format string, args ...interface{}) {
	log.Infof(prepareFormat(tag, format), args...)
}

func LogError(tag string, args ...interface{}) {
	log.Error(appendTagAndArgs(tag, args...)...)
}

func LogErrorf(tag string, format string, args ...interface{}) {
	log.Errorf(prepareFormat(tag, format), args...)
}

func LogWarn(tag string, args ...interface{}) {
	log.Warn(appendTagAndArgs(tag, args...)...)
}

func LogWarnf(tag string, format string, args ...interface{}) {
	log.Warnf(prepareFormat(tag, format), args...)
}

func LogDebug(tag string, args ...interface{}) {
	log.Debug(appendTagAndArgs(tag, args...)...)
}

func LogDebugf(tag string, format string, args ...interface{}) {
	log.Debugf(prepareFormat(tag, format), args...)
}

func LogFatal(tag string, args ...interface{}) {
	log.Fatal(appendTagAndArgs(tag, args...)...)
}

func LogFatalf(tag string, format string, args ...interface{}) {
	log.Fatalf(prepareFormat(tag, format), args...)
}

func SetLogLevel(level string) {
	l, _ := logrus.ParseLevel(level)
	log.SetLevel(l)
}

func getTag(tag string) string {
	return " ### [" + tag + "] ### "
}

func prepareFormat(tag string, format string) string {
	buffer := bytes.NewBufferString(getTag(tag))
	buffer.WriteString(format)
	return buffer.String()
}

func getMessage(args []interface{}) []interface{} {
	var res []interface{}
	for i, a := range args {
		switch a.(type) {
		case string:
			res = append(res, fmt.Sprintf("%s", a))
		case int:
			res = append(res, fmt.Sprintf("%d", a))
		default:
			res = append(res, fmt.Sprintf("%v", a))
		}
		if i+1 < len(args) {
			res = append(res, " ")
		}
	}
	return res
}

// appendTagAndArgs puts the tag and the args one after the other
func appendTagAndArgs(tag string, args ...interface{}) []interface{} {
	var argsNew []interface{}
	argsNew = append(argsNew, getTag(tag))
	argsNew = append(argsNew, getMessage(args)...)
	return argsNew
}
