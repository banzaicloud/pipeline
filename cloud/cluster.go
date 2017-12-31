package cloud

import (
	"fmt"
	"strconv"
	"time"

	"github.com/kris-nova/kubicorn/apis/cluster"
	"github.com/kris-nova/kubicorn/cutil"
	"github.com/kris-nova/kubicorn/cutil/initapi"
	"github.com/spf13/viper"
	azureClient "github.com/banzaicloud/azure-aks-client/client"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"github.com/kris-nova/kubicorn/cutil/logger"
	banzaiConstants "github.com/banzaicloud/banzai-types/constants"
	banzaiUtils "github.com/banzaicloud/banzai-types/utils"
	banzaiSimpleTypes "github.com/banzaicloud/banzai-types/components/database"
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
func CreateCluster(clusterType banzaiSimpleTypes.ClusterSimple) (*cluster.Cluster, error) {

	logger.Level = 4

	newCluster := GetAWSCluster(&clusterType)

	//Inject configuration parameters
	ssh_key_path := viper.GetString("dev.keypath")
	if ssh_key_path != "" {
		newCluster.SSH.PublicKeyPath = ssh_key_path
		banzaiUtils.LogDebug(banzaiConstants.TagCreateCluster, "Overwriting default SSH key path to:", newCluster.SSH.PublicKeyPath)
	}

	// ---- [ Init cluster ] ---- //
	banzaiUtils.LogInfo(banzaiConstants.TagCreateCluster, "Init cluster")
	newCluster, err := initapi.InitCluster(newCluster)

	if err != nil {
		banzaiUtils.LogInfo(banzaiConstants.TagCreateCluster, "Error during init cluster:", err)
		return nil, err
	} else {
		banzaiUtils.LogInfo(banzaiConstants.TagCreateCluster, "Init cluster succeeded")
	}

	// ---- [ Get Reconciler ] ---- //
	banzaiUtils.LogInfo(banzaiConstants.TagCreateCluster, "Get Reconciler")
	reconciler, err := cutil.GetReconciler(newCluster, &runtimeParam)

	if err != nil {
		banzaiUtils.LogInfo(banzaiConstants.TagCreateCluster, "Error during getting reconciler:", err)
		return nil, err
	} else {
		banzaiUtils.LogInfo(banzaiConstants.TagCreateCluster, "Get Reconciler succeeded")
	}

	// ---- [ Get expected state ] ---- //
	expected, err := reconciler.Expected(newCluster)
	if err != nil {
		banzaiUtils.LogInfo(banzaiConstants.TagCreateCluster, "Error during getting expected state:", err)
		return nil, err
	} else {
		banzaiUtils.LogInfo(banzaiConstants.TagCreateCluster, "Get expected state succeeded")
	}

	// ---- [ Get actual state ] ---- //
	actual, err := reconciler.Actual(newCluster)
	if err != nil {
		banzaiUtils.LogInfo(banzaiConstants.TagCreateCluster, "Error during getting actual state:", err)
		return nil, err
	} else {
		banzaiUtils.LogInfo(banzaiConstants.TagCreateCluster, "Get actual state succeeded")
	}

	// ---- [ Reconcile ] ---- //
	created, err := reconciler.Reconcile(actual, expected)
	if err != nil {
		banzaiUtils.LogInfo(banzaiConstants.TagCreateCluster, "Error during reconcile:", err)
		return nil, err
	} else {
		banzaiUtils.LogInfo(banzaiConstants.TagCreateCluster, "Reconcile succeeded")
	}

	if created == nil {
		banzaiUtils.LogInfo(banzaiConstants.TagCreateCluster, "Error during reconcile, created cluster is nil")
		return nil, errors.New("Error during reconcile")
	}

	banzaiUtils.LogDebug(banzaiConstants.TagCreateCluster, "Created cluster:", created.Name)

	banzaiUtils.LogInfo(banzaiConstants.TagCreateCluster, "Get state store")
	stateStore := getStateStoreForCluster(clusterType)
	if stateStore.Exists() {
		return nil, fmt.Errorf("State store [%s] exists, will not overwrite", clusterType.Name)
	}
	stateStore.Commit(created)

	return created, nil
}

// DeleteClusterAzure deletes cluster from azure
func DeleteClusterAzure(c *gin.Context, name string, resourceGroup string) bool {
	res, success := azureClient.DeleteCluster(name, resourceGroup)
	SetResponseBodyJson(c, res.StatusCode, res)
	return success
}

// DeleteCluster deletes a cluster from the cloud
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

// ReadCluster reads a persisted cluster from the statestore
func ReadCluster(cl banzaiSimpleTypes.ClusterSimple) (*cluster.Cluster, error) {

	stateStore := getStateStoreForCluster(cl)
	readCluster, err := stateStore.GetCluster()
	if err != nil {
		return nil, err
	}

	return readCluster, nil
}

// GetKubeConfig retrieves the K8S config
func GetKubeConfig(existing *cluster.Cluster) error {

	_, err := RetryGetConfig(existing, "")
	return err
}

// UpdateCluster updates a cluster in the cloud (e.g. autoscales)
func UpdateClusterAws(ccs banzaiSimpleTypes.ClusterSimple) (*cluster.Cluster, error) {

	logger.Level = 4

	banzaiUtils.LogInfo(banzaiConstants.TagUpdateCluster, "Get state store for cluster")
	stateStore := getStateStoreForCluster(ccs)
	banzaiUtils.LogDebug(banzaiConstants.TagUpdateCluster, "State store for cluster:", stateStore)

	// --- [ Get cluster ] --- //
	updateCluster, err := stateStore.GetCluster()
	banzaiUtils.LogDebug(banzaiConstants.TagUpdateCluster, "Get cluster")
	if err != nil {
		banzaiUtils.LogDebug(banzaiConstants.TagUpdateCluster, "Failed to load cluster:"+ccs.Name)
		return nil, err
	} else {
		banzaiUtils.LogDebug(banzaiConstants.TagUpdateCluster, "Get cluster succeeded:", updateCluster)
	}

	banzaiUtils.LogDebug(banzaiConstants.TagUpdateCluster, "Resizing cluster:"+ccs.Name)
	banzaiUtils.LogDebug(banzaiConstants.TagUpdateCluster, "Worker pool min size:"+strconv.Itoa(updateCluster.ServerPools[1].MinCount)+"=>"+strconv.Itoa(ccs.Amazon.NodeMinCount))
	banzaiUtils.LogDebug(banzaiConstants.TagUpdateCluster, "Worker pool max size:"+strconv.Itoa(updateCluster.ServerPools[1].MaxCount)+"=>"+strconv.Itoa(ccs.Amazon.NodeMaxCount))
	updateCluster.ServerPools[0].MinCount = 1
	updateCluster.ServerPools[0].MaxCount = 1
	updateCluster.ServerPools[1].MinCount = ccs.Amazon.NodeMinCount
	updateCluster.ServerPools[1].MaxCount = ccs.Amazon.NodeMaxCount

	// --- [ Get Reconciler ] --- //
	banzaiUtils.LogDebug(banzaiConstants.TagUpdateCluster, "Get reconciler")
	reconciler, err := cutil.GetReconciler(updateCluster, &runtimeParam)
	if err != nil {
		banzaiUtils.LogDebug(banzaiConstants.TagUpdateCluster, "Error during getting reconciler:", err)
		return nil, err
	} else {
		banzaiUtils.LogDebug(banzaiConstants.TagUpdateCluster, "Getting Reconciler succeeded")
	}

	/*actual, err := reconciler.Actual(updateCluster)
	if err != nil {
		logger.Info(err.Error())
		logger.Info("Error during getting expected state:", err)
		return nil, err
	}*/

	// --- [ Get expected state ] --- //
	banzaiUtils.LogDebug(banzaiConstants.TagUpdateCluster, "Get expected state")
	expected, err := reconciler.Expected(updateCluster)
	if err != nil {
		banzaiUtils.LogInfo(banzaiConstants.TagUpdateCluster, "Error during getting expected state:", err)
		return nil, err
	} else {
		banzaiUtils.LogDebug(banzaiConstants.TagUpdateCluster, "Getting expected state succeeded")
	}

	// --- [ Reconcile ] --- //
	banzaiUtils.LogDebug(banzaiConstants.TagUpdateCluster, "Reconcile")
	updated, err := reconciler.Reconcile(updateCluster, expected)
	if err != nil {
		banzaiUtils.LogDebug(banzaiConstants.TagUpdateCluster, "Error during reconcile:", err)
		return nil, err
	} else {
		banzaiUtils.LogDebug(banzaiConstants.TagUpdateCluster, "Reconcile succeeded")
	}

	banzaiUtils.LogInfo(banzaiConstants.TagUpdateCluster, "Commit state store")
	stateStore.Commit(updateCluster)
	return updated, nil
}

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
