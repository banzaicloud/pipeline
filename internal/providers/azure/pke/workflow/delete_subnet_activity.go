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

	"github.com/goph/emperror"
	"go.uber.org/cadence/activity"
)

// DeleteSubnetActivityName is the default registration name of the activity
const DeleteSubnetActivityName = "pke-azure-delete-subnet"

// DeleteSubnetActivity represents an activity for deleting a subnet
type DeleteSubnetActivity struct {
	azureClientFactory *AzureClientFactory
}

// DeleteSubnetActivityInput represents the input needed for executing a DeleteSubnetActivity
type DeleteSubnetActivityInput struct {
	OrganizationID    uint
	SecretID          string
	ClusterName       string
	ResourceGroupName string
	VNetName          string
	SubnetName        string
}

// MakeDeleteSubnetActivity returns a new DeleteSubnetActivity
func MakeDeleteSubnetActivity(azureClientFactory *AzureClientFactory) DeleteSubnetActivity {
	return DeleteSubnetActivity{
		azureClientFactory: azureClientFactory,
	}
}

func (a DeleteSubnetActivity) Execute(ctx context.Context, input DeleteSubnetActivityInput) (err error) {
	logger := activity.GetLogger(ctx).Sugar().With(
		"organization", input.OrganizationID,
		"resourceGroup", input.ResourceGroupName,
		"cluster", input.ClusterName,
		"vnet", input.VNetName,
		"subnet", input.SubnetName,
	)

	keyvals := []interface{}{
		"resourceGroup", input.ResourceGroupName,
		"vnet", input.VNetName,
		"subnet", input.SubnetName,
	}

	logger.Info("delete subnet")
	cc, err := a.azureClientFactory.New(input.OrganizationID, input.SecretID)
	if err = emperror.Wrap(err, "failed to create cloud connection"); err != nil {
		return
	}

	client := cc.GetSubnetsClient()

	// TODO: only delete subnet if it's owned by the cluster

	future, err := client.Delete(ctx, input.ResourceGroupName, input.VNetName, input.SubnetName)
	if err = emperror.WrapWith(err, "sending request to delete subnet failed", keyvals...); err != nil {
		if resp := future.Response(); resp != nil && resp.StatusCode == http.StatusNotFound {
			logger.Warn("subnet not found")
			return nil
		}
		return
	}

	logger.Debug("waiting for the completion of delete subnet operation")

	err = future.WaitForCompletionRef(ctx, client.Client)
	if err = emperror.WrapWith(err, "waiting for the completion of delete subnet operation failed", keyvals...); err != nil {
		if resp := future.Response(); resp != nil && resp.StatusCode == http.StatusNotFound {
			logger.Warn("subnet not found")
			return nil
		}
		return
	}

	logger.Debug("subnet deletion completed")

	return
}
