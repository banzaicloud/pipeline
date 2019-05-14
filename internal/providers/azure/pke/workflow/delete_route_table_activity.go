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

	"github.com/Azure/go-autorest/autorest/to"
	"github.com/goph/emperror"
	"go.uber.org/cadence/activity"
)

// DeleteRouteTableActivityName is the default registration name of the activity
const DeleteRouteTableActivityName = "pke-azure-delete-route-table"

// DeleteRouteTableActivity represents an activity for deleting a route table
type DeleteRouteTableActivity struct {
	azureClientFactory *AzureClientFactory
}

// DeleteRouteTableActivityInput represents the input needed for executing a DeleteRouteTableActivity
type DeleteRouteTableActivityInput struct {
	OrganizationID    uint
	SecretID          string
	ClusterName       string
	ResourceGroupName string
	RouteTableName    string
}

// MakeDeleteRouteTableActivity returns a new MakeDeleteRouteTableActivity
func MakeDeleteRouteTableActivity(azureClientFactory *AzureClientFactory) DeleteRouteTableActivity {
	return DeleteRouteTableActivity{
		azureClientFactory: azureClientFactory,
	}
}

func (a DeleteRouteTableActivity) Execute(ctx context.Context, input DeleteRouteTableActivityInput) (err error) {
	logger := activity.GetLogger(ctx).Sugar().With(
		"organization", input.OrganizationID,
		"cluster", input.ClusterName,
		"resourceGroup", input.ResourceGroupName,
		"routeTable", input.RouteTableName,
	)

	keyvals := []interface{}{
		"resourceGroup", input.ResourceGroupName,
		"routeTable", input.RouteTableName,
	}

	logger.Info("delete route table")
	cc, err := a.azureClientFactory.New(input.OrganizationID, input.SecretID)
	if err = emperror.Wrap(err, "failed to create cloud connection"); err != nil {
		return
	}

	client := cc.GetRouteTablesClient()

	// delete route table only if owned by current cluster
	logger.Debug("get route table details")

	rt, err := client.Get(ctx, input.ResourceGroupName, input.RouteTableName, "")
	if err != nil {
		if rt.StatusCode == http.StatusNotFound {
			logger.Warn("route table not found")
			return nil
		}

		return emperror.WrapWith(err, "failed to get route table details", keyvals...)
	}

	if !HasOwnedTag(input.ClusterName, to.StringMap(rt.Tags)) {
		logger.Info("skip deleting route table as it's not owned by cluster")
		return
	}

	future, err := client.Delete(ctx, input.ResourceGroupName, input.RouteTableName)
	if err = emperror.WrapWith(err, "sending request to delete route table failed", keyvals...); err != nil {
		if resp := future.Response(); resp != nil && resp.StatusCode == http.StatusNotFound {
			logger.Warn("route table not found")
			return nil
		}
		return
	}

	logger.Debug("waiting for the completion of delete route table operation")

	err = future.WaitForCompletionRef(ctx, client.Client)
	if err = emperror.WrapWith(err, "waiting for the completion of delete route table operation failed", keyvals...); err != nil {
		if resp := future.Response(); resp != nil && resp.StatusCode == http.StatusNotFound {
			logger.Warn("route table not found")
			return nil
		}
		return
	}

	logger.Debug("route table deletion completed")

	return
}
