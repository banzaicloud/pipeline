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
	"net"
	"net/http"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-01-01/network"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/banzaicloud/pipeline/internal/providers/azure/pke"
	"github.com/banzaicloud/pipeline/internal/providers/azure/pke/workflow"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	"github.com/banzaicloud/pipeline/pkg/k8sclient"
	pkgAzure "github.com/banzaicloud/pipeline/pkg/providers/azure"
	pkgSecret "github.com/banzaicloud/pipeline/pkg/secret"
	"github.com/banzaicloud/pipeline/secret"
	"github.com/goph/emperror"
	"github.com/sirupsen/logrus"
	"go.uber.org/cadence"
	"go.uber.org/cadence/client"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func MakeAzurePKEClusterDeleter(logger logrus.FieldLogger, store pke.AzurePKEClusterStore, workflowClient client.Client) AzurePKEClusterDeleter {
	return AzurePKEClusterDeleter{
		logger:         logger,
		store:          store,
		workflowClient: workflowClient,
	}
}

type AzurePKEClusterDeleter struct {
	logger         logrus.FieldLogger
	store          pke.AzurePKEClusterStore
	workflowClient client.Client
}

func (cd AzurePKEClusterDeleter) Delete(ctx context.Context, cluster pke.PKEOnAzureCluster, force bool) error {
	logger := cd.logger.WithField("clusterName", cluster.Name).WithField("clusterID", cluster.ID).WithField("force", force)
	logger.Info("Deleting cluster")

	if err := cd.store.SetStatus(cluster.ID, pkgCluster.Deleting, pkgCluster.DeletingMessage); err != nil {
		return emperror.Wrap(err, "failed to set cluster status")
	}

	// TODO: do not use global secret store
	sir, err := secret.Store.Get(cluster.OrganizationID, cluster.SecretID)
	if err != nil {
		return emperror.Wrap(err, "failed to get secret")
	}
	cc, err := pkgAzure.NewCloudConnection(&azure.PublicCloud, pkgAzure.NewCredentials(sir.Values))
	if err != nil {
		return emperror.Wrap(err, "failed to create cloud connection")
	}

	var k8sConfig []byte
	if len(cluster.K8sSecretID) > 0 {
		k8sConfigSir, err := secret.Store.Get(cluster.OrganizationID, cluster.K8sSecretID)
		if err != nil {
			return emperror.WrapWith(err, "can't get config from Vault", "secretID", cluster.K8sSecretID)
		}
		k8sConfig, err = base64.StdEncoding.DecodeString(k8sConfigSir.GetValue(pkgSecret.K8SConfig))
		if err != nil {
			return emperror.WrapWith(err, "can't decode Kubernetes config", "secretID", cluster.K8sSecretID)
		}
	}

	lb, err := cc.GetLoadBalancersClient().Get(ctx, cluster.ResourceGroup.Name, cluster.Name, "frontendIPConfigurations/publicIPAddress")
	if err != nil && lb.StatusCode != http.StatusNotFound {
		return emperror.Wrap(err, "failed to retrieve load balancer")
	}

	ssns := make([]string, len(cluster.NodePools))
	for i, np := range cluster.NodePools {
		ssns[i] = pke.GetVMSSName(cluster.Name, np.Name)
	}

	servicePips, err := getServicesPublicIPs(logger, k8sConfig)
	if err != nil {
		if !force {
			return emperror.Wrap(err, "failed to retrieve services public IPs")
		}

		logger.Warningln(emperror.Wrap(err, "failed to retrieve services public IPs"), " - continue deletion flow")
	}

	input := workflow.DeleteClusterWorkflowInput{
		OrganizationID:       cluster.OrganizationID,
		SecretID:             cluster.SecretID,
		ClusterID:            cluster.ID,
		ClusterName:          cluster.Name,
		ResourceGroupName:    cluster.ResourceGroup.Name,
		LoadBalancerName:     cluster.Name, // must be the same as the value passed to pke install master --kubernetes-cluster-name
		PublicIPAddressNames: collectPublicIPAddressNames(logger, lb, cluster.Name, servicePips),
		RouteTableName:       cluster.Name + "-route-table",
		ScaleSetNames:        ssns,
		SecurityGroupNames:   []string{cluster.Name + "-master-nsg", cluster.Name + "-worker-nsg"},
		VirtualNetworkName:   cluster.VirtualNetwork.Name,
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

	wfexec, err := cd.workflowClient.StartWorkflow(ctx, workflowOptions, workflow.DeleteClusterWorkflowName, input)
	if err != nil {
		return emperror.WrapWith(err, "failed to start cluster deletion workflow", "cluster", cluster.Name)
	}

	if err = cd.store.SetActiveWorkflowID(cluster.ID, wfexec.ID); err != nil {
		return emperror.WrapWith(err, "failed to set active workflow ID for cluster", "cluster", cluster.Name, "workflowID", wfexec.ID)
	}

	return nil
}

func (cd AzurePKEClusterDeleter) DeleteByID(ctx context.Context, clusterID uint, force bool) error {
	cl, err := cd.store.GetByID(clusterID)
	if err != nil {
		return emperror.Wrap(err, "failed to load cluster from data store")
	}
	return cd.Delete(ctx, cl, force)
}

func collectPublicIPAddressNames(logger logrus.FieldLogger, lb network.LoadBalancer, clusterName string, servicesPips []string) []string {
	names := make(map[string]bool)

	if lb.LoadBalancerPropertiesFormat != nil {
		if fics := lb.LoadBalancerPropertiesFormat.FrontendIPConfigurations; fics != nil {
			for _, fic := range *fics {
				if fic.FrontendIPConfigurationPropertiesFormat != nil {
					if pip := fic.FrontendIPConfigurationPropertiesFormat.PublicIPAddress; pip != nil {
						if name := to.String(pip.Name); name != "" {

							if workflow.HasOwnedTag(clusterName, to.StringMap(pip.Tags)) {
								names[name] = true
							} else {
								for i := range servicesPips {
									if servicesPips[i] == to.String(pip.IPAddress) {
										names[name] = true
									}
								}
							}
						}
					}
				}
			}
		}
	}

	result := make([]string, 0, len(names))
	for name := range names {
		logger.Debugln("mark pip for deletion", name)
		result = append(result, name)
	}
	return result
}

func getServicesPublicIPs(logger logrus.FieldLogger, k8sConfig []byte) ([]string, error) {
	if k8sConfig == nil {
		return nil, nil
	}

	client, err := k8sclient.NewClientFromKubeConfig(k8sConfig)
	if err != nil {
		return nil, emperror.Wrap(err, "failed to create a new Kubernetes client")
	}

	serviceList, err := client.CoreV1().Services(metav1.NamespaceAll).List(metav1.ListOptions{})
	if err != nil {
		return nil, emperror.Wrap(err, "failed to retrieve service list")
	}

	pips := make(map[string]bool)
	for _, service := range serviceList.Items {
		if service.Spec.Type == corev1.ServiceTypeLoadBalancer {
			for _, ing := range service.Status.LoadBalancer.Ingress {
				if ing.IP != "" && net.ParseIP(ing.IP) != nil {
					pips[ing.IP] = true
				}
			}
		}
	}

	publicIPs := make([]string, len(pips))
	for k := range pips {
		publicIPs = append(publicIPs, k)
	}

	return publicIPs, nil
}
