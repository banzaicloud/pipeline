package cluster

import (
	"github.com/banzaicloud/banzai-types/components"
	"github.com/sirupsen/logrus"
	"github.com/banzaicloud/banzai-types/constants"
	"github.com/banzaicloud/pipeline/model"
	"net/http"
	"gopkg.in/yaml.v2"
)

type DummyCluster struct {
	modelCluster *model.ClusterModel
	k8sConfig    *[]byte
	APIEndpoint  string
}

// CreateDummyClusterFromRequest creates ClusterModel struct from the request
func CreateDummyClusterFromRequest(request *components.CreateClusterRequest) (*DummyCluster, error) {
	log := logger.WithFields(logrus.Fields{"action": constants.TagCreateCluster})
	log.Debug("Create ClusterModel struct from the request")
	var cluster DummyCluster

	cluster.modelCluster = &model.ClusterModel{
		Name:             request.Name,
		Location:         request.Location,
		NodeInstanceType: request.NodeInstanceType,
		Cloud:            request.Cloud,
		Dummy: model.DummyClusterModel{
			KubernetesVersion: request.Properties.CreateClusterDummy.Node.KubernetesVersion,
			NodeCount:         request.Properties.CreateClusterDummy.Node.Count,
		},
	}
	return &cluster, nil
}

func (d *DummyCluster) CreateCluster() error {
	return nil // todo impl
}

func (d *DummyCluster) Persist() error {
	log.Infof("Model before save: %v", d.modelCluster)
	return d.modelCluster.Save()
}

func (d *DummyCluster) GetK8sConfig() (*[]byte, error) {
	//return nil, errors.New("")
	data, err := yaml.Marshal(createDummyConfig())
	if err != nil {
		return nil, err
	}
	d.k8sConfig = &data
	return &data, nil
}

func (d *DummyCluster) GetName() string {
	return d.modelCluster.Name
}

func (d *DummyCluster) GetType() string {
	return constants.Dummy
}

func (d *DummyCluster) GetStatus() (*components.GetClusterStatusResponse, error) {
	return &components.GetClusterStatusResponse{
		Status:           http.StatusOK,
		Name:             d.modelCluster.Name,
		Location:         d.modelCluster.Location,
		Cloud:            constants.Dummy,
		NodeInstanceType: d.modelCluster.NodeInstanceType,
		ResourceID:       d.GetID(),
	}, nil
}

func (d *DummyCluster) DeleteCluster() error {
	return nil // todo impl
}

func (d *DummyCluster) UpdateCluster(r *components.UpdateClusterRequest) error {
	return nil // todo impl
}

func (d *DummyCluster) GetID() uint {
	return d.modelCluster.ID
}

func (d *DummyCluster) GetModel() *model.ClusterModel {
	return d.modelCluster
}

func (d *DummyCluster) CheckEqualityToUpdate(r *components.UpdateClusterRequest) error {
	return nil // todo impl
}

func (d *DummyCluster) AddDefaultsToUpdate(r *components.UpdateClusterRequest) {
	// todo impl
}

func (d *DummyCluster) GetAPIEndpoint() (string, error) {
	return "http://cow.org:8080", nil
}

func (d *DummyCluster) DeleteFromDatabase() error {
	return d.modelCluster.Delete()
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
				User: userData{
				},
			},
		},
		CurrentContext: "federal-context",
		Kind:           "Config",
	}

}

func CreateDummyClusterFromModel(clusterModel *model.ClusterModel) (*DummyCluster, error) {
	log := logger.WithFields(logrus.Fields{"action": constants.TagGetCluster})
	log.Debug("Create ClusterModel struct from the request")
	dummyCluster := DummyCluster{
		modelCluster: clusterModel,
	}
	return &dummyCluster, nil
}
