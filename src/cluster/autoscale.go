// Copyright © 2018 Banzai Cloud
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

package cluster

import (
	"context"
	"fmt"

	"emperror.dev/errors"
	"github.com/Masterminds/semver/v3"
	"github.com/ghodss/yaml"
	"github.com/sirupsen/logrus"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/banzaicloud/pipeline/internal/global"
	"github.com/banzaicloud/pipeline/internal/providers/azure/pke"
	"github.com/banzaicloud/pipeline/internal/secret/secrettype"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	"github.com/banzaicloud/pipeline/pkg/k8sclient"
	"github.com/banzaicloud/pipeline/src/helm"
)

const (
	cloudProviderAzure          = "azure"
	cloudProviderAws            = "aws"
	expanderStrategy            = "least-waste"
	logLevel                    = "5"
	AzureVirtualMachineScaleSet = "vmss"
)

// Required for the selectArchNodeSelector func to compare k8s versions
var comparedK8sSemver *semver.Version

const releaseName = "autoscaler"

type deploymentAction string

const (
	install deploymentAction = "Install"
	upgrade deploymentAction = "Upgrade"
)

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

func init() {
	comparedK8sSemver, _ = semver.NewVersion("1.20.0")
}

func getAmazonNodeGroups(cluster CommonCluster) ([]nodeGroup, error) {
	var nodeGroups []nodeGroup

	scaleOptions := cluster.GetScaleOptions()
	scaleEnabled := scaleOptions != nil && scaleOptions.Enabled

	switch cluster.GetDistribution() {
	case pkgCluster.EKS:
		nodePools, err := GetEKSNodePools(cluster)
		if err != nil {
			return nil, err
		}

		for _, nodePool := range nodePools {
			// if ScaleOptions is enabled on cluster, ClusterAutoscaler is disabled on all node pools (except head) on Amazon
			if nodePool.Autoscaling && !scaleEnabled {
				nodeGroups = append(nodeGroups, nodeGroup{
					Name:    cluster.GetName() + ".node." + nodePool.Name,
					MinSize: nodePool.NodeMinCount,
					MaxSize: nodePool.NodeMaxCount,
				})
			}
		}
	case pkgCluster.PKE:
		pke, ok := cluster.(*EC2ClusterPKE)
		if !ok {
			return nil, errors.New("could not cast Amazon/PKE cluster to EC2ClusterPKE")
		}
		nodePools := pke.GetNodePools()
		for _, nodePool := range nodePools {
			if nodePool.Autoscaling && !scaleEnabled {
				nodeGroups = append(nodeGroups, nodeGroup{
					Name:    cluster.GetName() + ".node." + nodePool.Name,
					MinSize: nodePool.MinCount,
					MaxSize: nodePool.MaxCount,
				})
			}
		}
	}

	return nodeGroups, nil
}

func getAzureNodeGroups(cluster CommonCluster) ([]nodeGroup, error) {
	var nodeGroups []nodeGroup

	switch cluster.GetDistribution() {
	case pkgCluster.AKS:
		nodePools, err := GetAKSNodePools(cluster)
		if err != nil {
			return nil, err
		}

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

func createAutoscalingForEks(cluster CommonCluster, groups []nodeGroup) *autoscalingInfo {
	eksCertPath := "/etc/ssl/certs/ca-bundle.crt"
	nodeSelector, err := selectArchNodeSelector(cluster)
	if err != nil {
		log.Error(errors.WrapIfWithDetails(err, "unable to retrieve K8s version of cluster", "clusterID", cluster.GetID()))
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
	}
}

func getNodeResourceGroup(cluster CommonCluster) *string {
	kubeConfig, err := cluster.GetK8sConfig()
	if err != nil {
		log.Errorf("Error getting config: %s", err.Error())
		return nil
	}
	client, err := k8sclient.NewClientFromKubeConfig(kubeConfig)
	if err != nil {
		log.Errorf("Error getting k8s connection: %s", err.Error())
		return nil
	}

	response, err := client.CoreV1().Nodes().List(context.Background(), meta_v1.ListOptions{})
	log.Debugf("%s", response.String())
	if err != nil {
		log.Errorf("Error listing nodes: %s", err.Error())
		return nil
	}

	for _, node := range response.Items {
		for labelKey, labelValue := range node.Labels {
			if labelKey == "kubernetes.azure.com/cluster" {
				return &labelValue
			}
		}
	}
	return nil
}

func createAutoscalingForAzure(cluster CommonCluster, groups []nodeGroup, vmType string) *autoscalingInfo {
	clusterSecret, err := cluster.GetSecretWithValidation()
	if err != nil {
		return nil
	}

	nodeSelector, err := selectArchNodeSelector(cluster)
	if err != nil {
		log.Error(errors.WrapIfWithDetails(err, "unable to retrieve K8s version of cluster", "clusterID", cluster.GetID()))
	}

	autoscalingInfo := &autoscalingInfo{
		CloudProvider:     cloudProviderAzure,
		AutoscalingGroups: groups,
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
		nodeResourceGroup := getNodeResourceGroup(cluster)
		if nodeResourceGroup == nil {
			log.Errorf("Error nodeResourceGroup not found")
			return nil
		}

		resourceGroup, err := GetAKSResourceGroup(cluster)
		if err != nil {
			log.Errorf("could not get resource group: %s", err.Error())
		}

		autoscalingInfo.ResourceGroup = resourceGroup
		autoscalingInfo.NodeResourceGroup = *nodeResourceGroup

	case pkgCluster.PKE:
		i, ok := cluster.(interface {
			GetResourceGroupName() string
		})
		if !ok {
			return nil
		}
		autoscalingInfo.ResourceGroup = i.GetResourceGroupName()
		if len(vmType) > 0 {
			autoscalingInfo.VMType = vmType
		}
		sslCertHostPath := "/etc/kubernetes/pki/ca.crt"
		autoscalingInfo.SslCertHostPath = &sslCertHostPath
	}

	return autoscalingInfo
}

// DeployClusterAutoscaler post hook only for AWS & EKS & Azure for now
func DeployClusterAutoscaler(cluster CommonCluster, helmService HelmService) error {
	config := global.Config.Cluster.PostHook.Autoscaler
	if !config.Enabled {
		return nil
	}

	var nodeGroups []nodeGroup
	var err error

	switch cluster.GetCloud() {
	case pkgCluster.Amazon:
		// nodeGroups are the same for EKS & EC2
		nodeGroups, err = getAmazonNodeGroups(cluster)
	case pkgCluster.Azure:
		nodeGroups, err = getAzureNodeGroups(cluster)
	default:
		return nil
	}

	if err != nil {
		return errors.Wrap(err, "unable to fetch node pools")
	}

	if isAutoscalerDeployedAlready(releaseName, cluster.GetID(), helmService) {
		if len(nodeGroups) == 0 {
			// delete
			err := helmService.DeleteDeployment(context.TODO(), cluster.GetID(), releaseName, global.Config.Cluster.Namespace)
			if err != nil {
				log.Errorf("DeleteDeployment '%s' failed due to: %s", global.Config.Cluster.Autoscale.Charts.ClusterAutoscaler.Chart, err.Error())
				return err
			}
		} else {
			// upgrade
			return deployAutoscalerChart(cluster, nodeGroups, helmService, upgrade)
		}
	} else {
		if len(nodeGroups) == 0 {
			// do nothing
			log.Info("No node groups configured for autoscaling")
			return nil
		}
		// install
		return deployAutoscalerChart(cluster, nodeGroups, helmService, install)
	}

	return nil
}

func isAutoscalerDeployedAlready(releaseName string, clusterId uint, helmDeployer HelmService) bool {
	_, err := helmDeployer.GetDeployment(context.TODO(), clusterId, releaseName, global.Config.Cluster.Namespace)
	if err != nil {
		var notFoundErr *helm.DeploymentNotFoundError
		if errors.As(err, &notFoundErr) {
			return false
		}
		log.Errorf("ListDeployments for '%s' failed due to: %s", global.Config.Cluster.Autoscale.Charts.ClusterAutoscaler.Chart, err.Error())
		return false
	}
	return true
}

func deployAutoscalerChart(cluster CommonCluster, nodeGroups []nodeGroup, helmService HelmService, action deploymentAction) error {
	var values *autoscalingInfo
	switch cluster.GetDistribution() {
	case pkgCluster.EKS:
		values = createAutoscalingForEks(cluster, nodeGroups)
	case pkgCluster.AKS:
		values = createAutoscalingForAzure(cluster, nodeGroups, "")
	case pkgCluster.PKE:
		switch cluster.GetCloud() {
		case pkgCluster.Amazon:
			values = createAutoscalingForEks(cluster, nodeGroups)
		case pkgCluster.Azure:
			values = createAutoscalingForAzure(cluster, nodeGroups, AzureVirtualMachineScaleSet)
		}
	default:
		return nil
	}
	if values == nil {
		err := errors.New(fmt.Sprintf("Cluster autoscaler configuration error on %s %s", cluster.GetCloud(), cluster.GetDistribution()))
		log.Errorf(err.Error())
		return err
	}

	// set image tag & repo depending on K8s version
	values.Image = getImageVersion(cluster.GetID(), cluster)
	log.WithFields(logrus.Fields{"clusterID": cluster.GetID(), "imageTag": values.Image["tag"]}).Infof("deploy cluster autoscaler with image tag")

	yamlValues, err := yaml.Marshal(*values)
	if err != nil {
		log.Errorf("Error during values marshal: %s", err.Error())
		return err
	}

	chartName := global.Config.Cluster.Autoscale.Charts.ClusterAutoscaler.Chart
	chartVersion := global.Config.Cluster.Autoscale.Charts.ClusterAutoscaler.Version

	switch action {
	case install:
		err = helmService.ApplyDeploymentReuseValues(context.TODO(), cluster.GetID(), global.Config.Cluster.Namespace, chartName, releaseName, yamlValues, chartVersion, true)
	case upgrade:
		err = helmService.ApplyDeploymentReuseValues(context.TODO(), cluster.GetID(), global.Config.Cluster.Namespace, chartName, releaseName, yamlValues, chartVersion, true)
	default:
		return err
	}

	if err != nil {
		log.Errorf("%s of chart '%s' failed due to: %s", action, chartName, err.Error())
		return err
	}

	log.Infof("'%s' %sed", chartName, action)
	return nil
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

func getImageVersion(clusterID uint, cluster interface{}) map[string]string {
	var selectedImageVersion map[string]string

	k8sVersion, err := getK8sVersion(cluster)
	if err != nil {
		log.Error(errors.WrapIfWithDetails(err, "unable to retrieve K8s version of cluster", "clusterID", clusterID))
	} else {
		for _, imageVersion := range global.Config.Cluster.Autoscale.Charts.ClusterAutoscaler.ImageVersionConstraints {
			constraint, err := semver.NewConstraint(imageVersion.K8sVersion)
			if err != nil {
				log.Error(errors.WrapIf(err, fmt.Sprintf("invalid version constraint specified in config: %s", imageVersion.K8sVersion)))
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
		log.Debugf("no matching image found for clusterID: %v, using the latest version: %s", clusterID, imageVersion.Repository)
		selectedImageVersion = map[string]string{
			"repository": imageVersion.Repository,
			"tag":        imageVersion.Tag,
		}
	}

	return selectedImageVersion
}

// ToDo: This need to be removed when we no longer support k8s versions under 1.20.
func selectArchNodeSelector(cluster CommonCluster) (map[string]string, error) {
	k8sVersion, err := getK8sVersion(cluster)
	if err != nil {
		return nil, err
	}

	if k8sVersion.LessThan(comparedK8sSemver) {
		return map[string]string{"kubernetes.io/arch": "amd64"}, nil
	}

	return nil, nil
}
