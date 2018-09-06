// Copyright © 2018 Banzai Cloud
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cluster

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/banzaicloud/pipeline/auth"
	pipConfig "github.com/banzaicloud/pipeline/config"
	"github.com/banzaicloud/pipeline/dns"
	"github.com/banzaicloud/pipeline/helm"
	"github.com/banzaicloud/pipeline/internal/providers/azure"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	pkgCommon "github.com/banzaicloud/pipeline/pkg/common"
	pkgHelm "github.com/banzaicloud/pipeline/pkg/helm"
	"github.com/banzaicloud/pipeline/pkg/k8sutil"
	pkgSecret "github.com/banzaicloud/pipeline/pkg/secret"
	"github.com/banzaicloud/pipeline/secret"
	"github.com/banzaicloud/pipeline/utils"
	"github.com/ghodss/yaml"
	"github.com/go-errors/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"k8s.io/api/core/v1"
	"k8s.io/api/rbac/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	pkgHelmRelease "k8s.io/helm/pkg/proto/hapi/release"
)

//RunPostHooks calls posthook functions with created cluster
func RunPostHooks(postHooks []PostFunctioner, cluster CommonCluster) (err error) {

	log := log.WithFields(logrus.Fields{"cluster": cluster.GetName(), "org": cluster.GetOrganizationId()})

	for _, postHook := range postHooks {
		if postHook != nil {
			log.Infof("Start posthook function[%s]", postHook)
			err = postHook.Do(cluster)
			if err != nil {
				log.Errorf("Error during posthook function[%s]: %s", postHook, err.Error())
				postHook.Error(cluster, err)
				return
			}

			statusMsg := fmt.Sprintf("Posthook function finished: %s", postHook)
			err = cluster.UpdateStatus(pkgCluster.Creating, statusMsg)
			if err != nil {
				log.Errorf("Error during posthook status update in db [%s]: %s", postHook, err.Error())
				return
			}
		}
	}

	log.Info("Run all posthooks for cluster successfully.")

	err = cluster.UpdateStatus(pkgCluster.Running, pkgCluster.RunningMessage)

	if err != nil {
		log.Errorf("Error during posthook status update in db: %s", err.Error())
	}

	return
}

// PollingKubernetesConfig polls kubeconfig from the cloud
func PollingKubernetesConfig(cluster CommonCluster) ([]byte, error) {

	var err error

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

	grafanaAdminUsername := viper.GetString("monitor.grafanaAdminUsername")
	grafanaNamespace := viper.GetString(pipConfig.PipelineMonitorNamespace)
	// Grafana password generator
	grafanaAdminPass, err := secret.RandomString("randAlphaNum", 12)
	if err != nil {
		return errors.Errorf("Grafana admin user password generator failed: %T", err)
	}

	clusterNameTag := fmt.Sprintf("cluster:%s", cluster.GetName())
	clusterUidTag := fmt.Sprintf("clusterUID:%s", cluster.GetUID())

	createSecretRequest := secret.CreateSecretRequest{
		Name: fmt.Sprintf("cluster-%d-grafana", cluster.GetID()),
		Type: pkgSecret.PasswordSecretType,
		Values: map[string]string{
			pkgSecret.Username: grafanaAdminUsername,
			pkgSecret.Password: grafanaAdminPass,
		},
		Tags: []string{
			clusterNameTag,
			clusterUidTag,
			pkgSecret.TagBanzaiReadonly,
			"app:grafana",
			"release:pipeline-monitoring",
		},
	}

	secretID, err := secret.Store.CreateOrUpdate(cluster.GetOrganizationId(), &createSecretRequest)
	if err != nil {
		log.Errorf("Error during storing grafana secret: %s", err.Error())
		return err
	}
	log.Debugf("Grafana Secret Stored id: %s", secretID)

	orgId := cluster.GetOrganizationId()
	org, err := auth.GetOrganizationById(orgId)
	if err != nil {
		log.Errorf("Retrieving organization with id %d failed: %s", orgId, err.Error())
		return err
	}

	host := fmt.Sprintf("%s.%s.%s", cluster.GetName(), org.Name, viper.GetString(pipConfig.DNSBaseDomain))
	log.Debugf("grafana ingress host: %s", host)
	grafanaValues := map[string]interface{}{
		"grafana": map[string]interface{}{
			"adminUser":     grafanaAdminUsername,
			"adminPassword": grafanaAdminPass,
			"ingress":       map[string][]string{"hosts": {host}},
		},
	}
	grafanaValuesJson, err := json.Marshal(grafanaValues)
	if err != nil {
		return errors.Errorf("Json Convert Failed : %s", err.Error())
	}

	return installDeployment(cluster, grafanaNamespace, pkgHelm.BanzaiRepository+"/pipeline-cluster-monitor", "pipeline-monitoring", grafanaValuesJson, "InstallMonitoring", "")
}

// InstallLogging to install logging deployment
func InstallLogging(input interface{}, param pkgCluster.PostHookParam) error {
	const releaseTag = "release:pipeline-logging"
	cluster, ok := input.(CommonCluster)
	if !ok {
		return errors.Errorf("Wrong parameter type: %T", cluster)
	}

	var loggingParam pkgCluster.LoggingParam
	err := castToPostHookParam(&param, &loggingParam)
	if err != nil {
		return err
	}
	// This makes no sense since we can't check if it default false or set false
	//if !checkIfTLSRelatedValuesArePresent(&loggingParam.GenTLSForLogging) {
	//	return errors.Errorf("TLS related parameter is missing from request!")
	//}
	const namespace = "default"
	loggingParam.GenTLSForLogging.TLSEnabled = true
	// Set TLS default values (default True)
	if loggingParam.GenTLSForLogging.Namespace == "" {
		loggingParam.GenTLSForLogging.Namespace = namespace
	}
	if loggingParam.GenTLSForLogging.TLSHost == "" {
		loggingParam.GenTLSForLogging.TLSHost = "fluentd." + loggingParam.GenTLSForLogging.Namespace + ".svc.cluster.local"
	}
	if loggingParam.GenTLSForLogging.GenTLSSecretName == "" {
		loggingParam.GenTLSForLogging.GenTLSSecretName = fmt.Sprintf("logging-tls-%d", cluster.GetID())
	}
	if loggingParam.GenTLSForLogging.TLSEnabled {
		clusterUidTag := fmt.Sprintf("clusterUID:%s", cluster.GetUID())
		req := &secret.CreateSecretRequest{
			Name: loggingParam.GenTLSForLogging.GenTLSSecretName,
			Type: pkgSecret.TLSSecretType,
			Tags: []string{
				clusterUidTag,
				pkgSecret.TagBanzaiReadonly,
				releaseTag,
			},
			Values: map[string]string{
				pkgSecret.TLSHosts: loggingParam.GenTLSForLogging.TLSHost,
			},
		}
		_, err := secret.Store.GetOrCreate(cluster.GetOrganizationId(), req)
		if err != nil {
			return errors.Errorf("failed generate TLS secrets to logging operator: %s", err)
		}
		_, err = InstallSecrets(cluster,
			&pkgSecret.ListSecretsQuery{
				Type: pkgSecret.TLSSecretType,
				Tags: []string{
					clusterUidTag,
					releaseTag,
				},
			}, loggingParam.GenTLSForLogging.Namespace)
		if err != nil {
			return errors.Errorf("could not install created TLS secret to cluster: %s", err)
		}
	}
	operatorValues := map[string]interface{}{
		"tls": map[string]interface{}{
			"enabled":    "true",
			"secretName": loggingParam.GenTLSForLogging.GenTLSSecretName,
		},
	}
	operatorYamlValues, err := yaml.Marshal(operatorValues)
	if err != nil {
		return err
	}
	err = installDeployment(cluster, namespace, pkgHelm.BanzaiRepository+"/logging-operator", "pipeline-logging", operatorYamlValues, "InstallLogging", "")
	if err != nil {
		return err
	}

	// Determine the type of output plugin
	logSecret, err := secret.Store.Get(cluster.GetOrganizationId(), loggingParam.SecretId)
	if err != nil {
		return err
	}
	log.Infof("logging-hook secret type: %s", logSecret.Type)
	switch logSecret.Type {
	case pkgCluster.Amazon:
		installedSecretValues, err := InstallSecrets(cluster, &pkgSecret.ListSecretsQuery{IDs: []string{loggingParam.SecretId}}, loggingParam.GenTLSForLogging.Namespace)
		if err != nil {
			return err
		}
		loggingValues := map[string]interface{}{
			"bucketName": loggingParam.BucketName,
			"region":     loggingParam.Region,
			"secret": map[string]interface{}{
				"secretName": installedSecretValues[0].Name,
			},
		}
		marshaledValues, err := yaml.Marshal(loggingValues)
		if err != nil {
			return err
		}
		return installDeployment(cluster, namespace, pkgHelm.BanzaiRepository+"/s3-output", "pipeline-s3-output", marshaledValues, "ConfigureLoggingOutPut", "")
	case pkgCluster.Google:
		installedSecretValues, err := InstallSecrets(cluster, &pkgSecret.ListSecretsQuery{IDs: []string{loggingParam.SecretId}}, loggingParam.GenTLSForLogging.Namespace)
		if err != nil {
			return err
		}
		loggingValues := map[string]interface{}{
			"bucketName": loggingParam.BucketName,
			"secret": map[string]interface{}{
				"name": installedSecretValues[0].Name,
			},
		}
		marshaledValues, err := yaml.Marshal(loggingValues)
		if err != nil {
			return err
		}
		return installDeployment(cluster, namespace, pkgHelm.BanzaiRepository+"/gcs-output", "pipeline-gcs-output", marshaledValues, "ConfigureLoggingOutPut", "")
	case pkgCluster.Azure:

		sak, err := azure.StorageAccountKey(loggingParam.ResourceGroup, loggingParam.StorageAccount, logSecret, log)
		if err != nil {
			return err
		}

		clusterUidTag := fmt.Sprintf("clusterUID:%s", cluster.GetUID())

		genericSecretName := fmt.Sprintf("logging-generic-%d", cluster.GetID())
		req := &secret.CreateSecretRequest{
			Name: genericSecretName,
			Type: pkgSecret.GenericSecret,
			Tags: []string{
				clusterUidTag,
				pkgSecret.TagBanzaiReadonly,
				releaseTag,
			},
			Values: map[string]string{
				"storageAccountName": loggingParam.StorageAccount,
				"storageAccountKey":  sak,
			},
		}
		if _, err = secret.Store.GetOrCreate(cluster.GetOrganizationId(), req); err != nil {
			return errors.Errorf("failed generate Generic secrets to logging operator: %s", err)
		}

		_, err = InstallSecrets(cluster,
			&pkgSecret.ListSecretsQuery{
				Type: pkgSecret.GenericSecret,
				Tags: []string{
					clusterUidTag,
					releaseTag,
				},
			}, namespace)
		if err != nil {
			return errors.Errorf("could not install created Generic secret to cluster: %s", err)
		}

		loggingValues := map[string]interface{}{
			"bucketName": loggingParam.BucketName,
			"secret": map[string]interface{}{
				"name": genericSecretName,
			},
		}

		marshaledValues, err := yaml.Marshal(loggingValues)
		if err != nil {
			return err
		}

		return installDeployment(cluster, namespace, pkgHelm.BanzaiRepository+"/azure-output", "pipeline-azure-output", marshaledValues, "ConfigureLoggingOutPut", "")

	default:
		return fmt.Errorf("unexpected logging secret type: %s", logSecret.Type)
	}
	// Install output related secret

}

//func checkIfTLSRelatedValuesArePresent(v *pkgCluster.GenTLSForLogging) bool {
//	if v.TLSEnabled {
//		if v.TLSHost == "" || v.GenTLSSecretName == "" || v.Namespace == "" {
//			return false
//		}
//	}
//	return true
//}

func castToPostHookParam(data *pkgCluster.PostHookParam, output interface{}) (err error) {

	var bytes []byte
	bytes, err = json.Marshal(data)
	if err != nil {
		return
	}

	err = json.Unmarshal(bytes, &output)

	return
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

func installDeployment(cluster CommonCluster, namespace string, deploymentName string, releaseName string, values []byte, actionName string, chartVersion string) error {
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

	deployments, err := helm.ListDeployments(&releaseName, kubeConfig)
	if err != nil {
		log.Errorln("Unable to fetch deployments from helm:", err)
		return err
	}

	var foundRelease *pkgHelmRelease.Release

	if deployments != nil {
		for _, release := range deployments.Releases {
			if release.Name == releaseName {
				foundRelease = release
				break
			}
		}
	}

	if foundRelease != nil {
		switch foundRelease.GetInfo().GetStatus().GetCode() {
		case pkgHelmRelease.Status_DEPLOYED:
			log.Infof("'%s' is already installed", deploymentName)
			return nil
		case pkgHelmRelease.Status_FAILED:
			err = helm.DeleteDeployment(releaseName, kubeConfig)
			if err != nil {
				log.Errorf("Failed to deleted failed deployment '%s' due to: %s", deploymentName, err.Error())
				return err
			}
		}
	}

	_, err = helm.CreateDeployment(deploymentName, chartVersion, nil, namespace, releaseName, values, kubeConfig, helm.GenerateHelmRepoEnv(org.Name))
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
	return installDeployment(cluster, helm.DefaultNamespace, pkgHelm.BanzaiRepository+"/pipeline-cluster-ingress", "pipeline", nil, "InstallIngressController", "")
}

//InstallKubernetesDashboardPostHook post hooks can't return value, they can log error and/or update state?
func InstallKubernetesDashboardPostHook(input interface{}) error {
	cluster, ok := input.(CommonCluster)
	if !ok {
		return errors.Errorf("Wrong parameter type: %T", cluster)
	}

	k8sDashboardNameSpace := helm.SystemNamespace
	k8sDashboardReleaseName := "dashboard"
	var valuesJson []byte

	if cluster.RbacEnabled() {

		// create service account
		kubeConfig, err := cluster.GetK8sConfig()
		if err != nil {
			log.Errorf("Unable to fetch config for posthook: %s", err.Error())
			return err
		}

		client, err := helm.GetK8sConnection(kubeConfig)
		if err != nil {
			log.Errorf("Could not get kubernetes client: %s", err)
			return err
		}

		// service account
		k8sDashboardServiceAccountName := k8sDashboardReleaseName
		serviceAccount, err := k8sutil.GetOrCreateServiceAccount(log, client, k8sDashboardNameSpace, k8sDashboardServiceAccountName)
		if err != nil {
			return err
		}

		// cluster role based on https://github.com/helm/charts/blob/master/stable/kubernetes-dashboard/templates/role.yaml
		clusterRoleName := k8sDashboardReleaseName
		rules := []v1beta1.PolicyRule{
			// Allow to list all
			{
				APIGroups: []string{"*"},
				Resources: []string{"*"},
				Verbs:     []string{"list", "get"},
			},
			// # Allow Dashboard to create 'kubernetes-dashboard-key-holder' secret.
			{
				APIGroups: []string{""},
				Resources: []string{"secrets"},
				Verbs:     []string{"create"},
			},
			// # Allow Dashboard to list and create 'kubernetes-dashboard-settings' config map.
			{
				APIGroups: []string{""},
				Resources: []string{"configmaps"},
				Verbs:     []string{"create"},
			},
			// # Allow Dashboard to get, update and delete Dashboard exclusive secrets.
			{
				APIGroups:     []string{""},
				Resources:     []string{"secrets"},
				ResourceNames: []string{"kubernetes-dashboard-key-holder", fmt.Sprintf("kubernetes-dashboard-%s", k8sDashboardReleaseName)},
				Verbs:         []string{"update", "delete"},
			},
			// # Allow Dashboard to get and update 'kubernetes-dashboard-settings' config map.
			{
				APIGroups:     []string{""},
				Resources:     []string{"configmaps"},
				ResourceNames: []string{"kubernetes-dashboard-settings"},
				Verbs:         []string{"update"},
			},
			// # Allow Dashboard to get metrics from heapster.
			{
				APIGroups:     []string{""},
				Resources:     []string{"services"},
				ResourceNames: []string{"heapster"},
				Verbs:         []string{"proxy"},
			},
			{
				APIGroups:     []string{""},
				Resources:     []string{"services/proxy"},
				ResourceNames: []string{"heapster", "http:heapster:", "https:heapster:"},
				Verbs:         []string{"get"},
			},
		}

		clusterRole, err := k8sutil.GetOrCreateClusterRole(log, client, clusterRoleName, rules)
		if err != nil {
			return err
		}

		// cluster role binding
		clusterRoleBindingName := k8sDashboardReleaseName
		_, err = k8sutil.GetOrCreateClusterRoleBinding(log, client, clusterRoleBindingName, serviceAccount, clusterRole)
		if err != nil {
			return err
		}

		values := map[string]interface{}{
			"rbac": map[string]bool{
				"create":           false,
				"clusterAdminRole": false,
			},
			"serviceAccount": map[string]interface{}{
				"create": false,
				"name":   serviceAccount.Name,
			},
		}

		valuesJson, err = json.Marshal(values)
		if err != nil {
			return err
		}

	}

	return installDeployment(cluster, k8sDashboardNameSpace, pkgHelm.StableRepository+"/kubernetes-dashboard", k8sDashboardReleaseName, valuesJson, "InstallKubernetesDashboard", "")

}

func setAdminRights(client *kubernetes.Clientset, userName string) (err error) {

	name := "cluster-creator-admin-right"

	log := log.WithFields(logrus.Fields{"name": name, "user": userName})

	log.Info("cluster role creating")

	_, err = client.RbacV1beta1().ClusterRoleBindings().Create(
		&v1beta1.ClusterRoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name: name,
			},
			Subjects: []v1beta1.Subject{
				{
					Kind:     "User",
					Name:     userName,
					APIGroup: v1.GroupName,
				},
			},
			RoleRef: v1beta1.RoleRef{
				Kind:     "ClusterRole",
				Name:     "cluster-admin",
				APIGroup: v1beta1.GroupName,
			},
		})

	return
}

//InstallClusterAutoscalerPostHook post hook only for AWS & Azure for now
func InstallClusterAutoscalerPostHook(input interface{}) error {
	cluster, ok := input.(CommonCluster)
	if !ok {
		return errors.Errorf("Wrong parameter type: %T", cluster)
	}
	return DeployClusterAutoscaler(cluster)
}

//InstallHorizontalPodAutoscalerPostHook
func InstallHorizontalPodAutoscalerPostHook(input interface{}) error {
	cluster, ok := input.(CommonCluster)
	if !ok {
		return errors.Errorf("Wrong parameter type: %T", cluster)
	}
	infraNamespace := viper.GetString(pipConfig.PipelineMonitorNamespace)

	var valuesOverride []byte
	// install metricsServer for Amazon & Azure & Alibaba & Oracle
	switch cluster.GetCloud() {
	case pkgCluster.Amazon, pkgCluster.Azure, pkgCluster.Alibaba, pkgCluster.Oracle:
		values := map[string]map[string]string{
			"metricsServer": {
				"enabled": "true",
			},
		}
		marshalledValues, err := yaml.Marshal(values)
		if err != nil {
			return err
		}
		valuesOverride = marshalledValues
	}

	return installDeployment(cluster, infraNamespace, pkgHelm.BanzaiRepository+"/hpa-operator", "pipeline-hpa", valuesOverride, "InstallHorizontalPodAutoscaler", "")
}

//InstallPVCOperatorPostHook installs the PVC operator
func InstallPVCOperatorPostHook(input interface{}) error {
	cluster, ok := input.(CommonCluster)
	if !ok {
		return errors.Errorf("wrong parameter type: %T", cluster)
	}

	infraNamespace := viper.GetString(pipConfig.PipelineMonitorNamespace)

	return installDeployment(cluster, infraNamespace, pkgHelm.BanzaiRepository+"/pvc-operator", "pipeline-pvc-operator", nil, "InstallPVCOperator", "")
}

//UpdatePrometheusPostHook updates a configmap used by Prometheus
func UpdatePrometheusPostHook(_ interface{}) error {
	// TODO: return error instead? (preserved original behaviour during refactor)
	err := UpdatePrometheusConfig()
	if err != nil {
		log.Warnf("could not update prometheus configmap: %v", err)
	}

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

// SetupPrivileges setups privileges
func SetupPrivileges(input interface{}) error {

	cluster, ok := input.(CommonCluster)
	if !ok {
		return errors.Errorf("Wrong parameter type: %T", cluster)
	}

	// set admin rights (if needed)
	if cluster.NeedAdminRights() {

		kubeConfig, err := cluster.GetK8sConfig()
		if err != nil {
			return err
		}

		client, err := helm.GetK8sConnection(kubeConfig)
		if err != nil {
			return err
		}

		userName, err := cluster.GetKubernetesUserName()
		if err != nil {
			return err
		}

		if err := setAdminRights(client, userName); err != nil {
			return err
		}

	}

	return nil
}

// RegisterDomainPostHook registers a subdomain using the name of the current organization
// in external Dns service. It ensures that only one domain is registered per organization.
func RegisterDomainPostHook(input interface{}) error {
	commonCluster, ok := input.(CommonCluster)
	if !ok {
		return errors.Errorf("Wrong parameter type: %T", commonCluster)
	}

	domainBase := viper.GetString(pipConfig.DNSBaseDomain)
	route53SecretNamespace := viper.GetString(pipConfig.DNSSecretNamespace)

	orgId := commonCluster.GetOrganizationId()

	dnsSvc, err := dns.GetExternalDnsServiceClient()
	if err != nil {
		log.Errorf("Getting external dns service client failed: %s", err.Error())
		return err
	}

	if dnsSvc == nil {
		log.Info("Exiting as external dns service functionality is not enabled")
		return nil
	}

	org, err := auth.GetOrganizationById(orgId)
	if err != nil {
		log.Errorf("Retrieving organization with id %d failed: %s", orgId, err.Error())
		return err
	}

	domain := fmt.Sprintf("%s.%s", org.Name, domainBase)

	registered, err := dnsSvc.IsDomainRegistered(orgId, domain)
	if err != nil {
		log.Errorf("Checking if domain '%s' is already registered failed: %s", domain, err.Error())
		return err
	}

	if !registered {
		if err = dnsSvc.RegisterDomain(orgId, domain); err != nil {
			log.Errorf("Registering domain '%s' failed: %s", domain, err.Error())
			return err
		}
	} else {
		log.Infof("Domain '%s' already registered", domain)
	}

	secretSources, err := InstallSecrets(
		commonCluster,
		&pkgSecret.ListSecretsQuery{
			Type: pkgCluster.Amazon,
			Tags: []string{pkgSecret.TagBanzaiHidden},
		},
		route53SecretNamespace,
	)
	if err != nil {
		log.Errorf("Failed to install route53 secret into cluster: %s", err.Error())
		return err
	}

	route53Secret, err := secret.Store.GetByName(orgId, secretSources[0].Name)
	if err != nil {
		log.Errorf("Failed to get the route53 secret : %s", err.Error())
		return err
	}

	log.Info("route53 secret successfully installed into cluster.")

	externalDnsValues := map[string]interface{}{
		"rbac": map[string]bool{
			"create": commonCluster.RbacEnabled() == true,
		},
		"aws": map[string]string{
			"secretKey": route53Secret.Values[pkgSecret.AwsSecretAccessKey],
			"accessKey": route53Secret.Values[pkgSecret.AwsAccessKeyId],
			"region":    route53Secret.Values[pkgSecret.AwsRegion],
		},
		"domainFilters": []string{domain},
		"policy":        "sync",
		"txtOwnerId":    commonCluster.GetUID(),
	}

	externalDnsValuesJson, err := json.Marshal(externalDnsValues)
	if err != nil {
		return errors.Errorf("Json Convert Failed : %s", err.Error())
	}
	chartVersion := viper.GetString(pipConfig.DNSExternalDnsChartVersion)

	return installDeployment(commonCluster, route53SecretNamespace, pkgHelm.StableRepository+"/external-dns", "pipeline-dns", externalDnsValuesJson, "InstallExternalDNS", chartVersion)
}

// LabelNodes adds labels for all nodes
func LabelNodes(input interface{}) error {

	log.Info("start adding labels to nodes")

	commonCluster, ok := input.(CommonCluster)
	if !ok {
		return errors.Errorf("Wrong parameter type: %T", commonCluster)
	}

	log.Infof("get K8S config")
	kubeConfig, err := commonCluster.GetK8sConfig()
	if err != nil {
		return err
	}

	log.Info("get K8S connection")
	client, err := helm.GetK8sConnection(kubeConfig)
	if err != nil {
		return err
	}

	log.Info("list node names")
	nodeNames, err := commonCluster.ListNodeNames()
	if err != nil {
		return err
	}

	log.Infof("node names: %v", nodeNames)

	for name, nodes := range nodeNames {

		log.Infof("nodepool: [%s]", name)
		for _, nodeName := range nodes {
			log.Infof("add label to node [%s]", nodeName)
			if err := addLabelsToNode(client, nodeName, map[string]string{pkgCommon.LabelKey: name}); err != nil {
				log.Warnf("error during adding label to node [%s]: %s", nodeName, err.Error())
			}
		}
	}

	log.Info("add labels finished")

	return nil
}

// addLabelsToNode add label to the given node
func addLabelsToNode(client *kubernetes.Clientset, nodeName string, labels map[string]string) (err error) {

	tokens := make([]string, 0, len(labels))
	for k, v := range labels {
		tokens = append(tokens, "\""+k+"\":\""+v+"\"")
	}
	labelString := "{" + strings.Join(tokens, ",") + "}"
	patch := fmt.Sprintf(`{"metadata":{"labels":%v}}`, labelString)

	_, err = client.CoreV1().Nodes().Patch(nodeName, types.MergePatchType, []byte(patch))
	return
}
