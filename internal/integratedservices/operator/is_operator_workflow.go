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
package operator

import (
	"time"

	"emperror.dev/errors"
	"go.uber.org/cadence/workflow"
)

const IntegratedServiceOperatorInstallerWorkflowName = "integrated-service-operator-installer"

type Config struct {
	// repoName/ChartName format
	Chart     string `json:"chart"`
	Release   string `json:"release"`
	Version   string `json:"version"`
	Namespace string `json:"namespace"`
	BatchSize int    `json:"batchSize"`
}

type NextIDProvider func(uint) (uint, uint, error)

type IntegratedServicesOperatorInstallerWorkflowInput struct {
	LastClusterID uint
}

type IntegratedServicesOperatorWorkflow struct {
	config Config
}

func NewISOperatorWorkflow(config Config) IntegratedServicesOperatorWorkflow {
	return IntegratedServicesOperatorWorkflow{
		config: config,
	}
}

func (w IntegratedServicesOperatorWorkflow) Execute(ctx workflow.Context, input IntegratedServicesOperatorInstallerWorkflowInput) error {
	activityOptions := workflow.ActivityOptions{
		ScheduleToStartTimeout: 15 * time.Minute,
		StartToCloseTimeout:    5 * time.Minute,
		WaitForCancellation:    true,
	}
	ctx = workflow.WithActivityOptions(ctx, activityOptions)

	lastProcessedClusterID := input.LastClusterID
	for i := 0; i < w.config.BatchSize; i++ {
		// get the next cluster reference to be processed
		var clusterRef ClusterRef
		if err := workflow.ExecuteActivity(ctx, GetNextClusterRefActivityName, lastProcessedClusterID).Get(ctx, &clusterRef); err != nil {
			return errors.WrapIf(err, "failed to get the next cluster reference")
		}

		if clusterRef.NotFound {
			// this is the end criteria for the recursion (= all the clusters have been processed)
			return nil
		}

		// install / upgrade the  operator
		input := NewInstallerActivityInput(clusterRef.OrgID, clusterRef.ID)
		if err := workflow.ExecuteActivity(ctx, IntegratedServiceOperatorInstallerActivityName, input).Get(ctx, nil); err != nil {
			return errors.WrapIfWithDetails(err, "failed to install the  operator", "orgID", input.OrgID, "clusterID", input.ClusterID)
		}
		lastProcessedClusterID = input.ClusterID
	}

	// start a new workflow not to blow the history
	input.LastClusterID = lastProcessedClusterID
	return workflow.NewContinueAsNewError(ctx, w.Execute, input)
}

const SingleClusterIntegratedServiceOperatorInstallerWorkflowName = "single-cluster-integrated-service-operator-installer"

// SingleClusterIntegratedServiceOperatorInstallerWorkflow workflow component struct
type SingleClusterIntegratedServiceOperatorInstallerWorkflow struct {
}

// SingleClusterIntegratedServiceOperatorInstallerWorkflowInput input definition for the SingleClusterIntegratedServiceOperatorInstallerWorkflow
type SingleClusterIntegratedServiceOperatorInstallerWorkflowInput struct {
	ClusterID uint
	OrgID     uint
}

func NewSingleClusterIntegratedServiceOperatorInstallerWorkflow() SingleClusterIntegratedServiceOperatorInstallerWorkflow {
	return SingleClusterIntegratedServiceOperatorInstallerWorkflow{}
}

// Execute workflow function for installing the integrated service operator to a single cluster
func (s SingleClusterIntegratedServiceOperatorInstallerWorkflow) Execute(ctx workflow.Context, input SingleClusterIntegratedServiceOperatorInstallerWorkflowInput) error {
	activityOptions := workflow.ActivityOptions{
		ScheduleToStartTimeout: 15 * time.Minute,
		StartToCloseTimeout:    5 * time.Minute,
		WaitForCancellation:    true,
	}
	ctx = workflow.WithActivityOptions(ctx, activityOptions)

	// install / upgrade the  operator
	activityInput := NewInstallerActivityInput(input.OrgID, input.ClusterID)
	if err := workflow.ExecuteActivity(ctx, IntegratedServiceOperatorInstallerActivityName, activityInput).Get(ctx, nil); err != nil {
		return errors.WrapIfWithDetails(err, "failed to install the  operator", "orgID", input.OrgID, "clusterID", input.ClusterID)
	}

	return nil
}
