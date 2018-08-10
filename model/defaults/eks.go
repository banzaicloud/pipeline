package defaults

import (
	"github.com/banzaicloud/pipeline/database"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	"github.com/banzaicloud/pipeline/pkg/cluster/ec2"
	"github.com/banzaicloud/pipeline/pkg/cluster/eks"
	pkgCommon "github.com/banzaicloud/pipeline/pkg/common"
)

// EKSProfile describes an Amazon EKS cluster profile
type EKSProfile struct {
	DefaultModel
	Region    string                `gorm:"default:'us-west-2'"`
	Version   string                `gorm:"default:'1.10'"`
	NodePools []*EKSNodePoolProfile `gorm:"foreignkey:Name"`
}

// EKSNodePoolProfile describes an EKS cluster profile's nodepools
type EKSNodePoolProfile struct {
	AmazonNodePoolProfileBaseFields
	Image string `gorm:"default:'ami-0ea01e1d1dea65b5'"`
}

// TableName overrides EKSProfile's table name
func (EKSProfile) TableName() string {
	return DefaultEKSProfileTableName
}

// TableName overrides EKSNodePoolProfile's table name
func (EKSNodePoolProfile) TableName() string {
	return DefaultEKSNodePoolProfileTableName
}

// SaveInstance saves cluster profile into database
func (d *EKSProfile) SaveInstance() error {
	return save(d)
}

// GetCloud returns profile's cloud type
func (d *EKSProfile) GetCloud() string {
	return pkgCluster.Amazon
}

// GetDistribution returns profile's distribution type
func (d *EKSProfile) GetDistribution() string {
	return pkgCluster.EKS
}

// IsDefinedBefore returns true if database contains en entry with profile name
func (d *EKSProfile) IsDefinedBefore() bool {
	return database.GetDB().First(&d).RowsAffected != 0
}

// GetProfile load profile from database and converts ClusterProfileResponse
func (d *EKSProfile) GetProfile() *pkgCluster.ClusterProfileResponse {

	nodePools := make(map[string]*ec2.NodePool)
	for _, np := range d.NodePools {
		if np != nil {
			nodePools[np.NodeName] = &ec2.NodePool{
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
		Properties: &pkgCluster.ClusterProfileProperties{
			EKS: &eks.ClusterProfileEKS{
				Version:   d.Version,
				NodePools: nodePools,
			},
		},
	}

}

// UpdateProfile update profile's data with ClusterProfileRequest's data and if bool is true then update in the database
func (d *EKSProfile) UpdateProfile(r *pkgCluster.ClusterProfileRequest, withSave bool) error {

	if len(r.Location) != 0 {
		d.Region = r.Location
	}

	if r.Properties.EKS != nil {

		if len(r.Properties.EKS.Version) != 0 {
			d.Version = r.Properties.EKS.Version
		}

		if len(r.Properties.EKS.NodePools) != 0 {
			var nodePools []*EKSNodePoolProfile
			for npName, nodePool := range r.Properties.EKS.NodePools {

				spotPrice := ec2.DefaultSpotPrice
				instanceType := ec2.DefaultInstanceType
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

				nodePools = append(nodePools, &EKSNodePoolProfile{
					AmazonNodePoolProfileBaseFields: AmazonNodePoolProfileBaseFields{
						InstanceType: instanceType,
						Name:         d.Name,
						NodeName:     npName,
						SpotPrice:    spotPrice,
						Autoscaling:  nodePool.Autoscaling,
						MinCount:     minCount,
						MaxCount:     maxCount,
						Count:        count,
					},
					Image: image,
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

// AfterFind loads nodepools to profile
func (d *EKSProfile) AfterFind() error {
	log.Info("AfterFind eks profile... load node pools")
	return database.GetDB().Where(EKSNodePoolProfile{
		AmazonNodePoolProfileBaseFields: AmazonNodePoolProfileBaseFields{
			Name: d.Name,
		},
	}).Find(&d.NodePools).Error
}

// BeforeSave clears nodepools
func (d *EKSProfile) BeforeSave() error {
	log.Info("BeforeSave eks profile...")

	db := database.GetDB()
	var nodePools []*EKSNodePoolProfile
	err := db.Where(EKSNodePoolProfile{
		AmazonNodePoolProfileBaseFields: AmazonNodePoolProfileBaseFields{
			Name: d.Name,
		},
	}).Find(&nodePools).Delete(&nodePools).Error
	if err != nil {
		log.Errorf("Error during deleting saved nodepools: %s", err.Error())
	}

	return nil
}

// BeforeDelete deletes all nodepools to belongs to profile
func (d *EKSProfile) BeforeDelete() error {
	log.Info("BeforeDelete eks profile... delete all nodepool")

	var nodePools []*EKSNodePoolProfile
	return database.GetDB().Where(EKSNodePoolProfile{
		AmazonNodePoolProfileBaseFields: AmazonNodePoolProfileBaseFields{
			Name: d.Name,
		},
	}).Find(&nodePools).Delete(&nodePools).Error
}
