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
	"strings"

	"emperror.dev/errors"
	"github.com/aws/aws-sdk-go/aws/awserr"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"go.uber.org/cadence"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api/v1"

	"github.com/banzaicloud/pipeline/src/model"

	"github.com/banzaicloud/pipeline/src/secret"

	internalAmazon "github.com/banzaicloud/pipeline/internal/providers/amazon"
	pkgCloudformation "github.com/banzaicloud/pipeline/pkg/providers/amazon/cloudformation"
)

// ErrReasonStackFailed cadence custom error reason that denotes a stack operation that resulted a stack failure
const ErrReasonStackFailed = "CLOUDFORMATION_STACK_FAILED"

// getStackTags returns the tags that are placed onto CF template stacks.
// These tags  are propagated onto the resources created by the CF template.
func getStackTags(clusterName, stackType string) []*cloudformation.Tag {
	return append([]*cloudformation.Tag{
		{Key: aws.String("banzaicloud-pipeline-cluster-name"), Value: aws.String(clusterName)},
		{Key: aws.String("banzaicloud-pipeline-stack-type"), Value: aws.String(stackType)},
	}, internalAmazon.PipelineTags()...)
}

func getNodePoolStackTags(clusterName string) []*cloudformation.Tag {
	return getStackTags(clusterName, "nodepool")
}

func GenerateStackNameForCluster(clusterName string) string {
	return "pipeline-eks-" + clusterName
}

func generateStackNameForSubnet(clusterName, subnetCidr string) string {
	r := strings.NewReplacer(".", "-", "/", "-")
	return fmt.Sprintf("pipeline-eks-subnet-%s-%s", clusterName, r.Replace(subnetCidr))
}

func generateStackNameForIam(clusterName string) string {
	return "pipeline-eks-iam-" + clusterName
}

func GenerateSSHKeyNameForCluster(clusterName string) string {
	return "pipeline-eks-ssh-" + clusterName
}

func GenerateNodePoolStackName(clusterName string, poolName string) string {
	return "pipeline-eks-nodepool-" + clusterName + "-" + poolName
}

// getSecretName returns the name that identifies the  cluster user access key in Vault
func getSecretName(userName string) string {
	return fmt.Sprintf("%s-key", strings.ToLower(userName))
}

func generateK8sConfig(clusterName string, apiEndpoint string, certificateAuthorityData []byte,
	awsAccessKeyID string, awsSecretAccessKey string) *clientcmdapi.Config {
	return &clientcmdapi.Config{
		APIVersion: "v1",
		Clusters: []clientcmdapi.NamedCluster{
			{
				Name: clusterName,
				Cluster: clientcmdapi.Cluster{
					Server:                   apiEndpoint,
					CertificateAuthorityData: certificateAuthorityData,
				},
			},
		},
		Contexts: []clientcmdapi.NamedContext{
			{
				Name: clusterName,
				Context: clientcmdapi.Context{
					AuthInfo: "eks",
					Cluster:  clusterName,
				},
			},
		},
		AuthInfos: []clientcmdapi.NamedAuthInfo{
			{
				Name: "eks",
				AuthInfo: clientcmdapi.AuthInfo{
					Exec: &clientcmdapi.ExecConfig{
						APIVersion: "client.authentication.k8s.io/v1alpha1",
						Command:    "aws-iam-authenticator",
						Args:       []string{"token", "-i", clusterName},
						Env: []clientcmdapi.ExecEnvVar{
							{Name: "AWS_ACCESS_KEY_ID", Value: awsAccessKeyID},
							{Name: "AWS_SECRET_ACCESS_KEY", Value: awsSecretAccessKey},
						},
					},
				},
			},
		},
		Kind:           "Config",
		CurrentContext: clusterName,
	}
}

func generateRequestToken(uuid string, activityName string) string {
	token := uuid + "-" + activityName
	if len(token) > 64 {
		token = token[0:63]
	}
	return token
}

func packageCFError(err error, stackName string, clientRequestToken string, cloudformationClient *cloudformation.CloudFormation, errMessage string) error {
	var awsErr awserr.Error
	if errors.As(err, &awsErr) {
		if awsErr.Code() == request.WaiterResourceNotReadyErrorCode {
			err = pkgCloudformation.NewAwsStackFailure(err, stackName, clientRequestToken, cloudformationClient)
			err = errors.WrapIff(err, errMessage, stackName)
			if pkgCloudformation.IsErrorFinal(err) {
				return cadence.NewCustomError(ErrReasonStackFailed, err.Error())
			}
			return err
		}
	}
	return err
}

// EKSActivityInput holds common input data for all activities
type EKSActivityInput struct {
	OrganizationID uint
	SecretID       string

	Region string

	ClusterName string

	// 64 chars length unique unique identifier that identifies the create CloudFormation
	AWSClientRequestTokenBase string
}

// Subnet holds the fields of a Amazon subnet
type Subnet struct {
	SubnetID         string
	Cidr             string
	AvailabilityZone string
}

type AutoscaleGroup struct {
	Name             string
	NodeSpotPrice    string
	Autoscaling      bool
	NodeMinCount     int
	NodeMaxCount     int
	Count            int
	NodeImage        string
	NodeInstanceType string
	Labels           map[string]string
	Delete           bool
	Create           bool
	CreatedBy        uint
}

type SecretStore interface {
	Get(orgnaizationID uint, secretID string) (*secret.SecretItemResponse, error)
	GetByName(orgnaizationID uint, secretID string) (*secret.SecretItemResponse, error)
	Store(organizationID uint, request *secret.CreateSecretRequest) (string, error)
	Delete(organizationID uint, secretID string) error
	Update(organizationID uint, secretID string, request *secret.CreateSecretRequest) error
}

type Clusters interface {
	GetCluster(ctx context.Context, id uint) (EksCluster, error)
}

type EksCluster interface {
	GetEKSModel() *model.EKSClusterModel
	Persist() error
	SetStatus(string, string) error
	DeleteFromDatabase() error
	GetConfigSecretId() string
	SaveConfigSecretId(secretID string) error
}
