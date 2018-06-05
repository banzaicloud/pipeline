package model

import "github.com/jinzhu/gorm"

// ApplicationModel for Application
type ApplicationModel struct {
	gorm.Model
	Name           string `json:"name"`
	CatalogName    string `json:"catalogName"`
	CatalogVersion string `json:"catalogVersion"`
	Description    string `json:"description"`
	Icon           string `json:"icon"`
	OrganizationId uint   `json:"organizationId"`
	ClusterID      uint
	Deployments    []*Deployment `gorm:"foreignkey:application_id" json:"deployments"`
	Resources      string        `json:"resources"`
	Status         string        `json:"status"`
}

// Deployment for ApplicationModel
type Deployment struct {
	gorm.Model
	Name          string `json:"name"`
	Chart         string `json:"chart"`
	ReleaseName   string `json:"release_name"`
	Values        string `json:"values"`
	Status        string `json:"status"`
	WaitFor       string `json:"waitFor"`
	ApplicationID uint   `json:"applicationId"`
}

//Update Deployment
func (d *Deployment) Update(state string) error {
	return GetDB().Model(d).Update("status", state).Error
}

// Create Deployment
func (d *Deployment) Create() error {
	return GetDB().Create(d).Error
}

// GetCluster ApplicationModel
func (am ApplicationModel) GetCluster() ClusterModel {
	db := GetDB()
	var cluster ClusterModel
	db.First(&cluster, am.ClusterID)
	return cluster
}

//Save ApplicationModel the cluster to DB
func (am *ApplicationModel) Save() error {
	return GetDB().Save(&am).Error
}
