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

package objectstore

import (
	"context"
	"net/http"

	"github.com/banzaicloud/pipeline/pkg/providers/azure"

	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-02-01/resources"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type resourceGroup struct {
	client resources.GroupsClient
}

// NewResourceGroupClientFromSecret returns an Azure resource groups client from a secret.
func NewResourceGroupClientFromSecret(credentials azure.Credentials) resources.GroupsClient {
	return resources.NewGroupsClient(credentials.SubscriptionID)
}

// NewAuthorizedResourceGroupClientFromSecret returns an authorized Azure resource groups client from a secret.
func NewAuthorizedResourceGroupClientFromSecret(credentials azure.Credentials) (resourceGroup, error) {
	cred := NewClientCredentialsConfigFromSecret(credentials)

	client := NewResourceGroupClientFromSecret(credentials)

	authorizer, err := cred.Authorizer()
	if err != nil {
		return resourceGroup{client}, errors.Wrap(err, "failed to authorize")
	}

	client.Authorizer = authorizer

	return resourceGroup{client}, nil
}

// GetAllResourceGroups returns all resource groups using
// the Azure credentials referenced by the provided secret.
func (rg *resourceGroup) GetAllResourceGroups() ([]*resources.Group, error) {

	resourceGroupsPages, err := rg.client.List(context.TODO(), "", nil)
	if err != nil {
		return nil, err
	}

	var groups []*resources.Group
	for resourceGroupsPages.NotDone() {
		resourceGroupsChunk := resourceGroupsPages.Values()

		for i := 0; i < len(resourceGroupsChunk); i++ {
			groups = append(groups, &resourceGroupsChunk[i])
		}

		if err = resourceGroupsPages.NextWithContext(context.TODO()); err != nil {
			return nil, err
		}
	}

	return groups, nil
}

func (rg *resourceGroup) CreateResourceGroup(resourceGroup, location string, logger logrus.FieldLogger) error {
	logger = logger.WithField("resource_group", resourceGroup)

	logger.Info("creating resource group")

	res, _ := rg.client.Get(context.TODO(), resourceGroup)

	if res.StatusCode == http.StatusNotFound {
		result, err := rg.client.CreateOrUpdate(
			context.TODO(),
			resourceGroup,
			resources.Group{Location: to.StringPtr(location)},
		)
		if err != nil {
			return err
		}

		logger.Info(result.Status)
	}

	logger.Info("resource group created")

	return nil
}
