// Copyright © 2021 Banzai Cloud
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

package autoscaler

import (
	"context"
	"fmt"

	"emperror.dev/errors"
	"github.com/Masterminds/semver/v3"
	"github.com/ghodss/yaml"
	"go.uber.org/cadence/activity"
	"go.uber.org/zap"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/banzaicloud/pipeline/internal/global"
	"github.com/banzaicloud/pipeline/internal/providers/azure/azureadapter"
	"github.com/banzaicloud/pipeline/internal/providers/azure/pke"
	"github.com/banzaicloud/pipeline/internal/secret/secrettype"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	"github.com/banzaicloud/pipeline/pkg/k8sclient"
	"github.com/banzaicloud/pipeline/src/cluster"
)

const (
	cloudProviderAzure          = "azure"
	cloudProviderAws            = "aws"
	expanderStrategy            = "least-waste"
	logLevel                    = "5"
	AzureVirtualMachineScaleSet = "vmss"
)

const releaseName = "autoscaler"

// ClusterManager interface to access clusters.
type ClusterManager interface {
	GetClusterByIDOnly(ctx context.Context, clusterID uint) (cluster.CommonCluster, error)
}

type HelmService interface {
	ApplyDeploymentReuseValues(
		ctx context.Context,
		clusterID uint,
		namespace string,
		chartName string,
		releaseName string,
		values []byte,
		chartVersion string,
		reuseValues bool,
	) error
}

type DeployClusterAutoscalerActivityInput struct {
	ClusterID uint
}

type DeployClusterAutoscalerActivity struct {
	manager     ClusterManager
	helmService HelmService
}

type nodeGroup struct {
	Name    string `json:"name"`
	MinSize int    `json:"minSize"`
	MaxSize int    `json:"maxSize"`
}

type rbac struct {
	Create bool `json:"create"`
}

type azureInfo struct {
	ClientID          string `json:"azureClientID"`
	ClientSecret      string `json:"azureClientSecret"`
	SubscriptionID    string `json:"azureSubscriptionID"`
	TenantID          string `json:"azureTenantID"`
	ResourceGroup     string `json:"azureResourceGroup"`
	NodeResourceGroup string `json:"azureNodeResourceGroup"`
	ClusterName       string `json:"azureClusterName"`
	VMType            string `json:"azureVMType,omitempty"`
}

type autoDiscovery struct {
	ClusterName string   `json:"clusterName"`
	Tags        []string `json:"tags"`
}

type autoscalingInfo struct {
	CloudProvider     string            `json:"cloudProvider"`
	AutoscalingGroups []nodeGroup       `json:"autoscalingGroups"`
	ExtraArgs         map[string]string `json:"extraArgs"`
	Rbac              rbac              `json:"rbac"`
	AwsRegion         string            `json:"awsRegion"`
	AutoDiscovery     autoDiscovery     `json:"autoDiscovery"`
	SslCertPath       *string           `json:"sslCertPath,omitempty"`
	SslCertHostPath   *string           `json:"sslCertHostPath,omitempty"`
	Image             map[string]string `json:"image,omitempty"`
	NodeSelector      map[string]string `json:"nodeSelector,omitempty"`
	azureInfo
}

// nolint: gochecknoglobals
// Required for the newAMD64ArchNodeSelector func to compare k8s versions
var comparedK8sSemver *semver.Version = semver.MustParse("1.20.0")

func NewDeployClusterAutoscalerActivity(manager ClusterManager, helmService HelmService) *DeployClusterAutoscalerActivity {
	return &DeployClusterAutoscalerActivity{
		manager:     manager,
		helmService: helmService,
	}
}

func (a DeployClusterAutoscalerActivity) Execute(ctx context.Context, input DeployClusterAutoscalerActivityInput) error {
	config := global.Config.Cluster.PostHook.Autoscaler
	if !config.Enabled {
		return nil
	}

	info := activity.GetInfo(ctx)
	logger := activity.GetLogger(ctx).Sugar().With(
		"clusterID", input.ClusterID,
		"workflowID", info.WorkflowExecution.ID,
		"workflowRunID", info.WorkflowExecution.RunID,
	)

	cluster, err := a.manager.GetClusterByIDOnly(ctx, input.ClusterID)
	if err != nil {
		return err
	}

	err = a.deployAutoscalerChart(logger, cluster)
	if err != nil {
		logger.Error(err.Error())
		return err
	}
	return nil
}

func (a DeployClusterAutoscalerActivity) deployAutoscalerChart(logger *zap.SugaredLogger, cluster cluster.CommonCluster) error {
	var values *autoscalingInfo
	var err error

	// deploy cluster autoscaler only for AWS & Azure
	switch cluster.GetDistribution() {
	case pkgCluster.EKS:
		values, err = createAutoscalingForAws(cluster)
	case pkgCluster.AKS:
		values, err = createAutoscalingForAzure(cluster, "")
	case pkgCluster.PKE:
		switch cluster.GetCloud() {
		case pkgCluster.Amazon:
			values, err = createAutoscalingForAws(cluster)
		case pkgCluster.Azure:
			values, err = createAutoscalingForAzure(cluster, AzureVirtualMachineScaleSet)
		}
	default:
		return nil
	}

	if err != nil {
		return errors.WrapIfWithDetails(err, "Cluster Autoscaler configuration error", "cloud", cluster.GetCloud(), "distribution", cluster.GetDistribution())
	}

	// set image tag & repo depending on K8s version
	values.Image = getImageVersion(logger, cluster)

	logger.With("imageTag", values.Image["tag"]).
		Info("deploying Cluster Autoscaler")

	yamlValues, err := yaml.Marshal(*values)
	if err != nil {
		return errors.WrapIf(err, "Error during marshalling values for Cluster Autoscaler chart")
	}

	chartName := global.Config.Cluster.Autoscale.Charts.ClusterAutoscaler.Chart
	chartVersion := global.Config.Cluster.Autoscale.Charts.ClusterAutoscaler.Version

	err = a.helmService.ApplyDeploymentReuseValues(context.TODO(), cluster.GetID(), global.Config.Cluster.Namespace, chartName, releaseName, yamlValues, chartVersion, true)
	if err != nil {
		return err
	}

	return nil
}

func createAutoscalingForAws(cluster cluster.CommonCluster) (*autoscalingInfo, error) {
	eksCertPath := "/etc/ssl/certs/ca-bundle.crt"
	nodeSelector, err := newAMD64ArchNodeSelector(cluster)
	if err != nil {
		return nil, err
	}

	return &autoscalingInfo{
		CloudProvider: cloudProviderAws,
		ExtraArgs: map[string]string{
			"v":        logLevel,
			"expander": expanderStrategy,
		},
		Rbac:      rbac{Create: true},
		AwsRegion: cluster.GetLocation(),
		AutoDiscovery: autoDiscovery{
			ClusterName: cluster.GetName(),
			Tags: []string{
				"k8s.io/cluster-autoscaler/enabled",
				"kubernetes.io/cluster/" + cluster.GetName(),
			},
		},
		SslCertPath:  &eksCertPath,
		NodeSelector: nodeSelector,
	}, nil
}

func createAutoscalingForAzure(cluster cluster.CommonCluster, vmType string) (*autoscalingInfo, error) {
	clusterSecret, err := cluster.GetSecretWithValidation()
	if err != nil {
		return nil, nil
	}

	nodeSelector, err := newAMD64ArchNodeSelector(cluster)
	if err != nil {
		return nil, err
	}

	nodeGroups, err := getAzureNodeGroups(cluster)
	if err != nil {
		return nil, errors.WrapIf(err, "unable to fetch node groups")
	}

	autoscalingInfo := &autoscalingInfo{
		CloudProvider:     cloudProviderAzure,
		AutoscalingGroups: nodeGroups,
		ExtraArgs: map[string]string{
			"v":        logLevel,
			"expander": expanderStrategy,
		},
		Rbac: rbac{Create: true},
		azureInfo: azureInfo{
			ClientID:       clusterSecret.Values[secrettype.AzureClientID],
			ClientSecret:   clusterSecret.Values[secrettype.AzureClientSecret],
			SubscriptionID: clusterSecret.Values[secrettype.AzureSubscriptionID],
			TenantID:       clusterSecret.Values[secrettype.AzureTenantID],
			ClusterName:    cluster.GetName(),
		},
		NodeSelector: nodeSelector,
	}

	switch cluster.GetDistribution() {
	case pkgCluster.AKS:
		nodeResourceGroup, err := getNodeResourceGroup(cluster)
		if err != nil {
			return nil, errors.WrapIf(err, "Node resource group not found")
		}
		autoscalingInfo.NodeResourceGroup = *nodeResourceGroup

		i, ok := cluster.(interface {
			GetResourceGroupName() string
		})
		if !ok {
			return nil, errors.New("no GetResourceGroupName method found")
		}
		autoscalingInfo.ResourceGroup = i.GetResourceGroupName()

	case pkgCluster.PKE:
		i, ok := cluster.(interface {
			GetResourceGroupName() string
		})
		if !ok {
			return nil, errors.New("no GetResourceGroupName method found")
		}
		autoscalingInfo.ResourceGroup = i.GetResourceGroupName()

		if len(vmType) > 0 {
			autoscalingInfo.VMType = vmType
		}
		sslCertHostPath := "/etc/kubernetes/pki/ca.crt"
		autoscalingInfo.SslCertHostPath = &sslCertHostPath
	}

	return autoscalingInfo, nil
}

func getAzureNodeGroups(cluster cluster.CommonCluster) ([]nodeGroup, error) {
	var nodeGroups []nodeGroup

	switch cluster.GetDistribution() {
	case pkgCluster.AKS:
		i, ok := cluster.(interface {
			GetAKSNodePools() []*azureadapter.AKSNodePoolModel
		})
		if !ok {
			return nil, errors.New("AKS cluster does not implement method GetAKSNodePools")
		}

		nodePools := i.GetAKSNodePools()
		for _, nodePool := range nodePools {
			if nodePool.Autoscaling {
				nodeGroups = append(nodeGroups, nodeGroup{
					Name:    nodePool.Name,
					MinSize: nodePool.NodeMinCount,
					MaxSize: nodePool.NodeMaxCount,
				})
			}
		}
	case pkgCluster.PKE:
		i, ok := cluster.(interface {
			GetPKEOnAzureCluster() pke.Cluster
		})
		if !ok {
			return nil, errors.New("Azure/PKE cluster does not implement method GetPKEOnAzureCluster")
		}

		cl := i.GetPKEOnAzureCluster()
		for _, nodePool := range cl.NodePools {
			if nodePool.Autoscaling {
				nodeGroups = append(nodeGroups, nodeGroup{
					Name:    pke.GetVMSSName(cl.Name, nodePool.Name),
					MinSize: int(nodePool.Min),
					MaxSize: int(nodePool.Max),
				})
			}
		}
	}

	return nodeGroups, nil
}

func getNodeResourceGroup(cluster cluster.CommonCluster) (*string, error) {
	kubeConfig, err := cluster.GetK8sConfig()
	if err != nil {
		return nil, errors.WrapIf(err, "Error getting config: %s")
	}
	client, err := k8sclient.NewClientFromKubeConfig(kubeConfig)
	if err != nil {
		return nil, errors.WrapIf(err, "Error getting k8s connection")
	}

	response, err := client.CoreV1().Nodes().List(context.Background(), meta_v1.ListOptions{})
	if err != nil {
		return nil, errors.WrapIf(err, "Error listing nodes")
	}

	for _, node := range response.Items {
		for labelKey, labelValue := range node.Labels {
			if labelKey == "kubernetes.azure.com/cluster" {
				return &labelValue, nil
			}
		}
	}
	return nil, errors.WrapIf(err, "Node resource group not found on node label")
}

func getK8sVersion(cluster interface{}) (*semver.Version, error) {
	if c, ok := cluster.(interface{ GetKubernetesVersion() (string, error) }); ok {
		k8sVersion, err := c.GetKubernetesVersion()
		if err != nil {
			return nil, err
		}
		version, err := semver.NewVersion(k8sVersion)
		if err != nil {
			return nil, err
		}
		return version, nil
	}
	return nil, errors.New("no GetKubernetesVersion method found")
}

func getImageVersion(logger *zap.SugaredLogger, cluster interface{}) map[string]string {
	var selectedImageVersion map[string]string

	k8sVersion, err := getK8sVersion(cluster)
	if err != nil {
		logger.Error(errors.WrapIf(err, "unable to retrieve K8s version of cluster"))
	} else {
		for _, imageVersion := range global.Config.Cluster.Autoscale.Charts.ClusterAutoscaler.ImageVersionConstraints {
			constraint, err := semver.NewConstraint(imageVersion.K8sVersion)
			if err != nil {
				logger.Error(errors.WrapIf(err, fmt.Sprintf("invalid version constraint specified in config: %s", imageVersion.K8sVersion)))
			} else if constraint.Check(k8sVersion) {
				selectedImageVersion = map[string]string{
					"repository": imageVersion.Repository,
					"tag":        imageVersion.Tag,
				}
				break
			}
		}
	}

	// if no image found for k8s major.minor version choose the latest
	if selectedImageVersion == nil {
		l := len(global.Config.Cluster.Autoscale.Charts.ClusterAutoscaler.ImageVersionConstraints)
		imageVersion := global.Config.Cluster.Autoscale.Charts.ClusterAutoscaler.ImageVersionConstraints[l-1]
		logger.Debugf("no matching image found for cluster, using the latest version: %s", imageVersion.Repository)
		selectedImageVersion = map[string]string{
			"repository": imageVersion.Repository,
			"tag":        imageVersion.Tag,
		}
	}

	return selectedImageVersion
}

// ToDo: This need to be removed when we no longer support k8s versions under 1.20.
func newAMD64ArchNodeSelector(cluster cluster.CommonCluster) (map[string]string, error) {
	k8sVersion, err := getK8sVersion(cluster)
	if err != nil {
		return nil, errors.WrapIf(err, "unable to retrieve K8s version of cluster")
	}

	if k8sVersion.LessThan(comparedK8sSemver) {
		return map[string]string{"kubernetes.io/arch": "amd64"}, nil
	}

	return nil, nil
}
