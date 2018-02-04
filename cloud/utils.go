package cloud

import (
	"fmt"
	"github.com/banzaicloud/pipeline/notify"
	"github.com/kris-nova/kubicorn/apis/cluster"
	"github.com/kris-nova/kubicorn/cutil/logger"
	"io/ioutil"
	"os"
	"strings"
	"time"
)

//GetConfig retrieves K8S config
func GetConfig(existing *cluster.Cluster, localDir string) (string, error) {
	if localDir == "" {
		localDir = fmt.Sprintf("./statestore/%s/", existing.Name)
	}
	localPath, err := GetKubeConfigPath(localDir)
	if err != nil {
		return "", err
	}
	conf, err := GetAmazonKubernetesConfig(existing)
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(localPath); os.IsNotExist(err) {
		empty := []byte("")
		err := ioutil.WriteFile(localPath, empty, 0755)
		if err != nil {
			return "", err
		}
	}

	f, err := os.OpenFile(localPath, os.O_WRONLY, os.ModeAppend)
	if err != nil {
		return "", err
	}
	_, err = f.Write(conf)
	if err != nil {
		return "", err
	}
	defer f.Close()
	logger.Always("Wrote kubeconfig to [%s]", localPath)
	//TODO better solution
	writeKubernetesKeys(localPath, localDir)
	return localPath, nil
}

const (
	retryAttempts     = 150
	retrySleepSeconds = 5
)

//RetryGetConfig is retrying K8S config retrieval
func RetryGetConfig(existing *cluster.Cluster, localDir string) (string, error) {
	for i := 0; i <= retryAttempts; i++ {
		path, err := GetConfig(existing, localDir)
		if err != nil {
			if strings.Contains(err.Error(), "file does not exist") || strings.Contains(err.Error(), "getsockopt: connection refused") || strings.Contains(err.Error(), "unable to authenticate") {
				logger.Debug("Waiting for Kubernetes to come up.. #%s", err.Error())
				time.Sleep(time.Duration(retrySleepSeconds) * time.Second)
				continue
			}
			return "", err
		}
		notify.SlackNotify(fmt.Sprintf("Cluster Created: %s\n IP: %s", existing.Name, existing.KubernetesAPI.Endpoint))
		return path, err
	}
	return "", fmt.Errorf("Timeout writing kubeconfig")
}

func GetKubeConfigPath(path string) (string, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := os.Mkdir(path, 0777); err != nil {
			return "", err
		}
	}
	return fmt.Sprintf("%s/config", path), nil
}
