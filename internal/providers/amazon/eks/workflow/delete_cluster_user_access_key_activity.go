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
	"github.com/aws/aws-sdk-go/service/iam"
	"go.uber.org/cadence/activity"

	"github.com/banzaicloud/pipeline/secret"

	"github.com/banzaicloud/pipeline/pkg/amazon"
)

const DeleteClusterUserAccessKeyActivityName = "eks-delete-cluster-user-access-key"

// DeleteClusterUserAccessKeyActivity responsible for deleting cluster user access key in case if not default user
// &  cluster secret from secret store
type DeleteClusterUserAccessKeyActivity struct {
	awsSessionFactory *AWSSessionFactory
}

type DeleteClusterUserAccessKeyActivityInput struct {
	EKSActivityInput
	DefaultUser bool
}

//   DeleteClusterUserAccessKeyActivityOutput holds the output data of the DeleteStackActivity
type DeleteClusterUserAccessKeyActivityOutput struct {
}

//   NewDeleteClusterUserAccessKeyActivity instantiates a new DeleteClusterUserAccessKeyActivity
func NewDeleteClusterUserAccessKeyActivity(awsSessionFactory *AWSSessionFactory) *DeleteClusterUserAccessKeyActivity {
	return &DeleteClusterUserAccessKeyActivity{
		awsSessionFactory: awsSessionFactory,
	}
}

func (a *DeleteClusterUserAccessKeyActivity) Execute(ctx context.Context, input DeleteClusterUserAccessKeyActivityInput) error {
	logger := activity.GetLogger(ctx).Sugar().With(
		"organization", input.OrganizationID,
		"cluster", input.ClusterName,
	)

	// delete access key only if it's created by Pipeline (DefaultUser = false)
	if !input.DefaultUser {
		awsSession, err := a.awsSessionFactory.New(input.OrganizationID, input.SecretID, input.Region)
		if err = errors.WrapIf(err, "failed to create AWS session"); err != nil {
			return err
		}

		logger.Info("deleting cluster user access key")

		iamSvc := iam.New(awsSession)
		clusterUserName := aws.String(input.ClusterName)

		awsAccessKeys, err := amazon.GetUserAccessKeys(iamSvc, clusterUserName)

		if err != nil {
			if awsErr, ok := err.(awserr.Error); ok {
				if awsErr.Code() == iam.ErrCodeNoSuchEntityException {
					return nil
				}
			}
			logger.Errorf("querying IAM user '%s' access keys failed: %s", *clusterUserName, err)
			return errors.Wrapf(err, "querying IAM user '%s' access keys failed", *clusterUserName)
		}

		for _, awsAccessKey := range awsAccessKeys {
			if err := amazon.DeleteUserAccessKey(iamSvc, awsAccessKey.UserName, awsAccessKey.AccessKeyId); err != nil {
				if awsErr, ok := err.(awserr.Error); ok {
					if awsErr.Code() == iam.ErrCodeNoSuchEntityException {
						continue
					}
				}

				logger.Errorf("deleting Amazon user access key '%s', user '%s' failed: %s",
					aws.StringValue(awsAccessKey.AccessKeyId),
					aws.StringValue(awsAccessKey.UserName), err)

				return errors.Wrapf(err, "deleting Amazon access key '%s', user '%s' failed",
					aws.StringValue(awsAccessKey.AccessKeyId),
					aws.StringValue(awsAccessKey.UserName))
			}
		}
	}

	// delete secret from  store
	secretName := getSecretName(input.ClusterName)
	secretItem, err := a.awsSessionFactory.secretStore.GetByName(input.OrganizationID, secretName)

	if err != nil {
		if err == secret.ErrSecretNotExists {
			return nil
		}
		return errors.Wrapf(err, "retrieving secret with name '%s' from Vault failed", secretName)
	}

	err = a.awsSessionFactory.secretStore.Delete(input.OrganizationID, secretItem.ID)
	if err != nil {
		return err
	}
	return nil
}
