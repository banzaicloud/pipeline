package cloud

import (
	"fmt"
	"strconv"
	"time"

	azureClient "github.com/banzaicloud/azure-aks-client/client"
	banzaiSimpleTypes "github.com/banzaicloud/banzai-types/components/database"
	banzaiConstants "github.com/banzaicloud/banzai-types/constants"
	banzaiUtils "github.com/banzaicloud/banzai-types/utils"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/gin-gonic/gin"
	"github.com/kris-nova/kubicorn/apis/cluster"
	"github.com/kris-nova/kubicorn/cloud/amazon/awsSdkGo"
	"github.com/kris-nova/kubicorn/cutil"
	"github.com/kris-nova/kubicorn/cutil/logger"
)

const (
	apiSleepSeconds   = 5
	apiSocketAttempts = 40
)

var runtimeParam = cutil.RuntimeParameters{
	AwsProfile: "",
}

/**
func CloudInit(provider Provider, clusterType ClusterType) *cluster.Cluster {
	switch conf.Provider {
	case "aws":
		return GetAWSCluster(clusterType)
	case "digitalocean":
		return getDOCluster(clusterType)
	default:
		return GetAWSCluster(clusterType)
	}

}
**/

// CreateCluster creates a cluster in the cloud

// DeleteClusterAzure deletes cluster from azure
func DeleteClusterAzure(c *gin.Context, name string, resourceGroup string) bool {
	res, success := azureClient.DeleteCluster(name, resourceGroup)
	SetResponseBodyJson(c, res.StatusCode, res)
	return success
}

// DeleteClusterAmazon deletes a cluster from the cloud
func DeleteClusterAmazon(cs *banzaiSimpleTypes.ClusterSimple) (*cluster.Cluster, error) {

	logger.Level = 4

	// --- [ Get state store ] --- //
	banzaiUtils.LogInfo(banzaiConstants.TagDeleteCluster, "Get State store")
	stateStore := getStateStoreForCluster(*cs)
	if !stateStore.Exists() {
		banzaiUtils.LogWarn(banzaiConstants.TagDeleteCluster, "State store not exists")
		return nil, nil
	} else {
		banzaiUtils.LogInfo(banzaiConstants.TagDeleteCluster, "Get State store exists")
	}

	// --- [ Get cluster ] --- //
	banzaiUtils.LogInfo(banzaiConstants.TagDeleteCluster, "Get cluster")
	deleteCluster, err := stateStore.GetCluster()
	if err != nil {
		banzaiUtils.LogInfo(banzaiConstants.TagDeleteCluster, "Failed to load cluster:"+cs.Name)
		return nil, err
	} else {
		banzaiUtils.LogInfo(banzaiConstants.TagDeleteCluster, "Get cluster succeeded")
	}
	// --- [ Doing security group cleanup] --- //
	sdk, err := awsSdkGo.NewSdk(deleteCluster.Location, "")
	secGroupInput := &ec2.DescribeSecurityGroupsInput{
		Filters: []*ec2.Filter{
			{
				Name: aws.String("tag:KubernetesCluster"),
				Values: []*string{
					&deleteCluster.Name,
				},
			},
		},
	}
	securityGroup, err := sdk.Ec2.DescribeSecurityGroups(secGroupInput)
	if err != nil {
		banzaiUtils.LogInfo(banzaiConstants.TagDeleteCluster, "Error getting security groups: ", err)
	}
	for _, sg := range securityGroup.SecurityGroups {
		banzaiUtils.LogInfo(banzaiConstants.TagDeleteCluster, "Delete security group: ", *sg.GroupId)
		refSecurityGroupInput := &ec2.DescribeSecurityGroupsInput{
			Filters: []*ec2.Filter{
				{
					Name: aws.String("ip-permission.group-id"),
					Values: []*string{
						sg.GroupId,
					},
				},
			},
		}
		referencedSecurityGroup, err := sdk.Ec2.DescribeSecurityGroups(refSecurityGroupInput)
		if err != nil {
			banzaiUtils.LogInfo(banzaiConstants.TagDeleteCluster, "Error getting security groups: ", err.Error())
		}
		revokeIngress := &ec2.RevokeSecurityGroupIngressInput{
			GroupId: referencedSecurityGroup.SecurityGroups[0].GroupId,
			IpPermissions: []*ec2.IpPermission{
				{
					IpProtocol: aws.String("-1"),
					UserIdGroupPairs: []*ec2.UserIdGroupPair{
						{
							GroupId: sg.GroupId,
							VpcId:   sg.VpcId,
						},
					},
				},
			},
		}
		_, err = sdk.Ec2.RevokeSecurityGroupIngress(revokeIngress)
		if err != nil {
			banzaiUtils.LogInfo(banzaiConstants.TagDeleteCluster, "Delete security rule failed: ", err.Error())
		}
		_, err = sdk.Ec2.DeleteSecurityGroup(&ec2.DeleteSecurityGroupInput{GroupId: sg.GroupId})
		if err != nil {
			banzaiUtils.LogInfo(banzaiConstants.TagDeleteCluster, "Delete security group failed: ", err.Error())
		}
	}
	// --- [ Get Reconciler ] --- //
	banzaiUtils.LogInfo(banzaiConstants.TagDeleteCluster, "Get cluster")
	reconciler, err := cutil.GetReconciler(deleteCluster, &runtimeParam)
	if err != nil {
		banzaiUtils.LogInfo(banzaiConstants.TagDeleteCluster, "Error during getting reconciler:", err)
		return nil, err
	} else {
		banzaiUtils.LogInfo(banzaiConstants.TagDeleteCluster, "Get Reconciler succeeded")
	}

	// --- [ Destroy cluster ] --- //
	banzaiUtils.LogInfo(banzaiConstants.TagDeleteCluster, "Destroy cluster")
	_, err = reconciler.Destroy()
	if err != nil {
		banzaiUtils.LogInfo(banzaiConstants.TagDeleteCluster, "Error during reconciler destroy:", err)
		return nil, err
	}
	banzaiUtils.LogInfo(banzaiConstants.TagDeleteCluster, "Deleted cluster: ", deleteCluster.Name)

	banzaiUtils.LogInfo(banzaiConstants.TagDeleteCluster, "Destroy state store")
	stateStore.Destroy()
	return nil, nil
}

// GetKubeConfig retrieves the K8S config
func GetKubeConfig(existing *cluster.Cluster) error {

	_, err := RetryGetConfig(existing, "")
	return err
}

// UpdateClusterAws updates a cluster in the cloud (e.g. autoscales)

// Wait for K8S
func awaitKubernetesCluster(existing banzaiSimpleTypes.ClusterSimple) (bool, error) {
	success := false
	existingCluster, _ := getStateStoreForCluster(existing).GetCluster()

	for i := 0; i < apiSocketAttempts; i++ {
		_, err := IsKubernetesClusterAvailable(existingCluster)
		if err != nil {
			banzaiUtils.LogInfo(banzaiConstants.TagGetClusterInfo, "Attempting to open a socket to the Kubernetes API: %v...\n", err)
			time.Sleep(time.Duration(apiSleepSeconds) * time.Second)
			continue
		}
		success = true
	}
	if !success {
		return false, fmt.Errorf("Unable to connect to Kubernetes API")
	}
	return true, nil
}

// IsKubernetesClusterAvailable awaits for K8S cluster to be available
func IsKubernetesClusterAvailable(cluster *cluster.Cluster) (bool, error) {
	return assertTcpSocketAcceptsConnection(fmt.Sprintf("%s:%s", cluster.KubernetesAPI.Endpoint, cluster.KubernetesAPI.Port))
}
