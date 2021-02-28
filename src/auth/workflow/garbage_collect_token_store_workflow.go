// Copyright © 2021 Banzai Cloud
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

	"go.uber.org/cadence"
	"go.uber.org/cadence/workflow"

	"github.com/banzaicloud/pipeline/pkg/cadence/worker"
)

// GarbageCollectTokenStoreWorkflowName is the name of the token store garbage
// collection workflow.
const GarbageCollectTokenStoreWorkflowName = "auth-garbage-collect-token-store"

// GarbageCollectTokenStoreWorkflow defines a Cadence workflow encapsulating
// high level input-independent components required to garbage collect the token
// store.
type GarbageCollectTokenStoreWorkflow struct{}

// NewGarbageCollectTokenStoreWorkflow instantiates a token store garbage
// collection workflow.
func NewGarbageCollectTokenStoreWorkflow() *GarbageCollectTokenStoreWorkflow {
	return &GarbageCollectTokenStoreWorkflow{}
}

// Execute runs the workflow.
func (w GarbageCollectTokenStoreWorkflow) Execute(ctx workflow.Context) (err error) {
	logger := workflow.GetLogger(ctx)

	activityContext := workflow.WithActivityOptions(
		ctx,
		workflow.ActivityOptions{
			ScheduleToStartTimeout: 10 * time.Minute,
			StartToCloseTimeout:    time.Hour,
			WaitForCancellation:    true,
			RetryPolicy: &cadence.RetryPolicy{
				InitialInterval:          time.Hour,
				BackoffCoefficient:       1.0,
				ExpirationInterval:       6 * time.Hour,
				MaximumAttempts:          3,
				NonRetriableErrorReasons: []string{"cadenceInternal:Panic"},
			},
		},
	)

	err = startTokenStoreGC(activityContext)
	if err != nil {
		return err
	}

	logger.Info("TokenStore garbage collected")

	return nil
}

// Register registers the workflow in the worker.
func (w GarbageCollectTokenStoreWorkflow) Register(worker worker.Registry) {
	worker.RegisterWorkflowWithOptions(w.Execute, workflow.RegisterOptions{Name: GarbageCollectTokenStoreWorkflowName})
}
