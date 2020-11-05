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

	"emperror.dev/errors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/eks"
	"go.uber.org/cadence/activity"

	"github.com/banzaicloud/pipeline/internal/cluster/distribution/eks/eksprovider/workflow"
	"github.com/banzaicloud/pipeline/internal/cluster/infrastructure/aws/awsworkflow"
	"github.com/banzaicloud/pipeline/pkg/cadence/worker"
)

const UpdateClusterVersionActivityName = "eks-update-version"

// UpdateClusterVersionActivity responsible for updating an EKS cluster
type UpdateClusterVersionActivity struct {
	awsSessionFactory awsworkflow.AWSFactory
	eksFactory        workflow.EKSAPIFactory
}

// UpdateClusterVersionActivityInput holds data needed for updating an EKS cluster
type UpdateClusterVersionActivityInput struct {
	OrganizationID   uint
	ProviderSecretID string
	Region           string
	ClusterID        uint
	ClusterName      string

	Version string
}

// UpdateClusterVersionActivityOutput holds the output data of the UpdateEKSVersionActivityOutput
type UpdateClusterVersionActivityOutput struct {
	UpdateID string
}

// NewUpdateClusterVersionActivity instantiates a new EKS version update
func NewUpdateClusterVersionActivity(
	awsSessionFactory awsworkflow.AWSFactory, eksFactory workflow.EKSAPIFactory,
) *UpdateClusterVersionActivity {
	return &UpdateClusterVersionActivity{
		awsSessionFactory: awsSessionFactory,
		eksFactory:        eksFactory,
	}
}

// Register registers the activity in the worker.
func (a UpdateClusterVersionActivity) Register(worker worker.Registry) {
	worker.RegisterActivityWithOptions(a.Execute, activity.RegisterOptions{Name: UpdateClusterVersionActivityName})
}

func (a *UpdateClusterVersionActivity) Execute(ctx context.Context, input UpdateClusterVersionActivityInput) (*UpdateClusterVersionActivityOutput, error) {
	session, err := a.awsSessionFactory.New(input.OrganizationID, input.ProviderSecretID, input.Region)
	if err = errors.WrapIf(err, "failed to create AWS session"); err != nil {
		return nil, err
	}

	eksSvc := a.eksFactory.New(session)

	updateClusterVersionInput := &eks.UpdateClusterVersionInput{Name: aws.String(input.ClusterName), Version: aws.String(input.Version)}
	updateClusterVersionOutput, err := eksSvc.UpdateClusterVersion(updateClusterVersionInput)
	if err != nil {
		var awsErr awserr.Error
		if errors.As(err, &awsErr) {
			err = errors.New(awsErr.Message())
		}
		return nil, errors.WrapIfWithDetails(err, "failed to execute eks version update", "cluster", input.ClusterName)
	}

	output := UpdateClusterVersionActivityOutput{UpdateID: aws.StringValue(updateClusterVersionOutput.Update.Id)}

	return &output, nil
}
