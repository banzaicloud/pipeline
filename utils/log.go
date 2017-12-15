package utils

import (
	"github.com/sirupsen/logrus"
	"bytes"
	"fmt"
)

const (
	TagInit                  = "Init"
	TagCreateCluster         = "CreateCluster"
	TagValidateCreateCluster = "ValidateCreateCluster"
	TagValidateUpdateCluster = "ValidateUpdateCluster"
	TagGetClusterStatus      = "GetClusterStatus"
	TagUpdateCluster         = "UpdateCluster"
	TagGetCluster            = "GetCluster"
	TagDeleteCluster         = "DeleteCluster"
	TagDeleteDeployment      = "DeleteDeployment"
	TagCreateDeployment      = "CreateDeployment"
	TagListDeployments       = "ListDeployments"
	TagUpdatePrometheus      = "UpdatePrometheus"
	TagListClusters          = "ListClusters"
	TagGetClusterInfo        = "GetClusterInfo"
	TagFetchClusterConfig    = "FetchClusterConfig"
	TagGetTillerStatus       = "GetTillerStatus"
	TagFetchDeploymentStatus = "FetchDeploymentStatus"
	TagStatus                = "Status"
	TagSlack                 = "Slack"
	TagAuth                  = "Auth"
)

func LogInfo(log *logrus.Logger, tag string, args ... interface{}) {
	log.Info(getTag(tag), getMessage(args))
}

func LogError(log *logrus.Logger, tag string, args ... interface{}) {
	log.Error(getTag(tag), getMessage(args))
}

func LogWarn(log *logrus.Logger, tag string, args ... interface{}) {
	log.Warn(getTag(tag), getMessage(args))
}

func LogDebug(log *logrus.Logger, tag string, args ... interface{}) {
	log.Debug(getTag(tag), getMessage(args))
}

func getTag(tag string) string {
	return " ### [" + tag + "] ### "
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
