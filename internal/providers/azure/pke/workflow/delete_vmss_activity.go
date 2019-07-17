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
	"net/http"

	"emperror.dev/emperror"
	"github.com/Azure/go-autorest/autorest/to"
	"go.uber.org/cadence/activity"
)

// DeleteVMSSActivityName is the default registration name of the activity
const DeleteVMSSActivityName = "pke-azure-delete-vmss"

// DeleteVMSSActivity represents an activity for deleting a VMSS
type DeleteVMSSActivity struct {
	azureClientFactory *AzureClientFactory
}

// DeleteVMSSActivityInput represents the input needed for executing a DeleteVMSSActivity
type DeleteVMSSActivityInput struct {
	OrganizationID    uint
	SecretID          string
	ClusterName       string
	ResourceGroupName string
	VMSSName          string
}

// MakeDeleteVMSSActivity returns a new DeleteVMSSActivity
func MakeDeleteVMSSActivity(azureClientFactory *AzureClientFactory) DeleteVMSSActivity {
	return DeleteVMSSActivity{
		azureClientFactory: azureClientFactory,
	}
}

func (a DeleteVMSSActivity) Execute(ctx context.Context, input DeleteVMSSActivityInput) (err error) {
	logger := activity.GetLogger(ctx).Sugar().With(
		"organization", input.OrganizationID,
		"resourceGroup", input.ResourceGroupName,
		"cluster", input.ClusterName,
		"vmssName", input.VMSSName,
	)

	keyvals := []interface{}{
		"resourceGroup", input.ResourceGroupName,
		"vmssName", input.VMSSName,
	}

	logger.Info("delete virtual machine scale set")
	cc, err := a.azureClientFactory.New(input.OrganizationID, input.SecretID)
	if err = emperror.Wrap(err, "failed to create cloud connection"); err != nil {
		return
	}

	client := cc.GetVirtualMachineScaleSetsClient()

	// delete virtual machine scale set only of owned by current cluster
	logger.Debug("get virtual machine scale set details")

	vmss, err := client.Get(ctx, input.ResourceGroupName, input.VMSSName)
	if err != nil {
		if vmss.StatusCode == http.StatusNotFound {
			logger.Warn("virtual machine scale set not found")
			return nil
		}

		return emperror.WrapWith(err, "failed to get virtual machine scale set details", keyvals...)
	}

	if !HasOwnedTag(input.ClusterName, to.StringMap(vmss.Tags)) {
		logger.Info("skip deleting virtual machine scale set as it's not owned by cluster")
		return
	}

	future, err := client.Delete(ctx, input.ResourceGroupName, input.VMSSName)
	if err = emperror.WrapWith(err, "sending request to delete virtual machine scale set failed", keyvals...); err != nil {
		if resp := future.Response(); resp != nil && resp.StatusCode == http.StatusNotFound {
			logger.Warn("virtual machine scale set not found")
			return nil
		}
		return
	}

	logger.Debug("waiting for the completion of delete virtual machine scale set operation")

	err = future.WaitForCompletionRef(ctx, client.Client)
	if err = emperror.WrapWith(err, "waiting for the completion of delete virtual machine scale set operation failed", keyvals...); err != nil {
		if resp := future.Response(); resp != nil && resp.StatusCode == http.StatusNotFound {
			logger.Warn("virtual machine scale set not found")
			return nil
		}
		return
	}

	logger.Debug("virtual machine scale set deletion completed")

	return
}
