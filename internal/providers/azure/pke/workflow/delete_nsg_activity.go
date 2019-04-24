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

// DeleteNSGActivityName is the default registration name of the activity
const DeleteNSGActivityName = "pke-azure-delete-nsg"

// DeleteNSGActivity represents an activity for deleting a network security group
type DeleteNSGActivity struct {
	azureClientFactory *AzureClientFactory
}

// DeleteNSGActivityInput represents the input needed for executing a DeleteNSGActivity
type DeleteNSGActivityInput struct {
	OrganizationID    uint
	SecretID          string
	ClusterName       string
	ResourceGroupName string
	NSGName           string
}

// MakeDeleteNSGActivity returns a new DeleteNSGActivity
func MakeDeleteNSGActivity(azureClientFactory *AzureClientFactory) DeleteNSGActivity {
	return DeleteNSGActivity{
		azureClientFactory: azureClientFactory,
	}
}

func (a DeleteNSGActivity) Execute(ctx context.Context, input DeleteNSGActivityInput) (err error) {
	logger := activity.GetLogger(ctx).Sugar().With(
		"organization", input.OrganizationID,
		"cluster", input.ClusterName,
		"resourceGroup", input.ResourceGroupName,
		"nsgName", input.NSGName,
	)

	keyvals := []interface{}{
		"resourceGroup", input.ResourceGroupName,
		"nsgName", input.NSGName,
	}

	logger.Info("delete network security group")
	cc, err := a.azureClientFactory.New(input.OrganizationID, input.SecretID)
	if err = emperror.Wrap(err, "failed to create cloud connection"); err != nil {
		return
	}

	client := cc.GetSecurityGroupsClient()

	// delete network security group only if owned by current cluster
	logger.Debug("get network security group details")

	rt, err := client.Get(ctx, input.ResourceGroupName, input.NSGName, "")
	if err != nil {
		if rt.StatusCode == http.StatusNotFound {
			logger.Warn("network security group not found")
			return nil
		}

		return emperror.WrapWith(err, "failed to get network security group details", keyvals...)
	}

	if !hasOwnedTag(input.ClusterName, to.StringMap(rt.Tags)) {
		logger.Info("skip deleting route table as it's not owned by cluster")
		return
	}

	nsgProvisioningState := network.ProvisioningState(to.String(rt.ProvisioningState))
	if nsgProvisioningState == network.Deleting || nsgProvisioningState == network.Updating {
		return fmt.Errorf("can not delete network security group in %q provisioning state", nsgProvisioningState)
	}

	future, err := client.Delete(ctx, input.ResourceGroupName, input.NSGName)
	if err = emperror.WrapWith(err, "sending request to network security group failed", keyvals...); err != nil {
		return
	}

	logger.Debug("waiting for the completion of delete network security group operation")

	err = future.WaitForCompletionRef(ctx, client.Client)
	if err = emperror.WrapWith(err, "waiting for the completion of delete network security group operation failed", keyvals...); err != nil {
		return
	}

	logger.Debug("network security group deletion completed")

	return
}
