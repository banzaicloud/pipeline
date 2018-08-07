package defaults

import (
	"github.com/banzaicloud/pipeline/database"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	"github.com/banzaicloud/pipeline/pkg/cluster/amazon"
	"github.com/banzaicloud/pipeline/pkg/cluster/azure"
	"github.com/banzaicloud/pipeline/pkg/cluster/eks"
	"github.com/banzaicloud/pipeline/pkg/cluster/google"
	pkgCommon "github.com/banzaicloud/pipeline/pkg/common"
	oracle "github.com/banzaicloud/pipeline/pkg/providers/oracle/cluster"
)

// EKSProfile describes an Amazon EKS cluster profile
type EKSProfile struct {
	DefaultModel
	Region           string                `gorm:"default:'us-west-2'"`
	NodeImageId      string                `json:"nodeImageId,omitempty"`
	NodeInstanceType string                `json:"nodeInstanceType,omitempty"`
	Version          string                `json:"version,omitempty"`
	NodePools        []*AWSNodePoolProfile `gorm:"foreignkey:Name"`
}

// TableName overrides EKSProfile's table name
func (EKSProfile) TableName() string {
	return DefaultAmazonEksProfileTablaName
}

// SaveInstance saves cluster profile into database
func (d *EKSProfile) SaveInstance() error {
	return save(d)
}

// GetType returns profile's cloud type
func (d *EKSProfile) GetType() string {
	return pkgCluster.AmazonEKS
}

// IsDefinedBefore returns true if database contains en entry with profile name
func (d *EKSProfile) IsDefinedBefore() bool {
	return database.GetDB().First(&d).RowsAffected != 0
}

// GetProfile load profile from database and converts ClusterProfileResponse
func (d *EKSProfile) GetProfile() *pkgCluster.ClusterProfileResponse {

	nodePools := make(map[string]*amazon.NodePool)
	for _, np := range d.NodePools {
		if np != nil {
			nodePools[np.NodeName] = &amazon.NodePool{
				InstanceType: np.InstanceType,
				SpotPrice:    np.SpotPrice,
				Autoscaling:  np.Autoscaling,
				MinCount:     np.MinCount,
				MaxCount:     np.MaxCount,
				Count:        np.Count,
				Image:        np.Image,
			}
		}
	}

	return &pkgCluster.ClusterProfileResponse{
		Name:     d.DefaultModel.Name,
		Location: d.Region,
		Cloud:    pkgCluster.Amazon,
		Properties: struct {
			Amazon *amazon.ClusterProfileAmazon `json:"amazon,omitempty"`
			Azure  *azure.ClusterProfileAzure   `json:"azure,omitempty"`
			Eks    *eks.ClusterProfileEks       `json:"eks,omitempty"`
			Google *google.ClusterProfileGoogle `json:"google,omitempty"`
			Oracle *oracle.Cluster              `json:"oracle,omitempty"`
		}{
			Eks: &eks.ClusterProfileEks{
				Version: d.Version,
			},
		},
	}

}

// UpdateProfile update profile's data with ClusterProfileRequest's data and if bool is true then update in the database
func (d *EKSProfile) UpdateProfile(r *pkgCluster.ClusterProfileRequest, withSave bool) error {

	if len(r.Location) != 0 {
		d.Region = r.Location
	}

	if r.Properties.Eks != nil {

		if len(r.Properties.Eks.Version) != 0 {
			d.Version = r.Properties.Eks.Version
		}

		if len(r.Properties.Eks.NodePools) != 0 {
			var nodePools []*AWSNodePoolProfile
			for npName, nodePool := range r.Properties.Eks.NodePools {

				spotPrice := amazon.DefaultSpotPrice
				instanceType := amazon.DefaultInstanceType
				minCount := pkgCommon.DefaultNodeMinCount
				maxCount := pkgCommon.DefaultNodeMaxCount
				image := eks.DefaultImages[d.Region]

				if len(nodePool.SpotPrice) != 0 {
					spotPrice = nodePool.SpotPrice
				}

				if len(nodePool.InstanceType) != 0 {
					instanceType = nodePool.InstanceType
				}

				if nodePool.MinCount != 0 {
					minCount = nodePool.MinCount
				}

				if nodePool.MaxCount != 0 {
					maxCount = nodePool.MaxCount
				}

				if minCount > maxCount {
					minCount = pkgCommon.DefaultNodeMinCount
					maxCount = pkgCommon.DefaultNodeMaxCount
				}

				count := nodePool.Count
				if count == 0 {
					count = minCount
				}

				if len(nodePool.Image) != 0 {
					image = nodePool.Image
				}

				nodePools = append(nodePools, &AWSNodePoolProfile{
					InstanceType: instanceType,
					Name:         d.Name,
					NodeName:     npName,
					SpotPrice:    spotPrice,
					Autoscaling:  nodePool.Autoscaling,
					MinCount:     minCount,
					MaxCount:     maxCount,
					Count:        count,
					Image:        image,
				})

			}

			d.NodePools = nodePools
		}
	}
	if withSave {
		return d.SaveInstance()
	}
	d.Name = r.Name
	return nil
}

// DeleteProfile deletes cluster profile from database
func (d *EKSProfile) DeleteProfile() error {
	return database.GetDB().Delete(&d).Error
}
