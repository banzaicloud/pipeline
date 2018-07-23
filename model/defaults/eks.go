package defaults

import (
	"github.com/banzaicloud/pipeline/database"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	"github.com/banzaicloud/pipeline/pkg/cluster/amazon"
	"github.com/banzaicloud/pipeline/pkg/cluster/azure"
	"github.com/banzaicloud/pipeline/pkg/cluster/eks"
	"github.com/banzaicloud/pipeline/pkg/cluster/google"
	oracle "github.com/banzaicloud/pipeline/pkg/providers/oracle/cluster"
)

// EKSProfile describes an Amazon EKS cluster profile
type EKSProfile struct {
	DefaultModel
	Region           string `gorm:"default:'us-west-2'"`
	NodeImageId      string `json:"nodeImageId,omitempty"`
	NodeInstanceType string `json:"nodeInstanceType,omitempty"`
	Version          string `json:"version,omitempty"`
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
	return pkgCluster.Amazon
}

// IsDefinedBefore returns true if database contains en entry with profile name
func (d *EKSProfile) IsDefinedBefore() bool {
	return database.GetDB().First(&d).RowsAffected != int64(0)
}

// GetProfile load profile from database and converts ClusterProfileResponse
func (d *EKSProfile) GetProfile() *pkgCluster.ClusterProfileResponse {

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
				NodeImageId:      d.NodeImageId,
				NodeInstanceType: d.NodeInstanceType,
				Version:          d.Version,
			},
		},
	}

}

// UpdateProfile update profile's data with ClusterProfileRequest's data and if bool is true then update in the database
func (d *EKSProfile) UpdateProfile(r *pkgCluster.ClusterProfileRequest, withSave bool) error {

	//if len(r.Location) != 0 {
	//	d.Location = r.Location
	//}

	if r.Properties.Eks != nil {

		//TODO missing update body
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
