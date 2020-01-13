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

	"go.uber.org/cadence/workflow"
)

const (
	// ExpiryJobWorkflowName is the name the workflow is registered under
	ExpiryJobWorkflowName = "expiry-job"
	ExpiryJobSignalName   = "expiry-signal"
)

// ExpiryJobWorkflowInput defines the fixed inputs of the expiry workflow
type ExpiryJobWorkflowInput struct {
	ClusterID  uint
	ExpiryDate string
}

type ExpiryJobSignalInput struct {
	ClusterID  uint
	ExpiryDate string
}

// IntegratedServiceJobWorkflow executes integrated service jobs
func ExpiryJobWorkflow(ctx workflow.Context, input ExpiryJobWorkflowInput) error {

	expiryTime, err := time.ParseInLocation(time.RFC3339, input.ExpiryDate, time.Now().Location())
	if err != nil {
		return err
	}

	dur := expiryTime.Sub(workflow.Now(ctx))

	if err := workflow.Sleep(ctx, dur); err != nil {
		return err
	}

	activityInput := ExpiryActivityInput{
		ClusterID: input.ClusterID,
	}

	activityCtx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		ScheduleToStartTimeout: 15 * time.Minute,
		StartToCloseTimeout:    3 * time.Hour,
		WaitForCancellation:    true,
	})

	workflow.ExecuteActivity(activityCtx, ExpireActivityName, activityInput)

	return nil
}
