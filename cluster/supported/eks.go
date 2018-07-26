package supported

import (
	"github.com/banzaicloud/pipeline/cluster"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	pkgErrors "github.com/banzaicloud/pipeline/pkg/errors"
)

// EksInfo describes EKS with supported info
type EksInfo struct {
	BaseFields
}

// GetType returns cloud type
func (e *EksInfo) GetType() string {
	return pkgCluster.Amazon
}

// GetNameRegexp returns regexp for cluster name
func (e *EksInfo) GetNameRegexp() string {
	return pkgCluster.RegexpAWSName
}

// GetLocations returns supported locations
func (e *EksInfo) GetLocations() ([]string, error) {

	if len(e.SecretId) == 0 {
		return nil, pkgErrors.ErrorRequiredSecretId
	}

	regions, err := cluster.ListEksRegions(e.OrgId, e.SecretId)
	if err != nil {
		return nil, err
	}

	return regions, nil
}

// GetMachineTypes returns supported machine types
func (e *EksInfo) GetMachineTypes() (map[string]pkgCluster.MachineType, error) {
	return nil, pkgErrors.ErrorRequiredLocation
}

// GetMachineTypesWithFilter returns supported machine types by location
func (e *EksInfo) GetMachineTypesWithFilter(filter *pkgCluster.InstanceFilter) (map[string]pkgCluster.MachineType, error) {
	// todo NOTE: until aws sdk dont support getting instance types

	// this list must match the instance types listed under NodeInstanceType/AllowedValues in templates/amazon-eks-nodepool-cf.yaml
	response := make(map[string]pkgCluster.MachineType)
	response[filter.Location] = []string{
		"t2.small",
		"t2.medium",
		"t2.large",
		"t2.xlarge",
		"t2.2xlarge",
		"m3.medium",
		"m3.large",
		"m3.xlarge",
		"m3.2xlarge",
		"m4.large",
		"m4.xlarge",
		"m4.2xlarge",
		"m4.4xlarge",
		"m4.10xlarge",
		"m5.large",
		"m5.xlarge",
		"m5.2xlarge",
		"m5.4xlarge",
		"m5.12xlarge",
		"m5.24xlarge",
		"c4.large",
		"c4.xlarge",
		"c4.2xlarge",
		"c4.4xlarge",
		"c4.8xlarge",
		"c5.large",
		"c5.xlarge",
		"c5.2xlarge",
		"c5.4xlarge",
		"c5.9xlarge",
		"c5.18xlarge",
		"i3.large",
		"i3.xlarge",
		"i3.2xlarge",
		"i3.4xlarge",
		"i3.8xlarge",
		"i3.16xlarge",
		"r3.xlarge",
		"r3.2xlarge",
		"r3.4xlarge",
		"r3.8xlarge",
		"r4.large",
		"r4.xlarge",
		"r4.2xlarge",
		"r4.4xlarge",
		"r4.8xlarge",
		"r4.16xlarge",
		"x1.16xlarge",
		"x1.32xlarge",
		"p2.xlarge",
		"p2.8xlarge",
		"p2.16xlarge",
		"p3.2xlarge",
		"p3.8xlarge",
		"p3.16xlarge",
	}
	return response, nil
}

// GetKubernetesVersion returns supported k8s versions
func (e *EksInfo) GetKubernetesVersion(*pkgCluster.KubernetesFilter) (interface{}, error) {
	return "1.10", nil
}

// GetImages returns supported AMIs
func (e *EksInfo) GetImages(filter *pkgCluster.ImageFilter) (map[string][]string, error) {

	if len(e.SecretId) == 0 {
		return nil, pkgErrors.ErrorRequiredSecretId
	}

	if len(filter.Location) == 0 {
		return nil, pkgErrors.ErrorRequiredLocation
	}

	return cluster.ListEksImages(filter.Location)

}
