// The DummyCluster mocks create/update/delete functions. For testing and UI mocks.
package cluster

import (
	"github.com/banzaicloud/banzai-types/components"
	"github.com/banzaicloud/banzai-types/constants"
	"github.com/banzaicloud/pipeline/model"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

// DummyCluster struct for DC
type DummyCluster struct {
	modelCluster *model.ClusterModel
	k8sConfig    []byte
	APIEndpoint  string
}

// CreateDummyClusterFromRequest creates ClusterModel struct from the request
func CreateDummyClusterFromRequest(request *components.CreateClusterRequest, orgId uint) (*DummyCluster, error) {
	log := logger.WithFields(logrus.Fields{"action": constants.TagCreateCluster})
	log.Debug("Create ClusterModel struct from the request")
	var cluster DummyCluster

	cluster.modelCluster = &model.ClusterModel{
		Name:           request.Name,
		Location:       request.Location,
		Cloud:          request.Cloud,
		OrganizationId: orgId,
		SecretId:       request.SecretId,
		Dummy: model.DummyClusterModel{
			KubernetesVersion: request.Properties.CreateClusterDummy.Node.KubernetesVersion,
			NodeCount:         request.Properties.CreateClusterDummy.Node.Count,
		},
	}
	return &cluster, nil
}

//CreateCluster creates a new cluster
func (d *DummyCluster) CreateCluster() error {
	return nil
}

//Persist save the cluster model
func (d *DummyCluster) Persist(status string) error {
	log.Infof("Model before save: %v", d.modelCluster)
	return d.modelCluster.UpdateStatus(status)
}

//GetK8sConfig returns the Kubernetes config
func (d *DummyCluster) GetK8sConfig() ([]byte, error) {
	data, err := yaml.Marshal(createDummyConfig())
	if err != nil {
		return nil, err
	}
	d.k8sConfig = data
	return data, nil
}

//GetName returns the name of the cluster
func (d *DummyCluster) GetName() string {
	return d.modelCluster.Name
}

//GetType returns the cloud type of the cluster
func (d *DummyCluster) GetType() string {
	return constants.Dummy
}

//GetStatus gets cluster status
func (d *DummyCluster) GetStatus() (*components.GetClusterStatusResponse, error) {
	return &components.GetClusterStatusResponse{
		Status:     d.modelCluster.Status,
		Name:       d.modelCluster.Name,
		Location:   d.modelCluster.Location,
		Cloud:      constants.Dummy,
		ResourceID: d.GetID(),
		NodePools:  nil,
	}, nil
}

// DeleteCluster deletes cluster
func (d *DummyCluster) DeleteCluster() error {
	return nil
}

// UpdateCluster updates the dummy cluster
func (d *DummyCluster) UpdateCluster(r *components.UpdateClusterRequest) error {
	d.modelCluster.Dummy.KubernetesVersion = r.Dummy.Node.KubernetesVersion
	d.modelCluster.Dummy.NodeCount = r.Dummy.Node.Count
	return nil
}

//GetID returns the specified cluster id
func (d *DummyCluster) GetID() uint {
	return d.modelCluster.ID
}

//GetModel returns the whole clusterModel
func (d *DummyCluster) GetModel() *model.ClusterModel {
	return d.modelCluster
}

//CheckEqualityToUpdate validates the update request
func (d *DummyCluster) CheckEqualityToUpdate(r *components.UpdateClusterRequest) error {
	return nil
}

func (d *DummyCluster) AddDefaultsToUpdate(r *components.UpdateClusterRequest) {

}

//GetAPIEndpoint returns the Kubernetes Api endpoint
func (d *DummyCluster) GetAPIEndpoint() (string, error) {
	d.APIEndpoint = "http://cow.org:8080"
	return d.APIEndpoint, nil
}

//DeleteFromDatabase deletes model from the database
func (d *DummyCluster) DeleteFromDatabase() error {
	return d.modelCluster.Delete()
}

//GetOrg gets org where the cluster belongs
func (d *DummyCluster) GetOrg() uint {
	return d.modelCluster.OrganizationId
}

//GetSecretID retrieves the secretid
func (d *DummyCluster) GetSecretID() string {
	return d.modelCluster.SecretId
}

func createDummyConfig() *kubeConfig {
	return &kubeConfig{
		APIVersion: "v1",
		Clusters: []configCluster{
			{
				Cluster: dataCluster{
					Server: "http://cow.org:8080",
				},
				Name: "cow-cluster",
			}, {
				Cluster: dataCluster{
					Server: "https://horse.org:4443",
				},
				Name: "horse-cluster",
			},
			{
				Cluster: dataCluster{
					Server: "https://pig.org:443",
				},
				Name: "pig-cluster",
			},
		},
		Contexts: []configContext{
			{
				Context: contextData{
					Cluster: "horse-cluster",
					User:    "green-user",
				},
				Name: "federal-context",
			}, {
				Context: contextData{
					Cluster: "pig-cluster",
					User:    "black-user",
				},
				Name: "queen-anne-context",
			},
		},
		Users: []configUser{
			{
				Name: "blue-user",
				User: userData{
					Token: "blue-token",
				},
			},
			{
				Name: "green-user",
				User: userData{},
			},
		},
		CurrentContext: "federal-context",
		Kind:           "Config",
	}

}

//CreateDummyClusterFromModel creates the cluster from the model
func CreateDummyClusterFromModel(clusterModel *model.ClusterModel) (*DummyCluster, error) {
	log := logger.WithFields(logrus.Fields{"action": constants.TagGetCluster})
	log.Debug("Create ClusterModel struct from the request")
	dummyCluster := DummyCluster{
		modelCluster: clusterModel,
	}
	return &dummyCluster, nil
}

func (d *DummyCluster) UpdateStatus(status string) error {
	return d.modelCluster.UpdateStatus(status)
}

func (d *DummyCluster) GetClusterDetails() (*components.ClusterDetailsResponse, error) {
	status, err := d.GetStatus()
	if err != nil {
		return nil, err
	}

	return &components.ClusterDetailsResponse{
		Name: status.Name,
		Id:   status.ResourceID,
	}, nil
}

// ValidateCreationFields validates all field
func (d *DummyCluster) ValidateCreationFields(r *components.CreateClusterRequest) error {
	return nil
}
