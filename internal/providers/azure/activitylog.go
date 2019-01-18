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

package azure

import (
	"github.com/Azure/azure-sdk-for-go/services/monitor/mgmt/2017-09-01/insights"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/banzaicloud/azure-aks-client/cluster"
	"github.com/goph/emperror"
)

// NewActivityLogsClient instantiates a new Azure Activity Logs client using the specified credentials
func NewActivityLogsClient(creds *cluster.AKSCredential) (*insights.ActivityLogsClient, error) {
	authorizer, err := auth.NewClientCredentialsConfig(
		creds.ClientId,
		creds.ClientSecret,
		creds.TenantId).Authorizer()

	if err != nil {
		return nil, emperror.Wrap(err, "failed to instantiate new Authorizer from Azure client credentials")
	}

	activityLogClient := insights.NewActivityLogsClient(creds.SubscriptionId)
	activityLogClient.Authorizer = authorizer

	return &activityLogClient, nil
}
