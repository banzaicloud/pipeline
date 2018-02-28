package cluster

import (
	"fmt"
	htypes "github.com/banzaicloud/banzai-types/components/helm"
	"github.com/banzaicloud/banzai-types/constants"
	"github.com/banzaicloud/pipeline/helm"
	"github.com/banzaicloud/pipeline/utils"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"k8s.io/client-go/rest"
	"time"
)

// Calls posthook functions with created cluster
func RunPostHooks(functionList []func(cluster CommonCluster), createdCluster CommonCluster) {
	for _, i := range functionList {
		i(createdCluster)
	}
}

// Basic version of persisting keys TODO check if we need this from API or anywhere else
func PersistKubernetesKeys(cluster CommonCluster) {
	log = logger.WithFields(logrus.Fields{"action": "PersistKubernetesKeys"})
	configPath := fmt.Sprintf("%s/%s", viper.GetString("statestore.path"), cluster.GetName())
	log.Infof("Statestore path is: %s", configPath)
	var config *rest.Config
	retryCount := viper.GetInt("cloud.configRetryCount")
	retrySleepTime := viper.GetInt("cloud.configRetrySleep")
	var err error
	var kubeConfig *[]byte
	for i := 0; i < retryCount; i++ {
		kubeConfig, err = cluster.GetK8sConfig()
		if err != nil {
			log.Infof("Error getting kubernetes config attempt %s/%s: %s. Waiting %s seconds", i, retryCount, err.Error(), retrySleepTime)
			time.Sleep(time.Duration(retrySleepTime) * time.Second)
			continue
		}
		break
	}
	if err != nil {
		log.Errorf("Error getting kubernetes config : %s", err)
		return
	}
	config, err = helm.GetK8sClientConfig(kubeConfig)
	if err != nil {
		log.Errorf("Error parsing kubernetes config : %s", err)
		return
	}
	log.Infof("Starting to write kubernetes related certs/keys for: %s", configPath)
	if err := utils.WriteToFile(config.KeyData, configPath+"/client-key-data.pem"); err != nil {
		log.Errorf("Error writing file: %s", err.Error())
		return
	}
	if err := utils.WriteToFile(config.CertData, configPath+"/client-certificate-data.pem"); err != nil {
		log.Errorf("Error writing file: %s", err.Error())
		return
	}
	if err := utils.WriteToFile(config.CAData, configPath+"/certificate-authority-data.pem"); err != nil {
		log.Errorf("Error writing file: %s", err.Error())
		return
	}

	log.Infof("Writing kubernetes related certs/keys succeeded.")
}

//Post Hooks can't return value, they can log error and/or update state?
func InstallIngressControllerPostHook(cluster CommonCluster) {
	// --- [ Get K8S Config ] --- //
	log = logger.WithFields(logrus.Fields{"action": "InstallIngressController"})

	kubeConfig, err := cluster.GetK8sConfig()
	if err != nil {
		log.Errorf("Unable to fetch config for posthook: %s", err.Error())
		return
	}

	deploymentName := "banzaicloud-stable/pipeline-cluster-ingress"
	releaseName := "pipeline"

	_, err = helm.CreateDeployment(deploymentName, releaseName, nil, kubeConfig, cluster.GetName())
	if err != nil {
		log.Errorf("Deploying '%s' failed due to: ", deploymentName)
		log.Errorf("%s", err.Error())
		return
	}
	log.Infof("'%s' installed", deploymentName)
}

//PostHook functions with func(*cluster.Cluster) signature
func GetConfigPostHook(cluster CommonCluster) {
	log = logger.WithFields(logrus.Fields{"action": "PostHook"})
	createdCluster, err := cluster.GetK8sConfig()
	if err != nil {
		log.Errorf("error during get config post hook: %s", createdCluster)
		return
	}
}

func UpdatePrometheusPostHook(_ CommonCluster) {
	UpdatePrometheus()
}

func InstallHelmPostHook(cluster CommonCluster) {
	log = logger.WithFields(logrus.Fields{"action": "PostHook"})

	retryAttempts := viper.GetInt(constants.HELM_RETRY_ATTEMPT_CONFIG)
	retrySleepSeconds := viper.GetInt(constants.HELM_RETRY_SLEEP_SECONDS)

	helmInstall := &htypes.Install{
		Namespace:      "kube-system",
		ServiceAccount: "tiller",
		ImageSpec:      "gcr.io/kubernetes-helm/tiller:v2.7.2",
	}
	helmHome := viper.GetString("helm.home")
	kubeconfig, err := cluster.GetK8sConfig()
	if err != nil {
		log.Errorf("Error retrieving kubernetes config: %s", err.Error())
		return
	}

	err = helm.RetryHelmInstall(helmInstall, kubeconfig, helmHome)
	if err == nil {
		// Get K8S Config //
		kubeConfig, err := cluster.GetK8sConfig()
		if err != nil {
			return
		}
		log.Info("Getting K8S Config Succeeded")
		for i := 0; i <= retryAttempts; i++ {
			log.Debugf("Waiting for tiller to come up %d/%d", i, retryAttempts)
			_, err = helm.GetHelmClient(kubeConfig)
			if err == nil {
				return
			}
			time.Sleep(time.Duration(retrySleepSeconds) * time.Second)
		}
		log.Error("Timeout during waiting for tiller to get ready")
	}
}

func UpdatePrometheus() {
	log = logger.WithFields(logrus.Fields{"tag": constants.TagPrometheus})
	err := UpdatePrometheusConfig()
	if err != nil {
		log.Warn("Could not update prometheus configmap: %v", err)
	}
}
