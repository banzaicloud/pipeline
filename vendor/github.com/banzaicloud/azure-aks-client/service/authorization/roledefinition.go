package authorization

import (
	"github.com/Azure/azure-sdk-for-go/services/authorization/mgmt/2015-07-01/authorization"
	"context"
	"github.com/Azure/go-autorest/autorest"
)

// RoleDefinitionsClient provides role definitions
type RoleDefinitionsClient struct {
	client *authorization.RoleDefinitionsClient
}

// NewRoleDefinitionClient creates a new 'RoleDefinitionsClient' instance
func NewRoleDefinitionClient(authorizer autorest.Authorizer, subscriptionId string) *RoleDefinitionsClient {
	roleDefinitionClient := authorization.NewRoleDefinitionsClient(subscriptionId)
	roleDefinitionClient.Authorizer = authorizer
	return &RoleDefinitionsClient{
		client: &roleDefinitionClient,
	}
}

// ListRoleDefinitions gets all role definitions that are applicable at scope and above.
func (r *RoleDefinitionsClient) ListRoleDefinitions(scope string) ([]authorization.RoleDefinition, error) {
	page, err := r.client.List(context.Background(), scope, "")
	if err != nil {
		return nil, err
	}

	return page.Values(), nil
}
