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
	"github.com/Masterminds/semver/v3"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/eks"
	"github.com/ghodss/yaml"
	"go.uber.org/cadence/activity"
	v1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	storageUtil "k8s.io/kubernetes/pkg/apis/storage/util"

	"github.com/banzaicloud/pipeline/internal/providers/amazon"
	"github.com/banzaicloud/pipeline/pkg/cadence"
	"github.com/banzaicloud/pipeline/pkg/k8sclient"
)

const BootstrapActivityName = "eks-bootstrap"

const mapRolesTemplate = `    - rolearn: %s
      username: system:node:{{EC2PrivateDNSName}}
      groups:
      - system:bootstrappers
      - system:nodes
`

const mapUsersTemplate = `    - userarn: %s
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
	AuthConfigMap       string
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
		return nil, errors.WrapIf(err, "could not set 1.12 constraint for semver")
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
		err = createDefaultStorageClass(ctx, kubeClient, "kubernetes.io/aws-ebs", volumeBindingMode, nil)
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

	mapUsers := fmt.Sprintf(mapUsersTemplate, input.ClusterUserArn, userNameFromArn(input.ClusterUserArn))
	defaultAWSConfigMap := fmt.Sprintf(`apiVersion: v1
kind: ConfigMap
metadata:
  name: aws-auth
  namespace: kube-system
data:
  mapRoles: |
%s
  mapUsers: |
%s`, mapRoles, mapUsers)

	mergedConfigMap, err := mergeAuthConfigMaps(defaultAWSConfigMap, input.AuthConfigMap)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to merge config map")
	}

	_, err = kubeClient.CoreV1().ConfigMaps("kube-system").Create(ctx, mergedConfigMap, metav1.CreateOptions{})
	if k8serr.ReasonForError(err) == metav1.StatusReasonAlreadyExists {
		_, err = kubeClient.CoreV1().ConfigMaps("kube-system").Update(ctx, mergedConfigMap, metav1.UpdateOptions{})
		if err != nil {
			return nil, errors.WrapIfWithDetails(err, "failed to update config map", "configmap", mergedConfigMap.Name)
		}
	} else if err != nil {
		return nil, errors.WrapIfWithDetails(err, "failed to create config map", "configmap", mergedConfigMap.Name)
	}

	ds, err := kubeClient.AppsV1().DaemonSets("kube-system").Get(ctx, "aws-node", metav1.GetOptions{})
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

	_, err = kubeClient.AppsV1().DaemonSets("kube-system").Update(ctx, ds, metav1.UpdateOptions{})
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
func createDefaultStorageClass(ctx context.Context, kubernetesClient *kubernetes.Clientset, provisioner string, volumeBindingMode storagev1.VolumeBindingMode, parameters map[string]string) error {
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

	_, err := kubernetesClient.StorageV1().StorageClasses().Create(ctx, &defaultStorageClass, metav1.CreateOptions{})
	if k8serr.ReasonForError(err) == metav1.StatusReasonAlreadyExists {
		_, err = kubernetesClient.StorageV1().StorageClasses().Update(ctx, &defaultStorageClass, metav1.UpdateOptions{})
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

type mergeFunc func(key, defaultData, inputData string) ([]byte, error)

func mergeAuthConfigMaps(defaultConfigMap, inputConfigMap string) (*v1.ConfigMap, error) {
	var defaultAWSConfigMap v1.ConfigMap
	_, _, err := scheme.Codecs.UniversalDeserializer().Decode([]byte(defaultConfigMap), nil, &defaultAWSConfigMap)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to decode default config map")
	}

	var inputAWSConfigMap v1.ConfigMap
	_, _, err = scheme.Codecs.UniversalDeserializer().Decode([]byte(inputConfigMap), nil, &inputAWSConfigMap)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to decode input config map")
	}

	mergedConfigMap := defaultAWSConfigMap

	functions := map[string]mergeFunc{
		"mapUsers":    mergeDefaultMapWithInput,
		"mapRoles":    mergeDefaultMapWithInput,
		"mapAccounts": mergeDefaultStrArrayWithInput,
	}

	for key, merge := range functions {
		mergedValue, err := merge(key, defaultAWSConfigMap.Data[key], inputAWSConfigMap.Data[key])
		if err != nil {
			return nil, errors.WrapIf(err, "failed to merge "+key+" default value and input")
		}

		mergedConfigMap.Data[key] = string(mergedValue)
	}

	return &mergedConfigMap, nil
}

func mergeDefaultStrArrayWithInput(key, defaultData, inputData string) ([]byte, error) {
	var defaultValue []string
	err := yaml.Unmarshal([]byte(defaultData), &defaultValue)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to unmarshal "+key+" from: "+defaultData)
	}

	var inputValue []string
	err = yaml.Unmarshal([]byte(inputData), &inputValue)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to unmarshal "+key+" from: "+inputData)
	}

	var result []byte
	value := append(defaultValue, inputValue...)
	if value != nil {
		result, err = yaml.Marshal(value)
		if err != nil {
			return nil, errors.WrapIf(err, "failed to marshal "+key+" to yaml")
		}
	}

	return result, nil
}

func mergeDefaultMapWithInput(key, defaultData, inputData string) ([]byte, error) {
	var defaultValue []map[string]interface{}
	err := yaml.Unmarshal([]byte(defaultData), &defaultValue)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to unmarshal "+key+" from: "+defaultData)
	}

	var inputValue []map[string]interface{}
	err = yaml.Unmarshal([]byte(inputData), &inputValue)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to unmarshal "+key+" from: "+inputData)
	}

	var result []byte
	value := append(defaultValue, inputValue...)
	if value != nil {
		result, err = yaml.Marshal(value)
		if err != nil {
			return nil, errors.WrapIf(err, "failed to marshal "+key+" to yaml")
		}
	}

	return result, nil
}
