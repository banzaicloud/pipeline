package cluster

import (
	"fmt"
	htypes "github.com/banzaicloud/banzai-types/components/helm"
	"github.com/banzaicloud/banzai-types/constants"
	"github.com/banzaicloud/pipeline/auth"
	pipConfig "github.com/banzaicloud/pipeline/config"
	"github.com/banzaicloud/pipeline/helm"
	"github.com/banzaicloud/pipeline/utils"
	"github.com/go-errors/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"time"
)

var HookMap = map[string]func(interface{}) error{
	"PersistKubernetesKeys":            PersistKubernetesKeys,
	"UpdatePrometheusPostHook":         UpdatePrometheusPostHook,
	"InstallHelmPostHook":              InstallHelmPostHook,
	"InstallIngressControllerPostHook": InstallIngressControllerPostHook,
	"InstallMonitoring":                InstallMonitoring,
	"InstallLogging":                   InstallLogging,
}

//RunPostHooks calls posthook functions with created cluster
func RunPostHooks(functionList []func(interface{}) error, createdCluster CommonCluster) {
	for _, i := range functionList {
		i(createdCluster)
	}
}

func InstallMonitoring(input interface{}) error {
	var cluster CommonCluster
	if cluster, ok := input.(CommonCluster); !ok {
		return errors.Errorf("Wrong parameter type: %T", cluster)
	}
	//TODO install & ensure monitoring
	return installDeployment(cluster, "", "", nil)
}

func InstallLogging(input interface{}) error {
	var cluster CommonCluster
	if cluster, ok := input.(CommonCluster); !ok {
		return errors.Errorf("Wrong parameter type: %T", cluster)
	}
	//TODO install & ensure logging
	return installDeployment(cluster, "", "", nil)
}

//PersistKubernetesKeys is a basic version of persisting keys TODO check if we need this from API or anywhere else
func PersistKubernetesKeys(input interface{}) error {
	var cluster CommonCluster
	if cluster, ok := input.(CommonCluster); !ok {
		return errors.Errorf("Wrong parameter type: %T", cluster)
	}
	log = logger.WithFields(logrus.Fields{"action": "PersistKubernetesKeys"})
	configPath := pipConfig.GetStateStorePath(cluster.GetName())
	log.Infof("Statestore path is: %s", configPath)
	var config *rest.Config
	retryCount := viper.GetInt("cloud.configRetryCount")
	retrySleepTime := viper.GetInt("cloud.configRetrySleep")
	var err error
	var kubeConfig []byte
	for i := 0; i < retryCount; i++ {
		kubeConfig, err = cluster.GetK8sConfig()
		if err != nil {
			log.Infof("Error getting kubernetes config attempt %d/%d: %s. Waiting %d seconds", i, retryCount, err.Error(), retrySleepTime)
			time.Sleep(time.Duration(retrySleepTime) * time.Second)
			continue
		}
		break
	}
	if err != nil {
		log.Errorf("Error getting kubernetes config : %s", err)
		return err
	}
	log.Infof("Starting to write kubernetes config: %s", configPath)
	if err := utils.WriteToFile(kubeConfig, configPath+"/cluster.cfg"); err != nil {
		log.Errorf("Error writing file: %s", err.Error())
		return err
	}
	config, err = helm.GetK8sClientConfig(kubeConfig)
	if err != nil {
		log.Errorf("Error parsing kubernetes config : %s", err)
		return err
	}
	log.Infof("Starting to write kubernetes related certs/keys for: %s", configPath)
	if err := utils.WriteToFile(config.KeyData, configPath+"/client-key-data.pem"); err != nil {
		log.Errorf("Error writing file: %s", err.Error())
		return err
	}
	if err := utils.WriteToFile(config.CertData, configPath+"/client-certificate-data.pem"); err != nil {
		log.Errorf("Error writing file: %s", err.Error())
		return err
	}
	if err := utils.WriteToFile(config.CAData, configPath+"/certificate-authority-data.pem"); err != nil {
		log.Errorf("Error writing file: %s", err.Error())
		return err
	}

	configMapName := viper.GetString("monitor.configmap")
	configMapPath := viper.GetString("monitor.mountPath")
	if configMapName != "" && configMapPath != "" {
		log.Infof("save certificates to configmap: %s", configMapName)
		if err := saveKeysToConfigmap(config, configMapName, cluster.GetName()); err != nil {
			log.Errorf("error saving certs to configmap: %s", err)
			return err
		}
	}
	log.Infof("Writing kubernetes related certs/keys succeeded.")
	return nil
}

func saveKeysToConfigmap(config *rest.Config, configName string, clusterName string) error {
	client, err := helm.GetK8sInClusterConnection()
	if err != nil {
		return err
	}
	configmap, err := client.CoreV1().ConfigMaps("default").Get(configName, metav1.GetOptions{})
	configmap.Data[clusterName+"_client-key-data.pem"] = string(config.KeyData)
	configmap.Data[clusterName+"_client-certificate-data.pem"] = string(config.CertData)
	configmap.Data[clusterName+"_certificate-authority-data.pem"] = string(config.CAData)
	_, err = client.CoreV1().ConfigMaps("default").Update(configmap)
	if err != nil {
		return err
	}
	return nil
}

func installDeployment(cluster CommonCluster, deploymentName string, releaseName string, values []byte) error {
	// --- [ Get K8S Config ] --- //
	log = logger.WithFields(logrus.Fields{"action": "InstallIngressController"})

	kubeConfig, err := cluster.GetK8sConfig()
	if err != nil {
		log.Errorf("Unable to fetch config for posthook: %s", err.Error())
		return err
	}

	org, err := auth.GetOrganizationById(cluster.GetOrganizationId())
	if err != nil {
		log.Errorf("Error during getting organization: %s", err.Error())
		return err
	}

	_, err = helm.CreateDeployment(deploymentName, releaseName, nil, kubeConfig, helm.CreateEnvSettings(org.Name))
	if err != nil {
		log.Errorf("Deploying '%s' failed due to: %s", deploymentName, err.Error())
		return err
	}
	log.Infof("'%s' installed", deploymentName)
	return nil
}

//InstallIngressControllerPostHook post hooks can't return value, they can log error and/or update state?
func InstallIngressControllerPostHook(input interface{}) error {
	var cluster CommonCluster
	if cluster, ok := input.(CommonCluster); !ok {
		return errors.Errorf("Wrong parameter type: %T", cluster)
	}
	return installDeployment(cluster, "banzaicloud-stable/pipeline-cluster-ingress", "pipeline", nil)
}

//UpdatePrometheusPostHook updates a configmap used by Prometheus
func UpdatePrometheusPostHook(_ interface{}) error {
	UpdatePrometheus()
	return nil
}

//InstallHelmPostHook this posthook installs the helm related things
func InstallHelmPostHook(input interface{}) error {
	var cluster CommonCluster
	if cluster, ok := input.(CommonCluster); !ok {
		return errors.Errorf("Wrong parameter type: %T", cluster)
	}
	log = logger.WithFields(logrus.Fields{"action": "PostHook"})

	retryAttempts := viper.GetInt(constants.HELM_RETRY_ATTEMPT_CONFIG)
	retrySleepSeconds := viper.GetInt(constants.HELM_RETRY_SLEEP_SECONDS)

	helmInstall := &htypes.Install{
		Namespace:      "kube-system",
		ServiceAccount: "tiller",
		ImageSpec:      fmt.Sprintf("gcr.io/kubernetes-helm/tiller:%s", viper.GetString("helm.tillerVersion")),
	}
	kubeconfig, err := cluster.GetK8sConfig()
	if err != nil {
		log.Errorf("Error retrieving kubernetes config: %s", err.Error())
		return err
	}

	err = helm.RetryHelmInstall(helmInstall, kubeconfig)
	if err == nil {
		// Get K8S Config //
		kubeConfig, err := cluster.GetK8sConfig()
		if err != nil {
			return err
		}
		log.Info("Getting K8S Config Succeeded")
		for i := 0; i <= retryAttempts; i++ {
			log.Infof("Waiting for tiller to come up %d/%d", i, retryAttempts)
			_, err = helm.GetHelmClient(kubeConfig)
			if err == nil {
				return
			}
			log.Warnf("Error during getting helm client: %s", err.Error())
			time.Sleep(time.Duration(retrySleepSeconds) * time.Second)
		}
		log.Error("Timeout during waiting for tiller to get ready")
	} else {
		log.Errorf("Error during retry helm install: %s", err.Error())
	}
	return nil
}

//UpdatePrometheus updates a configmap used by Prometheus
func UpdatePrometheus() {
	log = logger.WithFields(logrus.Fields{"tag": constants.TagPrometheus})
	err := UpdatePrometheusConfig()
	if err != nil {
		log.Warn("Could not update prometheus configmap: %v", err)
	}
}
