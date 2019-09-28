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

	"emperror.dev/errors"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-10-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	"go.uber.org/cadence/activity"
)

// AssignRoleActivityName is the default registration name of the activity
const CreatePublicIPActivityName = "pke-azure-create-public-ip"

// AssignRoleActivity represents an activity for creating an Azure network security group
type CreatePublicIPActivity struct {
	azureClientFactory *AzureClientFactory
}

// MakeAssignRoleActivity returns a new CreateNSGActivity
func MakeCreatePublicIPActivity(azureClientFactory *AzureClientFactory) CreatePublicIPActivity {
	return CreatePublicIPActivity{
		azureClientFactory: azureClientFactory,
	}
}

type PublicIPAddress struct {
	Location string
	Name     string
	SKU      string
}

type CreatePublicIPActivityOutput struct {
	PublicIPAddressID string
	PublicIPAddress   string
	Name              string
}

type CreatePublicIPActivityInput struct {
	OrganizationID    uint
	SecretID          string
	ClusterName       string
	ResourceGroupName string
	PublicIPAddress   PublicIPAddress
}

func (a CreatePublicIPActivity) Execute(ctx context.Context, input CreatePublicIPActivityInput) (output CreatePublicIPActivityOutput, err error) {
	logger := activity.GetLogger(ctx).Sugar().With(
		"organization", input.OrganizationID,
		"cluster", input.ClusterName,
		"secret", input.SecretID,
		"resourceGroup", input.ResourceGroupName,
	)

	keyvals := []interface{}{
		"resourceGroup", input.ResourceGroupName,
	}

	cc, err := a.azureClientFactory.New(input.OrganizationID, input.SecretID)
	if err = errors.WrapIf(err, "failed to create cloud connection"); err != nil {
		return
	}

	params := input.getCreateOrUpdatePublicIPAddressParams()

	client := cc.GetPublicIPAddressesClient()

	future, err := client.CreateOrUpdate(ctx, input.ResourceGroupName, input.PublicIPAddress.Name, params)
	if err = errors.WrapIfWithDetails(err, "sending request to create or update public ip failed", keyvals...); err != nil {
		return
	}

	logger.Debug("waiting for the completion of create or update load public ip operation")

	err = future.WaitForCompletionRef(ctx, client.Client)
	if err = errors.WrapIfWithDetails(err, "waiting for the completion of create or update load public ip operation failed", keyvals...); err != nil {
		return
	}

	publicIP, err := future.Result(client.PublicIPAddressesClient)
	if err = errors.WrapIfWithDetails(err, "getting load balancer create or update result failed", keyvals...); err != nil {
		return
	}

	output.PublicIPAddressID = to.String(publicIP.ID)
	output.PublicIPAddress = to.String(publicIP.IPAddress)
	output.Name = to.String(publicIP.Name)

	return
}

func (input CreatePublicIPActivityInput) getCreateOrUpdatePublicIPAddressParams() network.PublicIPAddress {
	return network.PublicIPAddress{
		Location: to.StringPtr(input.PublicIPAddress.Location),
		PublicIPAddressPropertiesFormat: &network.PublicIPAddressPropertiesFormat{
			PublicIPAddressVersion:   network.IPv4,
			PublicIPAllocationMethod: network.Static,
		},
		Sku: &network.PublicIPAddressSku{
			Name: network.PublicIPAddressSkuName(input.PublicIPAddress.SKU),
		},
		Tags: getClusterTags(input.ClusterName),
	}
}
