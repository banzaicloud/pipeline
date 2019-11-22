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
	"fmt"

	"emperror.dev/errors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/iam"
	"go.uber.org/cadence/activity"

	"github.com/banzaicloud/pipeline/internal/secret/secrettype"
	"github.com/banzaicloud/pipeline/pkg/amazon"
	"github.com/banzaicloud/pipeline/pkg/cluster"
	"github.com/banzaicloud/pipeline/secret"
)

const CreateClusterUserAccessKeyActivityName = "eks-create-cluster-user-access-key"

// CreateClusterUserAccessKeyActivity responsible for creating IAM user access key for the cluster user
// and storing the access in secret store
type CreateClusterUserAccessKeyActivity struct {
	awsSessionFactory *AWSSessionFactory
}

// CreateClusterUserAccessKeyActivityInput holds data needed for setting up IAM user access key for the cluster user
type CreateClusterUserAccessKeyActivityInput struct {
	EKSActivityInput

	UserName       string
	UseDefaultUser bool
}

type CreateClusterUserAccessKeyActivityOutput struct {
	SecretID string
}

// NewCreateClusterUserAccessKeyActivity instantiates a CreateClusterUserAccessKeyActivity
func NewCreateClusterUserAccessKeyActivity(awsSessionFactory *AWSSessionFactory) *CreateClusterUserAccessKeyActivity {
	return &CreateClusterUserAccessKeyActivity{
		awsSessionFactory: awsSessionFactory,
	}
}

func (a *CreateClusterUserAccessKeyActivity) Execute(ctx context.Context, input CreateClusterUserAccessKeyActivityInput) (*CreateClusterUserAccessKeyActivityOutput, error) {
	logger := activity.GetLogger(ctx).Sugar().With(
		"organization", input.OrganizationID,
		"cluster", input.ClusterName,
		"user", input.UserName,
	)

	var accessKey *iam.AccessKey

	secretName := getSecretName(input.UserName)

	clusterUserAccessKeySecret, err := a.awsSessionFactory.GetSecretStore().GetByName(input.OrganizationID, secretName)
	if err != nil && err != secret.ErrSecretNotExists {
		return nil, errors.WrapIfWithDetails(err, "failed to verify if secret exists in secret store", "secretName", secretName)
	}

	if !input.UseDefaultUser {
		session, err := a.awsSessionFactory.New(input.OrganizationID, input.SecretID, input.Region)
		if err = errors.WrapIf(err, "failed to create AWS session"); err != nil {
			return nil, err
		}

		iamSvc := iam.New(session)
		clusterUserName := aws.String(input.UserName)

		createNewAccessKey := true

		userAccessKeys, err := amazon.GetUserAccessKeys(iamSvc, clusterUserName)
		if err = errors.WrapIfWithDetails(err, "failed to retrieve IAM user access keys for user", "user", input.UserName); err != nil {
			return nil, err
		}

		// if either the Amazon access key or it's corresponding secret is missing from secret store
		// we need to create(re-create in case of re-run) the Amazon access key
		// as the Amazon access secret can be obtained only at creation
		var userAccessKeyMap = make(map[string]*iam.AccessKeyMetadata)
		for _, userAccessKey := range userAccessKeys {
			userAccessKeyMap[aws.StringValue(userAccessKey.AccessKeyId)] = userAccessKey
		}

		if clusterUserAccessKeySecret != nil {
			if clusterUserAwsAccessKeyId, ok := clusterUserAccessKeySecret.Values[secrettype.AwsAccessKeyId]; ok {
				if _, ok := userAccessKeyMap[clusterUserAwsAccessKeyId]; ok {
					createNewAccessKey = false // the access key in Amazon and Vault matches, no need to create a new onw
				}
			}
		}

		if createNewAccessKey {
			if len(userAccessKeyMap) == 2 {
				// IAM user can not have more than 2 access keys
				for k := range userAccessKeyMap {
					err = amazon.DeleteUserAccessKey(iamSvc, clusterUserName, aws.String(k))
					if err = errors.WrapIfWithDetails(err, "couldn't delete IAM user access key", "user", input.UserName, "accessKeyID", k); err != nil {
						return nil, err
					}
				}
			}

			logger.Info("creating IAM user access key")

			accessKey, err = amazon.CreateUserAccessKey(iamSvc, clusterUserName)
			if err = errors.WrapIfWithDetails(err, "failed to create IAM user access key for user", "user", input.UserName); err != nil {
				return nil, err
			}
		} else {
			logger.Info("skip creating IAM user access as already exists")

			return &CreateClusterUserAccessKeyActivityOutput{
				SecretID: clusterUserAccessKeySecret.ID,
			}, nil
		}
	} else {
		logger.Debug("use IAM user access key of default user")

		awsCreds, err := a.awsSessionFactory.GetAWSCredentials(input.OrganizationID, input.SecretID, input.Region)
		if err = errors.WrapIf(err, "failed to retrieve AWS credentials"); err != nil {
			return nil, err
		}

		awsCredsFields, err := awsCreds.Get()
		if err = errors.WrapIf(err, "failed to AWS credential fields"); err != nil {
			return nil, err
		}

		// default user's access key is already set up and is out of the scope of this activity

		accessKey = &iam.AccessKey{
			AccessKeyId:     aws.String(awsCredsFields.AccessKeyID),
			SecretAccessKey: aws.String(awsCredsFields.SecretAccessKey),
		}
	}

	secretRequest := secret.CreateSecretRequest{
		Name: secretName,
		Type: cluster.Amazon,
		Values: map[string]string{
			secrettype.AwsAccessKeyId:     aws.StringValue(accessKey.AccessKeyId),
			secrettype.AwsSecretAccessKey: aws.StringValue(accessKey.SecretAccessKey),
		},
		Tags: []string{
			fmt.Sprintf("eksClusterUserAccessKey:%s", input.ClusterName),
			secret.TagBanzaiHidden,
		},
	}

	var secretID string
	if clusterUserAccessKeySecret != nil {
		ver := int(clusterUserAccessKeySecret.Version)
		secretRequest.Version = &ver

		if err = a.awsSessionFactory.GetSecretStore().Update(input.OrganizationID, clusterUserAccessKeySecret.ID, &secretRequest); err != nil {
			return nil, errors.WrapIff(err, "failed to update secret: %s", secretName)
		}
		secretID = clusterUserAccessKeySecret.ID
	} else {
		secretID, err = a.awsSessionFactory.GetSecretStore().Store(input.OrganizationID, &secretRequest)
		if err = errors.WrapIff(err, "failed to create secret: %s", secretName); err != nil {
			return nil, err
		}
	}

	return &CreateClusterUserAccessKeyActivityOutput{
		SecretID: secretID,
	}, nil
}
