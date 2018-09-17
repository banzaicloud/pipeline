// Copyright © 2018 Banzai Cloud
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
	"context"
	"errors"

	"github.com/banzaicloud/pipeline/internal/platform/zaplog"
	"go.uber.org/cadence/activity"
	"go.uber.org/zap"
)

const CreateResourceGroupActivityType = "azure-create-resource-group"

type CreateResourceGroupActivityContext struct {
	OrganizationID uint
	SecretID       string
	Location       string
	ResourceGroup  string
}

type CreateResourceGroupActivity struct {
	clientFactory *ResourceGroupClientFactory
}

func NewCreateResourceGroupActivity(clientFactory *ResourceGroupClientFactory) *CreateResourceGroupActivity {
	return &CreateResourceGroupActivity{
		clientFactory: clientFactory,
	}
}

func (a *CreateResourceGroupActivity) Name() string {
	return CreateResourceGroupActivityType
}

func (a *CreateResourceGroupActivity) Execute(ctx context.Context, activityContext CreateResourceGroupActivityContext) error {
	logger := activity.GetLogger(ctx).With( // TODO: add correlation ID from API request (if any)
		zap.Uint("organization-id", activityContext.OrganizationID),
		zap.String("secret-id", activityContext.SecretID),
		zap.String("location", activityContext.Location),
		zap.String("resource-group", activityContext.ResourceGroup),
	)

	logger.Info("creating resource group")

	logger.Debug("creating resource group client")

	client, err := a.clientFactory.New(activityContext.OrganizationID, activityContext.SecretID)
	if err != nil {
		zaplog.LogError(logger, err) // TODO: use error handler

		return errors.New("failed to initialize resource group client")
	}

	err = CreateResourceGroup(ctx, client, activityContext.ResourceGroup, activityContext.Location)
	if err != nil {
		zaplog.LogError(logger, err) // TODO: use error handler

		return errors.New("failed to create resource group")
	}

	logger.Info("resource group successfully created")

	return nil
}
