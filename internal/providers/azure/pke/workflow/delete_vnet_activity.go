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

	"emperror.dev/errors"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-10-01/network"
	"github.com/Azure/go-autorest/autorest/to"
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
	if err = errors.WrapIf(err, "failed to create cloud connection"); err != nil {
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

		return errors.WrapIfWithDetails(err, "failed to get virtual network details", keyvals...)
	}

	tags := to.StringMap(vnet.Tags)
	if !HasOwnedTag(input.ClusterName, tags) {
		logger.Info("skip deleting virtual network as it's not owned by cluster")

		tags = RemoveSharedTag(tags, input.ClusterName)
		future, err := client.UpdateTags(ctx, input.ResourceGroupName, input.VNetName, network.TagsObject{Tags: *to.StringMapPtr(tags)})
		if err = errors.WrapIfWithDetails(err, "sending request to update virtual network tags failed", keyvals...); err != nil {
			if resp := future.Response(); resp != nil && resp.StatusCode == http.StatusNotFound {
				logger.Warn("virtual network not found")
				return nil
			}
			return err
		}
		err = future.WaitForCompletionRef(ctx, client.Client)
		if err = errors.WrapIfWithDetails(err, "waiting for the completion of virtual network tags update operation failed", keyvals...); err != nil {
			if resp := future.Response(); resp != nil && resp.StatusCode == http.StatusNotFound {
				logger.Warn("virtual network not found")
				return nil
			}
			return err
		}
		return nil
	}

	future, err := client.Delete(ctx, input.ResourceGroupName, input.VNetName)
	if err = errors.WrapIfWithDetails(err, "sending request to delete virtual network failed", keyvals...); err != nil {
		if resp := future.Response(); resp != nil && resp.StatusCode == http.StatusNotFound {
			logger.Warn("virtual network not found")
			return nil
		}
		return
	}

	logger.Debug("waiting for the completion of delete virtual network operation")

	err = future.WaitForCompletionRef(ctx, client.Client)
	if err = errors.WrapIfWithDetails(err, "waiting for the completion of delete virtual network operation failed", keyvals...); err != nil {
		if resp := future.Response(); resp != nil && resp.StatusCode == http.StatusNotFound {
			logger.Warn("virtual network not found")
			return nil
		}
		return
	}

	logger.Debug("virtual network deletion completed")

	return
}
