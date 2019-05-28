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

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-10-01/compute"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/goph/emperror"
	"go.uber.org/cadence/activity"
)

// UpdateVMSSActivityName is the default registration name of the activity
const UpdateVMSSActivityName = "pke-azure-update-vmss"

// UpdateVMSSActivity represents an activity for deleting a VMSS
type UpdateVMSSActivity struct {
	azureClientFactory *AzureClientFactory
}

// UpdateVMSSActivityInput represents the input needed for executing a UpdateVMSSActivity
type UpdateVMSSActivityInput struct {
	OrganizationID    uint
	SecretID          string
	ClusterName       string
	ResourceGroupName string
	Changes           VirtualMachineScaleSetChanges
}

type VirtualMachineScaleSetChanges struct {
	Name          string
	InstanceCount uint
}

// MakeUpdateVMSSActivity returns a new UpdateVMSSActivity
func MakeUpdateVMSSActivity(azureClientFactory *AzureClientFactory) UpdateVMSSActivity {
	return UpdateVMSSActivity{
		azureClientFactory: azureClientFactory,
	}
}

func (a UpdateVMSSActivity) Execute(ctx context.Context, input UpdateVMSSActivityInput) (err error) {
	logger := activity.GetLogger(ctx).Sugar().With(
		"organization", input.OrganizationID,
		"resourceGroup", input.ResourceGroupName,
		"cluster", input.ClusterName,
		"vmssName", input.Changes.Name,
	)

	keyvals := []interface{}{
		"resourceGroup", input.ResourceGroupName,
		"vmssName", input.Changes.Name,
	}

	logger.Info("update virtual machine scale set")
	cc, err := a.azureClientFactory.New(input.OrganizationID, input.SecretID)
	if err = emperror.Wrap(err, "failed to create cloud connection"); err != nil {
		return
	}

	client := cc.GetVirtualMachineScaleSetsClient()

	// update virtual machine scale set only of owned by current cluster
	logger.Debug("get virtual machine scale set details")

	vmss, err := client.Get(ctx, input.ResourceGroupName, input.Changes.Name)
	if err != nil {
		if vmss.StatusCode == http.StatusNotFound {
			logger.Warn("virtual machine scale set not found")
			return nil
		}

		return emperror.WrapWith(err, "failed to get virtual machine scale set details", keyvals...)
	}

	if !HasOwnedTag(input.ClusterName, to.StringMap(vmss.Tags)) {
		logger.Info("skip updating virtual machine scale set as it's not owned by cluster")
		return
	}

	future, err := client.Update(ctx, input.ResourceGroupName, input.Changes.Name, compute.VirtualMachineScaleSetUpdate{
		Sku: &compute.Sku{
			Capacity: to.Int64Ptr(int64(input.Changes.InstanceCount)),
		},
	})
	if err = emperror.WrapWith(err, "sending request to update virtual machine scale set failed", keyvals...); err != nil {
		if resp := future.Response(); resp != nil && resp.StatusCode == http.StatusNotFound {
			logger.Warn("virtual machine scale set not found")
			return nil
		}
		return
	}

	logger.Debug("waiting for the completion of update virtual machine scale set operation")

	err = future.WaitForCompletionRef(ctx, client.Client)
	if err = emperror.WrapWith(err, "waiting for the completion of update virtual machine scale set operation failed", keyvals...); err != nil {
		if resp := future.Response(); resp != nil && resp.StatusCode == http.StatusNotFound {
			logger.Warn("virtual machine scale set not found")
			return nil
		}
		return
	}

	return
}
