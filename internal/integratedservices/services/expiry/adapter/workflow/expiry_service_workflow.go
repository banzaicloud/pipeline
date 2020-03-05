// Copyright © 2020 Banzai Cloud
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
	"time"

	"emperror.dev/errors"
	"go.uber.org/cadence/workflow"

	"github.com/banzaicloud/pipeline/internal/integratedservices/services/expiry"
)

const (
	ExpiryJobWorkflowName = "expiry-job"
)

// ExpiryJobWorkflowInput defines the fixed inputs of the expiry workflow
type ExpiryJobWorkflowInput struct {
	ClusterID  uint
	ExpiryDate string
}

// ExpiryJobWorkflow triggers the cluster deletion at a given date
func ExpiryJobWorkflow(ctx workflow.Context, input ExpiryJobWorkflowInput) error {
	sleepDuration, err := expiry.CalculateDuration(workflow.Now(ctx), input.ExpiryDate)
	if err != nil {
		return errors.WrapIf(err, "failed to calculate the expiry duration")
	}

	if err := workflow.Sleep(ctx, sleepDuration); err != nil {
		return errors.WrapIf(err, "sleep cancelled (possibly due to the workflow being cancelled")
	}

	activityInput := ExpiryActivityInput{
		ClusterID: input.ClusterID,
	}

	activityCtx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		ScheduleToStartTimeout: 5 * time.Minute,
		StartToCloseTimeout:    5 * time.Minute,
		WaitForCancellation:    true,
	})

	if err := workflow.ExecuteActivity(activityCtx, ExpireActivityName, activityInput).Get(activityCtx, nil); err != nil {
		return errors.WrapIfWithDetails(err, "failed to execute activity", "activity", ExpireActivityName)
	}

	return nil
}
