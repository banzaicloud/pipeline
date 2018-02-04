package cluster

import (
	htypes "github.com/banzaicloud/banzai-types/components/helm"
	"github.com/banzaicloud/banzai-types/constants"
	"github.com/banzaicloud/pipeline/helm"
	"github.com/banzaicloud/pipeline/monitor"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"time"
)

// Calls posthook functions with created cluster
func RunPostHooks(functionList []func(cluster CommonCluster), createdCluster CommonCluster) {
	for _, i := range functionList {
		i(createdCluster)
	}
}

//Post Hooks can't return value, they can log error and/or update state?
func InstallIngressControllerPostHook(cluster CommonCluster) {
	// --- [ Get K8S Config ] --- //
	log = logger.WithFields(logrus.Fields{"action": "InstallIngressController"})

	kubeConfig, err := cluster.GetK8sConfig()
	if err != nil {
		log.Error("Unable to fetch config for posthook")
		return
	}

	deploymentName := "banzaicloud-stable/pipeline-cluster-ingress"
	releaseName := "pipeline"

	_, err = helm.CreateDeployment(deploymentName, releaseName, nil, kubeConfig)
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
	err := helm.RetryHelmInstall(helmInstall, cluster, helmHome)
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
	err := monitor.UpdatePrometheusConfig()
	if err != nil {
		log.Warn("Could not update prometheus configmap: %v", err)
	}
}
