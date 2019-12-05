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
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/iam"
	"go.uber.org/cadence/activity"
)

const DeleteSshKeyActivityName = "eks-delete-ssh-key"

type DeleteSshKeyActivity struct {
	awsSessionFactory *AWSSessionFactory
}

type DeleteSshKeyActivityInput struct {
	EKSActivityInput
	SSHKeyName string
}

type DeleteSshKeyActivityOutput struct {
}

//   DeleteStackActivity instantiates a new DeleteStackActivity
func NewDeleteSshKeyActivity(awsSessionFactory *AWSSessionFactory) *DeleteSshKeyActivity {
	return &DeleteSshKeyActivity{
		awsSessionFactory: awsSessionFactory,
	}
}

func (a *DeleteSshKeyActivity) Execute(ctx context.Context, input DeleteSshKeyActivityInput) error {
	logger := activity.GetLogger(ctx).Sugar().With(
		"organization", input.OrganizationID,
		"cluster", input.ClusterName,
		"sshKeyName", input.SSHKeyName,
	)

	awsSession, err := a.awsSessionFactory.New(input.OrganizationID, input.SecretID, input.Region)
	if err = errors.WrapIf(err, "failed to create AWS session"); err != nil {
		return err
	}

	logger.Info("deleting ssh key")

	ec2srv := ec2.New(awsSession)
	deleteKeyPairInput := &ec2.DeleteKeyPairInput{
		KeyName: aws.String(input.SSHKeyName),
	}
	_, err = ec2srv.DeleteKeyPair(deleteKeyPairInput)
	if awsErr, ok := err.(awserr.Error); ok {
		if awsErr.Code() == iam.ErrCodeNoSuchEntityException {
			return nil
		}
	}
	return nil
}
