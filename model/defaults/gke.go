package defaults

import (
	"github.com/banzaicloud/banzai-types/constants"
	"github.com/banzaicloud/pipeline/model"
	"github.com/banzaicloud/banzai-types/components"
	"github.com/banzaicloud/banzai-types/components/google"
	"github.com/banzaicloud/banzai-types/components/azure"
	"github.com/banzaicloud/banzai-types/components/amazon"
)

type GKEProfile struct {
	DefaultModel
	Location         string `gorm:"default:'us-central1-a'"`
	NodeInstanceType string `gorm:"default:'n1-standard-1'"`
	NodeCount        int    `gorm:"default:1"`
	NodeVersion      string `gorm:"default:'1.7.12-gke.1'"`
	MasterVersion    string `gorm:"default:'1.7.12-gke.1'"`
}

func (*GKEProfile) TableName() string {
	return defaultGoogleProfileTablaName
}

func (d *GKEProfile) SaveInstance() error {
	return save(d)
}

func (d *GKEProfile) IsDefinedBefore() bool {
	return model.GetDB().First(&d).RowsAffected != int64(0)
}

func (d *GKEProfile) GetType() string {
	return constants.Google
}

func (d *GKEProfile) GetProfile() *components.ClusterProfileRespone {
	loadFirst(&d)

	return &components.ClusterProfileRespone{
		ProfileName:      d.DefaultModel.Name,
		Location:         d.Location,
		Cloud:            constants.Google,
		NodeInstanceType: d.NodeInstanceType,
		Properties: struct {
			Amazon *amazon.ClusterProfileAmazon `json:"amazon,omitempty"`
			Azure  *azure.ClusterProfileAzure   `json:"azure,omitempty"`
			Google *google.ClusterProfileGoogle `json:"google,omitempty"`
		}{
			Google: &google.ClusterProfileGoogle{
				Node: &google.GoogleNode{
					Count:   d.NodeCount,
					Version: d.NodeVersion,
				},
				Master: &google.GoogleMaster{
					Version: d.MasterVersion,
				},
			},
		},
	}
}

func (d *GKEProfile) UpdateProfile(r *components.ClusterProfileRequest) error {
	d.Location = r.Location
	d.NodeInstanceType = r.NodeInstanceType
	d.NodeCount = r.Properties.Google.Node.Count
	d.NodeVersion = r.Properties.Google.Node.Version
	d.MasterVersion = r.Properties.Google.Master.Version
	return d.SaveInstance()
}
