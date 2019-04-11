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

// CreateNSGActivityName is the default registration name of the activity
const CreateNSGActivityName = "pke-azure-create-nsg"

// CreateNSGActivity represents an activity for creating an Azure network security group
type CreateNSGActivity struct {
	azureClientFactory *AzureClientFactory
}

// MakeCreateNSGActivity returns a new CreateNSGActivity
func MakeCreateNSGActivity(azureClientFactory *AzureClientFactory) CreateNSGActivity {
	return CreateNSGActivity{
		azureClientFactory: azureClientFactory,
	}
}

// CreateNSGActivityInput represents the input needed for executing a CreateNSGActivity
type CreateNSGActivityInput struct {
	Name     string
	Location string
	Rules []SecurityRule

	ResourceGroupName string
	OrganizationID    uint
	ClusterName string
	SecretID          string
}

// SecurityRule represents a network security rule
type SecurityRule struct {
	Name                 string
	Access               string
	Description          string
	Destination          string
	DestinationPortRange string
	Direction            string
	Priority             int32
	Protocol             string
	Source               string
	SourcePortRange      string
}

// Execute performs the activity
func (a CreateNSGActivity) Execute(ctx context.Context, input CreateNSGActivityInput) (string, error) {
	nsgID := ""

	logger := activity.GetLogger(ctx).Sugar().With(
		"organization", input.OrganizationID,
		"cluster": input.ClusterName,
		"secret", input.SecretID,
		"resourceGroup", input.ResourceGroupName,
		"nsgName", input.Name,
	)

	keyvals := []interface{}{
		"resourceGroup", input.ResourceGroupName,
		"nsgName", input.Name,
	}

	logger.Info("create network security group")

	cc, err := a.azureClientFactory.New(input.OrganizationID, input.SecretID)
	if err != nil {
		return nsgID, emperror.Wrap(err, "failed to create cloud connection")
	}

	securityRules := make([]network.SecurityRule, len(input.Rules))
	for i, r := range input.Rules {
		securityRules[i] := network.SecurityRule{
			Name: &r.Name,
			SecurityRulePropertiesFormat: &network.SecurityRulePropertiesFormat{
				Access: network.SecurityRuleAccess(r.Access),
				Description: &r.Description,
				DestinationAddressPrefix: &r.Destination,
				DestinationPortRange: &r.DestinationPortRange,
				Direction: network.SecurityRuleDirection(r.Direction),
				Priority: &r.Priority,
				Protocol: network.SecurityRuleProtocol(r.Protocol),
				SourceAddressPrefix: &r.Source,
				SourcePortRange: &r.SourcePortRange,
			},
		}
	}

	tags := resourceTags(tagsFrom(getOwnedTag(input.ClusterName)))

	params := network.SecurityGroup{
		Location: &input.Location,
		SecurityGroupPropertiesFormat: network.SecurityGroupPropertiesFormat{
			SecurityRules: &securityRules,
		},
		Tags:     tags,
	}

	client := cc.GetSecurityGroupsClient()

	logger.Debug("sending request to create or update network security group")

	future, err := client.CreateOrUpdate(ctx, input.ResourceGroupName, input.Name, params)
	if err != nil {
		return nsgID, emperror.WrapWith(err, "sending request to create or update network security group failed", keyvals...)
	}

	logger.Debug("waiting for the completion of create or update network security group operation")

	err = future.WaitForCompletionRef(ctx, client.Client)
	if err != nil {
		return nsgID, emperror.WrapWith(err, "waiting for the completion of create or update network security group operation failed", keyvals...)
	}

	nsg, err := future.Result(client.SecurityGroupsClient)
	if err != nil {
		return nsgID, emperror.WrapWith(err, "getting network security group create or update result failed", keyvals...)
	}

	nsgID = stringDeref(nsg.ID)

	return nsgID, nil
}
