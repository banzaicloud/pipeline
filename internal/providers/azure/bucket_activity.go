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

const CreateBucketActivityType = "azure-create-bucket"

type CreateBucketActivityContext struct {
	OrganizationID uint
	SecretID       string
	ResourceGroup  string
	StorageAccount string
	Bucket         string
}

type CreateBucketActivity struct {
	accountClientFactory *StorageAccountClientFactory
}

func NewCreateBucketActivity(accountClientFactory *StorageAccountClientFactory) *CreateBucketActivity {
	return &CreateBucketActivity{
		accountClientFactory: accountClientFactory,
	}
}

func (a *CreateBucketActivity) Name() string {
	return CreateBucketActivityType
}

func (a *CreateBucketActivity) Execute(ctx context.Context, activityContext CreateBucketActivityContext) error {
	logger := activity.GetLogger(ctx).With( // TODO: add correlation ID from API request (if any)
		zap.Uint("organization-id", activityContext.OrganizationID),
		zap.String("secret-id", activityContext.SecretID),
		zap.String("resource-group", activityContext.ResourceGroup),
		zap.String("storage-account", activityContext.StorageAccount),
		zap.String("bucket", activityContext.Bucket),
	)

	logger.Info("creating bucket")

	logger.Debug("getting storage account client")

	client, err := a.accountClientFactory.New(activityContext.OrganizationID, activityContext.SecretID)
	if err != nil {
		zaplog.LogError(logger, err) // TODO: use error handler

		return errors.New("failed to initialize storage account client")
	}

	err = CreateBucket(ctx, client, activityContext.ResourceGroup, activityContext.StorageAccount, activityContext.Bucket)
	if err != nil {
		zaplog.LogError(logger, err) // TODO: use error handler

		return errors.New("failed to create bucket")
	}

	logger.Info("bucket successfully created")

	return nil
}
