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

// CreateVnetActivityName is the default registration name of the activity
const CreateVnetActivityName = "pke-azure-create-vnet"

// CreateVnetActivity represents an activity for creating an Azure virtual network
type CreateVnetActivity struct {
	azureClientFactory *AzureClientFactory
}

// MakeCreateVnetActivity returns a new CreateVnetActivity
func MakeCreateVnetActivity(azureClientFactory *AzureClientFactory) CreateVnetActivity {
	return CreateVnetActivity{
		azureClientFactory: azureClientFactory,
	}
}

// NewCreateVnetActivity returns a new CreateVnetActivity
func NewCreateVnetActivity(azureClientFactory *AzureClientFactory) *CreateVnetActivity {
	a := MakeCreateVnetActivity(azureClientFactory)
	return &a
}

// CreateVnetActivityInput represents the input needed for executing a CreateVnetActivity
type CreateVnetActivityInput struct {
	Name     string
	CIDR     string
	Location string

	ResourceGroupName string
	OrganizationID    uint
	SecretID          string
}

// Execute performs the activity
func (a CreateVnetActivity) Execute(ctx context.Context, input CreateVnetActivityInput) error {
	logger := activity.GetLogger(ctx).Sugar().With(
		"organization", input.OrganizationID,
		"secret", input.SecretID,
		"resourceGroup", input.ResourceGroupName,
		"networkName", input.Name,
		"networkLocation", input.Location,
	)

	keyvals := []interface{}{
		"resourceGroup", input.ResourceGroupName,
		"networkName", input.Name,
	}

	logger.Info("create virtual network")

	cc, err := a.azureClientFactory.New(input.OrganizationID, input.SecretID)
	if err != nil {
		return emperror.Wrap(err, "failed to create cloud connection")
	}

	tags := resourceTags(map[string]string{
		"creator": "banzai-pipeline",
	})

	cidrs := []string{input.CIDR}

	params := network.VirtualNetwork{
		Location: &input.Location,
		VirtualNetworkPropertiesFormat: &network.VirtualNetworkPropertiesFormat{
			AddressSpace: &network.AddressSpace{
				AddressPrefixes: &cidrs,
			},
		},
		Tags: tags,
	}

	logger.Debug("sending request to create or update virtual network")

	client := cc.GetVirtualNetworksClient()

	future, err := client.CreateOrUpdate(ctx, input.ResourceGroupName, input.Name, params)
	if err != nil {
		return emperror.WrapWith(err, "sending request to create or update virtual network failed", keyvals...)
	}

	logger.Debug("waiting for the completion of create or update virtual network operation")

	err = future.WaitForCompletionRef(ctx, client.Client)
	if err != nil {
		return emperror.WrapWith(err, "waiting for the completion of create or update virtual network operation failed", keyvals...)
	}

	return nil
}

// resourceTags converts map[string]string to map[string]*string
func resourceTags(tags map[string]string) map[string]*string {
	azTags := make(map[string]*string, len(tags))
	for k, v := range tags {
		v := v
		azTags[k] = &v
	}

	return azTags
}
