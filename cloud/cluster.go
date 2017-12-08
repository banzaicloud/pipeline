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
	"github.com/spf13/viper"
	azureClient "github.com/banzaicloud/azure-aks-client/client"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
)

const (
	apiSleepSeconds   = 5
	apiSocketAttempts = 40
)

var runtimeParam = cutil.RuntimeParameters{
	AwsProfile: "",
}

//ClusterType cluster definition for the API
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
	Tag                   string `json:"tag" binding:"required"`
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

//CreateCluster creates a cluster in the cloud
//func CreateCluster(clusterType ClusterType) (*cluster.Cluster, error) {
func CreateCluster(clusterType CreateClusterSimple) (*cluster.Cluster, error) {

	logger.Level = 4

	newCluster := getAWSCluster(clusterType)

	//Inject configuration parameters
	ssh_key_path := viper.GetString("dev.keypath")
	if ssh_key_path != "" {
		newCluster.SSH.PublicKeyPath = ssh_key_path
		logger.Debug("Overwriting default SSH key path to: %s", newCluster.SSH.PublicKeyPath)
	}

	newCluster, err := initapi.InitCluster(newCluster)

	if err != nil {
		logger.Info(err.Error())
		logger.Info("Error during init cluster:", err)
		return nil, err
	}

	reconciler, err := cutil.GetReconciler(newCluster, &runtimeParam)

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

	if created == nil {
		return nil, errors.New("Error during reconcile")
	}

	logger.Debug("Created cluster [%s]", created.Name)

	stateStore := getStateStoreForCluster(clusterType)
	if stateStore.Exists() {
		return nil, fmt.Errorf("State store [%s] exists, will not overwrite", clusterType.Name)
	}
	stateStore.Commit(created)

	return created, nil
}

func (cluster CreateClusterSimple) DeleteClusterAzure(c *gin.Context, name string, resourceGroup string) bool {
	res, err := azureClient.DeleteCluster(name, resourceGroup)
	if err != nil {
		SetResponseBodyJson(c, err.StatusCode, gin.H{"status": err.StatusCode, "message": err.Message})
		return false
	} else {
		SetResponseBodyJson(c, res.StatusCode, res)
		return true
	}
}

//DeleteCluster deletes a cluster from the cloud
func (cluster CreateClusterSimple) DeleteClusterAmazon() (*cluster.Cluster, error) {
	logger.Level = 4

	stateStore := getStateStoreForCluster(cluster)
	if !stateStore.Exists() {
		return nil, nil
	}

	deleteCluster, err := stateStore.GetCluster()
	if err != nil {
		logger.Info(err.Error())
		logger.Info("Failed to load cluster:" + cluster.Name)
		return nil, err
	}

	reconciler, err := cutil.GetReconciler(deleteCluster, &runtimeParam)
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

//ReadCluster reads a persisted cluster from the statestore
// todo írd át
func ReadClusterOld(clusterType ClusterType) (*cluster.Cluster, error) {

	stateStore := getStateStoreForClusterOld(clusterType)
	readCluster, err := stateStore.GetCluster()
	if err != nil {
		return nil, err
	}

	return readCluster, nil
}

func ReadCluster(cl CreateClusterSimple) (*cluster.Cluster, error) {

	stateStore := getStateStoreForCluster(cl)
	readCluster, err := stateStore.GetCluster()
	if err != nil {
		return nil, err
	}

	return readCluster, nil
}

//GetKubeConfig retrieves the K8S config
func GetKubeConfig(existing *cluster.Cluster) error {

	_, err := RetryGetConfig(existing, "")
	return err
}

//UpdateCluster updates a cluster in the cloud (e.g. autoscales)
func UpdateClusterAws(ccs CreateClusterSimple) (*cluster.Cluster, error) {

	logger.Level = 4

	stateStore := getStateStoreForCluster(ccs)

	updateCluster, err := stateStore.GetCluster()
	if err != nil {
		logger.Info(err.Error())
		logger.Info("Failed to load cluster:" + ccs.Name)
		return nil, err
	}

	logger.Info("Resizing cluster : " + ccs.Name)
	logger.Info("Worker pool min size: " + strconv.Itoa(updateCluster.ServerPools[1].MinCount) + " => " + strconv.Itoa(ccs.Amazon.NodeMinCount))
	logger.Info("Worker pool max size : " + strconv.Itoa(updateCluster.ServerPools[1].MaxCount) + " => " + strconv.Itoa(ccs.Amazon.NodeMaxCount))
	updateCluster.ServerPools[0].MinCount = 1
	updateCluster.ServerPools[0].MaxCount = 1
	updateCluster.ServerPools[1].MinCount = ccs.Amazon.NodeMinCount
	updateCluster.ServerPools[1].MaxCount = ccs.Amazon.NodeMaxCount

	reconciler, err := cutil.GetReconciler(updateCluster, &runtimeParam)
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

//Wait for K8S
func awaitKubernetesCluster(existing ClusterType) (bool, error) {
	success := false
	existingCluster, _ := getStateStoreForClusterOld(existing).GetCluster()

	for i := 0; i < apiSocketAttempts; i++ {
		_, err := IsKubernetesClusterAvailable(existingCluster)
		if err != nil {
			logger.Info("Attempting to open a socket to the Kubernetes API: %v...\n", err)
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

//IsKubernetesClusterAvailable awaits for K8S cluster to be available
func IsKubernetesClusterAvailable(cluster *cluster.Cluster) (bool, error) {
	return assertTcpSocketAcceptsConnection(fmt.Sprintf("%s:%s", cluster.KubernetesAPI.Endpoint, cluster.KubernetesAPI.Port))
}
