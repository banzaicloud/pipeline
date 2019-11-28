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

package action

import (
	"fmt"
	"strings"

	"emperror.dev/errors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/sirupsen/logrus"

	"github.com/banzaicloud/pipeline/internal/secret/secrettype"
	"github.com/banzaicloud/pipeline/pkg/amazon"
	"github.com/banzaicloud/pipeline/pkg/cluster"
	"github.com/banzaicloud/pipeline/src/secret"
	"github.com/banzaicloud/pipeline/src/utils"
)

var _ utils.RevocableAction = (*CreateClusterUserAccessKeyAction)(nil)

// CreateClusterUserAccessKeyAction describes the cluster user to create access key and secret for.
type CreateClusterUserAccessKeyAction struct {
	context *EksClusterCreateUpdateContext
	log     logrus.FieldLogger
}

//
func NewCreateClusterUserAccessKeyAction(log logrus.FieldLogger, creationContext *EksClusterCreateUpdateContext) *CreateClusterUserAccessKeyAction {
	return &CreateClusterUserAccessKeyAction{
		context: creationContext,
		log:     log,
	}
}

// GetName returns the name of this CreateClusterUserAccessKeyAction
func (a *CreateClusterUserAccessKeyAction) GetName() string {
	return "CreateClusterUserAccessKeyAction"
}

// ExecuteAction executes this CreateClusterUserAccessKeyAction
func (a *CreateClusterUserAccessKeyAction) ExecuteAction(input interface{}) (output interface{}, err error) {
	a.log.Infoln("EXECUTE CreateClusterUserAccessKeyAction, cluster user name: ", a.context.ClusterName)

	iamSvc := iam.New(a.context.Session)
	clusterUserName := aws.String(a.context.ClusterName)

	accessKey, err := amazon.CreateUserAccessKey(iamSvc, clusterUserName)
	if err != nil {
		return nil, err
	}

	a.context.ClusterUserAccessKeyId = aws.StringValue(accessKey.AccessKeyId)
	a.context.ClusterUserSecretAccessKey = aws.StringValue(accessKey.SecretAccessKey)

	return nil, nil
}

// UndoAction rolls back this CreateClusterUserAccessKeyAction
func (a *CreateClusterUserAccessKeyAction) UndoAction() error {
	a.log.Infof("EXECUTE UNDO CreateClusterUserAccessKeyAction, deleting cluster user access key: %s", a.context.ClusterUserAccessKeyId)

	iamSvc := iam.New(a.context.Session)
	clusterUserName := aws.String(a.context.ClusterName)

	err := amazon.DeleteUserAccessKey(iamSvc, clusterUserName, aws.String(a.context.ClusterUserAccessKeyId))
	return err
}

// ---

var _ utils.RevocableAction = (*PersistClusterUserAccessKeyAction)(nil)

// PersistClusterUserAccessKeyAction describes the cluster user access key to be persisted
type PersistClusterUserAccessKeyAction struct {
	context        *EksClusterCreateUpdateContext
	organizationID uint
	log            logrus.FieldLogger
}

// NewPersistClusterUserAccessKeyAction creates a new PersistClusterUserAccessKeyAction
func NewPersistClusterUserAccessKeyAction(log logrus.FieldLogger, context *EksClusterCreateUpdateContext, orgID uint) *PersistClusterUserAccessKeyAction {
	return &PersistClusterUserAccessKeyAction{
		context:        context,
		organizationID: orgID,
		log:            log,
	}
}

// GetName returns the name of this PersistClusterUserAccessKeyAction
func (a *PersistClusterUserAccessKeyAction) GetName() string {
	return "PersistClusterUserAccessKeyAction"
}

// getSecretName returns the name that identifies the  cluster user access key in Vault
func getSecretName(userName string) string {
	return fmt.Sprintf("%s-key", strings.ToLower(userName))
}

// GetClusterUserAccessKeyIdAndSecretVault returns the AWS access key and access key secret from Vault
// for cluster user name
func GetClusterUserAccessKeyIdAndSecretVault(organizationID uint, userName string) (string, string, error) {
	secretName := getSecretName(userName)
	secretItem, err := secret.Store.GetByName(organizationID, secretName)
	if err != nil {
		return "", "", errors.WrapWithDetails(err, "failed to get secret from Vault", "secret", secretName)
	}
	clusterUserAccessKeyId := secretItem.GetValue(secrettype.AwsAccessKeyId)
	clusterUserSecretAccessKey := secretItem.GetValue(secrettype.AwsSecretAccessKey)

	return clusterUserAccessKeyId, clusterUserSecretAccessKey, nil
}

// ExecuteAction executes this PersistClusterUserAccessKeyAction
func (a *PersistClusterUserAccessKeyAction) ExecuteAction(input interface{}) (output interface{}, err error) {
	a.log.Info("EXECUTE PersistClusterUserAccessKeyAction")

	secretName := getSecretName(a.context.ClusterName)
	secretRequest := secret.CreateSecretRequest{
		Name: secretName,
		Type: cluster.Amazon,
		Values: map[string]string{
			secrettype.AwsAccessKeyId:     a.context.ClusterUserAccessKeyId,
			secrettype.AwsSecretAccessKey: a.context.ClusterUserSecretAccessKey,
		},
		Tags: []string{
			fmt.Sprintf("eksClusterUserAccessKey:%s", a.context.ClusterName),
			secret.TagBanzaiHidden,
		},
	}

	if _, err := secret.Store.Store(a.organizationID, &secretRequest); err != nil {
		return nil, errors.WrapIff(err, "failed to create/update secret: %s", secretName)
	}

	return nil, nil
}

// UndoAction rools back this PersistClusterUserAccessKeyAction
func (a *PersistClusterUserAccessKeyAction) UndoAction() error {
	a.log.Info("EXECUTE UNDO PersistClusterUserAccessKeyAction")

	secretItem, err := secret.Store.GetByName(a.organizationID, getSecretName(a.context.ClusterName))

	if err != nil && err != secret.ErrSecretNotExists {
		return err
	}

	if secretItem != nil {
		return secret.Store.Delete(a.organizationID, secretItem.ID)
	}

	return nil
}

var _ utils.Action = (*DeleteClusterUserAccessKeyAction)(nil)

// DeleteClusterUserAccessKeyAction deletes all access keys of cluster user
type DeleteClusterUserAccessKeyAction struct {
	context *EksClusterDeletionContext
	log     logrus.FieldLogger
}

// NewDeleteClusterUserAccessKeyAction creates a new DeleteClusterUserAccessKeyAction
func NewDeleteClusterUserAccessKeyAction(log logrus.FieldLogger, context *EksClusterDeletionContext) *DeleteClusterUserAccessKeyAction {
	return &DeleteClusterUserAccessKeyAction{
		context: context,
		log:     log,
	}
}

// GetName returns the name of this DeleteClusterUserAccessKeyAction
func (a *DeleteClusterUserAccessKeyAction) GetName() string {
	return "DeleteClusterUserAccessKeyAction"
}

func (a *DeleteClusterUserAccessKeyAction) ExecuteAction(input interface{}) (output interface{}, err error) {
	iamSvc := iam.New(a.context.Session)
	clusterUserName := aws.String(a.context.ClusterName)

	a.log.Infof("EXECUTE DeleteClusterUserAccessKeyAction: %q", *clusterUserName)

	awsAccessKeys, err := amazon.GetUserAccessKeys(iamSvc, clusterUserName)

	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			if awsErr.Code() == iam.ErrCodeNoSuchEntityException {
				return nil, nil
			}
		}
		a.log.Errorf("querying IAM user '%s' access keys failed: %s", *clusterUserName, err)
		return nil, errors.Wrapf(err, "querying IAM user '%s' access keys failed", *clusterUserName)
	}

	for _, awsAccessKey := range awsAccessKeys {
		if err := amazon.DeleteUserAccessKey(iamSvc, awsAccessKey.UserName, awsAccessKey.AccessKeyId); err != nil {
			if awsErr, ok := err.(awserr.Error); ok {
				if awsErr.Code() == iam.ErrCodeNoSuchEntityException {
					continue
				}
			}

			a.log.Errorf("deleting Amazon user access key '%s', user '%s' failed: %s",
				aws.StringValue(awsAccessKey.AccessKeyId),
				aws.StringValue(awsAccessKey.UserName), err)

			return nil, errors.Wrapf(err, "deleting Amazon access key '%s', user '%s' failed",
				aws.StringValue(awsAccessKey.AccessKeyId),
				aws.StringValue(awsAccessKey.UserName))
		}
	}

	return nil, nil
}

// --

var _ utils.Action = (*DeleteClusterUserAccessKeySecretAction)(nil)

// DeleteClusterUserAccessKeySecretAction deletes cluster user access key from Vault
type DeleteClusterUserAccessKeySecretAction struct {
	context        *EksClusterDeletionContext
	organizationID uint
	log            logrus.FieldLogger
}

// NewDeleteClusterUserAccessKeySecretAction creates a new DeleteClusterUserAccessKeySecretAction
func NewDeleteClusterUserAccessKeySecretAction(log logrus.FieldLogger, context *EksClusterDeletionContext, orgID uint) *DeleteClusterUserAccessKeySecretAction {
	return &DeleteClusterUserAccessKeySecretAction{
		context:        context,
		organizationID: orgID,
		log:            log,
	}
}

// GetName returns the name of this DeleteClusterUserAccessKeySecretAction
func (a *DeleteClusterUserAccessKeySecretAction) GetName() string {
	return "DeleteClusterUserAccessKeySecretAction"
}

// ExecuteAction executes this DeleteClusterUserAccessKeySecretAction
func (a *DeleteClusterUserAccessKeySecretAction) ExecuteAction(input interface{}) (interface{}, error) {
	a.log.Infoln("EXECUTE DeleteClusterUserAccessKeySecretAction")

	secretName := getSecretName(a.context.ClusterName)
	secretItem, err := secret.Store.GetByName(a.organizationID, secretName)

	if err != nil {
		if err == secret.ErrSecretNotExists {
			return nil, nil
		}
		return nil, errors.Wrapf(err, "retrieving secret with name '%s' from Vault failed", secretName)
	}

	err = secret.Store.Delete(a.organizationID, secretItem.ID)

	return nil, err
}
