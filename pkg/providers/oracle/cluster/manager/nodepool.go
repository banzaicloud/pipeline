package manager

import (
	"github.com/banzaicloud/pipeline/pkg/providers/oracle/model"
	"github.com/oracle/oci-go-sdk/containerengine"
)

// SyncNodePools keeps the cluster node pools in state with the model
func (cm *ClusterManager) SyncNodePools(clusterModel *model.Cluster) error {

	cm.oci.GetLogger().Infof("Syncing Node Pools states of Cluster[%s]", clusterModel.Name)

	for _, np := range clusterModel.NodePools {
		if np.Add {
			if err := cm.AddNodePool(clusterModel, np); err != nil {
				return err
			}
		} else if np.Delete {
			if err := cm.DeleteNodePool(clusterModel, np); err != nil {
				return err
			}
		} else {
			if err := cm.UpdateNodePool(clusterModel, np); err != nil {
				return err
			}
		}
	}

	ce, err := cm.oci.NewContainerEngineClient()
	if err != nil {
		return err
	}

	return ce.WaitingForClusterNodePoolActiveState(clusterModel.OCID)
}

// UpdateNodePool updates node pool in a cluster
func (cm *ClusterManager) UpdateNodePool(clusterModel *model.Cluster, np *model.NodePool) error {

	ce, err := cm.oci.NewContainerEngineClient()
	if err != nil {
		return err
	}

	nodePools, err := ce.ListClusterNodePoolsByName(clusterModel.OCID, np.Name)
	if err != nil {
		return err
	}

	if len(nodePools) != 1 {
		return nil
	}

	nodePool := nodePools[0]

	cm.oci.GetLogger().Infof("Updating NodePool[%s]", *nodePool.Name)

	request := containerengine.UpdateNodePoolRequest{
		NodePoolId: nodePool.Id,
		UpdateNodePoolDetails: containerengine.UpdateNodePoolDetails{
			Name:              &np.Name,
			KubernetesVersion: &np.Version,
			QuantityPerSubnet: &np.QuantityPerSubnet,
		},
	}
	for _, subnet := range np.Subnets {
		request.UpdateNodePoolDetails.SubnetIds = append(request.UpdateNodePoolDetails.SubnetIds, subnet.SubnetID)
	}
	for _, label := range np.Labels {
		request.UpdateNodePoolDetails.InitialNodeLabels = append(request.UpdateNodePoolDetails.InitialNodeLabels, containerengine.KeyValue{
			Key: &label.Name, Value: &label.Value,
		})
	}

	_, err = ce.UpdateNodePool(request)
	if err != nil {
		return err
	}

	return nil
}

// DeleteNodePool deletes a node pool from a cluster
func (cm *ClusterManager) DeleteNodePool(clusterModel *model.Cluster, np *model.NodePool) error {

	cm.oci.GetLogger().Infof("Deleting NodePool[%s]", np.Name)

	ce, err := cm.oci.NewContainerEngineClient()
	if err != nil {
		return err
	}

	return ce.DeleteClusterNodePoolByName(clusterModel.OCID, np.Name)
}

// AddNodePool creates a new node pool in a cluster
func (cm *ClusterManager) AddNodePool(clusterModel *model.Cluster, np *model.NodePool) error {

	ce, err := cm.oci.NewContainerEngineClient()
	if err != nil {
		return err
	}

	nodePools, err := ce.ListClusterNodePoolsByName(clusterModel.OCID, np.Name)
	if err != nil {
		return err
	}

	if len(nodePools) > 0 {
		return nil
	}

	cm.oci.GetLogger().Infof("Adding Node Pool[%s] to Cluster[%s]", np.Name, clusterModel.Name)

	// create NodePool
	createNodePoolReq := containerengine.CreateNodePoolRequest{}
	createNodePoolReq.CompartmentId = &cm.oci.CompartmentOCID
	createNodePoolReq.Name = &np.Name
	createNodePoolReq.ClusterId = &clusterModel.OCID
	createNodePoolReq.KubernetesVersion = &np.Version
	createNodePoolReq.NodeImageName = &np.Image
	createNodePoolReq.NodeShape = &np.Shape
	createNodePoolReq.QuantityPerSubnet = &np.QuantityPerSubnet

	for _, subnet := range np.Subnets {
		createNodePoolReq.SubnetIds = append(createNodePoolReq.SubnetIds, subnet.SubnetID)
	}
	for _, label := range np.Labels {
		createNodePoolReq.InitialNodeLabels = append(createNodePoolReq.InitialNodeLabels, containerengine.KeyValue{
			Key: &label.Name, Value: &label.Value,
		})
	}

	nodepoolOCID, err := ce.CreateNodePool(createNodePoolReq)
	if err != nil {
		return err
	}

	np.OCID = nodepoolOCID

	return nil
}
