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

package pkeawsworkflow

import (
	"time"

	"go.uber.org/cadence/workflow"

	"github.com/banzaicloud/pipeline/internal/cluster/clusterworkflow"
)

// TODO: this is temporary
func setClusterStatus(ctx workflow.Context, clusterID uint, status, statusMessage string) error {
	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		ScheduleToStartTimeout: 10 * time.Minute,
		StartToCloseTimeout:    2 * time.Minute,
		WaitForCancellation:    true,
	})

	return workflow.ExecuteActivity(ctx, clusterworkflow.SetClusterStatusActivityName, clusterworkflow.SetClusterStatusActivityInput{
		ClusterID:     clusterID,
		Status:        status,
		StatusMessage: statusMessage,
	}).Get(ctx, nil)
}
