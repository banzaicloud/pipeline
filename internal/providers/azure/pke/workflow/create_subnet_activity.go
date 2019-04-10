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

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-01-01/network"
	"github.com/goph/emperror"
	"go.uber.org/cadence/activity"
)

// CreateSubnetActivityName is the default registration name of the activity
const CreateSubnetActivityName = "pke-azure-create-subnet"

// CreateSubnetActivity represents an activity for creating an Azure virtual subnetwork
type CreateSubnetActivity struct {
	azureClientFactory *AzureClientFactory
}

// MakeCreateSubnetActivity returns a new CreateSubnetActivity
func MakeCreateSubnetActivity(azureClientFactory *AzureClientFactory) CreateSubnetActivity {
	return CreateSubnetActivity{
		azureClientFactory: azureClientFactory,
	}
}

// CreateSubnetActivityInput represents the input needed for executing a CreateSubnetActivity
type CreateSubnetActivityInput struct {
	Name string
	CIDR string

	VirtualNetworkName string
	ResourceGroupName  string
	OrganizationID     uint
	SecretID           string
}

// Execute performs the activity
func (a CreateSubnetActivity) Execute(ctx context.Context, input CreateSubnetActivityInput) error {
	logger := activity.GetLogger(ctx).Sugar().With(
		"organization", input.OrganizationID,
		"secret", input.SecretID,
		"resourceGroup", input.ResourceGroupName,
		"networkName", input.VirtualNetworkName,
		"subnetName", input.Name,
	)

	keyvals := []interface{}{
		"resourceGroup", input.ResourceGroupName,
		"networkName", input.VirtualNetworkName,
		"subnetName", input.Name,
	}

	logger.Info("create virtual subnetwork")

	cc, err := a.azureClientFactory.New(input.OrganizationID, input.SecretID)
	if err != nil {
		return emperror.Wrap(err, "failed to create cloud connection")
	}

	client := cc.GetSubnetsClient()

	params := network.Subnet{
		SubnetPropertiesFormat: &network.SubnetPropertiesFormat{
			AddressPrefix: &input.CIDR,
		},
	}
	future, err := client.CreateOrUpdate(ctx, input.ResourceGroupName, input.VirtualNetworkName, input.Name, params)
	if err != nil {
		return emperror.WrapWith(err, "sending request to create or update virtual subnetwork failed", keyvals...)
	}

	logger.Debug("waiting for the completion of create or update virtual subnetwork operation")

	err = future.WaitForCompletionRef(ctx, client.Client)
	if err != nil {
		return emperror.WrapWith(err, "waiting for the completion of create or update virtual subnetwork operation failed", keyvals...)
	}

	return nil
}
