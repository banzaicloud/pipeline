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
	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/banzaicloud/pipeline/internal/global"
	pkgCommon "github.com/banzaicloud/pipeline/pkg/common"
	"github.com/banzaicloud/pipeline/pkg/k8sclient"
	"github.com/banzaicloud/pipeline/pkg/k8sutil"
)

type KubernetesDashboardPostHook struct {
	helmServiceInjector
	Priority
	ErrorHandler
}

func (ph *KubernetesDashboardPostHook) Do(cluster CommonCluster) error {
	config := global.Config.Cluster.PostHook.Dashboard
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

type InitSpotConfigPostHook struct {
	helmServiceInjector
	Priority
	ErrorHandler
}

// InitSpotConfig creates a ConfigMap to store spot related config and installs the scheduler and the spot webhook charts
func (ph *InitSpotConfigPostHook) Do(cluster CommonCluster) error {
	config := global.Config.Cluster.PostHook.Spotconfig
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

	ctx := context.Background()

	_, err := client.CoreV1().ConfigMaps(systemNs).Get(ctx, pkgCommon.SpotConfigMapKey, metav1.GetOptions{})
	if err != nil {
		if apiErrors.IsNotFound(err) {
			_, err = client.CoreV1().ConfigMaps(systemNs).Create(ctx, &v1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name: pkgCommon.SpotConfigMapKey,
				},
				Data: make(map[string]string),
			}, metav1.CreateOptions{})
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
