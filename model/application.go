package model

import (
	"time"

	"github.com/banzaicloud/pipeline/config"
	"github.com/banzaicloud/pipeline/database"
)

var log = config.Logger()

// Application for Application
type Application struct {
	ID             uint          `json:"id" gorm:"primary_key"`
	CreatedAt      time.Time     `json:"createdAt"`
	UpdatedAt      time.Time     `json:"updatedAt"`
	DeletedAt      *time.Time    `json:"-" sql:"index"`
	Name           string        `json:"name"`
	CatalogName    string        `json:"catalogName"`
	CatalogVersion string        `json:"catalogVersion"`
	Description    string        `json:"description"`
	Icon           string        `json:"icon"`
	OrganizationId uint          `json:"organizationId"`
	ClusterID      uint          `json:"clusterId"`
	Deployments    []*Deployment `gorm:"foreignkey:application_id" json:"deployments"`
	Resources      string        `json:"resources"`
	Status         string        `json:"status"`
	Message        string        `json:"message"`
	CreatedBy      uint          `json:"creatorId"`
}

// Deployment for Application
type Deployment struct {
	ID            uint       `json:"id" gorm:"primary_key"`
	CreatedAt     time.Time  `json:"createdAt"`
	UpdatedAt     time.Time  `json:"updatedAt"`
	DeletedAt     *time.Time `json:"-" sql:"index"`
	Name          string     `json:"name"`
	Chart         string     `json:"chart"`
	ReleaseName   string     `json:"releaseName"`
	Values        string     `json:"values"`
	Status        string     `json:"status"`
	Message       string     `json:"message"`
	ApplicationID uint       `json:"applicationId"`
}

//Update Deployment
func (d *Deployment) Update(update Deployment) error {
	err := database.GetDB().Model(d).Update(update).Error
	if err != nil {
		log.Error(err)
	}
	return err
}

// Create Deployment
func (d *Deployment) Create() error {
	err := database.GetDB().Create(d).Error
	if err != nil {
		log.Error(err)
	}
	return err
}

// GetCluster Application
func (am Application) GetCluster() (*ClusterModel, error) {
	db := database.GetDB()
	var cluster ClusterModel
	err := db.First(&cluster, am.ClusterID).Error
	return &cluster, err
}

//Save Application the cluster to DB
func (am *Application) Save() error {
	err := database.GetDB().Save(&am).Error
	if err != nil {
		log.Error(err)
	}
	return err
}

// Delete deletes application from the DB
func (am *Application) Delete() error {
	err := database.GetDB().Delete(&am).Error
	if err != nil {
		log.Error(err)
	}
	return err
}

// Update update fields for Application
func (am *Application) Update(update Application) error {
	err := database.GetDB().Model(am).Update(update).Error
	if err != nil {
		log.Error(err)
	}
	return err
}
