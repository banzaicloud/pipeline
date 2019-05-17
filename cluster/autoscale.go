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
	"fmt"

	"github.com/banzaicloud/pipeline/internal/providers/azure/pke"
	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
	v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sHelm "k8s.io/helm/pkg/helm"

	"github.com/banzaicloud/pipeline/auth"
	"github.com/banzaicloud/pipeline/config"
	"github.com/banzaicloud/pipeline/helm"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	"github.com/banzaicloud/pipeline/pkg/k8sclient"
	pkgSecret "github.com/banzaicloud/pipeline/pkg/secret"
)

const cloudProviderAzure = "azure"
const cloudProviderAws = "aws"
const autoScalerChart = "banzaicloud-stable/cluster-autoscaler"
const expanderStrategy = "least-waste"
const logLevel = "5"
const AzureVirtualMachineScaleSet = "vmss"

const releaseName = "autoscaler"

type deploymentAction string

const install deploymentAction = "Install"
const upgrade deploymentAction = "Upgrade"

type nodeGroup struct {
	Name    string `json:"name"`
	MinSize int    `json:"minSize"`
	MaxSize int    `json:"maxSize"`
}

type rbac struct {
	Create bool `json:"create"`
}

type azureInfo struct {
	ClientID          string `json:"clientID"`
	ClientSecret      string `json:"clientSecret"`
	SubscriptionID    string `json:"subscriptionID"`
	TenantID          string `json:"tenantID"`
	ResourceGroup     string `json:"resourceGroup"`
	NodeResourceGroup string `json:"nodeResourceGroup"`
	ClusterName       string `json:"clusterName"`
	VMType            string `json:"vmType,omitempty"`
}

type autoDiscovery struct {
	ClusterName string `json:"clusterName"`
}

type autoscalingInfo struct {
	CloudProvider     string            `json:"cloudProvider"`
	AutoscalingGroups []nodeGroup       `json:"autoscalingGroups"`
	ExtraArgs         map[string]string `json:"extraArgs"`
	Rbac              rbac              `json:"rbac"`
	AwsRegion         string            `json:"awsRegion"`
	Azure             azureInfo         `json:"azure"`
	AutoDiscovery     autoDiscovery     `json:"autoDiscovery"`
	SslCertPath       *string           `json:"sslCertPath,omitempty"`
	SslCertHostPath   *string           `json:"sslCertHostPath,omitempty"`
	Affinity          v1.Affinity       `json:"affinity,omitempty"`
	Tolerations       []v1.Toleration   `json:"tolerations,omitempty"`
}

func getAmazonNodeGroups(cluster CommonCluster) ([]nodeGroup, error) {
	var nodeGroups []nodeGroup

	headNodePoolName := viper.GetString(config.PipelineHeadNodePoolName)
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
			if nodePool.Autoscaling && (nodePool.Name == headNodePoolName || !scaleEnabled) {
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
			if nodePool.Autoscaling && (nodePool.Name == headNodePoolName || !scaleEnabled) {
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
			GetPKEOnAzureCluster() pke.PKEOnAzureCluster
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
		},
		SslCertPath: &eksCertPath,
		Affinity:    GetHeadNodeAffinity(cluster),
		Tolerations: GetHeadNodeTolerations(),
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

	response, err := client.CoreV1().Nodes().List(meta_v1.ListOptions{})
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

	autoscalingInfo := &autoscalingInfo{
		CloudProvider:     cloudProviderAzure,
		AutoscalingGroups: groups,
		ExtraArgs: map[string]string{
			"v":        logLevel,
			"expander": expanderStrategy,
		},
		Rbac: rbac{Create: true},
		Azure: azureInfo{
			ClientID:       clusterSecret.Values[pkgSecret.AzureClientID],
			ClientSecret:   clusterSecret.Values[pkgSecret.AzureClientSecret],
			SubscriptionID: clusterSecret.Values[pkgSecret.AzureSubscriptionID],
			TenantID:       clusterSecret.Values[pkgSecret.AzureTenantID],
			ClusterName:    cluster.GetName(),
		},
		Affinity:    GetHeadNodeAffinity(cluster),
		Tolerations: GetHeadNodeTolerations(),
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

		autoscalingInfo.Azure.ResourceGroup = resourceGroup
		autoscalingInfo.Azure.NodeResourceGroup = *nodeResourceGroup

	case pkgCluster.PKE:
		i, ok := cluster.(interface {
			GetResourceGroupName() string
		})
		if !ok {
			return nil
		}
		autoscalingInfo.Azure.ResourceGroup = i.GetResourceGroupName()
		if len(vmType) > 0 {
			autoscalingInfo.Azure.VMType = vmType
		}
		sslCertHostPath := "/etc/kubernetes/pki/ca.crt"
		autoscalingInfo.SslCertHostPath = &sslCertHostPath
	}

	return autoscalingInfo
}

//DeployClusterAutoscaler post hook only for AWS & EKS & Azure for now
func DeployClusterAutoscaler(cluster CommonCluster) error {

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

	kubeConfig, err := cluster.GetK8sConfig()
	if err != nil {
		log.Errorf("Unable to fetch K8S config %s", err.Error())
		return err
	}

	if isAutoscalerDeployedAlready(releaseName, kubeConfig) {
		// no need to upgrade in case of EKS since we're using nodepool autodiscovery
		if _, isEks := cluster.(*EKSCluster); isEks {
			return nil
		}
		if len(nodeGroups) == 0 {
			// delete
			err := helm.DeleteDeployment(releaseName, kubeConfig)
			if err != nil {
				log.Errorf("DeleteDeployment '%s' failed due to: %s", autoScalerChart, err.Error())
				return err
			}
		} else {
			// upgrade
			return deployAutoscalerChart(cluster, nodeGroups, kubeConfig, upgrade)
		}
	} else {
		if len(nodeGroups) == 0 {
			// do nothing
			log.Info("No node groups configured for autoscaling")
			return nil
		}
		// install
		return deployAutoscalerChart(cluster, nodeGroups, kubeConfig, install)

	}

	return nil
}

func isAutoscalerDeployedAlready(releaseName string, kubeConfig []byte) bool {
	deployments, err := helm.ListDeployments(&releaseName, "", kubeConfig)
	if err != nil {
		log.Errorf("ListDeployments for '%s' failed due to: %s", autoScalerChart, err.Error())
		return false
	}
	for _, release := range deployments.GetReleases() {
		if release.Name == releaseName {
			return true
		}
	}
	return false
}

func deployAutoscalerChart(cluster CommonCluster, nodeGroups []nodeGroup, kubeConfig []byte, action deploymentAction) error {
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
	yamlValues, err := yaml.Marshal(*values)
	if err != nil {
		log.Errorf("Error during values marshal: %s", err.Error())
		return err
	}
	org, err := auth.GetOrganizationById(cluster.GetOrganizationId())
	if err != nil {
		log.Errorf("Error during getting organization: %s", err.Error())
		return err
	}

	switch action {
	case install:
		_, err = helm.CreateDeployment(autoScalerChart, "", nil, helm.SystemNamespace, releaseName, false, nil, kubeConfig, helm.GenerateHelmRepoEnv(org.Name), k8sHelm.ValueOverrides(yamlValues))
	case upgrade:
		_, err = helm.UpgradeDeployment(releaseName, autoScalerChart, "", nil, yamlValues, false, kubeConfig, helm.GenerateHelmRepoEnv(org.Name))
	default:
		return err
	}

	if err != nil {
		log.Errorf("%s of chart '%s' failed due to: %s", action, autoScalerChart, err.Error())
		return err
	}

	log.Infof("'%s' %sed", autoScalerChart, action)
	return nil
}
