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

	"github.com/banzaicloud/pipeline/internal/platform/zaplog"
	"github.com/jinzhu/gorm"
	"go.uber.org/cadence"
	"go.uber.org/cadence/activity"
	"go.uber.org/zap"
)

const CreateBucketRollbackActivityType = "azure-create-bucket-rollback"

type CreateBucketRollbackActivityContext struct {
	BucketID uint
}

type CreateBucketRollbackActivity struct {
	db *gorm.DB
}

func NewCreateBucketRollbackActivity(db *gorm.DB) *CreateBucketRollbackActivity {
	return &CreateBucketRollbackActivity{
		db: db,
	}
}

func (a *CreateBucketRollbackActivity) Name() string {
	return CreateBucketRollbackActivityType
}

func (a *CreateBucketRollbackActivity) Execute(ctx context.Context, activityContext CreateBucketRollbackActivityContext) error {
	logger := activity.GetLogger(ctx).With( // TODO: add correlation ID from API request (if any)
		zap.Uint("bucket-id", activityContext.BucketID),
	)

	logger.Info("deleting bucket state")

	err := DeleteBucketState(a.db, activityContext.BucketID)
	if err != nil {
		zaplog.LogError(logger, err) // TODO: use error handler

		return cadence.NewCustomError("pipeline:DatabaseFailure", "failed to delete bucket state")
	}

	logger.Info("bucket state successfully deleted")

	return nil
}
