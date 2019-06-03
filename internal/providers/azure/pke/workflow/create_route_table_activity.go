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

// CreateRouteTableActivityName is the default registration name of the activity
const CreateRouteTableActivityName = "pke-azure-create-route-table"

// CreateRouteTableActivity represents an activity for creating an Azure route table
type CreateRouteTableActivity struct {
	azureClientFactory *AzureClientFactory
}

// MakeCreateRouteTableActivity returns a new CreateRouteTableActivity
func MakeCreateRouteTableActivity(azureClientFactory *AzureClientFactory) CreateRouteTableActivity {
	return CreateRouteTableActivity{
		azureClientFactory: azureClientFactory,
	}
}

// CreateRouteTableActivityInput represents the input needed for executing a CreateRouteTableActivity
type CreateRouteTableActivityInput struct {
	OrganizationID    uint
	SecretID          string
	ClusterName       string
	ResourceGroupName string
	RouteTable        RouteTable
}

// RouteTable represents an Azure route table
type RouteTable struct {
	ID       string
	Name     string
	Location string
}

// CreateRouteTableActivityOutput represents the output of executing a CreateRouteTableActivity
type CreateRouteTableActivityOutput struct {
	RouteTableID   string
	RouteTableName string
}

// Execute performs the activity
func (a CreateRouteTableActivity) Execute(ctx context.Context, input CreateRouteTableActivityInput) (output CreateRouteTableActivityOutput, err error) {
	logger := activity.GetLogger(ctx).Sugar().With(
		"organization", input.OrganizationID,
		"cluster", input.ClusterName,
		"secret", input.SecretID,
		"resourceGroup", input.ResourceGroupName,
		"routeTable", input.RouteTable.Name,
	)

	keyvals := []interface{}{
		"resourceGroup", input.ResourceGroupName,
		"routeTable", input.RouteTable.Name,
	}

	logger.Info("create route table")

	output.RouteTableName = input.RouteTable.Name

	cc, err := a.azureClientFactory.New(input.OrganizationID, input.SecretID)
	if err = emperror.Wrap(err, "failed to create cloud connection"); err != nil {
		return
	}

	params := input.getCreateOrUpdateRouteTableParams()

	client := cc.GetRouteTablesClient()

	logger.Debug("sending request to create or update route table")

	future, err := client.CreateOrUpdate(ctx, input.ResourceGroupName, input.RouteTable.Name, params)
	if err = emperror.WrapWith(err, "sending request to create or update route table failed", keyvals...); err != nil {
		return
	}

	logger.Debug("waiting for the completion of create or update route table operation")

	err = future.WaitForCompletionRef(ctx, client.Client)
	if err = emperror.WrapWith(err, "waiting for the completion of create or update route table operation failed", keyvals...); err != nil {
		return
	}

	rt, err := future.Result(client.RouteTablesClient)
	if err = emperror.WrapWith(err, "getting route table create or update result failed", keyvals...); err != nil {
		return
	}

	output.RouteTableID = to.String(rt.ID)

	return
}

func (input CreateRouteTableActivityInput) getCreateOrUpdateRouteTableParams() network.RouteTable {
	return network.RouteTable{
		Location: to.StringPtr(input.RouteTable.Location),
		Tags:     *to.StringMapPtr(getOwnedTag(input.ClusterName).Map()),
	}
}
