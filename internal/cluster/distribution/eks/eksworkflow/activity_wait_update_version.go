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

package eksworkflow

import (
	"context"
	"time"

	"emperror.dev/errors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/eks"
	"go.uber.org/cadence/activity"

	"github.com/banzaicloud/pipeline/internal/cluster/distribution/eks/eksprovider/workflow"
	"github.com/banzaicloud/pipeline/internal/cluster/infrastructure/aws/awsworkflow"
	"github.com/banzaicloud/pipeline/pkg/cadence/worker"
)

const WaitUpdateClusterVersionActivityName = "eks-wait-update-version"

// WaitUpdateClusterVersionActivity responsible for updating an EKS cluster
type WaitUpdateClusterVersionActivity struct {
	awsSessionFactory awsworkflow.AWSFactory
	eksFactory        workflow.EKSAPIFactory
}

// WaitUpdateClusterVersionActivityInput holds data needed for waiting for an EKS cluster update
type WaitUpdateClusterVersionActivityInput struct {
	Region           string
	OrganizationID   uint
	ProviderSecretID string
	ClusterName      string

	UpdateID string
}

// NewWaitUpdateClusterVersionActivity instantiates a new EKS version update waiting activity
func NewWaitUpdateClusterVersionActivity(
	awsSessionFactory awsworkflow.AWSFactory, eksFactory workflow.EKSAPIFactory,
) *WaitUpdateClusterVersionActivity {
	return &WaitUpdateClusterVersionActivity{
		awsSessionFactory: awsSessionFactory,
		eksFactory:        eksFactory,
	}
}

// Register registers the activity in the worker.
func (a WaitUpdateClusterVersionActivity) Register(worker worker.Registry) {
	worker.RegisterActivityWithOptions(a.Execute, activity.RegisterOptions{Name: WaitUpdateClusterVersionActivityName})
}

func (a *WaitUpdateClusterVersionActivity) Execute(ctx context.Context, input WaitUpdateClusterVersionActivityInput) error {
	session, err := a.awsSessionFactory.New(input.OrganizationID, input.ProviderSecretID, input.Region)
	if err = errors.WrapIf(err, "failed to create AWS session"); err != nil {
		return err
	}

	eksSvc := a.eksFactory.New(session)

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			describeUpdateInput := &eks.DescribeUpdateInput{Name: aws.String(input.ClusterName), UpdateId: aws.String(input.UpdateID)}
			describeUpdateOutput, err := eksSvc.DescribeUpdate(describeUpdateInput)
			if err != nil {
				return errors.WrapIfWithDetails(err, "failed to execute describe eks version update", "cluster", input.ClusterName)
			}

			switch aws.StringValue(describeUpdateOutput.Update.Status) {
			case eks.UpdateStatusCancelled:
				return errors.NewWithDetails("eks version update cancelled", "cluster", input.ClusterName)
			case eks.UpdateStatusFailed:
				var err error
				for _, e := range describeUpdateOutput.Update.Errors {
					err = errors.Combine(err, errors.New(aws.StringValue(e.ErrorMessage)))
				}

				return errors.WrapIfWithDetails(err, "eks version update failed", "cluster", input.ClusterName)
			case eks.UpdateStatusSuccessful:
				return nil
			}

		case <-ctx.Done():
			return nil
		}
	}
}
