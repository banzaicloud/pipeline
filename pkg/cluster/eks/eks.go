package eks

import (
	pkgAmazon "github.com/banzaicloud/pipeline/pkg/cluster/amazon"
	pkgErrors "github.com/banzaicloud/pipeline/pkg/errors"
)

// CreateClusterEks describes Pipeline's Amazon EKS fields of a CreateCluster request
type CreateClusterEks struct {
	Version   string                         `json:"version,omitempty"`
	NodePools map[string]*pkgAmazon.NodePool `json:"nodePools,omitempty"`
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

	// validate K8s version
	if !isValidVersion(eks.Version) {
		return pkgErrors.ErrorNotValidKubernetesVersion
	}

	for _, np := range eks.NodePools {
		if err := np.Validate(); err != nil {
			return err
		}
	}

	return nil
}

// AddDefaults puts default values to optional field(s)
func (eks *CreateClusterEks) AddDefaults(location string) error {
	if eks == nil {
		return pkgErrors.ErrorAmazonEksFieldIsEmpty
	}

	defaultImage := DefaultImages[location]

	if len(eks.NodePools) == 0 {
		return pkgErrors.ErrorAmazonEksNodePoolFieldIsEmpty
	}

	for i, np := range eks.NodePools {
		if len(np.Image) == 0 {
			eks.NodePools[i].Image = defaultImage
		}
	}

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

// isValidVersion validates the given K8S version
func isValidVersion(version string) bool {
	if len(version) == 0 {
		return true
	}

	// currently only Kubernetes version 1.10 is supported by EKS
	// TODO check if there is an AWS API that can tell us supported Kubernetes versions
	return "1.10" == version

}

// CertificateAuthority is a helper struct for AWS kube config JSON parsing
type CertificateAuthority struct {
	Data string `json:"data,omitempty"`
}

// ClusterProfileEks describes an Amazon EKS profile
type ClusterProfileEks struct {
	Version string `json:"version,omitempty"`
}

// CreateAmazonEksObjectStoreBucketProperties describes the properties of
// S3 bucket creation request
type CreateAmazonEksObjectStoreBucketProperties struct {
	Location string `json:"location" binding:"required"`
}
