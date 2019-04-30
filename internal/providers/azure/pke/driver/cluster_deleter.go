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
	"time"

	"github.com/banzaicloud/pipeline/secret"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-01-01/network"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/banzaicloud/pipeline/internal/providers/azure/pke"
	"github.com/banzaicloud/pipeline/internal/providers/azure/pke/workflow"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	pkgAzure "github.com/banzaicloud/pipeline/pkg/providers/azure"
	"github.com/goph/emperror"
	"github.com/sirupsen/logrus"
	"go.uber.org/cadence/client"
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

func (cd AzurePKEClusterDeleter) Delete(ctx context.Context, cluster pke.PKEOnAzureCluster) error {
	cd.logger.WithField("clusterName", cluster.Name).WithField("clusterID", cluster.ID).Info("Deleting cluster")

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
	lb, err := cc.GetLoadBalancersClient().Get(ctx, cluster.ResourceGroup.Name, cluster.Name, "frontendIPConfigurations/publicIPAddress")
	if err != nil {
		return emperror.Wrap(err, "failed to retrieve load balancer")
	}

	input := workflow.DeleteClusterWorkflowInput{
		OrganizationID:       cluster.OrganizationID,
		SecretID:             cluster.SecretID,
		ClusterID:            cluster.ID,
		ClusterName:          cluster.Name,
		ResourceGroupName:    cluster.ResourceGroup.Name,
		LoadBalancerName:     cluster.Name, // must be the same as the value passed to pke install master --kubernetes-cluster-name
		PublicIPAddressNames: collectPublicIPAddressNames(lb),
		RouteTableName:       cluster.Name + "-route-table",
		ScaleSetNames:        []string{cluster.Name + "-master-vmss", cluster.Name + "-worker-vmss"},
		SecurityGroupNames:   []string{cluster.Name + "-master-nsg", cluster.Name + "-worker-nsg"},
		VirtualNetworkName:   cluster.Name + "-vnet",
	}

	workflowOptions := client.StartWorkflowOptions{
		TaskList:                     "pipeline",
		ExecutionStartToCloseTimeout: 40 * time.Minute, // TODO: lower timeout
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

func (cd AzurePKEClusterDeleter) DeleteByID(ctx context.Context, clusterID uint) error {
	cl, err := cd.store.GetByID(clusterID)
	if err != nil {
		return emperror.Wrap(err, "failed to load cluster from data store")
	}
	return cd.Delete(ctx, cl)
}

func collectPublicIPAddressNames(lb network.LoadBalancer) []string {
	names := make(map[string]bool)

	if lb.LoadBalancerPropertiesFormat != nil {
		if fics := lb.LoadBalancerPropertiesFormat.FrontendIPConfigurations; fics != nil {
			for _, fic := range *fics {
				if fic.FrontendIPConfigurationPropertiesFormat != nil {
					if pip := fic.FrontendIPConfigurationPropertiesFormat.PublicIPAddress; pip != nil {
						if name := to.String(pip.Name); name != "" {
							names[name] = true
						}
					}
				}
			}
		}
	}

	result := make([]string, 0, len(names))
	for name := range names {
		result = append(result, name)
	}
	return result
}
