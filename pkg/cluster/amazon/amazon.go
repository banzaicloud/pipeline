package amazon

import (
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	pkgErrors "github.com/banzaicloud/pipeline/pkg/errors"
)

// CreateClusterAmazon describes Pipeline's Amazon fields of a CreateCluster request
type CreateClusterAmazon struct {
	NodePools map[string]*NodePool `json:"nodePools,omitempty"`
	Master    *CreateAmazonMaster  `json:"master,omitempty"`
}

// CreateAmazonMaster describes Amazon's master fields of a CreateCluster request
type CreateAmazonMaster struct {
	InstanceType string `json:"instanceType"`
	Image        string `json:"image"`
}

// NodePool describes Amazon's node fields of a CreateCluster/Update request
type NodePool struct {
	InstanceType string `json:"instanceType"`
	SpotPrice    string `json:"spotPrice"`
	Autoscaling  bool   `json:"autoscaling"`
	MinCount     int    `json:"minCount"`
	MaxCount     int    `json:"maxCount"`
	Count        int    `json:"count"`
	Image        string `json:"image"`
}

// UpdateClusterAmazon describes Amazon's node fields of an UpdateCluster request
type UpdateClusterAmazon struct {
	NodePools map[string]*NodePool `json:"nodePools,omitempty"`
}

// Validate checks Amazon's node fields
func (a *NodePool) Validate() error {
	// ---- [ Node image check ] ---- //
	if len(a.InstanceType) == 0 {
		return pkgErrors.ErrorAmazonInstancetypeFieldIsEmpty
	}

	// ---- [ Node image check ] ---- //
	if len(a.Image) == 0 {
		return pkgErrors.ErrorAmazonImageFieldIsEmpty
	}

	// ---- [ Min & Max count fields are required in case of autoscaling ] ---- //
	if a.Autoscaling {

		if a.MinCount == 0 {
			return pkgErrors.ErrorMinFieldRequiredError
		}
		if a.MaxCount == 0 {
			return pkgErrors.ErrorMaxFieldRequiredError
		}

	} else {
		// ---- [ Node min count check ] ---- //
		if a.MinCount == 0 {
			a.MinCount = pkgCluster.DefaultNodeMinCount
		}

		// ---- [ Node max count check ] ---- //
		if a.MaxCount == 0 {
			a.MaxCount = pkgCluster.DefaultNodeMaxCount
		}
	}

	// ---- [ Node min count <= max count check ] ---- //
	if a.MaxCount < a.MinCount {
		return pkgErrors.ErrorNodePoolMinMaxFieldError
	}

	if a.Count == 0 {
		a.Count = a.MinCount
	}

	// ---- [ Node spot price ] ---- //
	if len(a.SpotPrice) == 0 {
		a.SpotPrice = pkgCluster.AmazonDefaultNodeSpotPrice
	}

	return nil
}

// Validate validates Amazon cluster create request
func (amazon *CreateClusterAmazon) Validate() error {
	if amazon == nil {
		return pkgErrors.ErrorAmazonFieldIsEmpty
	}
	if amazon.Master == nil {
		return pkgErrors.ErrorAmazonMasterFieldIsEmpty
	}
	if amazon.Master.Image == "" {
		return pkgErrors.ErrorAmazonImageFieldIsEmpty
	}

	if amazon.Master.InstanceType == "" {
		amazon.Master.InstanceType = pkgCluster.AmazonDefaultMasterInstanceType
	}

	if len(amazon.NodePools) == 0 {
		return pkgErrors.ErrorAmazonNodePoolFieldIsEmpty
	}

	for _, np := range amazon.NodePools {
		if err := np.Validate(); err != nil {
			return err
		}
	}

	return nil
}

// AddDefaults puts default values to optional field(s)
func (amazon *CreateClusterAmazon) AddDefaults() error {

	if amazon == nil {
		return pkgErrors.ErrorAmazonFieldIsEmpty
	}

	if amazon.Master == nil {
		amazon.Master = &CreateAmazonMaster{
			InstanceType: pkgCluster.AmazonDefaultMasterInstanceType,
			Image:        pkgCluster.AmazonDefaultImage,
		}
	}

	if len(amazon.NodePools) == 0 {
		return pkgErrors.ErrorAmazonNodePoolFieldIsEmpty
	}

	for i, np := range amazon.NodePools {
		if len(np.Image) == 0 {
			amazon.NodePools[i].Image = pkgCluster.AmazonDefaultImage
		}
	}

	return nil
}

// Validate validates the update request (only amazon part). If any of the fields is missing, the method fills
// with stored data.
func (a *UpdateClusterAmazon) Validate() error {

	// ---- [ Amazon field check ] ---- //
	if a == nil {
		return pkgErrors.ErrorAmazonFieldIsEmpty
	}

	if len(a.NodePools) == 0 {
		return pkgErrors.ErrorAmazonNodePoolFieldIsEmpty
	}

	for _, np := range a.NodePools {
		if err := np.Validate(); err != nil {
			return err
		}
	}

	return nil
}

// ClusterProfileAmazon describes an Amazon profile
type ClusterProfileAmazon struct {
	Master    *AmazonProfileMaster `json:"master,omitempty"`
	NodePools map[string]*NodePool `json:"nodePools,omitempty"`
}

// AmazonProfileMaster describes an Amazon profile's master fields
type AmazonProfileMaster struct {
	InstanceType string `json:"instanceType"`
	Image        string `json:"image"`
}

// CreateAmazonObjectStoreBucketProperties describes the properties of
// S3 bucket creation request
type CreateAmazonObjectStoreBucketProperties struct {
	Location string `json:"location" binding:"required"`
}
