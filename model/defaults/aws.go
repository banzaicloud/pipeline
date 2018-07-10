package defaults

import (
	"github.com/banzaicloud/pipeline/database"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	pkgAmazon "github.com/banzaicloud/pipeline/pkg/cluster/amazon"
	pkgAzure "github.com/banzaicloud/pipeline/pkg/cluster/azure"
	"github.com/banzaicloud/pipeline/pkg/cluster/eks"
	pkgGoogle "github.com/banzaicloud/pipeline/pkg/cluster/google"
	pkgCommon "github.com/banzaicloud/pipeline/pkg/common"
	oracle "github.com/banzaicloud/pipeline/pkg/providers/oracle/cluster"
)

// AWSProfile describes an Amazon cluster profile
type AWSProfile struct {
	DefaultModel
	Location           string                `gorm:"default:'eu-west-1'"`
	MasterInstanceType string                `gorm:"default:'m4.xlarge'"`
	MasterImage        string                `gorm:"default:'ami-4d485ca7'"`
	NodePools          []*AWSNodePoolProfile `gorm:"foreignkey:Name"`
}

// AWSNodePoolProfile describes an Amazon cluster profile's nodepools
type AWSNodePoolProfile struct {
	ID           uint   `gorm:"primary_key"`
	InstanceType string `gorm:"default:'m4.xlarge'"`
	Name         string `gorm:"unique_index:idx_model_name"`
	NodeName     string `gorm:"unique_index:idx_model_name"`
	SpotPrice    string `gorm:"default:'0.2'"`
	Autoscaling  bool   `gorm:"default:false"`
	MinCount     int    `gorm:"default:1"`
	MaxCount     int    `gorm:"default:2"`
	Count        int    `gorm:"default:1"`
	Image        string `gorm:"default:'ami-4d485ca7'"`
}

// TableName overrides AWSNodePoolProfile's table name
func (AWSNodePoolProfile) TableName() string {
	return DefaultAmazonNodePoolProfileTablaName
}

// TableName overrides AWSProfile's table name
func (AWSProfile) TableName() string {
	return DefaultAmazonProfileTablaName
}

// SaveInstance saves cluster profile into database
func (d *AWSProfile) SaveInstance() error {
	return save(d)
}

// GetType returns profile's cloud type
func (d *AWSProfile) GetType() string {
	return pkgCluster.Amazon
}

// IsDefinedBefore returns true if database contains en entry with profile name
func (d *AWSProfile) IsDefinedBefore() bool {
	return database.GetDB().First(&d).RowsAffected != int64(0)
}

// AfterFind loads nodepools to profile
func (d *AWSProfile) AfterFind() error {
	log.Info("AfterFind aws profile... load node pools")
	return database.GetDB().Where(AWSNodePoolProfile{Name: d.Name}).Find(&d.NodePools).Error
}

// BeforeSave clears nodepools
func (d *AWSProfile) BeforeSave() error {
	log.Info("BeforeSave aws profile...")

	db := database.GetDB()
	var nodePools []*AWSNodePoolProfile
	err := db.Where(AWSNodePoolProfile{
		Name: d.Name,
	}).Find(&nodePools).Delete(&nodePools).Error
	if err != nil {
		log.Errorf("Error during deleting saved nodepools: %s", err.Error())
	}

	return nil
}

// BeforeDelete deletes all nodepools to belongs to profile
func (d *AWSProfile) BeforeDelete() error {
	log.Info("BeforeDelete aws profile... delete all nodepool")

	var nodePools []*AWSNodePoolProfile
	return database.GetDB().Where(AWSNodePoolProfile{
		Name: d.Name,
	}).Find(&nodePools).Delete(&nodePools).Error
}

// GetProfile load profile from database and converts ClusterProfileResponse
func (d *AWSProfile) GetProfile() *pkgCluster.ClusterProfileResponse {

	nodePools := make(map[string]*pkgAmazon.NodePool)
	for _, np := range d.NodePools {
		if np != nil {
			nodePools[np.NodeName] = &pkgAmazon.NodePool{
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
		Location: d.Location,
		Cloud:    pkgCluster.Amazon,
		Properties: struct {
			Amazon *pkgAmazon.ClusterProfileAmazon `json:"amazon,omitempty"`
			Azure  *pkgAzure.ClusterProfileAzure   `json:"azure,omitempty"`
			Eks    *eks.ClusterProfileEks          `json:"eks,omitempty"`
			Google *pkgGoogle.ClusterProfileGoogle `json:"google,omitempty"`
			Oracle *oracle.Cluster                 `json:"oracle,omitempty"`
		}{
			Amazon: &pkgAmazon.ClusterProfileAmazon{
				NodePools: nodePools,
				Master: &pkgAmazon.ProfileMaster{
					InstanceType: d.MasterInstanceType,
					Image:        d.MasterImage,
				},
			},
		},
	}

}

// UpdateProfile update profile's data with ClusterProfileRequest's data and if bool is true then update in the database
func (d *AWSProfile) UpdateProfile(r *pkgCluster.ClusterProfileRequest, withSave bool) error {

	if len(r.Location) != 0 {
		d.Location = r.Location
	}

	if r.Properties.Amazon != nil {

		if len(r.Properties.Amazon.NodePools) != 0 {
			var nodePools []*AWSNodePoolProfile
			for npName, nodePool := range r.Properties.Amazon.NodePools {

				spotPrice := pkgAmazon.DefaultSpotPrice
				instanceType := pkgAmazon.DefaultInstanceType
				minCount := pkgCommon.DefaultNodeMinCount
				maxCount := pkgCommon.DefaultNodeMaxCount
				image := pkgAmazon.DefaultImages[d.Location]

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

		if r.Properties.Amazon.Master != nil {
			if len(r.Properties.Amazon.Master.InstanceType) != 0 {
				d.MasterInstanceType = r.Properties.Amazon.Master.InstanceType
			}

			if len(r.Properties.Amazon.Master.Image) != 0 {
				d.MasterImage = r.Properties.Amazon.Master.Image
			}
		}
	}
	if withSave {
		return d.SaveInstance()
	}
	d.Name = r.Name
	return nil
}

// DeleteProfile deletes cluster profile from database
func (d *AWSProfile) DeleteProfile() error {
	return database.GetDB().Delete(&d).Error
}
