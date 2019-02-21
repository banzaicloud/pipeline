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

package manager

import (
	"github.com/oracle/oci-go-sdk/common"
	"github.com/oracle/oci-go-sdk/containerengine"

	pkgCommon "github.com/banzaicloud/pipeline/pkg/common"
	"github.com/banzaicloud/pipeline/pkg/providers/oracle/model"
	"github.com/banzaicloud/pipeline/pkg/providers/oracle/oci"
)

// SyncNodePools keeps the cluster node pools in state with the model
func (cm *ClusterManager) SyncNodePools(clusterModel *model.Cluster) error {

	cm.oci.GetLogger().Infof("Syncing Node Pools states of Cluster[%s]", clusterModel.Name)

	nodePools := clusterModel.NodePools

	ce, err := cm.oci.NewContainerEngineClient()
	if err != nil {
		return err
	}

	waitForChange := false
	nodepoolNamesToCheck := make(map[string]bool, 0)
	for _, np := range nodePools {
		if !np.Delete {
			nodepoolNamesToCheck[np.Name] = true
		}
		if waitForChange == false && !np.Delete {
			waitForChange = true
		}
		if np.Add {
			if err := cm.AddNodePool(clusterModel, np); err != nil {
				return err
			}
		} else if !np.Delete {
			if err := cm.UpdateNodePool(clusterModel, np); err != nil {
				return err
			}
		}
	}

	// waiting for add/update operations to finish
	if waitForChange {
		err = ce.WaitingForClusterNodePoolActiveState(&clusterModel.OCID, nodepoolNamesToCheck)
		if err != nil {
			return err
		}
	}

	waitForDelete := false
	for _, np := range nodePools {
		if np.Delete {
			if !waitForDelete {
				waitForDelete = true
			}
			if err := cm.DeleteNodePool(clusterModel, np); err != nil {
				return err
			}
		}
	}

	if waitForDelete {
		err = ce.WaitingForClusterNodePoolActiveState(&clusterModel.OCID, make(map[string]bool, 0))
		if err != nil {
			return err
		}
	}

	return nil
}

// UpdateNodePool updates node pool in a cluster
func (cm *ClusterManager) UpdateNodePool(clusterModel *model.Cluster, np *model.NodePool) error {

	ce, err := cm.oci.NewContainerEngineClient()
	if err != nil {
		return err
	}

	nodePool, err := ce.GetNodePoolByName(&clusterModel.OCID, np.Name)
	if err != nil && !oci.IsEntityNotFoundError(err) {
		return err
	}

	if nodePool.Id == nil {
		return nil
	}

	cm.oci.GetLogger().Infof("Updating NodePool[%s]", *nodePool.Name)

	request := containerengine.UpdateNodePoolRequest{
		NodePoolId: nodePool.Id,
		UpdateNodePoolDetails: containerengine.UpdateNodePoolDetails{
			Name:              &np.Name,
			KubernetesVersion: &np.Version,
			QuantityPerSubnet: common.Int(int(np.QuantityPerSubnet)),
		},
	}
	for _, subnet := range np.Subnets {
		request.UpdateNodePoolDetails.SubnetIds = append(request.UpdateNodePoolDetails.SubnetIds, subnet.SubnetID)
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

	return ce.DeleteNodePoolByName(&clusterModel.OCID, np.Name)
}

// AddNodePool creates a new node pool in a cluster
func (cm *ClusterManager) AddNodePool(clusterModel *model.Cluster, np *model.NodePool) error {

	ce, err := cm.oci.NewContainerEngineClient()
	if err != nil {
		return err
	}

	nodePool, err := ce.GetNodePoolByName(&clusterModel.OCID, np.Name)
	if err != nil && !oci.IsEntityNotFoundError(err) {
		return err
	}

	if nodePool.Id != nil {
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
	createNodePoolReq.QuantityPerSubnet = common.Int(int(np.QuantityPerSubnet))
	createNodePoolReq.SshPublicKey = common.String(clusterModel.SSHPubKey)

	for _, subnet := range np.Subnets {
		createNodePoolReq.SubnetIds = append(createNodePoolReq.SubnetIds, subnet.SubnetID)
	}

	// only add node pool name label, other are added by NodeLabelController
	createNodePoolReq.InitialNodeLabels = append(createNodePoolReq.InitialNodeLabels, containerengine.KeyValue{
		Key: common.String(pkgCommon.LabelKey), Value: common.String(np.Name),
	})

	nodepoolOCID, err := ce.CreateNodePool(createNodePoolReq)
	if err != nil {
		return err
	}

	np.OCID = nodepoolOCID

	return nil
}
