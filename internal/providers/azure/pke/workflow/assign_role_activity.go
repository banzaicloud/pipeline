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

	"emperror.dev/emperror"
	"github.com/Azure/azure-sdk-for-go/profiles/latest/authorization/mgmt/authorization"
	"github.com/Azure/go-autorest/autorest/to"
	"go.uber.org/cadence/activity"
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
	RoleAssignment    RoleAssignment
}

type RoleAssignment struct {
	Name        string
	PrincipalID string
	RoleName    string
}

func (a AssignRoleActivity) Execute(ctx context.Context, input AssignRoleActivityInput) error {
	logger := activity.GetLogger(ctx).Sugar().With(
		"organization", input.OrganizationID,
		"cluster", input.ClusterName,
		"secret", input.SecretID,
		"resourceGroup", input.ResourceGroupName,
	)

	cc, err := a.azureClientFactory.New(input.OrganizationID, input.SecretID)
	if err = emperror.Wrap(err, "failed to create cloud connection"); err != nil {
		return err
	}

	resourceGroup, err := cc.GetGroupsClient().Get(ctx, input.ResourceGroupName)
	if err != nil {
		return emperror.WrapWith(err, "failed to retrieve resource group", "resourceGroup", input.ResourceGroupName)
	}

	scope := *resourceGroup.ID

	role, err := cc.GetRoleDefinitionsClient().FindByRoleName(ctx, scope, input.RoleAssignment.RoleName)
	if err != nil {
		return emperror.WrapWith(err, "failed to find role by name", "scope", scope, "roleName", input.RoleAssignment.RoleName)
	}

	params := input.getRoleAssignmentCreateParams(*role)

	result, err := cc.GetRoleAssignmentsClient().Create(ctx, scope, input.RoleAssignment.Name, params)
	if result.Response.StatusCode == http.StatusConflict {
		logger.Infof("Role [%s] is already assigned to principal [%s] as [%s]", input.RoleAssignment.RoleName, input.RoleAssignment.PrincipalID, result.ID)
		return nil
	}
	return emperror.WrapWith(err, "failed to create role assignment", "scope", scope, "roleName", input.RoleAssignment.RoleName, "principalID", input.RoleAssignment.PrincipalID)
}

func (input AssignRoleActivityInput) getRoleAssignmentCreateParams(role authorization.RoleDefinition) authorization.RoleAssignmentCreateParameters {
	return authorization.RoleAssignmentCreateParameters{
		Properties: &authorization.RoleAssignmentProperties{
			PrincipalID:      to.StringPtr(input.RoleAssignment.PrincipalID),
			RoleDefinitionID: role.ID,
		},
	}
}
