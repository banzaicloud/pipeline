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
	"net/http"
	"time"

	"emperror.dev/errors"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-10-01/network"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"go.uber.org/cadence"
	"go.uber.org/cadence/client"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/banzaicloud/pipeline/auth"
	"github.com/banzaicloud/pipeline/config"
	"github.com/banzaicloud/pipeline/internal/cluster/metrics"
	"github.com/banzaicloud/pipeline/internal/providers/azure/pke"
	"github.com/banzaicloud/pipeline/internal/providers/azure/pke/workflow"
	intSecret "github.com/banzaicloud/pipeline/internal/secret"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	"github.com/banzaicloud/pipeline/pkg/k8sclient"
	pkgAzure "github.com/banzaicloud/pipeline/pkg/providers/azure"
	"github.com/banzaicloud/pipeline/secret"
)

func MakeAzurePKEClusterDeleter(events ClusterDeleterEvents, kubeProxyCache KubeProxyCache, logger logrus.FieldLogger, secrets SecretStore, statusChangeDurationMetric metrics.ClusterStatusChangeDurationMetric, store pke.AzurePKEClusterStore, workflowClient client.Client) AzurePKEClusterDeleter {
	return AzurePKEClusterDeleter{
		events:                     events,
		kubeProxyCache:             kubeProxyCache,
		logger:                     logger,
		secrets:                    secrets,
		statusChangeDurationMetric: statusChangeDurationMetric,
		store:                      store,
		workflowClient:             workflowClient,
	}
}

type AzurePKEClusterDeleter struct {
	events                     ClusterDeleterEvents
	kubeProxyCache             KubeProxyCache
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

type KubeProxyCache interface {
	Delete(clusterUID string)
}

func (cd AzurePKEClusterDeleter) Delete(ctx context.Context, cluster pke.PKEOnAzureCluster, forced bool) error {
	logger := cd.logger.WithField("clusterName", cluster.Name).WithField("clusterID", cluster.ID).WithField("forced", forced)
	logger.Info("deleting cluster")

	sir, err := cd.secrets.Get(cluster.OrganizationID, cluster.SecretID)
	if err = errors.WrapIf(err, "failed to get cluster secret from secret store"); err != nil {
		if forced {
			cd.logger.Error(err)
		} else {
			return err
		}
	}

	var loadBalancers []network.LoadBalancer
	var lbClient *pkgAzure.LoadBalancersClient
	var pipClient *pkgAzure.PublicIPAddressesClient

	if sir != nil {
		conn, err := pkgAzure.NewCloudConnection(&azure.PublicCloud, pkgAzure.NewCredentials(sir.Values))
		if err = errors.WrapIf(err, "failed to create cloud connection"); err != nil {
			if forced {
				cd.logger.Error(err)
			} else {
				return err
			}
		} else {
			lbClient = conn.GetLoadBalancersClient()
			pipClient = conn.GetPublicIPAddressesClient()
		}

		loadBalancers, err = collectClusterLoadBalancers(ctx, lbClient, cluster)
		if err = errors.Wrap(err, "couldn't list the load balancers used by the cluster"); err != nil {
			if forced {
				cd.logger.Error(err)
			} else {
				return err
			}
		}
	}

	pipNames, err := collectPublicIPAddressNames(ctx, logger, pipClient, cd.secrets, loadBalancers, cluster, forced)
	if err = errors.WrapIf(err, "failed to collect public IP address resource names"); err != nil {
		if forced {
			cd.logger.Error(err)
		} else {
			return err
		}
	}

	// delete only owned Load Balancers
	var loadBalancerNames []string
	for _, lb := range loadBalancers {
		if workflow.HasOwnedTag(cluster.Name, to.StringMap(lb.Tags)) {
			loadBalancerNames = append(loadBalancerNames, to.String(lb.Name))
		}
	}

	input := workflow.DeleteClusterWorkflowInput{
		OrganizationID:       cluster.OrganizationID,
		SecretID:             cluster.SecretID,
		ClusterID:            cluster.ID,
		ClusterName:          cluster.Name,
		ClusterUID:           cluster.UID,
		K8sSecretID:          cluster.K8sSecretID,
		ResourceGroupName:    cluster.ResourceGroup.Name,
		LoadBalancerNames:    loadBalancerNames,
		PublicIPAddressNames: pipNames,
		RouteTableName:       pke.GetRouteTableName(cluster.Name),
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
		return errors.WrapIf(err, "failed to set cluster status")
	}

	timer, err := cd.getClusterStatusChangeDurationTimer(cluster)
	if err = errors.WrapIf(err, "failed to start status change duration metric timer"); err != nil {
		if forced {
			cd.logger.Error(err)
			timer = metrics.NoopDurationMetricTimer{}
		} else {
			return err
		}
	}

	wfrun, err := cd.workflowClient.ExecuteWorkflow(ctx, workflowOptions, workflow.DeleteClusterWorkflowName, input)
	if err = errors.WrapIfWithDetails(err, "failed to start cluster deletion workflow", "cluster", cluster.Name); err != nil {
		_ = cd.store.SetStatus(cluster.ID, pkgCluster.Error, err.Error())
		return err
	}

	go func() {
		defer timer.RecordDuration()

		ctx := context.Background()

		if err := wfrun.Get(ctx, nil); err != nil {
			cd.logger.Errorf("cluster deleting workflow failed: %v", err)
			return
		}
		cd.kubeProxyCache.Delete(cluster.UID)
		cd.events.ClusterDeleted(cluster.OrganizationID, cluster.Name)
	}()

	if err = cd.store.SetActiveWorkflowID(cluster.ID, wfrun.GetID()); err != nil {
		return errors.WrapIfWithDetails(err, "failed to set active workflow ID for cluster", "cluster", cluster.Name, "workflowID", wfrun.GetID())
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
			return nil, errors.WrapIf(err, "Error during getting organization.")
		}
		values.OrganizationName = org.Name
		values.ClusterName = cluster.Name
	}
	return cd.statusChangeDurationMetric.StartTimer(values), nil
}

func (cd AzurePKEClusterDeleter) DeleteByID(ctx context.Context, clusterID uint, forced bool) error {
	cl, err := cd.store.GetByID(clusterID)
	if err != nil {
		return errors.WrapIf(err, "failed to load cluster from data store")
	}
	return cd.Delete(ctx, cl, forced)
}

func collectPublicIPAddressNames(ctx context.Context, logger logrus.FieldLogger, pipClient *pkgAzure.PublicIPAddressesClient, secrets SecretStore, loadBalancers []network.LoadBalancer, cluster pke.PKEOnAzureCluster, forced bool) ([]string, error) {
	names := make(map[string]bool)
	for _, lb := range loadBalancers {
		names = gatherOwnedPublicIPAddressNames(lb, cluster.Name, names)
	}

	pipList, err := pipClient.List(ctx, cluster.ResourceGroup.Name)
	if err != nil {
		return nil, errors.WrapIfWithDetails(err, "failed to list Azure public IP address resources in resource group", "resourceGroup", cluster.ResourceGroup.Name)
	}

	var pips []network.PublicIPAddress
	for pipList.NotDone() {
		for _, pip := range pipList.Values() {
			pips = append(pips, pip)
		}

		if err := errors.Wrap(pipList.NextWithContext(ctx), "failed to get Azure public IP address resources"); err != nil {
			if forced {
				logger.Warning(err)
			} else {
				return nil, err
			}
		}
	}

	names = gatherClusterPublicIPAddressNames(pips, cluster.Name, names)

	names, err = gatherK8sServicePublicIPs(pips, cluster, secrets, names)
	if err = errors.WrapIf(err, "failed to gather k8s services' public IP addresses"); err != nil {
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

// gatherClusterPublicIPAddressNames collect public ips that are tagged with kubernetes-cluster-name : <cluster name> which
// was introduced in Kubernetes version 1.15
func gatherClusterPublicIPAddressNames(publicAddresses []network.PublicIPAddress, clusterName string, names map[string]bool) map[string]bool {
	if names == nil {
		names = make(map[string]bool)
	}

	for _, pip := range publicAddresses {
		if v, ok := pip.Tags["kubernetes-cluster-name"]; ok && to.String(v) == clusterName {
			names[to.String(pip.Name)] = true
		}
	}

	return names
}

func gatherK8sServicePublicIPs(publicAddresses []network.PublicIPAddress, cluster pke.PKEOnAzureCluster, secrets SecretStore, names map[string]bool) (map[string]bool, error) {
	if cluster.K8sSecretID == "" {
		return names, nil
	}

	k8sConfig, err := intSecret.MakeKubeSecretStore(secrets).Get(cluster.OrganizationID, cluster.K8sSecretID)
	if err != nil {
		return names, errors.WrapIf(err, "failed to get k8s config")
	}

	ipToName := make(map[string]string)
	for _, pip := range publicAddresses {
		if to.String(pip.Name) != "" && to.String(pip.IPAddress) != "" {
			ipToName[to.String(pip.IPAddress)] = to.String(pip.Name)
		}
	}

	k8sClient, err := k8sclient.NewClientFromKubeConfig(k8sConfig)
	if err != nil {
		return names, errors.WrapIf(err, "failed to create a new Kubernetes client")
	}

	serviceList, err := k8sClient.CoreV1().Services(metav1.NamespaceAll).List(metav1.ListOptions{})
	if serviceList == nil || err != nil {
		return names, errors.WrapIf(err, "failed to retrieve service list")
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

func collectClusterLoadBalancers(ctx context.Context, client *pkgAzure.LoadBalancersClient, cluster pke.PKEOnAzureCluster) ([]network.LoadBalancer, error) {
	if client == nil {
		return nil, nil
	}
	lbs, err := client.List(ctx, cluster.ResourceGroup.Name)
	if err = errors.WrapIf(err, "failed to list load balancers"); err != nil {
		return nil, err
	}

	var clusterLoadBalancers []network.LoadBalancer
	for lbs.NotDone() {
		for _, lb := range lbs.Values() {
			if workflow.HasOwnedTag(cluster.Name, to.StringMap(lb.Tags)) || workflow.HasSharedTag(cluster.Name, to.StringMap(lb.Tags)) {
				lbDetails, err := client.Get(ctx, cluster.ResourceGroup.Name, to.String(lb.Name), "frontendIPConfigurations/publicIPAddress")
				if err != nil {
					if lb.StatusCode == http.StatusNotFound {
						continue
					}
					return nil, errors.WrapIf(err, "failed to retrieve load balancer")
				}

				clusterLoadBalancers = append(clusterLoadBalancers, lbDetails)
			}
		}
		err = lbs.NextWithContext(ctx)
		if err != nil {
			return nil, errors.WrapIf(err, "retrieving load balancers failed")
		}
	}
	return clusterLoadBalancers, nil
}
