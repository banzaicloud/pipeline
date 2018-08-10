package accs

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
	InstanceType       string `json:"instanceType"`
	SystemDiskCategory string `json:"systemDiskCategory,omitempty"`
	SystemDiskSize     int    `json:"systemDiskSize,omitempty"`
	LoginPassword      string `json:"loginPassword,omitempty"`
	Count              int    `json:"count"`
	Image              string `json:"image"`
}

type NodePools map[string]*NodePool

// CreateClusterACCS
type CreateClusterACCS struct {
	RegionID                 string    `json:"regionId"`
	ZoneID                   string    `json:"zoneId"`
	MasterInstanceType       string    `json:"masterInstanceType,omitempty"`
	MasterSystemDiskCategory string    `json:"masterSystemDiskCategory,omitempty"`
	MasterSystemDiskSize     int       `json:"masterSystemDiskSize,omitempty"`
	KeyPair                  string    `json:"keyPair,omitempty"`
	NodePools                NodePools `json:"nodePools,omitempty"`
}

// UpdateClusterACCS describes Alibaba's node fields of an UpdateCluster request
type UpdateClusterACCS struct {
	NodePools NodePools `json:"nodePools,omitempty"`
}

// AddDefaults puts default values to optional field(s)
func (c *CreateClusterACCS) AddDefaults() error {
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
		if np.InstanceType == "" {
			c.NodePools[i].InstanceType = DefaultWorkerInstanceType
		}
		if np.SystemDiskCategory == "" {
			c.NodePools[i].SystemDiskCategory = DefaultWorkerSystemDiskCategory
		}
		if np.SystemDiskSize < DefaultWorkerSystemDiskSize {
			c.NodePools[i].SystemDiskSize = DefaultWorkerSystemDiskSize
		}
		if np.Image == "" {
			c.NodePools[i].Image = DefaultImage
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
		if np.Count < 1 {
			return pkgErrors.ErrorAlibabaMinNumberOfNodes
		}
		if np.Image != DefaultImage {
			return pkgErrors.ErrorNotValidNodeImage
		}
	}
	return nil
}

func (c *CreateClusterACCS) Validate() error {
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

func (c *UpdateClusterACCS) Validate() error {
	if c == nil {
		return pkgErrors.ErrorAlibabaFieldIsEmpty
	}

	return ValidateNodePools(c.NodePools)
}

// ClusterProfileACCS describes an Alibaba CS profile
type ClusterProfileACCS struct {
	RegionID  string               `json:"regionId"`
	ZoneID    string               `json:"zoneId"`
	NodePools map[string]*NodePool `json:"nodePools,omitempty"`
}
