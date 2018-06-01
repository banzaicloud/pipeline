package defaults

import (
	"github.com/banzaicloud/banzai-types/components"
	"github.com/banzaicloud/banzai-types/components/amazon"
	"github.com/banzaicloud/banzai-types/components/azure"
	"github.com/banzaicloud/banzai-types/components/google"
	"github.com/banzaicloud/banzai-types/constants"
	"github.com/banzaicloud/pipeline/model"
)

// AWSProfile describes an Amazon cluster profile
type AWSProfile struct {
	DefaultModel
	Location           string                `gorm:"default:'eu-west-1'"`
	MasterInstanceType string                `gorm:"default:'m4.xlarge'"`
	MasterImage        string                `gorm:"default:'ami-16bfeb6f'"`
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
	Image        string `gorm:"default:'ami-16bfeb6f'"`
}

const (
	defaultSpotPrice    = "0.2"
	defaultInstanceType = "m4.xlarge"
	defaultMinCount     = 1
	defaultMaxCount     = 2
	defaultImage        = "ami-16bfeb6f"
)

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
	return constants.Amazon
}

// IsDefinedBefore returns true if database contains en entry with profile name
func (d *AWSProfile) IsDefinedBefore() bool {
	return model.GetDB().First(&d).RowsAffected != int64(0)
}

// AfterFind loads nodepools to profile
func (d *AWSProfile) AfterFind() error {
	log.Info("AfterFind aws profile... load node pools")
	return model.GetDB().Where(AWSNodePoolProfile{Name: d.Name}).Find(&d.NodePools).Error
}

// BeforeSave clears nodepools
func (d *AWSProfile) BeforeSave() error {
	log.Info("BeforeSave aws profile...")

	db := model.GetDB()
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
	return model.GetDB().Where(AWSNodePoolProfile{
		Name: d.Name,
	}).Find(&nodePools).Delete(&nodePools).Error
}

// GetProfile load profile from database and converts ClusterProfileResponse
func (d *AWSProfile) GetProfile() *components.ClusterProfileResponse {

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

	return &components.ClusterProfileResponse{
		Name:     d.DefaultModel.Name,
		Location: d.Location,
		Cloud:    constants.Amazon,
		Properties: struct {
			Amazon *amazon.ClusterProfileAmazon `json:"amazon,omitempty"`
			Azure  *azure.ClusterProfileAzure   `json:"azure,omitempty"`
			Google *google.ClusterProfileGoogle `json:"google,omitempty"`
		}{
			Amazon: &amazon.ClusterProfileAmazon{
				NodePools: nodePools,
				Master: &amazon.AmazonProfileMaster{
					InstanceType: d.MasterInstanceType,
					Image:        d.MasterImage,
				},
			},
		},
	}

}

// UpdateProfile update profile's data with ClusterProfileRequest's data and if bool is true then update in the database
func (d *AWSProfile) UpdateProfile(r *components.ClusterProfileRequest, withSave bool) error {

	if len(r.Location) != 0 {
		d.Location = r.Location
	}

	if r.Properties.Amazon != nil {

		if len(r.Properties.Amazon.NodePools) != 0 {
			var nodePools []*AWSNodePoolProfile
			for npName, nodePool := range r.Properties.Amazon.NodePools {

				spotPrice := defaultSpotPrice
				instanceType := defaultInstanceType
				minCount := defaultMinCount
				maxCount := defaultMaxCount
				image := defaultImage

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
					minCount = defaultMinCount
					maxCount = defaultMaxCount
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
	return model.GetDB().Delete(&d).Error
}
