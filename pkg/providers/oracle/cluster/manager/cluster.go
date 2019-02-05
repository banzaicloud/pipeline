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
	"github.com/banzaicloud/pipeline/pkg/providers/oracle/model"
	"github.com/oracle/oci-go-sdk/common"
	"github.com/oracle/oci-go-sdk/containerengine"
	"github.com/pkg/errors"
)

// CreateCluster creates a new cluster
func (cm *ClusterManager) CreateCluster(clusterModel *model.Cluster) error {

	ce, err := cm.oci.NewContainerEngineClient()
	if err != nil {
		return err
	}

	clusters, err := ce.GetClustersByName(clusterModel.Name)
	if err != nil {
		return err
	}

	clusters = ce.FilterClustersByNotInState(clusters, containerengine.ClusterSummaryLifecycleStateDeleted)

	if len(clusters) > 0 {
		return errors.Errorf("cluster[%s] already exists", clusterModel.Name)
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

	cluster, err := cm.GetCluster(clusterModel)
	if err != nil {
		return err
	}

	if cluster.LifecycleState == containerengine.ClusterLifecycleStateDeleted {
		return errors.Errorf("cluster[%s] was deleted", *cluster.Name)
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

	cluster, err := cm.GetCluster(clusterModel)
	if err != nil {
		return err
	}

	if cluster.LifecycleState == containerengine.ClusterLifecycleStateDeleted {
		return errors.Errorf("cluster[%s] was already deleted", *cluster.Name)
	}

	req := containerengine.DeleteClusterRequest{
		ClusterId: cluster.Id,
	}

	cm.oci.GetLogger().Infof("Deleting cluster[%s]", *cluster.Name)

	return ce.DeleteCluster(req)
}

// GetCluster gets a cluster by ID or name
func (cm *ClusterManager) GetCluster(model *model.Cluster) (cluster containerengine.Cluster, err error) {

	if model.OCID != "" {
		return cm.GetClusterByID(&model.OCID)
	}

	return cm.GetClusterByName(model.Name)
}

// GetClusterByID gets cluster by ID
func (cm *ClusterManager) GetClusterByID(id *string) (cluster containerengine.Cluster, err error) {

	ce, err := cm.oci.NewContainerEngineClient()
	if err != nil {
		return cluster, err
	}

	return ce.GetClusterByID(id)
}

// GetClusterByName gets cluster by name
func (cm *ClusterManager) GetClusterByName(name string) (cluster containerengine.Cluster, err error) {

	ce, err := cm.oci.NewContainerEngineClient()
	if err != nil {
		return cluster, err
	}

	c, err := ce.GetClusterByName(name)
	if err != nil {
		return cluster, err
	}

	return ce.GetClusterByID(c.Id)
}
