package model

type CatalogModel struct {
	ID             uint          `gorm:"primary_key"`
	Name           string        `json:"name"`
	OrganizationId uint          `gorm:"unique_index:idx_unique_id"`
	Cluster        ClusterModel  `json:"cluster"`
	Deployments    []Deployments `json:"deployments"`
}

type Deployments struct {
	Name        string `json:"name"`
	State       string `json:"state"`
	ReleaseName string `json:"release_name"`
}

func QueryCatalog(filter map[string]interface{}) ([]CatalogModel, error) {
	var catalogs []CatalogModel
	err := db.Where(filter).Find(&catalogs).Error
	if err != nil {
		return nil, err
	}
	return catalogs, nil
}

//Save the cluster to DB
func (cm *CatalogModel) Save() error {
	db := GetDB()
	err := db.Save(&cm).Error
	if err != nil {
		return err
	}
	return nil
}
