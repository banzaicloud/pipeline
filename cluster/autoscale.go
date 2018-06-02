package cluster

import (
	"github.com/banzaicloud/pipeline/helm"
	"github.com/banzaicloud/pipeline/secret"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const cloudProviderAzure = "azure"
const cloudProviderAws = "aws"
const autoSalerChart = "banzaicloud-stable/cluster-autoscaler"
const expanderStrategy = "least-waste"
const logLevel = "5"

type nodeGroup struct {
	Name    string `json:"name"`
	MinSize int    `json:"minSize"`
	MaxSize int    `json:"maxSize"`
}

type rbac struct {
	Create bool `json:"create"`
}

type awsInfo struct {
	AwsRegion string `json:"awsRegion"`
}

type azureInfo struct {
	ClientID          string `json:"clientID"`
	ClientSecret      string `json:"clientSecret"`
	SubscriptionID    string `json:"subscriptionID"`
	TenantID          string `json:"tenantID"`
	ResourceGroup     string `json:"resourceGroup"`
	NodeResourceGroup string `json:"nodeResourceGroup"`
	ClusterName       string `json:"clusterName"`
}

type autoscalingInfo struct {
	CloudProvider     string            `json:"cloudProvider"`
	AutoscalingGroups []nodeGroup       `json:"autoscalingGroups"`
	ExtraArgs         map[string]string `json:"extraArgs"`
	Rbac              rbac              `json:"rbac"`
	AwsRegion         string            `json:"awsRegion"`
	Azure             azureInfo         `json:"azure"`
}

func getAmazonNodeGroups(cluster CommonCluster) []nodeGroup {
	var nodeGroups []nodeGroup
	for _, nodePool := range cluster.GetModel().Amazon.NodePools {
		if nodePool.Autoscaling {
			nodeGroups = append(nodeGroups, nodeGroup{
				Name:    cluster.GetName() + ".node." + nodePool.Name,
				MinSize: nodePool.NodeMinCount,
				MaxSize: nodePool.NodeMaxCount,
			})
		}
	}
	return nodeGroups
}

func getAzureNodeGroups(cluster CommonCluster) []nodeGroup {
	var nodeGroups []nodeGroup
	for _, nodePool := range cluster.GetModel().Azure.NodePools {
		if nodePool.Autoscaling {
			nodeGroups = append(nodeGroups, nodeGroup{
				Name:    nodePool.Name,
				MinSize: nodePool.NodeMinCount,
				MaxSize: nodePool.NodeMaxCount,
			})
		}
	}
	return nodeGroups
}

func createAutoscalingForAmazon(cluster CommonCluster, groups []nodeGroup) *autoscalingInfo {
	return &autoscalingInfo{
		CloudProvider:     cloudProviderAws,
		AutoscalingGroups: groups,
		ExtraArgs: map[string]string{
			"v":        logLevel,
			"expander": expanderStrategy,
		},
		Rbac:      rbac{Create: true},
		AwsRegion: cluster.GetModel().Location,
	}
}

func getNodeResourceGroup(cluster CommonCluster) *string {
	kubeConfig, err := cluster.GetK8sConfig()
	if err != nil {
		log.Errorf("Error getting config: %s", err.Error())
		return nil
	}
	client, err := helm.GetK8sConnection(kubeConfig)
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

func createAutoscalingForAzure(cluster CommonCluster, groups []nodeGroup) *autoscalingInfo {
	clusterSecret, err := cluster.GetSecretWithValidation()
	if err != nil {
		return nil
	}

	nodeResourceGroup := getNodeResourceGroup(cluster)
	if nodeResourceGroup == nil {
		log.Errorf("Error nodeResourceGroup not found")
		return nil
	}

	return &autoscalingInfo{
		CloudProvider:     cloudProviderAzure,
		AutoscalingGroups: groups,
		ExtraArgs: map[string]string{
			"v":        logLevel,
			"expander": expanderStrategy,
		},
		Rbac: rbac{Create: true},
		Azure: azureInfo{
			ClientID:          clusterSecret.Values[secret.AzureClientId],
			ClientSecret:      clusterSecret.Values[secret.AzureClientSecret],
			SubscriptionID:    clusterSecret.Values[secret.AzureSubscriptionId],
			TenantID:          clusterSecret.Values[secret.AzureTenantId],
			ResourceGroup:     cluster.GetModel().Azure.ResourceGroup,
			NodeResourceGroup: *nodeResourceGroup,
			ClusterName:       cluster.GetName(),
		},
	}
}
