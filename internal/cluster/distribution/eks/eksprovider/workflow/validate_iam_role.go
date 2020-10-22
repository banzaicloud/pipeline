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
	"context"

	"emperror.dev/errors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/iam"

	"github.com/banzaicloud/pipeline/internal/cluster/infrastructure/aws/awsworkflow"
)

const ValidateIAMRoleActivityName = "eks-validate-iam-role"

//  ValidateIAMRoleActivity responsible for validating IAM role
type ValidateIAMRoleActivity struct {
	awsSessionFactory *awsworkflow.AWSSessionFactory
}

//  ValidateIAMRoleActivityInput holds data needed to validate IAM Role
type ValidateIAMRoleActivityInput struct {
	EKSActivityInput

	ClusterRoleID string
}

//  ValidateIAMRoleActivityOutput holds the output data of ValidateIAMRoleActivity
type ValidateIAMRoleActivityOutput struct {
}

//  NewValidateIAMRoleActivity instantiates a new  ValidateIAMRoleActivity
func NewValidateIAMRoleActivity(awsSessionFactory *awsworkflow.AWSSessionFactory) *ValidateIAMRoleActivity {
	return &ValidateIAMRoleActivity{
		awsSessionFactory: awsSessionFactory,
	}
}

func (a *ValidateIAMRoleActivity) Execute(ctx context.Context, input ValidateIAMRoleActivityInput) (*ValidateIAMRoleActivityOutput, error) {
	awsSession, err := a.awsSessionFactory.New(input.OrganizationID, input.SecretID, input.Region)
	if err = errors.WrapIf(err, "failed to create AWS session"); err != nil {
		return nil, err
	}

	id, _ := splitResourceId(input.ClusterRoleID)

	iamSession := iam.New(awsSession)
	if _, err := iamSession.GetRole(&iam.GetRoleInput{
		RoleName: aws.String(id),
	}); err != nil {
		return nil, errors.WrapIfWithDetails(err, "invalid cluster role ID", "roleID", input.ClusterRoleID)
	}

	return &ValidateIAMRoleActivityOutput{}, nil
}
