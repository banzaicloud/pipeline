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

	"emperror.dev/errors"
	"go.uber.org/cadence"
	"go.uber.org/cadence/workflow"

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
	return workflow.ExecuteActivity(ctx, SetClusterStatusActivityName, SetClusterStatusActivityInput{
		ClusterID:     clusterID,
		Status:        status,
		StatusMessage: statusMessage,
	}).Get(ctx, nil)
}

func extractErrorDetail(err error) error {
	if cadence.IsCustomError(err) {
		cerr := err.(*cadence.CustomError)
		if cerr.HasDetails() {
			var errDetails string
			if err = errors.WrapIf(cerr.Details(&errDetails), "couldn't get error details"); err != nil {
				return err
			}

			return errors.New(errDetails)
		}
	}
	return err
}

func SetClusterErrorStatus(ctx workflow.Context, clusterID uint, err error) error {
	return SetClusterStatus(ctx, clusterID, pkgCluster.Error, extractErrorDetail(err).Error())
}
