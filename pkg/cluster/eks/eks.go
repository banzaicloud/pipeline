package eks

import (
	pkgCommon "github.com/banzaicloud/pipeline/pkg/common"
	pkgErrors "github.com/banzaicloud/pipeline/pkg/errors"
)

// CreateClusterEks describes Pipeline's Amazon EKS fields of a CreateCluster request
type CreateClusterEks struct {
	NodeImageId      string `json:"nodeImageId,omitempty"`
	NodeInstanceType string `json:"nodeInstanceType,omitempty"`
	Version          string `json:"version,omitempty"`
	MinCount         int    `json:"minCount,omitempty"`
	MaxCount         int    `json:"maxCount,omitempty"`
}

// UpdateClusterAmazonEKS describes Amazon EKS's node fields of an UpdateCluster request
type UpdateClusterAmazonEKS struct {
	//TODO missing fields
}

// Validate validates Amazon EKS cluster create request
func (eks *CreateClusterEks) Validate() error {
	if eks == nil {
		return pkgErrors.ErrorAmazonEksFieldIsEmpty
	}

	// TODO: support nodepools

	/* if len(eks.NodePools) == 0 {
		  return pkgErrors.ErrorAmazonEksNodePoolFieldIsEmpty
	  }

	 for _, np := range eks.NodePools {
		if err := np.Validate(); err != nil {
			return err
		}
	}

	return nil
	}*/

	// ---- [ Node min count check ] ---- //
	if eks.MinCount == 0 {
		eks.MinCount = pkgCommon.DefaultNodeMinCount
	}

	// ---- [ Node max count check ] ---- //
	if eks.MaxCount == 0 {
		eks.MaxCount = pkgCommon.DefaultNodeMaxCount
	}

	// ---- [ Node min count <= max count check ] ---- //
	if eks.MaxCount < eks.MinCount {
		return pkgErrors.ErrorNodePoolMinMaxFieldError
	}

	if len(eks.NodeInstanceType) == 0 {
		eks.NodeInstanceType = DefaultInstanceType
	}

	// TODO: support spot price // ---- [ Node spot price ] ---- //

	return nil
}

// AddDefaults puts default values to optional field(s)
func (eks *CreateClusterEks) AddDefaults(location string) error {
	if eks == nil {
		return pkgErrors.ErrorAmazonEksFieldIsEmpty
	}

	defaultImage := DefaultImages[location]

	// TODO: add nodepool support
	/*
		if len(eks.NodePools) == 0 {
			return pkgErrors.ErrorAmazonEksNodePoolFieldIsEmpty
		}

		for i, np := range eks.NodePools {
			if len(np.Image) == 0 {
				eks.NodePools[i].Image = defaultImage
			}
		}
	*/
	eks.NodeImageId = defaultImage

	return nil
}

// Validate validates the update request (only EKS part). If any of the fields is missing, the method fills
// with stored data.
func (eks *UpdateClusterAmazonEKS) Validate() error {

	// ---- [ Amazon EKS field check ] ---- //
	if eks == nil {
		return pkgErrors.ErrorAmazonEksFieldIsEmpty
	}

	//TODO missing validate body

	return nil
}

// CertificateAuthority is a helper struct for AWS kube config JSON parsing
type CertificateAuthority struct {
	Data string `json:"data,omitempty"`
}

// ClusterProfileEks describes an Amazon EKS profile
type ClusterProfileEks struct {
	NodeImageId      string `json:"nodeImageId,omitempty"`
	NodeInstanceType string `json:"nodeInstanceType,omitempty"`
	Version          string `json:"version,omitempty"`
	NodeMinCount     int    `json:"nodeMinCount,omitempty"`
	NodeMaxCount     int    `json:"nodeMaxCount,omitempty"`
}

// CreateAmazonEksObjectStoreBucketProperties describes the properties of
// S3 bucket creation request
type CreateAmazonEksObjectStoreBucketProperties struct {
	Location string `json:"location" binding:"required"`
}
