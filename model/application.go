package model

import "github.com/jinzhu/gorm"

type ApplicationModel struct {
	gorm.Model
	Name           string `json:"name"`
	CatalogName    string `json:"catalogName"`
	CatalogVersion string `json:"catalogVersion"`
	Description    string `json:"description"`
	Icon           string `json:"icon"`
	OrganizationId uint   `json:"organizationId"`
	ClusterID      uint
	Deployments    []*Deployment `gorm:"foreignkey:ApplicationID" json:"deployments"`
	Resources      string        `json:"resources"`
	Status         string        `json:"status"`
}

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

func (d *Deployment) Update(state string) error {
	return GetDB().Model(d).Update("status", state).Error
}

func (d *Deployment) Create() error {
	return GetDB().Create(d).Error
}

func (am ApplicationModel) GetCluster() ClusterModel {
	db := GetDB()
	var cluster ClusterModel
	db.First(&cluster, am.ClusterID)
	return cluster
}

func QueryCatalog(filter map[string]interface{}) ([]ApplicationModel, error) {
	var catalogs []ApplicationModel
	err := db.Where(filter).Find(&catalogs).Error
	if err != nil {
		return nil, err
	}
	return catalogs, nil
}

//Save the cluster to DB
func (cm *ApplicationModel) Save() error {
	return GetDB().Save(&cm).Error
}

//
//resources:
//- vcpu: 32
//memory: 32G
//filter:
//- gpu
//- ena
//- boostable
//- high-iops
//# Szazalekosan hany szazaleka legyen on-demand
//on_demand_percentage: 50%
//# hasonlo meretu instance tipusokat ajanljon.
//same_size: true

//depends:
//- monitor:
//type: crd
//values:
//- prometheuses.monitoring.coreos.com
//- servicemonitors.monitoring.coreos.com
//- alertmanagers.monitoring.coreos.com
//namespace: monitoring
//charts:
//- name: pipeline-cluster-monitor
//repository: alias:banzaicloud-stable
//version: 0.0.1
//- logging:
//type: chart
//namespace: default
//charts:
//- name: pipeline-cluster-log
//repository: alias:banzaicloud-stable
//version: 0.0.1
