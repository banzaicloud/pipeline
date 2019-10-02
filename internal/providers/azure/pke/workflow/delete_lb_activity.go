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

	"emperror.dev/errors"
	"github.com/Azure/go-autorest/autorest/to"
	"go.uber.org/cadence/activity"
)

// DeleteLoadBalancerActivityName is the default registration name of the activity
const DeleteLoadBalancerActivityName = "pke-azure-delete-load-balancer"

// DeleteLoadBalancerActivity represents an activity for deleting an Azure load balancer
type DeleteLoadBalancerActivity struct {
	azureClientFactory *AzureClientFactory
}

// DeleteLoadBalancerActivityInput represents the input needed for executing a DeleteLoadBalancerActivity
type DeleteLoadBalancerActivityInput struct {
	OrganizationID    uint
	SecretID          string
	ClusterName       string
	ResourceGroupName string
	LoadBalancerName  string
}

// MakeDeleteLoadBalancerActivity returns a new CreateLoadBalancerActivity
func MakeDeleteLoadBalancerActivity(azureClientFactory *AzureClientFactory) DeleteLoadBalancerActivity {
	return DeleteLoadBalancerActivity{
		azureClientFactory: azureClientFactory,
	}
}

func (a DeleteLoadBalancerActivity) Execute(ctx context.Context, input DeleteLoadBalancerActivityInput) (err error) {
	logger := activity.GetLogger(ctx).Sugar().With(
		"organization", input.OrganizationID,
		"cluster", input.ClusterName,
		"resourceGroup", input.ResourceGroupName,
		"loadBalancer", input.LoadBalancerName,
	)

	keyvals := []interface{}{
		"resourceGroup", input.ResourceGroupName,
		"loadBalancer", input.LoadBalancerName,
	}

	logger.Info("delete load balancer")

	cc, err := a.azureClientFactory.New(input.OrganizationID, input.SecretID)
	if err = errors.WrapIf(err, "failed to create cloud connection"); err != nil {
		return
	}

	client := cc.GetLoadBalancersClient()

	// delete load balancer only if is owned by this cluster
	logger.Debug("get load balancer details")

	lb, err := client.Get(ctx, input.ResourceGroupName, input.LoadBalancerName, "")
	if err != nil {
		if lb.StatusCode == http.StatusNotFound {
			logger.Warn("load balancer not found")
			return nil
		}

		return errors.WrapIfWithDetails(err, "failed to get load balancer details", keyvals...)
	}

	if !HasOwnedTag(input.ClusterName, to.StringMap(lb.Tags)) {
		logger.Info("skip deleting load balancer as it's not owned by cluster")
		return
	}

	future, err := client.Delete(ctx, input.ResourceGroupName, input.LoadBalancerName)
	if err = errors.WrapIfWithDetails(err, "sending request to delete load balancer failed", keyvals...); err != nil {
		if resp := future.Response(); resp != nil && resp.StatusCode == http.StatusNotFound {
			logger.Warn("load balancer not found")
			return nil
		}
		return
	}

	logger.Debug("waiting for the completion of delete load balancer operation")

	err = future.WaitForCompletionRef(ctx, client.Client)
	if err = errors.WrapIfWithDetails(err, "waiting for the completion of delete load balancer operation failed", keyvals...); err != nil {
		if resp := future.Response(); resp != nil && resp.StatusCode == http.StatusNotFound {
			logger.Warn("load balancer not found")
			return nil
		}
		return
	}

	logger.Debug("load balancer deletion completed")

	return
}
