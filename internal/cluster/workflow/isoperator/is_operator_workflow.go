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
package isoperator

import (
	"time"

	"emperror.dev/errors"
	"go.uber.org/cadence/workflow"

	"github.com/banzaicloud/pipeline/internal/cluster"
)

const IntegratedServiceOperatorInstallerWorkflowName = "integrated-service-operator-installer"

type Config struct {
	Enabled      bool   `json:"enabled"`
	RepoURL      string `json:"repoUrl"`
	RepoName     string `json:"repoName"`
	ChartVersion string `json:"chartVersion"`
	ChartName    string `json:"chartName"`
	ReleaseName  string `json:"releaseName"`
	Namespace    string `json:"namespace"`
	BatchSize    int    `json:"batchSize"`
}

type NextIDProvider func(uint) (uint, uint, error)

type IntegratedSesrvicesOperatorInstallerWorkflowInput struct {
	LastClusterID uint
}

type IntegratedSesrvicesOperatorWorkflow struct {
	config Config
}

func NewISOperatorWorkflow(config Config) IntegratedSesrvicesOperatorWorkflow {
	return IntegratedSesrvicesOperatorWorkflow{
		config: config,
	}
}

func (w IntegratedSesrvicesOperatorWorkflow) Execute(ctx workflow.Context, input IntegratedSesrvicesOperatorInstallerWorkflowInput) error {
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
			if cluster.IsNotFoundError(err) {
				// all clusters have been processed, success flow!
				return nil
			}
			return errors.WrapIf(err, "failed to get the next cluster reference")
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
