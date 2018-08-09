package defaults

import (
	"github.com/banzaicloud/pipeline/database"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	"github.com/banzaicloud/pipeline/pkg/cluster/accs"
	"github.com/banzaicloud/pipeline/pkg/cluster/aks"
	"github.com/banzaicloud/pipeline/pkg/cluster/ec2"
	"github.com/banzaicloud/pipeline/pkg/cluster/eks"
	"github.com/banzaicloud/pipeline/pkg/cluster/gke"
	oracle "github.com/banzaicloud/pipeline/pkg/providers/oracle/cluster"
)

// AKSProfile describes an Azure cluster profile
type AKSProfile struct {
	DefaultModel
	Location          string                `gorm:"default:'eastus'"`
	KubernetesVersion string                `gorm:"default:'1.9.2'"`
	NodePools         []*AKSNodePoolProfile `gorm:"foreignkey:Name"`
}

// AKSNodePoolProfile describes an Azure cluster profile's nodepools
type AKSNodePoolProfile struct {
	ID               uint   `gorm:"primary_key"`
	Autoscaling      bool   `gorm:"default:false"`
	MinCount         int    `gorm:"default:1"`
	MaxCount         int    `gorm:"default:2"`
	Count            int    `gorm:"default:1"`
	NodeInstanceType string `gorm:"default:'Standard_D4_v2'"`
	Name             string `gorm:"unique_index:idx_model_name"`
	NodeName         string `gorm:"unique_index:idx_model_name"`
}

// TableName overrides AKSNodePoolProfile's table name
func (AKSNodePoolProfile) TableName() string {
	return DefaultAKSNodePoolProfileTableName
}

// TableName overrides AKSProfile's table name
func (AKSProfile) TableName() string {
	return DefaultAKSProfileTableName
}

// AfterFind loads nodepools to profile
func (d *AKSProfile) AfterFind() error {
	log.Info("AfterFind aks profile... load node pools")
	return database.GetDB().Where(AKSNodePoolProfile{Name: d.Name}).Find(&d.NodePools).Error
}

// BeforeSave clears nodepools
func (d *AKSProfile) BeforeSave() error {
	log.Info("BeforeSave aks profile...")

	db := database.GetDB()
	var nodePools []*AKSNodePoolProfile
	err := db.Where(AKSNodePoolProfile{
		Name: d.Name,
	}).Find(&nodePools).Delete(&nodePools).Error
	if err != nil {
		log.Errorf("Error during deleting saved nodepools: %s", err.Error())
	}

	return nil
}

// BeforeDelete deletes all nodepools to belongs to profile
func (d *AKSProfile) BeforeDelete() error {
	log.Info("BeforeDelete aks profile... delete all nodepool")

	var nodePools []*AKSNodePoolProfile
	return database.GetDB().Where(AKSNodePoolProfile{
		Name: d.Name,
	}).Find(&nodePools).Delete(&nodePools).Error
}

// SaveInstance saves cluster profile into database
func (d *AKSProfile) SaveInstance() error {
	return save(d)
}

// IsDefinedBefore returns true if database contains en entry with profile name
func (d *AKSProfile) IsDefinedBefore() bool {
	return database.GetDB().First(&d).RowsAffected != int64(0)
}

// GetCloud returns profile's cloud type
func (d *AKSProfile) GetCloud() string {
	return pkgCluster.Azure
}

// GetDistribution returns profile's distribution type
func (d *AKSProfile) GetDistribution() string {
	return pkgCluster.AKS
}

// GetProfile load profile from database and converts ClusterProfileResponse
func (d *AKSProfile) GetProfile() *pkgCluster.ClusterProfileResponse {

	nodePools := make(map[string]*aks.NodePoolCreate)
	for _, np := range d.NodePools {
		if np != nil {
			nodePools[np.NodeName] = &aks.NodePoolCreate{
				Autoscaling:      np.Autoscaling,
				MinCount:         np.MinCount,
				MaxCount:         np.MaxCount,
				Count:            np.Count,
				NodeInstanceType: np.NodeInstanceType,
			}
		}
	}

	return &pkgCluster.ClusterProfileResponse{
		Name:     d.DefaultModel.Name,
		Location: d.Location,
		Cloud:    pkgCluster.Azure,
		Properties: struct {
			ACCS *accs.ClusterProfileACCS `json:"accs,omitempty"`
			EC2  *ec2.ClusterProfileEC2   `json:"ec2,omitempty"`
			EKS  *eks.ClusterProfileEKS   `json:"eks,omitempty"`
			AKS  *aks.ClusterProfileAKS   `json:"aks,omitempty"`
			GKE  *gke.ClusterProfileGKE   `json:"gke,omitempty"`
			OKE  *oracle.Cluster          `json:"oracle,omitempty"`
		}{
			AKS: &aks.ClusterProfileAKS{
				KubernetesVersion: d.KubernetesVersion,
				NodePools:         nodePools,
			},
		},
	}
}

// UpdateProfile update profile's data with ClusterProfileRequest's data and if bool is true then update in the database
func (d *AKSProfile) UpdateProfile(r *pkgCluster.ClusterProfileRequest, withSave bool) error {
	if len(r.Location) != 0 {
		d.Location = r.Location
	}

	if r.Properties.AKS != nil {

		if len(r.Properties.AKS.KubernetesVersion) != 0 {
			d.KubernetesVersion = r.Properties.AKS.KubernetesVersion
		}

		if len(r.Properties.AKS.NodePools) != 0 {

			var nodePools []*AKSNodePoolProfile
			for name, np := range r.Properties.AKS.NodePools {
				nodePools = append(nodePools, &AKSNodePoolProfile{
					Autoscaling:      np.Autoscaling,
					MinCount:         np.MinCount,
					MaxCount:         np.MaxCount,
					Count:            np.Count,
					NodeInstanceType: np.NodeInstanceType,
					Name:             d.Name,
					NodeName:         name,
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
func (d *AKSProfile) DeleteProfile() error {
	return database.GetDB().Delete(&d).Error
}
