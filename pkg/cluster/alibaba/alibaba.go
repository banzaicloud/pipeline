package alibaba

import (
	pkgErrors "github.com/banzaicloud/pipeline/pkg/errors"
)

const (
	DefaultMasterInstanceType       = "ecs.sn1.large"
	DefaultMasterSystemDiskCategory = "cloud_efficiency"
	DefaultMasterSystemDiskSize     = 40
	DefaultWorkerInstanceType       = "ecs.sn1.large"
	DefaultWorkerSystemDiskCategory = "cloud_efficiency"
	DefaultWorkerSystemDiskSize     = 40
	DefaultImage                    = "centos_7"
)

// NodePool describes Alibaba's node fields of a CreateCluster/Update request
type NodePool struct {
	WorkerInstanceType       string `json:"worker_instance_type,omitempty"`
	WorkerSystemDiskCategory string `json:"worker_system_disk_category,omitempty"`
	WorkerSystemDiskSize     int    `json:"worker_system_disk_size,omitempty"`
	LoginPassword            string `json:"login_password,omitempty"`
	ImageID                  string `json:"image_id,omitempty"`
	NumOfNodes               int    `json:"num_of_nodes"`
}

type NodePools map[string]*NodePool

// CreateClusterAlibaba
// TODO: decide to use cameCase instead of original alibaba field names.
type CreateClusterAlibaba struct {
	RegionID                 string    `json:"region_id"`
	ZoneID                   string    `json:"zoneid"`
	MasterInstanceType       string    `json:"master_instance_type,omitempty"`
	MasterSystemDiskCategory string    `json:"master_system_disk_category,omitempty"`
	MasterSystemDiskSize     int       `json:"master_system_disk_size,omitempty"`
	KeyPair                  string    `json:"key_pair,omitempty"`
	NodePools                NodePools `json:"nodePools,omitempty"`
}

// UpdateClusterAlibaba describes Alibaba's node fields of an UpdateCluster request
type UpdateClusterAlibaba struct {
	NodePools NodePools `json:"nodePools,omitempty"`
}

// AddDefaults puts default values to optional field(s)
func (c *CreateClusterAlibaba) AddDefaults() error {
	if c.MasterInstanceType == "" {
		c.MasterInstanceType = DefaultMasterInstanceType
	}
	if c.MasterSystemDiskCategory == "" {
		c.MasterSystemDiskCategory = DefaultMasterSystemDiskCategory
	}
	if c.MasterSystemDiskSize < DefaultMasterSystemDiskSize {
		c.MasterSystemDiskSize = DefaultMasterSystemDiskSize
	}

	if len(c.NodePools) == 0 {
		return pkgErrors.ErrorAlibabaNodePoolFieldIsEmpty
	}
	for i, np := range c.NodePools {
		if np.WorkerInstanceType == "" {
			c.NodePools[i].WorkerInstanceType = DefaultWorkerInstanceType
		}
		if np.WorkerSystemDiskCategory == "" {
			c.NodePools[i].WorkerSystemDiskCategory = DefaultWorkerSystemDiskCategory
		}
		if np.WorkerSystemDiskSize < DefaultWorkerSystemDiskSize {
			c.NodePools[i].WorkerSystemDiskSize = DefaultWorkerSystemDiskSize
		}
		if np.ImageID == "" {
			c.NodePools[i].ImageID = DefaultImage
		}
	}

	return nil
}

func ValidateNodePools(nps NodePools) error {
	if len(nps) == 0 {
		return pkgErrors.ErrorAlibabaNodePoolFieldIsEmpty
	}

	// Alibaba only supports one type for nodes in a cluster.
	if len(nps) > 1 {
		return pkgErrors.ErrorAlibabaNodePoolFieldLenError
	}

	for _, np := range nps {
		if np.NumOfNodes < 1 {
			return pkgErrors.ErrorAlibabaMinNumberOfNodes
		}
		if np.ImageID != DefaultImage {
			return pkgErrors.ErrorNotValidNodeImage
		}
	}
	return nil
}

func (c *CreateClusterAlibaba) Validate() error {
	if c == nil {
		return pkgErrors.ErrorAlibabaFieldIsEmpty
	}
	if c.RegionID == "" {
		return pkgErrors.ErrorAlibabaRegionIDFieldIsEmpty
	}
	if c.ZoneID == "" {
		return pkgErrors.ErrorAlibabaZoneIDFieldIsEmpty
	}
	if c.NodePools == nil {
		return pkgErrors.ErrorAlibabaNodePoolFieldIsEmpty
	}

	return ValidateNodePools(c.NodePools)
}

func (c *UpdateClusterAlibaba) Validate() error {
	if c == nil {
		return pkgErrors.ErrorAlibabaFieldIsEmpty
	}

	return ValidateNodePools(c.NodePools)
}

// ClusterProfileAlibaba describes an Alibaba profile
type ClusterProfileAlibaba struct {
	RegionID  string               `json:"region_id"`
	ZoneID    string               `json:"zoneid"`
	NodePools map[string]*NodePool `json:"nodePools,omitempty"`
}

// CreateAlibabaObjectStoreBucketProperties describes the properties of
// an OSS bucket creation request
type CreateAlibabaObjectStoreBucketProperties struct {
	Location string `json:"location" binding:"required"`
}
