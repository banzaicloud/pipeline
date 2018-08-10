package cluster

import (
	"github.com/banzaicloud/pipeline/auth"
	"github.com/banzaicloud/pipeline/helm"
	"github.com/banzaicloud/pipeline/model"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	pkgSecret "github.com/banzaicloud/pipeline/pkg/secret"
	"github.com/ghodss/yaml"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const cloudProviderAzure = "azure"
const cloudProviderAws = "aws"
const autoScalerChart = "banzaicloud-stable/cluster-autoscaler"
const expanderStrategy = "least-waste"
const logLevel = "5"

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
}

type autoscalingInfo struct {
	CloudProvider     string            `json:"cloudProvider"`
	AutoscalingGroups []nodeGroup       `json:"autoscalingGroups"`
	ExtraArgs         map[string]string `json:"extraArgs"`
	Rbac              rbac              `json:"rbac"`
	AwsRegion         string            `json:"awsRegion"`
	Azure             azureInfo         `json:"azure"`
	AutoDiscovery     map[string]string `json:"autoDiscovery"`
	SslCertPath       string            `json:"sslCertPath"`
}

func getAmazonNodeGroups(cluster CommonCluster) []nodeGroup {
	var nodeGroups []nodeGroup

	var nodePools []*model.AmazonNodePoolsModel
	switch cluster.GetDistribution() {
	case pkgCluster.EC2:
		nodePools = cluster.GetModel().EC2.NodePools
	case pkgCluster.EKS:
		nodePools = cluster.GetModel().EKS.NodePools
	}

	for _, nodePool := range nodePools {
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
	for _, nodePool := range cluster.GetModel().AKS.NodePools {
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

func createAutoscalingForEc2(cluster CommonCluster, groups []nodeGroup) *autoscalingInfo {
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

func createAutoscalingForEks(cluster CommonCluster, groups []nodeGroup) *autoscalingInfo {
	return &autoscalingInfo{
		CloudProvider: cloudProviderAws,
		ExtraArgs: map[string]string{
			"v":        logLevel,
			"expander": expanderStrategy,
		},
		Rbac:      rbac{Create: true},
		AwsRegion: cluster.GetModel().Location,
		AutoDiscovery: map[string]string{
			"clusterName": cluster.GetName(),
		},
		SslCertPath: "/etc/ssl/certs/ca-bundle.crt",
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
			ClientID:          clusterSecret.Values[pkgSecret.AzureClientId],
			ClientSecret:      clusterSecret.Values[pkgSecret.AzureClientSecret],
			SubscriptionID:    clusterSecret.Values[pkgSecret.AzureSubscriptionId],
			TenantID:          clusterSecret.Values[pkgSecret.AzureTenantId],
			ResourceGroup:     cluster.GetModel().AKS.ResourceGroup,
			NodeResourceGroup: *nodeResourceGroup,
			ClusterName:       cluster.GetName(),
		},
	}
}

//DeployClusterAutoscaler post hook only for AWS & EKS & Azure for now
func DeployClusterAutoscaler(cluster CommonCluster) error {

	var nodeGroups []nodeGroup

	switch cluster.GetCloud() {
	case pkgCluster.Amazon:
		// nodeGroups are the same for EKS & EC2
		nodeGroups = getAmazonNodeGroups(cluster)
	case pkgCluster.Azure:
		nodeGroups = getAzureNodeGroups(cluster)
	default:
		return nil
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
	deployments, err := helm.ListDeployments(&releaseName, kubeConfig)
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
	case pkgCluster.EC2:
		values = createAutoscalingForEc2(cluster, nodeGroups)
	case pkgCluster.AKS:
		values = createAutoscalingForAzure(cluster, nodeGroups)
	default:
		return nil
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
		_, err = helm.CreateDeployment(autoScalerChart, "", helm.SystemNamespace, releaseName, yamlValues, kubeConfig, helm.GenerateHelmRepoEnv(org.Name))
	case upgrade:
		_, err = helm.UpgradeDeployment(releaseName, autoScalerChart, "", yamlValues, false, kubeConfig, helm.GenerateHelmRepoEnv(org.Name))
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
