package model

type ApplicationModel struct {
	ID             uint         `gorm:"primary_key"`
	Name           string       `json:"name"`
	OrganizationId uint         `gorm:"unique_index:idx_unique_id"`
	Cluster        ClusterModel `json:"cluster"`
	Deployments    []Deployment `json:"deployments"`
	Resources      string       `json:"resources"`
}

type Deployment struct {
	Name        string `json:"name"`
	Chart       string `json:"chart"`
	ReleaseName string `json:"release_name"`
	Values      string `json:"values"`
	Status      string `json:"status"`
	WaitFor     string `json:"wait_for"`
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
