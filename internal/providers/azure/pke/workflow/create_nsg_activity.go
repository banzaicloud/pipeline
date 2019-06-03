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
	OrganizationID    uint
	SecretID          string
	ClusterName       string
	ResourceGroupName string
	SecurityGroup     SecurityGroup
}

// SecurityGroup represents a network security group
type SecurityGroup struct {
	Name     string
	Location string
	Rules    []SecurityRule
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

// CreateNSGActivityOutput represents the output of executing a CreateNSGActivity
type CreateNSGActivityOutput struct {
	NetworkSecurityGroupID string
}

// Execute performs the activity
func (a CreateNSGActivity) Execute(ctx context.Context, input CreateNSGActivityInput) (output CreateNSGActivityOutput, err error) {
	logger := activity.GetLogger(ctx).Sugar().With(
		"organization", input.OrganizationID,
		"cluster", input.ClusterName,
		"secret", input.SecretID,
		"resourceGroup", input.ResourceGroupName,
		"nsgName", input.SecurityGroup.Name,
	)

	keyvals := []interface{}{
		"resourceGroup", input.ResourceGroupName,
		"nsgName", input.SecurityGroup.Name,
	}

	logger.Info("create network security group")

	cc, err := a.azureClientFactory.New(input.OrganizationID, input.SecretID)
	if err = emperror.Wrap(err, "failed to create cloud connection"); err != nil {
		return
	}

	params := input.getCreateOrUpdateSecurityGroupParams()

	client := cc.GetSecurityGroupsClient()

	logger.Debug("sending request to create or update network security group")

	future, err := client.CreateOrUpdate(ctx, input.ResourceGroupName, input.SecurityGroup.Name, params)
	if err = emperror.WrapWith(err, "sending request to create or update network security group failed", keyvals...); err != nil {
		return
	}

	logger.Debug("waiting for the completion of create or update network security group operation")

	err = future.WaitForCompletionRef(ctx, client.Client)
	if err = emperror.WrapWith(err, "waiting for the completion of create or update network security group operation failed", keyvals...); err != nil {
		return
	}

	nsg, err := future.Result(client.SecurityGroupsClient)
	if err = emperror.WrapWith(err, "getting network security group create or update result failed", keyvals...); err != nil {
		return
	}

	output.NetworkSecurityGroupID = to.String(nsg.ID)

	return
}

func (input CreateNSGActivityInput) getCreateOrUpdateSecurityGroupParams() network.SecurityGroup {
	securityRules := make([]network.SecurityRule, len(input.SecurityGroup.Rules))
	for i, r := range input.SecurityGroup.Rules {
		securityRules[i] = network.SecurityRule{
			Name: to.StringPtr(r.Name),
			SecurityRulePropertiesFormat: &network.SecurityRulePropertiesFormat{
				Access:                   network.SecurityRuleAccess(r.Access),
				Description:              to.StringPtr(r.Description),
				DestinationAddressPrefix: to.StringPtr(r.Destination),
				DestinationPortRange:     to.StringPtr(r.DestinationPortRange),
				Direction:                network.SecurityRuleDirection(r.Direction),
				Priority:                 to.Int32Ptr(r.Priority),
				Protocol:                 network.SecurityRuleProtocol(r.Protocol),
				SourceAddressPrefix:      to.StringPtr(r.Source),
				SourcePortRange:          to.StringPtr(r.SourcePortRange),
			},
		}
	}

	return network.SecurityGroup{
		Location: &input.SecurityGroup.Location,
		SecurityGroupPropertiesFormat: &network.SecurityGroupPropertiesFormat{
			SecurityRules: &securityRules,
		},
		Tags: *to.StringMapPtr(getOwnedTag(input.ClusterName).Map()),
	}
}
