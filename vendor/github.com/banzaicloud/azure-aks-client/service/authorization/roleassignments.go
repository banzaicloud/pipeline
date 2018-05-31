package authorization

import (
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/satori/go.uuid"
	"github.com/Azure/azure-sdk-for-go/services/authorization/mgmt/2015-07-01/authorization"
	"context"
	"fmt"
	"github.com/Azure/go-autorest/autorest"
)

// RoleAssignmentsClient responsible for role assignments
type RoleAssignmentsClient struct {
	client *authorization.RoleAssignmentsClient
}

// NewRoleAssignmentsClient creates a new 'RoleAssignmentsClient' instance
func NewRoleAssignmentsClient(authorizer autorest.Authorizer, subscriptionId string) *RoleAssignmentsClient {

	roleAssignmentsClient := authorization.NewRoleAssignmentsClient(subscriptionId)
	roleAssignmentsClient.Authorizer = authorizer

	return &RoleAssignmentsClient{
		client: &roleAssignmentsClient,
	}
}

// CreateRoleAssignment creates a role assignment
func (r *RoleAssignmentsClient) CreateRoleAssignment(scope, roleDefinitionID, principalID string) (authorization.RoleAssignment, error) {

	roleAssignmentName := uuid.NewV1().String()
	return r.client.Create(context.Background(), scope, roleAssignmentName, authorization.RoleAssignmentCreateParameters{
		Properties: &authorization.RoleAssignmentProperties{
			RoleDefinitionID: to.StringPtr(roleDefinitionID),
			PrincipalID:      to.StringPtr(principalID),
		},
	})

}

// GetRoleAssignmentByAssignedTo filters all role assignments for the subscription
func (r *RoleAssignmentsClient) GetRoleAssignmentByAssignedTo(principalID string) ([]authorization.RoleAssignment, error) {
	filter := fmt.Sprintf("assignedTo('%s')", principalID)
	return r.listAssignments(filter)
}

// listAssignments gets all role assignments for the subscription
func (r *RoleAssignmentsClient) ListRoleAssignments() ([]authorization.RoleAssignment, error) {
	return r.listAssignments("")
}

// listAssignments gets all role assignments for the subscription
func (r *RoleAssignmentsClient) listAssignments(filter string) ([]authorization.RoleAssignment, error) {
	page, err := r.client.List(context.Background(), filter)
	if err != nil {
		return nil, err
	}
	return page.Values(), err
}

// DeleteRoleAssignments deletes a role assignment
func (r *RoleAssignmentsClient) DeleteRoleAssignments(scope, roleAssignmentName string) (authorization.RoleAssignment, error) {
	return r.client.Delete(context.Background(), scope, roleAssignmentName)
}
