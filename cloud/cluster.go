package cloud

import (
	"fmt"
	"strconv"
	"time"

	"github.com/jinzhu/gorm"
	"github.com/kris-nova/kubicorn/apis/cluster"
	"github.com/kris-nova/kubicorn/cutil"
	"github.com/kris-nova/kubicorn/cutil/initapi"
	"github.com/kris-nova/kubicorn/cutil/logger"
)

const (
	ApiSleepSeconds   = 5
	ApiSocketAttempts = 40
)

type ClusterType struct {
	gorm.Model
	Name                  string `json:"name" binding:"required" gorm:"unique"`
	Location              string `json:"location" binding:"required"`
	NodeInstanceType      string `json:"nodeInstanceType" binding:"required"`
	MasterInstanceType    string `json:"masterInstanceType" binding:"required"`
	NodeInstanceSpotPrice string `json:"nodeInstanceSpotPrice"`
	NodeMin               int    `json:"nodeMin" binding:"required"`
	NodeMax               int    `json:"nodeMax" binding:"required"`
	MasterImage           string `json:"masterImage" binding:"required"`
	NodeImage             string `json:"nodeImage" binding:"required"`
}

/**
func CloudInit(provider Provider, clusterType ClusterType) *cluster.Cluster {
	switch conf.Provider {
	case "aws":
		return getAWSCluster(clusterType)
	case "digitalocean":
		return getDOCluster(clusterType)
	default:
		return getAWSCluster(clusterType)
	}

}
**/

func CreateCluster(clusterType ClusterType) (*cluster.Cluster, error) {

	logger.Level = 4

	newCluster := getAWSCluster(clusterType)
	newCluster, err := initapi.InitCluster(newCluster)

	if err != nil {
		logger.Info(err.Error())
		logger.Info("Error during init cluster:", err)
		return nil, err
	}

	reconciler, err := cutil.GetReconciler(newCluster)

	if err != nil {
		logger.Info(err.Error())
		logger.Info("Error during getting reconciler:", err)
		return nil, err
	}

	expected, err := reconciler.Expected(newCluster)
	if err != nil {
		logger.Info(err.Error())
		logger.Info("Error during getting expected state:", err)
		return nil, err
	}
	actual, err := reconciler.Actual(newCluster)
	if err != nil {
		logger.Info(err.Error())
		logger.Info("Error during getting actual state:", err)
		return nil, err
	}
	created, err := reconciler.Reconcile(actual, expected)
	if err != nil {
		logger.Info(err.Error())
		logger.Info("Error during reconcile:", err)
		return nil, err
	}
	logger.Debug("Created cluster [%s]", created.Name)

	stateStore := getStateStoreForCluster(clusterType)
	if stateStore.Exists() {
		return nil, fmt.Errorf("State store [%s] exists, will not overwrite", clusterType.Name)
	}
	stateStore.Commit(created)

	return created, nil
}

func DeleteCluster(clusterType ClusterType) (*cluster.Cluster, error) {
	logger.Level = 4

	stateStore := getStateStoreForCluster(clusterType)
	if !stateStore.Exists() {
		return nil, nil
	}

	deleteCluster, err := stateStore.GetCluster()
	if err != nil {
		logger.Info(err.Error())
		logger.Info("Failed to load cluster:" + clusterType.Name)
		return nil, err
	}

	reconciler, err := cutil.GetReconciler(deleteCluster)
	if err != nil {
		logger.Info(err.Error())
		logger.Info("Error during getting reconciler:", err)
		return nil, err
	}

	_, err = reconciler.Destroy()
	if err != nil {
		logger.Info(err.Error())
		logger.Info("Error during reconciler destroy:", err)
		return nil, err
	}
	logger.Info("Deleted cluster [%s]", deleteCluster.Name)

	stateStore.Destroy()
	return nil, nil
}

func ReadCluster(clusterType ClusterType) (*cluster.Cluster, error) {

	stateStore := getStateStoreForCluster(clusterType)
	readCluster, err := stateStore.GetCluster()
	if err != nil {
		return nil, err
	}

	return readCluster, nil
}

func GetKubeConfig(existing *cluster.Cluster) error {

	_, err := RetryGetConfig(existing, "")
	return err
}

func UpdateCluster(clusterType ClusterType) (*cluster.Cluster, error) {

	logger.Level = 4

	stateStore := getStateStoreForCluster(clusterType)

	updateCluster, err := stateStore.GetCluster()
	if err != nil {
		logger.Info(err.Error())
		logger.Info("Failed to load cluster:" + clusterType.Name)
		return nil, err
	}

	logger.Info("Resizing cluster : " + clusterType.Name)
	logger.Info("Worker pool min size: " + strconv.Itoa(updateCluster.ServerPools[1].MinCount) + " => " + strconv.Itoa(clusterType.NodeMin))
	logger.Info("Worker pool max size : " + strconv.Itoa(updateCluster.ServerPools[1].MaxCount) + " => " + strconv.Itoa(clusterType.NodeMax))
	updateCluster.ServerPools[0].MinCount = 1
	updateCluster.ServerPools[0].MaxCount = 1
	updateCluster.ServerPools[1].MinCount = clusterType.NodeMin
	updateCluster.ServerPools[1].MaxCount = clusterType.NodeMax

	reconciler, err := cutil.GetReconciler(updateCluster)
	if err != nil {
		logger.Info(err.Error())
		logger.Info("Error during getting reconciler:", err)
		return nil, err
	}

	/*actual, err := reconciler.Actual(updateCluster)
	if err != nil {
		logger.Info(err.Error())
		logger.Info("Error during getting expected state:", err)
		return nil, err
	}*/
	expected, err := reconciler.Expected(updateCluster)
	if err != nil {
		logger.Info(err.Error())
		logger.Info("Error during getting expected state:", err)
		return nil, err
	}

	updated, err := reconciler.Reconcile(updateCluster, expected)
	if err != nil {
		logger.Info(err.Error())
		logger.Info("Error during reconcile:", err)
		return nil, err
	}

	stateStore.Commit(updateCluster)
	return updated, nil
}

func AwaitKubernetesCluster(existing ClusterType) (bool, error) {
	success := false
	existingCluster, _ := getStateStoreForCluster(existing).GetCluster()

	for i := 0; i < ApiSocketAttempts; i++ {
		_, err := IsKubernetesClusterAvailable(existingCluster)
		if err != nil {
			logger.Info("Attempting to open a socket to the Kubernetes API: %v...\n", err)
			time.Sleep(time.Duration(ApiSleepSeconds) * time.Second)
			continue
		}
		success = true
	}
	if !success {
		return false, fmt.Errorf("Unable to connect to Kubernetes API")
	}
	return true, nil
}

func IsKubernetesClusterAvailable(cluster *cluster.Cluster) (bool, error) {
	return assertTcpSocketAcceptsConnection(fmt.Sprintf("%s:%s", cluster.KubernetesAPI.Endpoint, cluster.KubernetesAPI.Port))
}
