// Copyright © 2019 Banzai Cloud
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

package driver

import (
	"context"
	"encoding/base64"
	"errors"
	"net/http"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-01-01/network"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/banzaicloud/pipeline/auth"
	"github.com/banzaicloud/pipeline/config"
	"github.com/banzaicloud/pipeline/internal/cluster/metrics"
	"github.com/banzaicloud/pipeline/internal/cluster/statestore"
	"github.com/banzaicloud/pipeline/internal/providers/azure/pke"
	"github.com/banzaicloud/pipeline/internal/providers/azure/pke/workflow"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	"github.com/banzaicloud/pipeline/pkg/k8sclient"
	pkgAzure "github.com/banzaicloud/pipeline/pkg/providers/azure"
	pkgSecret "github.com/banzaicloud/pipeline/pkg/secret"
	"github.com/banzaicloud/pipeline/secret"
	"github.com/goph/emperror"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"go.uber.org/cadence"
	"go.uber.org/cadence/client"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func MakeAzurePKEClusterDeleter(events ClusterDeleterEvents, logger logrus.FieldLogger, secrets SecretStore, statusChangeDurationMetric metrics.ClusterStatusChangeDurationMetric, store pke.AzurePKEClusterStore, workflowClient client.Client) AzurePKEClusterDeleter {
	return AzurePKEClusterDeleter{
		events:                     events,
		logger:                     logger,
		secrets:                    secrets,
		statusChangeDurationMetric: statusChangeDurationMetric,
		store:                      store,
		workflowClient:             workflowClient,
	}
}

type AzurePKEClusterDeleter struct {
	events                     ClusterDeleterEvents
	logger                     logrus.FieldLogger
	secrets                    SecretStore
	statusChangeDurationMetric metrics.ClusterStatusChangeDurationMetric
	store                      pke.AzurePKEClusterStore
	workflowClient             client.Client
}

type SecretStore interface {
	Get(orgnaizationID uint, secretID string) (*secret.SecretItemResponse, error)
}

type ClusterDeleterEvents interface {
	ClusterDeleted(organizationID uint, clusterName string)
}

func (cd AzurePKEClusterDeleter) Delete(ctx context.Context, cluster pke.PKEOnAzureCluster, forced bool) error {
	logger := cd.logger.WithField("clusterName", cluster.Name).WithField("clusterID", cluster.ID).WithField("forced", forced)
	logger.Info("Deleting cluster")

	k8sConfig, err := getK8sConfig(cd.secrets, cluster.OrganizationID, cluster.K8sSecretID)
	if err != nil {
		return emperror.Wrap(err, "failed to get k8s config")
	}
	pipNames, err := collectPublicIPAddressNames(ctx, logger, cd.secrets, cluster, k8sConfig, forced)
	if err != nil {
		return emperror.Wrap(err, "failed to collect public IP address resource names")
	}

	input := workflow.DeleteClusterWorkflowInput{
		OrganizationID:       cluster.OrganizationID,
		SecretID:             cluster.SecretID,
		ClusterID:            cluster.ID,
		ClusterName:          cluster.Name,
		K8sConfig:            k8sConfig,
		ResourceGroupName:    cluster.ResourceGroup.Name,
		LoadBalancerName:     cluster.Name, // must be the same as the value passed to pke install master --kubernetes-cluster-name
		PublicIPAddressNames: pipNames,
		RouteTableName:       cluster.Name + "-route-table",
		ScaleSetNames:        getVMSSNames(cluster),
		SecurityGroupNames:   []string{cluster.Name + "-master-nsg", cluster.Name + "-worker-nsg"},
		VirtualNetworkName:   cluster.VirtualNetwork.Name,
		Forced:               forced,
	}

	retryPolicy := &cadence.RetryPolicy{
		InitialInterval:    time.Second * 3,
		BackoffCoefficient: 2,
		ExpirationInterval: time.Minute * 3,
		MaximumAttempts:    5,
	}

	workflowOptions := client.StartWorkflowOptions{
		TaskList:                     "pipeline",
		ExecutionStartToCloseTimeout: 40 * time.Minute, // TODO: lower timeout
		RetryPolicy:                  retryPolicy,
	}

	if err := cd.store.SetStatus(cluster.ID, pkgCluster.Deleting, pkgCluster.DeletingMessage); err != nil {
		return emperror.Wrap(err, "failed to set cluster status")
	}

	timer, err := cd.getClusterStatusChangeDurationTimer(cluster)
	if err = emperror.Wrap(err, "failed to start status change duration metric timer"); err != nil {
		if forced {
			cd.logger.Error(err)
			timer = metrics.NoopDurationMetricTimer{}
		} else {
			return err
		}
	}

	wfrun, err := cd.workflowClient.ExecuteWorkflow(ctx, workflowOptions, workflow.DeleteClusterWorkflowName, input)
	if err = emperror.WrapWith(err, "failed to start cluster deletion workflow", "cluster", cluster.Name); err != nil {
		cd.store.SetStatus(cluster.ID, pkgCluster.Error, err.Error())
		return err
	}

	go func() {
		defer timer.RecordDuration()

		if err := wfrun.Get(ctx, nil); err != nil {
			cd.logger.Error("cluster deleting workflow failed", err)
			return
		}
		// TODO: delete KubeProxy
		statestore.CleanStateStore(cluster.Name)
		cd.events.ClusterDeleted(cluster.OrganizationID, cluster.Name)
	}()

	if err = cd.store.SetActiveWorkflowID(cluster.ID, wfrun.GetID()); err != nil {
		return emperror.WrapWith(err, "failed to set active workflow ID for cluster", "cluster", cluster.Name, "workflowID", wfrun.GetID())
	}

	return nil
}

func (cd AzurePKEClusterDeleter) getClusterStatusChangeDurationTimer(cluster pke.PKEOnAzureCluster) (metrics.DurationMetricTimer, error) {
	values := metrics.ClusterStatusChangeDurationMetricValues{
		ProviderName: pkgCluster.Azure,
		LocationName: cluster.Location,
		Status:       pkgCluster.Deleting,
	}
	if viper.GetBool(config.MetricsDebug) {
		org, err := auth.GetOrganizationById(cluster.OrganizationID)
		if err != nil {
			return nil, emperror.Wrap(err, "Error during getting organization.")
		}
		values.OrganizationName = org.Name
		values.ClusterName = cluster.Name
	}
	return cd.statusChangeDurationMetric.StartTimer(values), nil
}

func (cd AzurePKEClusterDeleter) DeleteByID(ctx context.Context, clusterID uint, forced bool) error {
	cl, err := cd.store.GetByID(clusterID)
	if err != nil {
		return emperror.Wrap(err, "failed to load cluster from data store")
	}
	return cd.Delete(ctx, cl, forced)
}

func getK8sConfig(secrets SecretStore, organizationID uint, k8sSecretID string) ([]byte, error) {
	if secrets == nil || k8sSecretID == "" {
		return nil, nil
	}
	sir, err := secrets.Get(organizationID, k8sSecretID)
	if err != nil {
		return nil, emperror.Wrap(err, "failed to get k8s config from secret store")
	}
	k8sConfig, err := base64.StdEncoding.DecodeString(sir.GetValue(pkgSecret.K8SConfig))
	if err != nil {
		return nil, emperror.Wrap(err, "can't decode Kubernetes config")
	}
	return k8sConfig, nil
}

func collectPublicIPAddressNames(ctx context.Context, logger logrus.FieldLogger, secrets SecretStore, cluster pke.PKEOnAzureCluster, k8sConfig []byte, forced bool) ([]string, error) {
	sir, err := secrets.Get(cluster.OrganizationID, cluster.SecretID)
	if err != nil {
		return nil, emperror.Wrap(err, "failed to get cluster secret from secret store")
	}
	cc, err := pkgAzure.NewCloudConnection(&azure.PublicCloud, pkgAzure.NewCredentials(sir.Values))
	if err != nil {
		return nil, emperror.Wrap(err, "failed to create cloud connection")
	}

	names := make(map[string]bool)

	lb, err := cc.GetLoadBalancersClient().Get(ctx, cluster.ResourceGroup.Name, cluster.Name, "frontendIPConfigurations/publicIPAddress")
	if err != nil {
		if lb.StatusCode == http.StatusNotFound {
			return nil, nil
		}
		return nil, emperror.Wrap(err, "failed to retrieve load balancer")
	}
	names = gatherOwnedPublicIPAddressNames(lb, cluster.Name, names)

	names, err = gatherK8sServicePublicIPs(ctx, cc.GetPublicIPAddressesClient(), cluster, k8sConfig, names)
	if emperror.Wrap(err, "failed to gather k8s services' public IP addresses"); err != nil {
		if forced {
			logger.Warning(err)
		} else {
			return nil, err
		}
	}

	result := make([]string, 0, len(names))
	for name := range names {
		result = append(result, name)
	}
	return result, nil
}

func gatherOwnedPublicIPAddressNames(lb network.LoadBalancer, clusterName string, names map[string]bool) map[string]bool {
	if names == nil {
		names = make(map[string]bool)
	}

	if lb.LoadBalancerPropertiesFormat != nil {
		if fics := lb.LoadBalancerPropertiesFormat.FrontendIPConfigurations; fics != nil {
			for _, fic := range *fics {
				if fic.FrontendIPConfigurationPropertiesFormat != nil {
					if pip := fic.FrontendIPConfigurationPropertiesFormat.PublicIPAddress; pip != nil {
						if name := to.String(pip.Name); name != "" && workflow.HasOwnedTag(clusterName, to.StringMap(pip.Tags)) {
							names[name] = true
						}
					}
				}
			}
		}
	}

	return names
}

func gatherK8sServicePublicIPs(ctx context.Context, client *pkgAzure.PublicIPAddressesClient, cluster pke.PKEOnAzureCluster, k8sConfig []byte, names map[string]bool) (map[string]bool, error) {
	if k8sConfig == nil {
		return names, errors.New("no k8s config")
	}

	resPage, err := client.List(ctx, cluster.ResourceGroup.Name)
	if err != nil {
		return names, emperror.WrapWith(err, "failed to list Azure public IP address resources in resource group", "resourceGroup", cluster.ResourceGroup.Name)
	}

	ipToName := make(map[string]string)
	for {
		for _, pip := range resPage.Values() {
			if to.String(pip.Name) != "" && to.String(pip.IPAddress) != "" {
				ipToName[to.String(pip.IPAddress)] = to.String(pip.Name)
			}
		}
		if resPage.NotDone() {
			resPage.Next()
		} else {
			break
		}
	}

	k8sClient, err := k8sclient.NewClientFromKubeConfig(k8sConfig)
	if err != nil {
		return names, emperror.Wrap(err, "failed to create a new Kubernetes client")
	}

	serviceList, err := k8sClient.CoreV1().Services(metav1.NamespaceAll).List(metav1.ListOptions{})
	if serviceList == nil || err != nil {
		return names, emperror.Wrap(err, "failed to retrieve service list")
	}

	if names == nil {
		names = make(map[string]bool)
	}

	for _, service := range serviceList.Items {
		if service.Spec.Type == corev1.ServiceTypeLoadBalancer {
			for _, ing := range service.Status.LoadBalancer.Ingress {
				if name := ipToName[ing.IP]; name != "" {
					names[name] = true
				}
			}
		}
	}

	return names, nil
}

func getVMSSNames(cluster pke.PKEOnAzureCluster) []string {
	names := make([]string, len(cluster.NodePools))
	for i, np := range cluster.NodePools {
		names[i] = pke.GetVMSSName(cluster.Name, np.Name)
	}
	return names
}
