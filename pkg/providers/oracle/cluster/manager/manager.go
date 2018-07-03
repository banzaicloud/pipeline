package manager

import (
	"github.com/banzaicloud/pipeline/pkg/providers/oracle/model"
	"github.com/banzaicloud/pipeline/pkg/providers/oracle/oci"
)

type ClusterManager struct {
	oci *oci.OCI
}

// NewClusterManager creates a new ClusterManager
func NewClusterManager(oci *oci.OCI) *ClusterManager {

	return &ClusterManager{
		oci: oci,
	}
}

// ManageOKECluster manages an OKE cluster specified in a model.Cluster
func (cm *ClusterManager) ManageOKECluster(clusterModel *model.Cluster) error {

	// Creating
	if clusterModel.OCID == "" && !clusterModel.Delete {
		return cm.CreateCluster(clusterModel)
	}

	// Deleting
	if clusterModel.Delete {
		return cm.DeleteCluster(clusterModel)
	}

	// Updating
	return cm.UpdateCluster(clusterModel)
}
