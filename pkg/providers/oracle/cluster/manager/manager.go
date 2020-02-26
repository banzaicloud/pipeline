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
	"github.com/banzaicloud/pipeline/pkg/providers/oracle/oci"
)

// ClusterManager for managing Cluster state
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
