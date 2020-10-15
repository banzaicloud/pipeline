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

package workflow

import (
	"context"
	"time"

	"go.uber.org/cadence/workflow"

	pkgCadence "github.com/banzaicloud/pipeline/pkg/cadence"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
)

const SetClusterStatusActivityName = "eks-set-cluster-status"

type SetClusterStatusActivity struct {
	manager Clusters
}

func NewSetClusterStatusActivity(manager Clusters) SetClusterStatusActivity {
	return SetClusterStatusActivity{
		manager: manager,
	}
}

type SetClusterStatusActivityInput struct {
	ClusterID     uint
	Status        string
	StatusMessage string
}

func (a SetClusterStatusActivity) Execute(ctx context.Context, input SetClusterStatusActivityInput) error {
	cluster, err := a.manager.GetCluster(ctx, input.ClusterID)
	if err != nil {
		return err
	}
	return cluster.SetStatus(input.Status, input.StatusMessage)
}

func SetClusterStatus(ctx workflow.Context, clusterID uint, status, statusMessage string) error {
	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		ScheduleToStartTimeout: 10 * time.Minute,
		StartToCloseTimeout:    2 * time.Minute,
		WaitForCancellation:    true,
	})

	return workflow.ExecuteActivity(ctx, SetClusterStatusActivityName, SetClusterStatusActivityInput{
		ClusterID:     clusterID,
		Status:        status,
		StatusMessage: statusMessage,
	}).Get(ctx, nil)
}

func SetClusterErrorStatus(ctx workflow.Context, clusterID uint, err error) error {
	return SetClusterStatus(ctx, clusterID, pkgCluster.Error, pkgCadence.UnwrapError(err).Error())
}
