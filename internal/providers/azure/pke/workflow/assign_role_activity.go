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
	"fmt"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/authorization/mgmt/authorization"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/gofrs/uuid"
	"github.com/goph/emperror"
)

// AssignRoleActivityName is the default registration name of the activity
const AssignRoleActivityName = "pke-azure-assign-role"

// AssignRoleActivity represents an activity for creating an Azure network security group
type AssignRoleActivity struct {
	azureClientFactory *AzureClientFactory
}

// MakeAssignRoleActivity returns a new CreateNSGActivity
func MakeAssignRoleActivity(azureClientFactory *AzureClientFactory) AssignRoleActivity {
	return AssignRoleActivity{
		azureClientFactory: azureClientFactory,
	}
}

type AssignRoleActivityInput struct {
	OrganizationID    uint
	SecretID          string
	ClusterName       string
	ResourceGroupName string
	PrincipalID       string
}

func (a AssignRoleActivity) Execute(ctx context.Context, input AssignRoleActivityInput) error {

	cc, err := a.azureClientFactory.New(input.OrganizationID, input.SecretID)
	if err = emperror.Wrap(err, "failed to create cloud connection"); err != nil {
		return err
	}
	scope := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s", cc.GetSubscriptionID(), input.ResourceGroupName)
	client := cc.GetRoleAssignmentsClient()
	role, err := cc.GetRoleDefinitionsClient().FindByRoleName(ctx, scope, "Contributor")
	if err != nil {
		return err
	}
	_, err = client.Create(
		ctx,
		input.ResourceGroupName,
		uuid.Must(uuid.NewV1()).String(),
		authorization.RoleAssignmentCreateParameters{
			Properties: &authorization.RoleAssignmentProperties{
				PrincipalID:      to.StringPtr(input.PrincipalID),
				RoleDefinitionID: role.ID,
			},
		})
	return err
}
