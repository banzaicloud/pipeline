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

// DeletePublicIPActivityName is the default registration name of the activity
const DeletePublicIPActivityName = "pke-azure-delete-public-ip"

// DeletePublicIPActivity represents an activity for deleting a public IP
type DeletePublicIPActivity struct {
	azureClientFactory *AzureClientFactory
}

// DeleteVNetActivityInput represents the input needed for executing a DeletePublicIPActivity
type DeletePublicIPActivityInput struct {
	OrganizationID      uint
	SecretID            string
	ClusterName         string
	ResourceGroupName   string
	PublicIPAddressName string
}

// MakeDeletePublicIPActivity returns a new DeletePublicIPActivity
func MakeDeletePublicIPActivity(azureClientFactory *AzureClientFactory) DeletePublicIPActivity {
	return DeletePublicIPActivity{
		azureClientFactory: azureClientFactory,
	}
}

func (a DeletePublicIPActivity) Execute(ctx context.Context, input DeletePublicIPActivityInput) (err error) {
	logger := activity.GetLogger(ctx).Sugar().With(
		"organization", input.OrganizationID,
		"cluster", input.ClusterName,
		"resourceGroup", input.ResourceGroupName,
		"publicIPAddress", input.PublicIPAddressName,
	)

	keyvals := []interface{}{
		"resourceGroup", input.ResourceGroupName,
		"publicIPAddress", input.PublicIPAddressName,
	}

	logger.Info("delete public ip")

	cc, err := a.azureClientFactory.New(input.OrganizationID, input.SecretID)
	if err = emperror.Wrap(err, "failed to create cloud connection"); err != nil {
		return
	}

	client := cc.GetPublicIPAddressesClient()

	logger.Debug("get public ip details")

	pip, err := client.Get(ctx, input.ResourceGroupName, input.PublicIPAddressName, "")
	if err != nil {
		if pip.StatusCode == http.StatusNotFound {
			logger.Warn("public ip not found")
			return nil
		}

		return emperror.WrapWith(err, "failed to get public ip details", keyvals...)
	}

	if !hasOwnedTag(input.ClusterName, to.StringMap(pip.Tags)) {
		logger.Info("skip deleting public ip as it's not owned by cluster")
		return
	}

	pipProvisioningState := network.ProvisioningState(to.String(pip.ProvisioningState))
	if pipProvisioningState == network.Deleting || pipProvisioningState == network.Updating {
		return fmt.Errorf("can not delete public ip in %q provisioning state", pipProvisioningState)
	}

	future, err := client.Delete(ctx, input.ResourceGroupName, input.PublicIPAddressName)
	if err = emperror.WrapWith(err, "sending request to delete public ip failed", keyvals...); err != nil {
		return
	}

	logger.Debug("waiting for the completion of delete public ip operation")

	err = future.WaitForCompletionRef(ctx, client.Client)
	if err = emperror.WrapWith(err, "waiting for the completion of delete public ip operation failed", keyvals...); err != nil {
		return
	}

	logger.Debug("public ip deletion completed")

	return
}
