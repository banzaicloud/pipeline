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
	"encoding/base64"
	"fmt"

	"emperror.dev/errors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/eks"
	"github.com/ghodss/yaml"
	"go.uber.org/cadence/activity"
	"go.uber.org/zap"

	"github.com/banzaicloud/pipeline/internal/cluster/infrastructure/aws/awsworkflow"
	"github.com/banzaicloud/pipeline/internal/secret/secrettype"
	"github.com/banzaicloud/pipeline/src/secret"
	"github.com/banzaicloud/pipeline/src/utils"
)

const SaveK8sConfigActivityName = "eks-save-k8s-config"

type SaveK8sConfigActivityInput struct {
	ClusterID   uint
	ClusterUID  string
	ClusterName string

	OrganizationID   uint
	ProviderSecretID string
	UserSecretID     string
	Region           string
}

type SaveK8sConfigActivity struct {
	awsSessionFactory *awsworkflow.AWSSessionFactory
	manager           Clusters
}

func NewSaveK8sConfigActivity(awsSessionFactory *awsworkflow.AWSSessionFactory, manager Clusters) SaveK8sConfigActivity {
	return SaveK8sConfigActivity{
		awsSessionFactory: awsSessionFactory,
		manager:           manager,
	}
}

func (a SaveK8sConfigActivity) Execute(ctx context.Context, input SaveK8sConfigActivityInput) (string, error) {
	logger := activity.GetLogger(ctx).Sugar().With("clusterId", input.ClusterID)

	commonCluster, err := a.manager.GetCluster(ctx, input.ClusterID)
	if err != nil {
		return "", err
	}

	if secretID := commonCluster.GetConfigSecretId(); secretID != "" {
		logger.Info("config is already present in Vault")

		return secretID, nil
	}

	awsSession, err := a.awsSessionFactory.New(input.OrganizationID, input.ProviderSecretID, input.Region)
	if err = errors.WrapIf(err, "failed to create AWS session"); err != nil {
		return "", err
	}
	eksSvc := eks.New(
		awsSession,
		aws.NewConfig().
			WithLogger(aws.LoggerFunc(
				func(args ...interface{}) {
					logger.Debug(args)
				})).
			WithLogLevel(aws.LogDebugWithHTTPBody),
	)

	activityInfo := activity.GetInfo(ctx)

	// On the first attempt try to get an existing config
	if activityInfo.Attempt == 0 {
		logger.Info("trying to get config for the first time")

		config, err := a.getK8sConfig(eksSvc, input)
		if err == nil && len(config) > 0 {
			logger.Info("saving existing config")

			if err := a.storeConfig(logger, commonCluster, config, input); err != nil {
				return "", err
			}
			return commonCluster.GetConfigSecretId(), nil
		}
	}

	return commonCluster.GetConfigSecretId(), nil
}

func (a *SaveK8sConfigActivity) getK8sConfig(eksSvc *eks.EKS, input SaveK8sConfigActivityInput) ([]byte, error) {
	describeClusterInput := &eks.DescribeClusterInput{
		Name: aws.String(input.ClusterName),
	}

	clusterInfo, err := eksSvc.DescribeCluster(describeClusterInput)
	if err != nil {
		return nil, err
	}
	cluster := clusterInfo.Cluster
	if cluster == nil {
		return nil, errors.New("unable to get EKS Cluster info")
	}

	apiEndpoint := aws.StringValue(cluster.Endpoint)
	certificateAuthorityData, err := base64.StdEncoding.DecodeString(aws.StringValue(cluster.CertificateAuthority.Data))
	if err != nil {
		return nil, err
	}

	awsCreds, err := a.awsSessionFactory.GetAWSCredentials(input.OrganizationID, input.UserSecretID, input.Region)
	if err = errors.WrapIf(err, "failed to retrieve AWS credentials"); err != nil {
		return nil, err
	}

	awsCredsFields, err := awsCreds.Get()
	if err = errors.WrapIf(err, "failed to AWS credential fields"); err != nil {
		return nil, err
	}

	k8sCfg := generateK8sConfig(input.ClusterName, apiEndpoint, certificateAuthorityData, awsCredsFields.AccessKeyID, awsCredsFields.SecretAccessKey)
	kubeConfig, err := yaml.Marshal(k8sCfg)
	if err != nil {
		return nil, err
	}
	return kubeConfig, nil
}

func (a *SaveK8sConfigActivity) storeConfig(logger *zap.SugaredLogger, cluster EksCluster, raw []byte, input SaveK8sConfigActivityInput) error {
	configYaml := string(raw)
	encodedConfig := utils.EncodeStringToBase64(configYaml)

	clusterUidTag := fmt.Sprintf("clusterUID:%s", input.ClusterUID)

	createSecretRequest := secret.CreateSecretRequest{
		Name: fmt.Sprintf("cluster-%d-config", input.ClusterID),
		Type: secrettype.Kubernetes,
		Values: map[string]string{
			secrettype.K8SConfig: encodedConfig,
		},
		Tags: []string{
			secret.TagKubeConfig,
			secret.TagBanzaiReadonly,
			clusterUidTag,
		},
	}

	secretID := secret.GenerateSecretID(&createSecretRequest)

	// Try to get the secret version first
	if _, err := secret.Store.Get(input.OrganizationID, secretID); err != nil && err != secret.ErrSecretNotExists {
		return err
	}

	err := secret.Store.Update(input.OrganizationID, secretID, &createSecretRequest)
	if err != nil {
		return err
	}

	logger.Info("Kubeconfig stored in vault")

	logger.Info("Update cluster model in DB with config secret id")
	if err := cluster.SaveConfigSecretId(secretID); err != nil {
		logger.Errorf("Error during saving config secret id: %s", err.Error())
		return err
	}

	return nil
}
