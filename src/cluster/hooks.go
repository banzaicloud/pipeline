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
	"context"
	"fmt"
	"sync"

	"emperror.dev/errors"
	"github.com/ghodss/yaml"
	"github.com/mitchellh/mapstructure"
	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	arkAPI "github.com/banzaicloud/pipeline/internal/ark/api"
	arkPosthook "github.com/banzaicloud/pipeline/internal/ark/posthook"
	"github.com/banzaicloud/pipeline/internal/global"
	"github.com/banzaicloud/pipeline/internal/hollowtrees"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	pkgCommon "github.com/banzaicloud/pipeline/pkg/common"
	"github.com/banzaicloud/pipeline/pkg/k8sclient"
	"github.com/banzaicloud/pipeline/pkg/k8sutil"
)

func castToPostHookParam(data pkgCluster.PostHookParam, output interface{}) error {
	return mapstructure.Decode(data, output)
}

type KubernetesDashboardPostHook struct {
	helmServiceInjector
	Priority
	ErrorHandler
}

func (ph *KubernetesDashboardPostHook) Do(cluster CommonCluster) error {
	var config = global.Config.Cluster.PostHook.Dashboard
	if !config.Enabled {
		return nil
	}

	k8sDashboardNameSpace := global.Config.Cluster.Namespace
	k8sDashboardReleaseName := "dashboard"
	var valuesJson []byte

	if cluster.RbacEnabled() {
		// create service account
		kubeConfig, err := cluster.GetK8sConfig()
		if err != nil {
			log.Errorf("Unable to fetch config for posthook: %s", err.Error())
			return err
		}

		client, err := k8sclient.NewClientFromKubeConfig(kubeConfig)
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
		rules := []rbacv1.PolicyRule{
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

		valuesJson, err = yaml.Marshal(values)
		if err != nil {
			return err
		}
	}

	return ph.helmService.ApplyDeployment(context.Background(), cluster.GetID(), k8sDashboardNameSpace, config.Chart, k8sDashboardReleaseName, valuesJson, config.Version)
}

type ClusterAutoscalerPostHook struct {
	helmServiceInjector
	Priority
	ErrorHandler
}

// InstallClusterAutoscalerPostHook post hook only for AWS & Azure for now
func (ph *ClusterAutoscalerPostHook) Do(cluster CommonCluster) error {
	if ph.helmService == nil {
		return errors.New("missing helm service dependency")
	}
	return DeployClusterAutoscaler(cluster, ph.helmService)
}

func metricsServerIsInstalled(cluster CommonCluster) bool {
	kubeConfig, err := cluster.GetK8sConfig()
	if err != nil {
		log.Errorf("Getting cluster config failed: %s", err.Error())
		return false
	}

	client, err := k8sclient.NewClientFromKubeConfig(kubeConfig)
	if err != nil {
		log.Errorf("Getting K8s client failed: %s", err.Error())
		return false
	}

	serverGroups, err := client.ServerGroups()
	for _, group := range serverGroups.Groups {
		if group.Name == "metrics.k8s.io" {
			for _, v := range group.Versions {
				if v.GroupVersion == "metrics.k8s.io/v1beta1" {
					return true
				}
			}
		}
	}
	return false
}

// make sure the injector interface is implemented
var _ HookWithParamsFactory = &RestoreFromBackupPosthook{}

type RestoreFromBackupPosthook struct {
	helmServiceInjector
	Priority
	ErrorHandler

	params pkgCluster.PostHookParam
}

func (ph *RestoreFromBackupPosthook) Create(params pkgCluster.PostHookParam) PostFunctioner {
	return &RestoreFromBackupPosthook{
		Priority:     ph.Priority,
		ErrorHandler: ErrorHandler{},
		params:       params,
	}
}

// RestoreFromBackup restores an ARK backup
func (ph *RestoreFromBackupPosthook) Do(cluster CommonCluster) error {
	var params arkAPI.RestoreFromBackupParams
	err := castToPostHookParam(ph.params, &params)
	if err != nil {
		return err
	}

	return arkPosthook.RestoreFromBackup(
		params,
		cluster,
		global.DB(),
		log,
		errorHandler,
		global.Config.Cluster.DisasterRecovery.Ark.RestoreWaitTimeout,
		ph.helmService,
	)
}

type InitSpotConfigPostHook struct {
	helmServiceInjector
	Priority
	ErrorHandler
}

// InitSpotConfig creates a ConfigMap to store spot related config and installs the scheduler and the spot webhook charts
func (ph *InitSpotConfigPostHook) Do(cluster CommonCluster) error {
	var config = global.Config.Cluster.PostHook.Spotconfig
	if !config.Enabled {
		return nil
	}

	spot, err := isSpotCluster(cluster)
	if err != nil {
		return errors.WrapIf(err, "failed to check if cluster has spot instances")
	}

	if !spot {
		log.Debug("cluster doesn't have spot priced instances, spot post hook won't run")
		return nil
	}

	pipelineSystemNamespace := global.Config.Cluster.Namespace

	kubeConfig, err := cluster.GetK8sConfig()
	if err != nil {
		return errors.WrapIf(err, "failed to get Kubernetes config")
	}

	client, err := k8sclient.NewClientFromKubeConfig(kubeConfig)
	if err != nil {
		return errors.WrapIf(err, "failed to get Kubernetes clientset from kubeconfig")
	}

	err = initializeSpotConfigMap(client, pipelineSystemNamespace)
	if err != nil {
		return errors.WrapIf(err, "failed to initialize spot ConfigMap")
	}

	values := map[string]interface{}{}
	marshalledValues, err := yaml.Marshal(values)
	if err != nil {
		return errors.WrapIf(err, "failed to marshal yaml values")
	}

	err = ph.helmService.InstallDeployment(context.Background(), cluster.GetID(), pipelineSystemNamespace, config.Charts.Scheduler.Chart, "spot-scheduler", marshalledValues, config.Charts.Scheduler.Version, false)
	if err != nil {
		return errors.WrapIf(err, "failed to install the spot-scheduler deployment")
	}
	err = ph.helmService.InstallDeployment(context.Background(), cluster.GetID(), pipelineSystemNamespace, config.Charts.Webhook.Chart, "spot-webhook", marshalledValues, config.Charts.Webhook.Version, true)
	if err != nil {
		return errors.WrapIf(err, "failed to install the spot-config-webhook deployment")
	}
	return nil
}

func isSpotCluster(cluster CommonCluster) (bool, error) {
	status, err := cluster.GetStatus()
	if err != nil {
		return false, errors.WrapIf(err, "failed to get cluster status")
	}
	return status.Spot, nil
}

func initializeSpotConfigMap(client *kubernetes.Clientset, systemNs string) error {
	log.Debug("initializing ConfigMap to store spot configuration")
	_, err := client.CoreV1().ConfigMaps(systemNs).Get(pkgCommon.SpotConfigMapKey, metav1.GetOptions{})
	if err != nil {
		if apiErrors.IsNotFound(err) {
			_, err = client.CoreV1().ConfigMaps(systemNs).Create(&v1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name: pkgCommon.SpotConfigMapKey,
				},
				Data: make(map[string]string),
			})
			if err != nil {
				return errors.WrapIf(err, "failed to create spot ConfigMap")
			}
		} else {
			return errors.WrapIf(err, "failed to retrieve spot ConfigMap")
		}
	}
	log.Info("finished initializing spot ConfigMap")
	return nil
}

type HorizontalPodAutoscalerPostHook struct {
	helmServiceInjector
	Priority
	ErrorHandler
}

func (hpa *HorizontalPodAutoscalerPostHook) Do(cluster CommonCluster) error {
	var config = global.Config.Cluster

	if !config.PostHook.HPA.Enabled {
		return nil
	}

	promServiceName := config.Autoscale.HPA.Prometheus.ServiceName
	prometheusPort := global.Config.Cluster.Autoscale.HPA.Prometheus.LocalPort

	infraNamespace := config.Autoscale.Namespace
	serviceContext := config.Autoscale.HPA.Prometheus.ServiceContext

	values := map[string]interface{}{
		"kube-metrics-adapter": map[string]interface{}{
			"prometheus": map[string]interface{}{
				"url": fmt.Sprintf("http://%s.%s.svc:%d/%s", promServiceName, infraNamespace, prometheusPort, serviceContext),
			},
			"enableExternalMetricsApi": true,
			"enableCustomMetricsApi":   false,
		},
	}

	// install metricsServer only if metrics.k8s.io endpoint is not available already
	if !metricsServerIsInstalled(cluster) {
		log.Infof("Metrics Server is not installed, installing")

		metricsServerValues := make(map[string]interface{}, 0)
		metricsServerValues["enabled"] = true

		// use InternalIP on VSphere
		if cluster.GetCloud() == pkgCluster.Vsphere {
			metricsServerValues["args"] = []string{
				"--kubelet-preferred-address-types=InternalIP",
			}
		}

		values["metrics-server"] = metricsServerValues
	} else {
		log.Infof("Metrics Server is already installed")
	}

	mergedValues, err := mergeValues(values, config.Autoscale.Charts.HPAOperator.Values)
	if err != nil {
		return errors.WrapIf(err, "failed to merge hpa-operator chart values with config")
	}
	return hpa.helmService.ApplyDeployment(context.Background(), cluster.GetID(), infraNamespace, config.Autoscale.Charts.HPAOperator.Chart, "hpa-operator", mergedValues, config.Autoscale.Charts.HPAOperator.Version)
}

type InstanceTerminationHandlerPostHook struct {
	helmServiceInjector
	Priority
	ErrorHandler
}

func (ith InstanceTerminationHandlerPostHook) Do(cluster CommonCluster) error {
	var config = global.Config.Cluster.PostHook.ITH
	if !global.Config.Pipeline.Enterprise || !config.Enabled {
		return nil
	}

	cloud := cluster.GetCloud()

	if cloud != pkgCluster.Amazon && cloud != pkgCluster.Google {
		return nil
	}

	pipelineSystemNamespace := global.Config.Cluster.Namespace

	values := map[string]interface{}{
		"tolerations": []v1.Toleration{
			{
				Operator: v1.TolerationOpExists,
			},
		},
		"hollowtreesNotifier": map[string]interface{}{
			"enabled": false,
		},
	}

	scaleOptions := cluster.GetScaleOptions()
	if scaleOptions != nil && scaleOptions.Enabled == true {
		tokenSigningKey := global.Config.Hollowtrees.TokenSigningKey
		if tokenSigningKey == "" {
			err := errors.New("no Hollowtrees token signkey specified")
			errorHandler.Handle(err)
			return err
		}

		generator := hollowtrees.NewTokenGenerator(
			global.Config.Auth.Token.Issuer,
			global.Config.Auth.Token.Audience,
			global.Config.Hollowtrees.TokenSigningKey,
		)
		_, token, err := generator.Generate(cluster.GetID(), cluster.GetOrganizationId(), nil)
		if err != nil {
			err = errors.WrapIf(err, "could not generate JWT token for instance termination handler")
			errorHandler.Handle(err)
			return err
		}

		values["hollowtreesNotifier"] = map[string]interface{}{
			"enabled":        true,
			"URL":            global.Config.Hollowtrees.Endpoint + "/alerts",
			"organizationID": cluster.GetOrganizationId(),
			"clusterID":      cluster.GetID(),
			"clusterName":    cluster.GetName(),
			"jwtToken":       token,
		}
	}

	marshalledValues, err := yaml.Marshal(values)
	if err != nil {
		return errors.WrapIf(err, "failed to marshal yaml values")
	}

	return ith.helmService.ApplyDeployment(context.Background(), cluster.GetID(), pipelineSystemNamespace, config.Chart, "ith", marshalledValues, config.Version)
}

// helmServiceInjector component implementing the helm service injector
// designed to be embedded into posthook structs
type helmServiceInjector struct {
	helmService HelmService
	sync.Mutex
}

// InjectHelmService injects the service to be used by the "parent" struct
func (h *helmServiceInjector) InjectHelmService(helmService HelmService) {
	h.Lock()
	defer h.Unlock()

	if h.helmService == nil {
		h.helmService = helmService
	}
}
