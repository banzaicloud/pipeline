package eks

import (
	pkgErrors "github.com/banzaicloud/pipeline/pkg/errors"
)

// CreateClusterEks describes Pipeline's Amazon EKS fields of a CreateCluster request
type CreateClusterEks struct {
	NodeImageId      string `json:"nodeImageId,omitempty"`
	NodeInstanceType string `json:"nodeInstanceType,omitempty"`
	Version          string `json:"version,omitempty"`
	NodeMinCount     int    `json:"nodeMinCount,omitempty"`
	NodeMaxCount     int    `json:"nodeMaxCount,omitempty"`
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
	//TODO missing validate

	return nil
}

// Validate validates the update request (only amazon part). If any of the fields is missing, the method fills
// with stored data.
func (a *UpdateClusterAmazonEKS) Validate() error {

	// ---- [ Amazon EKS field check ] ---- //
	if a == nil {
		return pkgErrors.ErrorAmazonEksFieldIsEmpty
	}

	//TODO missing validate body

	return nil
}

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
