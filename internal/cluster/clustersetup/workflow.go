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

package clustersetup

import (
	"time"

	"go.uber.org/cadence"
	"go.uber.org/cadence/workflow"
)

// WorkflowName can be used to reference the cluster setup workflow.
const WorkflowName = "cluster-setup"

// Workflow orchestrates the post-creation cluster setup flow.
type Workflow struct {
	// InstallInit
	InstallInitManifest bool
}

// WorkflowInput is the input for a cluster setup workflow.
type WorkflowInput struct {
	Cluster      Cluster
	Organization Organization
}

// Cluster represents a Kubernetes cluster.
type Cluster struct {
	ID   uint
	UID  string
	Name string
}

// Organization contains information about the organization a cluster belongs to.
type Organization struct {
	ID   uint
	Name string
}

// Execute executes the cluster setup workflow.
func (w Workflow) Execute(ctx workflow.Context, input WorkflowInput) error {
	// Default timeouts and retries
	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		ScheduleToStartTimeout: 20 * time.Minute,
		StartToCloseTimeout:    30 * time.Minute,
		WaitForCancellation:    true,
		RetryPolicy: &cadence.RetryPolicy{
			InitialInterval:    2 * time.Second,
			BackoffCoefficient: 1.5,
			MaximumInterval:    30 * time.Second,
			MaximumAttempts:    30,
		},
	})

	// Install the cluster manifest to the cluster (if configured)
	if w.InstallInitManifest {
		activityInput := InitManifestActivityInput{
			Cluster:      input.Cluster,
			Organization: input.Organization,
		}

		err := workflow.ExecuteActivity(ctx, InitManifestActivityName, activityInput).Get(ctx, nil)
		if err != nil {
			return err
		}
	}

	return nil
}
