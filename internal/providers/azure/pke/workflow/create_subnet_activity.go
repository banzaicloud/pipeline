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

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-10-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/goph/emperror"
	"go.uber.org/cadence/activity"
)

// CreateSubnetActivityName is the default registration name of the activity
const CreateSubnetActivityName = "pke-azure-create-subnet"

// CreateSubnetActivity represents an activity for creating an Azure subnet
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
	OrganizationID     uint
	SecretID           string
	ClusterName        string
	ResourceGroupName  string
	VirtualNetworkName string
	Subnet             Subnet
}

type CreateSubnetActivityOutput struct {
	SubnetID string
}

// Execute performs the activity
func (a CreateSubnetActivity) Execute(ctx context.Context, input CreateSubnetActivityInput) (output CreateSubnetActivityOutput, err error) {
	logger := activity.GetLogger(ctx).Sugar().With(
		"organization", input.OrganizationID,
		"cluster", input.ClusterName,
		"secret", input.SecretID,
		"resourceGroup", input.ResourceGroupName,
		"networkName", input.VirtualNetworkName,
		"subnetName", input.Subnet.Name,
	)

	keyvals := []interface{}{
		"resourceGroup", input.ResourceGroupName,
		"networkName", input.VirtualNetworkName,
		"subnetName", input.Subnet.Name,
	}

	logger.Info("create subnet")

	cc, err := a.azureClientFactory.New(input.OrganizationID, input.SecretID)
	if err = emperror.Wrap(err, "failed to create cloud connection"); err != nil {
		return
	}

	logger.Debug("sending request to create or update subnet")

	client := cc.GetSubnetsClient()

	var nsg *network.SecurityGroup
	if input.Subnet.NetworkSecurityGroupID != "" {
		nsg = &network.SecurityGroup{
			ID: to.StringPtr(input.Subnet.NetworkSecurityGroupID),
		}
	}
	params := network.Subnet{
		SubnetPropertiesFormat: &network.SubnetPropertiesFormat{
			AddressPrefix:        to.StringPtr(input.Subnet.CIDR),
			NetworkSecurityGroup: nsg,
			RouteTable: &network.RouteTable{
				ID: to.StringPtr(input.Subnet.RouteTableID),
			},
		},
	}

	future, err := client.CreateOrUpdate(ctx, input.ResourceGroupName, input.VirtualNetworkName, input.Subnet.Name, params)
	if err = emperror.WrapWith(err, "sending request to create or update subnet failed", keyvals...); err != nil {
		return
	}

	logger.Debug("waiting for the completion of create or update subnet operation")

	err = future.WaitForCompletionRef(ctx, client.Client)
	if err = emperror.WrapWith(err, "waiting for the completion of create or update subnet operation failed", keyvals...); err != nil {
		return
	}

	subnet, err := future.Result(client.SubnetsClient)
	if err = emperror.WrapWith(err, "getting virtual network create or update result failed", keyvals...); err != nil {
		return
	}

	output.SubnetID = to.String(subnet.ID)

	return
}
