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

package workflow

import (
	"context"
	"fmt"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-10-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/goph/emperror"
	"go.uber.org/cadence/activity"
)

// DeleteVNetActivityName is the default registration name of the activity
const DeleteVNetActivityName = "pke-azure-delete-vnet"

// DeleteVNetActivity represents an activity for deleting a VNet
type DeleteVNetActivity struct {
	azureClientFactory *AzureClientFactory
}

// DeleteVNetActivityInput represents the input needed for executing a DeleteVNetActivity
type DeleteVNetActivityInput struct {
	OrganizationID    uint
	SecretID          string
	ClusterName       string
	ResourceGroupName string
	VNetName          string
}

// MakeDeleteVNetActivity returns a new DeleteVNetActivity
func MakeDeleteVNetActivity(azureClientFactory *AzureClientFactory) DeleteVNetActivity {
	return DeleteVNetActivity{
		azureClientFactory: azureClientFactory,
	}
}

func (a DeleteVNetActivity) Execute(ctx context.Context, input DeleteVNetActivityInput) (err error) {
	logger := activity.GetLogger(ctx).Sugar().With(
		"organization", input.OrganizationID,
		"resourceGroup", input.ResourceGroupName,
		"cluster", input.ClusterName,
		"vnet", input.VNetName,
	)

	keyvals := []interface{}{
		"resourceGroup", input.ResourceGroupName,
		"vnet", input.VNetName,
	}

	logger.Info("delete virtual network")
	cc, err := a.azureClientFactory.New(input.OrganizationID, input.SecretID)
	if err = emperror.Wrap(err, "failed to create cloud connection"); err != nil {
		return
	}

	client := cc.GetVirtualNetworksClient()

	// delete virtual network only of owned by current cluster
	logger.Debug("get virtual network details")

	vnet, err := client.Get(ctx, input.ResourceGroupName, input.VNetName, "")
	if err != nil {
		if vnet.StatusCode == http.StatusNotFound {
			logger.Warn("virtual network not found")
			return nil
		}

		return emperror.WrapWith(err, "failed to get virtual network details", keyvals...)
	}

	if !hasOwnedTag(input.ClusterName, to.StringMap(vnet.Tags)) {
		logger.Info("skip deleting virtual network as it's not owned by cluster")
		return
	}

	vnetProvisioningState := network.ProvisioningState(to.String(vnet.ProvisioningState))
	if vnetProvisioningState == network.Deleting || vnetProvisioningState == network.Updating {
		return fmt.Errorf("can not delete virtual network in %q provisioning state", vnetProvisioningState)
	}

	future, err := client.Delete(ctx, input.ResourceGroupName, input.VNetName)
	if err = emperror.WrapWith(err, "sending request to delete virtual network failed", keyvals...); err != nil {
		return
	}

	logger.Debug("waiting for the completion of delete virtual network operation")

	err = future.WaitForCompletionRef(ctx, client.Client)
	if err = emperror.WrapWith(err, "waiting for the completion of delete virtual network operation failed", keyvals...); err != nil {
		return
	}

	logger.Debug("virtual network deletion completed")

	return
}
