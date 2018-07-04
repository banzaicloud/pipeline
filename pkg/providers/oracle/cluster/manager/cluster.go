package manager

import (
	"fmt"

	"github.com/banzaicloud/pipeline/pkg/providers/oracle/model"
	"github.com/oracle/oci-go-sdk/common"
	"github.com/oracle/oci-go-sdk/containerengine"
)

// CreateCluster creates a new cluster
func (cm *ClusterManager) CreateCluster(clusterModel *model.Cluster) error {

	ce, err := cm.oci.NewContainerEngineClient()
	if err != nil {
		return err
	}

	clusters, err := ce.ListClusterByName(clusterModel.Name)
	if err != nil {
		return err
	}

	clusters = ce.FilterClustersByNotInState(clusters, containerengine.ClusterSummaryLifecycleStateDeleted)

	if len(clusters) > 0 {
		return fmt.Errorf("Cluster[%s] already exists", clusterModel.Name)
	}

	req := containerengine.CreateClusterRequest{}
	req.Name = &clusterModel.Name
	req.CompartmentId = &cm.oci.CompartmentOCID
	req.VcnId = &clusterModel.VCNID
	req.KubernetesVersion = &clusterModel.Version
	req.Options = &containerengine.ClusterCreateOptions{
		ServiceLbSubnetIds: []string{clusterModel.LBSubnetID1, clusterModel.LBSubnetID2},
	}

	cm.oci.GetLogger().Infof("Creating cluster[%s]", clusterModel.Name)
	clusterOCID, err := ce.CreateCluster(req)
	if err != nil {
		return err
	}

	clusterModel.OCID = clusterOCID

	return cm.SyncNodePools(clusterModel)
}

// UpdateCluster updates the cluster
func (cm *ClusterManager) UpdateCluster(clusterModel *model.Cluster) error {

	cluster, err := cm.GetCluster(clusterModel.OCID)
	if err != nil {
		return err
	}

	if cluster.LifecycleState == containerengine.ClusterLifecycleStateDeleted {
		return fmt.Errorf("Cluster[%s] was deleted", *cluster.Name)
	}

	ce, err := cm.oci.NewContainerEngineClient()
	if err != nil {
		return err
	}

	update := false
	req := containerengine.UpdateClusterRequest{
		ClusterId:            cluster.Id,
		UpdateClusterDetails: containerengine.UpdateClusterDetails{},
	}

	if *cluster.KubernetesVersion != clusterModel.Version {
		update = true
		req.KubernetesVersion = common.String(clusterModel.Version)
	}
	if *cluster.Name != clusterModel.Name {
		update = true
		req.UpdateClusterDetails.Name = common.String(clusterModel.Name)
	}

	cm.oci.GetLogger().Infof("Updating cluster[%s]", *cluster.Name)

	if update {
		_, err := ce.UpdateCluster(req)
		if err != nil {
			return err
		}
	}

	return cm.SyncNodePools(clusterModel)
}

// DeleteCluster deletes a cluster
func (cm *ClusterManager) DeleteCluster(clusterModel *model.Cluster) error {

	ce, err := cm.oci.NewContainerEngineClient()
	if err != nil {
		return err
	}

	cluster, err := cm.GetCluster(clusterModel.OCID)
	if err != nil {
		return err
	}

	if cluster.LifecycleState == containerengine.ClusterLifecycleStateDeleted {
		return fmt.Errorf("Cluster[%s] was already deleted", *cluster.Name)
	}

	req := containerengine.DeleteClusterRequest{
		ClusterId: cluster.Id,
	}

	cm.oci.GetLogger().Infof("Deleting cluster[%s]", *cluster.Name)

	return ce.DeleteCluster(req)
}

// GetCluster gets cluster info by OCID
func (cm *ClusterManager) GetCluster(OCID string) (cluster containerengine.Cluster, err error) {

	ce, err := cm.oci.NewContainerEngineClient()
	if err != nil {
		return cluster, err
	}

	return ce.GetCluster(OCID)
}
