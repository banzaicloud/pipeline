package defaults

import (
	"github.com/banzaicloud/banzai-types/constants"
	"github.com/banzaicloud/pipeline/model"
	"github.com/banzaicloud/banzai-types/components"
	"github.com/banzaicloud/banzai-types/components/amazon"
	"github.com/banzaicloud/banzai-types/components/azure"
	"github.com/banzaicloud/banzai-types/components/google"
)

// todo maybe this could be private
type DefaultAWS struct {
	DefaultModel
	Location           string `gorm:"default:'eu-west-1'"`
	NodeInstanceType   string `gorm:"default:'m4.xlarge'"`
	NodeImage          string `gorm:"default:'ami-06d1667f'"`
	MasterInstanceType string `gorm:"default:'m4.xlarge'"`
	MasterImage        string `gorm:"default:'ami-06d1667f'"`
	NodeSpotPrice      string `gorm:"default:'0.2'"`
	NodeMinCount       int    `gorm:"default:1"`
	NodeMaxCount       int    `gorm:"default:2"`
}

func (*DefaultAWS) TableName() string {
	return "amazon_default_profile"
}

func (d *DefaultAWS) SaveDefaultInstance() error {
	return save(d)
}

func (d *DefaultAWS) GetType() string {
	return constants.Amazon
}

func (d *DefaultAWS) IsDefinedBefore() bool {
	database := model.GetDB()
	database.First(&d)
	return d.ID != 0
}

func (d *DefaultAWS) GetDefaultProfile() *components.CreateClusterRequest {
	loadFirst(&d)

	return &components.CreateClusterRequest{
		Name:             "", // todo good?
		Location:         d.Location,
		Cloud:            constants.Amazon,
		NodeInstanceType: d.NodeInstanceType,
		Properties: struct {
			CreateClusterAmazon *amazon.CreateClusterAmazon `json:"amazon,omitempty"`
			CreateClusterAzure  *azure.CreateClusterAzure   `json:"azure,omitempty"`
			CreateClusterGoogle *google.CreateClusterGoogle `json:"google,omitempty"`
		}{
			CreateClusterAmazon: &amazon.CreateClusterAmazon{
				Node: &amazon.CreateAmazonNode{
					SpotPrice: d.NodeSpotPrice,
					MinCount:  d.NodeMinCount,
					MaxCount:  d.NodeMaxCount,
					Image:     d.NodeImage,
				},
				Master: &amazon.CreateAmazonMaster{
					InstanceType: d.MasterInstanceType,
					Image:        d.MasterImage,
				},
			},
		},
	}

}
