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
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api/v1"

	"github.com/banzaicloud/pipeline/secret"

	internalAmazon "github.com/banzaicloud/pipeline/internal/providers/amazon"
)

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

func generateStackNameForCluster(clusterName string) string {
	return "pipeline-eks-" + clusterName
}

func generateStackNameForSubnet(clusterName, subnetCidr string) string {
	r := strings.NewReplacer(".", "-", "/", "-")
	return fmt.Sprintf("pipeline-eks-subnet-%s-%s", clusterName, r.Replace(subnetCidr))
}

func generateStackNameForIam(clusterName string) string {
	return "pipeline-eks-iam-" + clusterName
}

func generateSSHKeyNameForCluster(clusterName string) string {
	return "pipeline-eks-ssh-" + clusterName
}

func generateNodePoolStackName(clusterName string, asgName string) string {
	return "pipeline-eks-nodepool-" + clusterName + "-" + asgName
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
}

type SecretStore interface {
	Get(orgnaizationID uint, secretID string) (*secret.SecretItemResponse, error)
	GetByName(orgnaizationID uint, secretID string) (*secret.SecretItemResponse, error)
	Store(organizationID uint, request *secret.CreateSecretRequest) (string, error)
}
