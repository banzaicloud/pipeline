package defaults

import (
	"github.com/banzaicloud/pipeline/database"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	pkgAzure "github.com/banzaicloud/pipeline/pkg/cluster/aks"
	"github.com/banzaicloud/pipeline/pkg/cluster/alibaba"
	pkgAmazon "github.com/banzaicloud/pipeline/pkg/cluster/ec2"
	"github.com/banzaicloud/pipeline/pkg/cluster/eks"
	pkgGoogle "github.com/banzaicloud/pipeline/pkg/cluster/gke"
	pkgCommon "github.com/banzaicloud/pipeline/pkg/common"
	oracle "github.com/banzaicloud/pipeline/pkg/providers/oracle/cluster"
)

// EC2Profile describes an Amazon cluster profile
type EC2Profile struct {
	DefaultModel
	Location           string                `gorm:"default:'eu-west-1'"`
	MasterInstanceType string                `gorm:"default:'m4.xlarge'"`
	MasterImage        string                `gorm:"default:'ami-4d485ca7'"`
	NodePools          []*EC2NodePoolProfile `gorm:"foreignkey:Name"`
}

// EC2NodePoolProfile describes an EC2 cluster profile's nodepools
type EC2NodePoolProfile struct {
	AmazonNodePoolProfileBaseFields
	Image string `gorm:"default:'ami-4d485ca7'"`
}

// TableName overrides AmazonNodePoolProfile's table name
func (EC2NodePoolProfile) TableName() string {
	return DefaultEC2NodePoolProfileTableName
}

// TableName overrides EC2Profile's table name
func (EC2Profile) TableName() string {
	return DefaultEC2ProfileTableName
}

// SaveInstance saves cluster profile into database
func (d *EC2Profile) SaveInstance() error {
	return save(d)
}

// GetCloud returns profile's cloud type
func (d *EC2Profile) GetCloud() string {
	return pkgCluster.Amazon
}

// GetDistribution returns profile's distribution type
func (d *EC2Profile) GetDistribution() string {
	return pkgCluster.EC2
}

// IsDefinedBefore returns true if database contains en entry with profile name
func (d *EC2Profile) IsDefinedBefore() bool {
	return database.GetDB().First(&d).RowsAffected != int64(0)
}

// AfterFind loads nodepools to profile
func (d *EC2Profile) AfterFind() error {
	log.Info("AfterFind ec2 profile... load node pools")
	return database.GetDB().Where(EC2NodePoolProfile{
		AmazonNodePoolProfileBaseFields: AmazonNodePoolProfileBaseFields{
			Name: d.Name,
		},
	}).Find(&d.NodePools).Error
}

// BeforeSave clears nodepools
func (d *EC2Profile) BeforeSave() error {
	log.Info("BeforeSave ec2 profile...")

	db := database.GetDB()
	var nodePools []*EC2NodePoolProfile
	err := db.Where(EC2NodePoolProfile{
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
func (d *EC2Profile) BeforeDelete() error {
	log.Info("BeforeDelete ec2 profile... delete all nodepool")

	var nodePools []*EC2NodePoolProfile
	return database.GetDB().Where(EC2NodePoolProfile{
		AmazonNodePoolProfileBaseFields: AmazonNodePoolProfileBaseFields{
			Name: d.Name,
		},
	}).Find(&nodePools).Delete(&nodePools).Error
}

// GetProfile load profile from database and converts ClusterProfileResponse
func (d *EC2Profile) GetProfile() *pkgCluster.ClusterProfileResponse {

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
			Alibaba *alibaba.ClusterProfileAlibaba `json:"alibaba,omitempty"`
			EC2     *pkgAmazon.ClusterProfileEC2   `json:"ec2,omitempty"`
			EKS     *eks.ClusterProfileEKS         `json:"eks,omitempty"`
			AKS     *pkgAzure.ClusterProfileAKS    `json:"aks,omitempty"`
			GKE     *pkgGoogle.ClusterProfileGKE   `json:"gke,omitempty"`
			OKE     *oracle.Cluster                `json:"oracle,omitempty"`
		}{
			EC2: &pkgAmazon.ClusterProfileEC2{
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
func (d *EC2Profile) UpdateProfile(r *pkgCluster.ClusterProfileRequest, withSave bool) error {

	if len(r.Location) != 0 {
		d.Location = r.Location
	}

	if r.Properties.EC2 != nil {

		if len(r.Properties.EC2.NodePools) != 0 {
			var nodePools []*EC2NodePoolProfile
			for npName, nodePool := range r.Properties.EC2.NodePools {

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

				nodePools = append(nodePools, &EC2NodePoolProfile{
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

		if r.Properties.EC2.Master != nil {
			if len(r.Properties.EC2.Master.InstanceType) != 0 {
				d.MasterInstanceType = r.Properties.EC2.Master.InstanceType
			}

			if len(r.Properties.EC2.Master.Image) != 0 {
				d.MasterImage = r.Properties.EC2.Master.Image
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
func (d *EC2Profile) DeleteProfile() error {
	return database.GetDB().Delete(&d).Error
}
