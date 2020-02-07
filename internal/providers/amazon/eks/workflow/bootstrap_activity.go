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
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"emperror.dev/errors"
	"github.com/Masterminds/semver"
	"go.uber.org/cadence/activity"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/eks"
	"github.com/ghodss/yaml"
	v1 "k8s.io/api/core/v1"

	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	storageUtil "k8s.io/kubernetes/pkg/apis/storage/util"

	k8serr "k8s.io/apimachinery/pkg/api/errors"

	"github.com/banzaicloud/pipeline/internal/providers/amazon"
	"github.com/banzaicloud/pipeline/pkg/cadence"
	"github.com/banzaicloud/pipeline/pkg/k8sclient"
)

const BootstrapActivityName = "eks-bootstrap"

const mapRolesTemplate = `- rolearn: %s
  username: system:node:{{EC2PrivateDNSName}}
  groups:
  - system:bootstrappers
  - system:nodes
`

const mapUsersTemplate = `- userarn: %s
  username: %s
  groups:
  - system:masters
`

// nolint: gochecknoglobals
var iamRoleARNRegexp = regexp.MustCompile("arn:aws:iam::([0-9]{12}):role")

func findResourceInRoleARN(iamRoleARN string) (string, string) {
	arnMeta := iamRoleARNRegexp.Find([]byte(iamRoleARN))
	return string(arnMeta), strings.TrimPrefix(iamRoleARN, string(arnMeta))
}

// CreateEksControlPlaneActivity creates aws-auth map & default StorageClass on cluster
type BootstrapActivity struct {
	awsSessionFactory *AWSSessionFactory
}

// BootstrapActivityInput holds input data
type BootstrapActivityInput struct {
	EKSActivityInput

	KubernetesVersion   string
	NodeInstanceRoleArn string
	ClusterUserArn      string
}

// BootstrapActivityOutput holds the output data
type BootstrapActivityOutput struct {
}

// BootstrapActivity instantiates a new BootstrapActivity
func NewBootstrapActivity(awsSessionFactory *AWSSessionFactory) *BootstrapActivity {
	return &BootstrapActivity{
		awsSessionFactory: awsSessionFactory,
	}
}

func (a *BootstrapActivity) Execute(ctx context.Context, input BootstrapActivityInput) (*BootstrapActivityOutput, error) {
	logger := activity.GetLogger(ctx).Sugar().With(
		"organization", input.OrganizationID,
		"cluster", input.ClusterName,
		"region", input.Region,
		"version", input.KubernetesVersion,
	)

	awsSession, err := a.awsSessionFactory.New(input.OrganizationID, input.SecretID, input.Region)
	if err = errors.WrapIf(err, "failed to create AWS session"); err != nil {
		return nil, err
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

	kubeClient, err := a.getKubeClient(eksSvc, input)
	if err = errors.WrapIf(err, "failed to retrieve K8s client"); err != nil {
		return nil, err
	}

	constraint, err := semver.NewConstraint(">= 1.12")
	if err != nil {
		return nil, errors.WrapIf(err, "could not set  1.12 constraint for semver")
	}
	kubeVersion, err := semver.NewVersion(input.KubernetesVersion)
	if err != nil {
		return nil, errors.WrapIf(err, "could not set eks version for semver check")
	}
	var volumeBindingMode storagev1.VolumeBindingMode
	if constraint.Check(kubeVersion) {
		volumeBindingMode = storagev1.VolumeBindingWaitForFirstConsumer
	} else {
		volumeBindingMode = storagev1.VolumeBindingImmediate
	}

	storageClassConstraint, err := semver.NewConstraint("< 1.11")
	if err != nil {
		return nil, errors.WrapIf(err, "could not set  1.11 constraint for semver")
	}

	if storageClassConstraint.Check(kubeVersion) {
		// create default storage class
		err = createDefaultStorageClass(kubeClient, "kubernetes.io/aws-ebs", volumeBindingMode, nil)
		if err != nil {
			return nil, errors.WrapIfWithDetails(err, "failed to create default storage class",
				"provisioner", "kubernetes.io/aws-ebs",
				"bindingMode", volumeBindingMode)
		}
	}

	logger.Debug("creating aws-auth configmap")

	mapRoles := fmt.Sprintf(mapRolesTemplate, input.NodeInstanceRoleArn)

	// The aws-iam-authenticator doesn't handle path currently in role mappings:
	// https://github.com/kubernetes-sigs/aws-iam-authenticator/issues/268
	// Once this bug gets fixed our code won't work, so we are making it future
	// compatible by adding the role id with and without path to the mapping.
	roleMeta, roleResource := findResourceInRoleARN(input.NodeInstanceRoleArn)
	roleID, rolePath := splitResourceId(roleResource)
	if rolePath != "/" {
		mapRoles += fmt.Sprintf(mapRolesTemplate, roleMeta+"/"+roleID)
	}

	awsAuthConfigMap := v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: "aws-auth"},
		Data: map[string]string{
			"mapRoles": mapRoles,
			"mapUsers": fmt.Sprintf(mapUsersTemplate, input.ClusterUserArn, userNameFromArn(input.ClusterUserArn)),
		},
	}

	_, err = kubeClient.CoreV1().ConfigMaps("kube-system").Create(&awsAuthConfigMap)
	if k8serr.ReasonForError(err) == metav1.StatusReasonAlreadyExists {
		_, err = kubeClient.CoreV1().ConfigMaps("kube-system").Update(&awsAuthConfigMap)
		if err != nil {
			return nil, errors.WrapIfWithDetails(err, "failed to create config map", "configmap", awsAuthConfigMap.Name)
		}
	} else if err != nil {
		return nil, errors.WrapIfWithDetails(err, "failed to create config map", "configmap", awsAuthConfigMap.Name)
	}

	ds, err := kubeClient.AppsV1().DaemonSets("kube-system").Get("aws-node", metav1.GetOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get CNI driver daemonset")
	}

	tags := map[string]string{}

	var envVars []v1.EnvVar

	for _, envVar := range ds.Spec.Template.Spec.Containers[0].Env {
		if envVar.Name == "ADDITIONAL_ENI_TAGS" {
			// omit invalid JSONs
			_ = json.Unmarshal([]byte(envVar.Value), &tags)

			continue
		}

		envVars = append(envVars, envVar)
	}

	for _, tag := range amazon.PipelineTags() {
		tags[aws.StringValue(tag.Key)] = aws.StringValue(tag.Value)
	}

	tagBody, err := json.Marshal(tags)
	if err != nil {
		return nil, cadence.NewClientError(err)
	}

	ds.Spec.Template.Spec.Containers[0].Env = append(envVars, v1.EnvVar{
		Name:  "ADDITIONAL_ENI_TAGS",
		Value: string(tagBody),
	})

	_, err = kubeClient.AppsV1().DaemonSets("kube-system").Update(ds)
	if err != nil {
		return nil, errors.Wrap(err, "failed to update CNI driver daemonset")
	}

	outParams := BootstrapActivityOutput{}
	return &outParams, nil
}

func (a *BootstrapActivity) getKubeClient(eksSvc *eks.EKS, input BootstrapActivityInput) (*kubernetes.Clientset, error) {
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

	awsCreds, err := a.awsSessionFactory.GetAWSCredentials(input.OrganizationID, input.SecretID, input.Region)
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

	restKubeConfig, err := k8sclient.NewClientConfig(kubeConfig)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to create K8S config object")
	}

	kubeClient, err := kubernetes.NewForConfig(restKubeConfig)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to create K8S client")
	}

	return kubeClient, nil
}

// CreateDefaultStorageClass creates a default storage class as some clusters are not created with
// any storage classes or with default one
func createDefaultStorageClass(kubernetesClient *kubernetes.Clientset, provisioner string, volumeBindingMode storagev1.VolumeBindingMode, parameters map[string]string) error {
	defaultStorageClass := storagev1.StorageClass{
		ObjectMeta: metav1.ObjectMeta{
			Name: "default",
			Annotations: map[string]string{
				storageUtil.IsDefaultStorageClassAnnotation: "true",
			},
		},
		VolumeBindingMode: &volumeBindingMode,
		Provisioner:       provisioner,
		Parameters:        parameters,
	}

	_, err := kubernetesClient.StorageV1().StorageClasses().Create(&defaultStorageClass)
	if k8serr.ReasonForError(err) == metav1.StatusReasonAlreadyExists {
		_, err = kubernetesClient.StorageV1().StorageClasses().Update(&defaultStorageClass)
		if err != nil {
			return errors.WrapIf(err, "create storage class failed")
		}
	}

	return nil
}

func userNameFromArn(arn string) string {
	idx := strings.LastIndex(arn, "/")

	return arn[idx+1:]
}
