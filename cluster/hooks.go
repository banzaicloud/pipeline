package cluster

import (
	"fmt"
	"github.com/banzaicloud/pipeline/auth"
	pipConfig "github.com/banzaicloud/pipeline/config"
	"github.com/banzaicloud/pipeline/dns"
	"github.com/banzaicloud/pipeline/helm"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	pkgHelm "github.com/banzaicloud/pipeline/pkg/helm"
	"github.com/banzaicloud/pipeline/utils"
	"github.com/go-errors/errors"
	"github.com/spf13/viper"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"sync"
	"time"
)

// muxOrgDomain is a mutex used to sync access to external Dns service
// in order to avoid registering the same domain twice
var muxOrgDomain sync.Mutex

//RunPostHooks calls posthook functions with created cluster
func RunPostHooks(functionList []PostFunctioner, createdCluster CommonCluster) {
	var err error
	for _, i := range functionList {
		if i != nil {

			if err == nil {
				log.Infof("Start posthook function[%s]", i)
				err = i.Do(createdCluster)
				if err != nil {
					log.Errorf("Error during posthook function[%s]: %s", i, err.Error())
				}
			}

			if err != nil {
				i.Error(createdCluster, err)
			}

		}
	}

}

// PollingKubernetesConfig polls kubeconfig from the cloud
func PollingKubernetesConfig(cluster CommonCluster) ([]byte, error) {

	var err error
	status := pkgCluster.Creating

	for status == pkgCluster.Creating {

		log.Infof("Cluster status: %s", status)
		sr, err := cluster.GetStatus()
		if err != nil {
			return nil, err
		}
		status = sr.Status

		err = cluster.ReloadFromDatabase()
		if err != nil {
			return nil, err
		}
		time.Sleep(time.Second * 5)
	}

	retryCount := viper.GetInt("cloud.configRetryCount")
	retrySleepTime := viper.GetInt("cloud.configRetrySleep")

	var kubeConfig []byte
	for i := 0; i < retryCount; i++ {
		kubeConfig, err = cluster.DownloadK8sConfig()
		if err != nil {
			log.Infof("Error getting kubernetes config attempt %d/%d: %s. Waiting %d seconds", i, retryCount, err.Error(), retrySleepTime)
			time.Sleep(time.Duration(retrySleepTime) * time.Second)
			continue
		}
		break
	}

	return kubeConfig, err
}

// WaitingForTillerComeUp waits until till to come up
func WaitingForTillerComeUp(kubeConfig []byte) error {

	retryAttempts := viper.GetInt(pkgHelm.HELM_RETRY_ATTEMPT_CONFIG)
	retrySleepSeconds := viper.GetInt(pkgHelm.HELM_RETRY_SLEEP_SECONDS)

	for i := 0; i <= retryAttempts; i++ {
		log.Infof("Waiting for tiller to come up %d/%d", i, retryAttempts)
		_, err := helm.GetHelmClient(kubeConfig)
		if err == nil {
			return nil
		}
		log.Warnf("Error during getting helm client: %s", err.Error())
		time.Sleep(time.Duration(retrySleepSeconds) * time.Second)
	}
	return errors.New("Timeout during waiting for tiller to get ready")
}

// InstallMonitoring to install monitoring deployment
func InstallMonitoring(input interface{}) error {
	cluster, ok := input.(CommonCluster)
	if !ok {
		return errors.Errorf("Wrong parameter type: %T", cluster)
	}
	return installDeployment(cluster, helm.DefaultNamespace, pkgHelm.BanzaiRepository+"/pipeline-cluster-monitor", "pipeline-monitoring", nil, "InstallMonitoring")
}

// InstallLogging to install logging deployment
func InstallLogging(input interface{}) error {
	cluster, ok := input.(CommonCluster)
	if !ok {
		return errors.Errorf("Wrong parameter type: %T", cluster)
	}
	return installDeployment(cluster, helm.DefaultNamespace, pkgHelm.BanzaiRepository+"/pipeline-cluster-logging", "pipeline-logging", nil, "InstallLogging")
}

//PersistKubernetesKeys is a basic version of persisting keys TODO check if we need this from API or anywhere else
func PersistKubernetesKeys(input interface{}) error {
	cluster, ok := input.(CommonCluster)
	if !ok {
		return errors.Errorf("Wrong parameter type: %T", cluster)
	}
	configPath := pipConfig.GetStateStorePath(cluster.GetName())
	log.Infof("Statestore path is: %s", configPath)
	var config *rest.Config

	kubeConfig, err := cluster.GetK8sConfig()

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
	if err != nil {
		return err
	}

	configmap.Data[clusterName+"_client-key-data.pem"] = string(config.KeyData)
	configmap.Data[clusterName+"_client-certificate-data.pem"] = string(config.CertData)
	configmap.Data[clusterName+"_certificate-authority-data.pem"] = string(config.CAData)
	_, err = client.CoreV1().ConfigMaps("default").Update(configmap)
	if err != nil {
		return err
	}
	return nil
}

func installDeployment(cluster CommonCluster, namespace string, deploymentName string, releaseName string, values []byte, actionName string) error {
	// --- [ Get K8S Config ] --- //
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

	_, err = helm.CreateDeployment(deploymentName, namespace, releaseName, values, kubeConfig, helm.GenerateHelmRepoEnv(org.Name))
	if err != nil {
		log.Errorf("Deploying '%s' failed due to: %s", deploymentName, err.Error())
		return err
	}
	log.Infof("'%s' installed", deploymentName)
	return nil
}

//InstallIngressControllerPostHook post hooks can't return value, they can log error and/or update state?
func InstallIngressControllerPostHook(input interface{}) error {
	cluster, ok := input.(CommonCluster)
	if !ok {
		return errors.Errorf("Wrong parameter type: %T", cluster)
	}
	return installDeployment(cluster, helm.DefaultNamespace, pkgHelm.BanzaiRepository+"/pipeline-cluster-ingress", "pipeline", nil, "InstallIngressController")
}

//InstallClusterAutoscalerPostHook post hook only for AWS & Azure for now
func InstallClusterAutoscalerPostHook(input interface{}) error {
	cluster, ok := input.(CommonCluster)
	if !ok {
		return errors.Errorf("Wrong parameter type: %T", cluster)
	}
	return DeployClusterAutoscaler(cluster)
}

//UpdatePrometheusPostHook updates a configmap used by Prometheus
func UpdatePrometheusPostHook(_ interface{}) error {
	UpdatePrometheus()
	return nil
}

//InstallHelmPostHook this posthook installs the helm related things
func InstallHelmPostHook(input interface{}) error {
	cluster, ok := input.(CommonCluster)
	if !ok {
		return errors.Errorf("Wrong parameter type: %T", cluster)
	}

	helmInstall := &pkgHelm.Install{
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
		log.Info("Getting K8S Config Succeeded")

		if err := WaitingForTillerComeUp(kubeconfig); err != nil {
			return err
		}

	} else {
		log.Errorf("Error during retry helm install: %s", err.Error())
	}
	return nil
}

//UpdatePrometheus updates a configmap used by Prometheus
func UpdatePrometheus() {
	err := UpdatePrometheusConfig()
	if err != nil {
		log.Warn("Could not update prometheus configmap: %v", err)
	}
}

// StoreKubeConfig saves kubeconfig into vault
func StoreKubeConfig(input interface{}) error {

	cluster, ok := input.(CommonCluster)
	if !ok {
		return errors.Errorf("Wrong parameter type: %T", cluster)
	}

	config, err := PollingKubernetesConfig(cluster)
	if err != nil {
		log.Errorf("Error downloading kubeconfig: %s", err.Error())
		return err
	}

	return StoreKubernetesConfig(cluster, config)
}

// RegisterDomainPostHook registers a subdomain using the name of the current organisation
// in external Dns service. It ensures that only one domain is registered per organisation.
func RegisterDomainPostHook(input interface{}) error {
	cluster, ok := input.(CommonCluster)
	if !ok {
		return errors.Errorf("Wrong parameter type: %T", cluster)
	}

	region := ""       // TODO: this should come from config or vault
	awsSecretId := ""  // TODO: this should come from vault
	awsSecretKey := "" // TODO: this should come form vault

	// If no aws credentials for Route53 provided in Vault than exit as this functionality is not enabled
	if len(region) == 0 || len(awsSecretId) == 0 || len(awsSecretKey) == 0 {
		return nil
	}

	domainBase := viper.GetString("organization.domain")

	orgId := cluster.GetOrganizationId()

	// sync domain registration to avoid duplicates in case more clusters are created in parallel in the same org
	muxOrgDomain.Lock()

	defer muxOrgDomain.Unlock()

	dnsSvc, err := dns.NewExternalDnsServiceClient(region, awsSecretId, awsSecretKey)
	if err != nil {
		log.Errorf("Creating external dns service client failed: %s", err.Error())
		return err
	}

	org, err := auth.GetOrganizationById(orgId)
	if err != nil {
		log.Errorf("Retrieving organisation with id %d failed: %s", orgId, err.Error())
		return err
	}

	domain := fmt.Sprintf("%s.%s", org.Name, domainBase)

	registered, err := dnsSvc.IsDomainRegistered(orgId, domain)
	if err != nil {
		log.Errorf("Checking if domain '%s' is already regsitered failed: %s", domain, err.Error())
		return err
	}

	if registered {
		return nil // already registered, nothing to do
	}

	if err = dnsSvc.RegisterDomain(orgId, domain); err != nil {
		log.Errorf("Registering domain '%s' failed: %s", domain, err.Error())
		return err
	}

	return nil
}
