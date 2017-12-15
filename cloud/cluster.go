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
	"github.com/sirupsen/logrus"
	"github.com/kris-nova/kubicorn/cutil/logger"
	"github.com/banzaicloud/pipeline/utils"
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
func CreateCluster(clusterType ClusterSimple, log *logrus.Logger) (*cluster.Cluster, error) {

	logger.Level = 4

	newCluster := clusterType.GetAWSCluster()

	//Inject configuration parameters
	ssh_key_path := viper.GetString("dev.keypath")
	if ssh_key_path != "" {
		newCluster.SSH.PublicKeyPath = ssh_key_path
		utils.LogDebug(log, utils.TagCreateCluster, "Overwriting default SSH key path to:", newCluster.SSH.PublicKeyPath)
	}

	// ---- [ Init cluster ] ---- //
	utils.LogInfo(log, utils.TagCreateCluster, "Init cluster")
	newCluster, err := initapi.InitCluster(newCluster)

	if err != nil {
		utils.LogInfo(log, utils.TagCreateCluster, "Error during init cluster:", err)
		return nil, err
	} else {
		utils.LogInfo(log, utils.TagCreateCluster, "Init cluster succeeded")
	}

	// ---- [ Get Reconciler ] ---- //
	utils.LogInfo(log, utils.TagCreateCluster, "Get Reconciler")
	reconciler, err := cutil.GetReconciler(newCluster, &runtimeParam)

	if err != nil {
		utils.LogInfo(log, utils.TagCreateCluster, "Error during getting reconciler:", err)
		return nil, err
	} else {
		utils.LogInfo(log, utils.TagCreateCluster, "Get Reconciler succeeded")
	}

	// ---- [ Get expected state ] ---- //
	expected, err := reconciler.Expected(newCluster)
	if err != nil {
		utils.LogInfo(log, utils.TagCreateCluster, "Error during getting expected state:", err)
		return nil, err
	} else {
		utils.LogInfo(log, utils.TagCreateCluster, "Get expected state succeeded")
	}

	// ---- [ Get actual state ] ---- //
	actual, err := reconciler.Actual(newCluster)
	if err != nil {
		utils.LogInfo(log, utils.TagCreateCluster, "Error during getting actual state:", err)
		return nil, err
	} else {
		utils.LogInfo(log, utils.TagCreateCluster, "Get actual state succeeded")
	}

	// ---- [ Reconcile ] ---- //
	created, err := reconciler.Reconcile(actual, expected)
	if err != nil {
		utils.LogInfo(log, utils.TagCreateCluster, "Error during reconcile:", err)
		return nil, err
	} else {
		utils.LogInfo(log, utils.TagCreateCluster, "Reconcile succeeded")
	}

	if created == nil {
		utils.LogInfo(log, utils.TagCreateCluster, "Error during reconcile, created cluster is nil")
		return nil, errors.New("Error during reconcile")
	}

	utils.LogDebug(log, utils.TagCreateCluster, "Created cluster:", created.Name)

	utils.LogInfo(log, utils.TagCreateCluster, "Get state store")
	stateStore := getStateStoreForCluster(clusterType)
	if stateStore.Exists() {
		return nil, fmt.Errorf("State store [%s] exists, will not overwrite", clusterType.Name)
	}
	stateStore.Commit(created)

	return created, nil
}

// DeleteClusterAzure deletes cluster from azure
func (cs *ClusterSimple) DeleteClusterAzure(c *gin.Context, name string, resourceGroup string) bool {
	res, err := azureClient.DeleteCluster(name, resourceGroup)
	if err != nil {
		SetResponseBodyJson(c, err.StatusCode, gin.H{"status": err.StatusCode, "message": err.Message})
		return false
	} else {
		SetResponseBodyJson(c, res.StatusCode, res)
		return true
	}
}

// DeleteCluster deletes a cluster from the cloud
func (cs *ClusterSimple) DeleteClusterAmazon(log *logrus.Logger) (*cluster.Cluster, error) {

	logger.Level = 4

	// --- [ Get state store ] --- //
	utils.LogInfo(log, utils.TagDeleteCluster, "Get State store")
	stateStore := getStateStoreForCluster(*cs)
	if !stateStore.Exists() {
		utils.LogWarn(log, utils.TagDeleteCluster, "State store not exists")
		return nil, nil
	} else {
		utils.LogInfo(log, utils.TagDeleteCluster, "Get State store exists")
	}

	// --- [ Get cluster ] --- //
	utils.LogInfo(log, utils.TagDeleteCluster, "Get cluster")
	deleteCluster, err := stateStore.GetCluster()
	if err != nil {
		utils.LogInfo(log, utils.TagDeleteCluster, "Failed to load cluster:"+cs.Name)
		return nil, err
	} else {
		utils.LogInfo(log, utils.TagDeleteCluster, "Get cluster succeeded")
	}

	// --- [ Get Reconciler ] --- //
	utils.LogInfo(log, utils.TagDeleteCluster, "Get cluster")
	reconciler, err := cutil.GetReconciler(deleteCluster, &runtimeParam)
	if err != nil {
		utils.LogInfo(log, utils.TagDeleteCluster, "Error during getting reconciler:", err)
		return nil, err
	} else {
		utils.LogInfo(log, utils.TagDeleteCluster, "Get Reconciler succeeded")
	}

	// --- [ Destroy cluster ] --- //
	utils.LogInfo(log, utils.TagDeleteCluster, "Destroy cluster")
	_, err = reconciler.Destroy()
	if err != nil {
		utils.LogInfo(log, utils.TagDeleteCluster, "Error during reconciler destroy:", err)
		return nil, err
	}
	utils.LogInfo(log, utils.TagDeleteCluster, "Deleted cluster [%s]", deleteCluster.Name)

	utils.LogInfo(log, utils.TagDeleteCluster, "Destroy state store")
	stateStore.Destroy()
	return nil, nil
}

// ReadCluster reads a persisted cluster from the statestore
func ReadCluster(cl ClusterSimple) (*cluster.Cluster, error) {

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
func UpdateClusterAws(ccs ClusterSimple, log *logrus.Logger) (*cluster.Cluster, error) {

	logger.Level = 4

	utils.LogInfo(log, utils.TagUpdateCluster, "Get state store for cluster")
	stateStore := getStateStoreForCluster(ccs)
	utils.LogDebug(log, utils.TagUpdateCluster, "State store for cluster:", stateStore)

	// --- [ Get cluster ] --- //
	updateCluster, err := stateStore.GetCluster()
	utils.LogDebug(log, utils.TagUpdateCluster, "Get cluster")
	if err != nil {
		utils.LogDebug(log, utils.TagUpdateCluster, "Failed to load cluster:"+ccs.Name)
		return nil, err
	} else {
		utils.LogDebug(log, utils.TagUpdateCluster, "Get cluster succeeded:", updateCluster)
	}

	utils.LogDebug(log, utils.TagUpdateCluster, "Resizing cluster:"+ccs.Name)
	utils.LogDebug(log, utils.TagUpdateCluster, "Worker pool min size:"+strconv.Itoa(updateCluster.ServerPools[1].MinCount)+"=>"+strconv.Itoa(ccs.Amazon.NodeMinCount))
	utils.LogDebug(log, utils.TagUpdateCluster, "Worker pool max size:"+strconv.Itoa(updateCluster.ServerPools[1].MaxCount)+"=>"+strconv.Itoa(ccs.Amazon.NodeMaxCount))
	updateCluster.ServerPools[0].MinCount = 1
	updateCluster.ServerPools[0].MaxCount = 1
	updateCluster.ServerPools[1].MinCount = ccs.Amazon.NodeMinCount
	updateCluster.ServerPools[1].MaxCount = ccs.Amazon.NodeMaxCount

	// --- [ Get Reconciler ] --- //
	utils.LogDebug(log, utils.TagUpdateCluster, "Get reconciler")
	reconciler, err := cutil.GetReconciler(updateCluster, &runtimeParam)
	if err != nil {
		utils.LogDebug(log, utils.TagUpdateCluster, "Error during getting reconciler:", err)
		return nil, err
	} else {
		utils.LogDebug(log, utils.TagUpdateCluster, "Getting Reconciler succeeded")
	}

	/*actual, err := reconciler.Actual(updateCluster)
	if err != nil {
		logger.Info(err.Error())
		logger.Info("Error during getting expected state:", err)
		return nil, err
	}*/

	// --- [ Get expected state ] --- //
	utils.LogDebug(log, utils.TagUpdateCluster, "Get expected state")
	expected, err := reconciler.Expected(updateCluster)
	if err != nil {
		utils.LogInfo(log, utils.TagUpdateCluster, "Error during getting expected state:", err)
		return nil, err
	} else {
		utils.LogDebug(log, utils.TagUpdateCluster, "Getting expected state succeeded")
	}

	// --- [ Reconcile ] --- //
	utils.LogDebug(log, utils.TagUpdateCluster, "Reconcile")
	updated, err := reconciler.Reconcile(updateCluster, expected)
	if err != nil {
		utils.LogDebug(log, utils.TagUpdateCluster, "Error during reconcile:", err)
		return nil, err
	} else {
		utils.LogDebug(log, utils.TagUpdateCluster, "Reconcile succeeded")
	}

	utils.LogInfo(log, utils.TagUpdateCluster, "Commit state store")
	stateStore.Commit(updateCluster)
	return updated, nil
}

// Wait for K8S
func awaitKubernetesCluster(existing ClusterSimple, log *logrus.Logger) (bool, error) {
	success := false
	existingCluster, _ := getStateStoreForCluster(existing).GetCluster()

	for i := 0; i < apiSocketAttempts; i++ {
		_, err := IsKubernetesClusterAvailable(existingCluster)
		if err != nil {
			log.Info("Attempting to open a socket to the Kubernetes API: %v...\n", err)
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
