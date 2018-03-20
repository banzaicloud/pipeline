package cluster

import (
	"encoding/base64"
	azureClient "github.com/banzaicloud/azure-aks-client/client"
	azureCluster "github.com/banzaicloud/azure-aks-client/cluster"
	"github.com/banzaicloud/banzai-types/components"
	bTypes "github.com/banzaicloud/banzai-types/components"
	banzaiAzureTypes "github.com/banzaicloud/banzai-types/components/azure"
	"github.com/banzaicloud/banzai-types/constants"
	"github.com/banzaicloud/pipeline/model"
	"github.com/banzaicloud/pipeline/utils"
	"github.com/go-errors/errors"
	"github.com/sirupsen/logrus"
	"net/http"
)

//CreateAKSClusterFromRequest creates ClusterModel struct from the request
func CreateAKSClusterFromRequest(request *components.CreateClusterRequest, orgId uint) (*AKSCluster, error) {
	log := logger.WithFields(logrus.Fields{"action": constants.TagCreateCluster})
	log.Debug("Create ClusterModel struct from the request")
	var cluster AKSCluster

	cluster.modelCluster = &model.ClusterModel{
		Name:             request.Name,
		Location:         request.Location,
		NodeInstanceType: request.NodeInstanceType,
		Cloud:            request.Cloud,
		OrganizationId:   orgId,
		Azure: model.AzureClusterModel{
			ResourceGroup:     request.Properties.CreateClusterAzure.Node.ResourceGroup,
			AgentCount:        request.Properties.CreateClusterAzure.Node.AgentCount,
			AgentName:         request.Properties.CreateClusterAzure.Node.AgentName,
			KubernetesVersion: request.Properties.CreateClusterAzure.Node.KubernetesVersion,
		},
	}
	return &cluster, nil
}

//AKSCluster struct for AKS cluster
type AKSCluster struct {
	azureCluster *banzaiAzureTypes.Value //Don't use this directly
	modelCluster *model.ClusterModel
	k8sConfig    *[]byte
	APIEndpoint  string
}

func (c *AKSCluster) GetOrg() uint {
	return c.modelCluster.OrganizationId
}

//GetAPIEndpoint returns the Kubernetes Api endpoint
func (c *AKSCluster) GetAPIEndpoint() (string, error) {
	if c.APIEndpoint != "" {
		return c.APIEndpoint, nil
	}
	cluster, err := c.GetAzureCluster()
	if err != nil {
		return "", err
	}
	c.APIEndpoint = cluster.Properties.Fqdn
	return c.APIEndpoint, nil
}

//CreateCluster creates a new cluster
func (c *AKSCluster) CreateCluster() error {

	log := logger.WithFields(logrus.Fields{"action": constants.TagCreateCluster})

	r := azureCluster.CreateClusterRequest{
		Name:              c.modelCluster.Name,
		Location:          c.modelCluster.Location,
		VMSize:            c.modelCluster.NodeInstanceType,
		ResourceGroup:     c.modelCluster.Azure.ResourceGroup,
		AgentCount:        c.modelCluster.Azure.AgentCount,
		AgentName:         c.modelCluster.Azure.AgentName,
		KubernetesVersion: c.modelCluster.Azure.KubernetesVersion,
	}
	client, err := azureClient.GetAKSClient(nil)
	if err != nil {
		return err
	}

	client.With(log.Logger)

	// call creation
	createdCluster, err := client.CreateUpdateCluster(r)
	if err != nil {
		// creation failed
		// todo status code!??
		return err
	}
	// creation success
	log.Info("Cluster created successfully!")

	c.azureCluster = &createdCluster.Value

	// save to database
	if err := c.Persist(); err != nil {
		log.Errorf("Cluster save failed! %s", err.Error())
	}

	// polling cluster
	pollingResult, err := client.PollingCluster(r.Name, r.ResourceGroup)
	if err != nil {
		// polling error
		// todo status code!??
		return err
	}
	log.Info("Cluster is ready...")
	c.azureCluster = &pollingResult.Value
	return nil
}

//Persist save the cluster model
func (c *AKSCluster) Persist() error {
	return c.modelCluster.Save()
}

//GetK8sConfig returns the Kubernetes config
func (c *AKSCluster) GetK8sConfig() (*[]byte, error) {
	if c.k8sConfig != nil {
		return c.k8sConfig, nil
	}
	client, err := azureClient.GetAKSClient(nil)
	if err != nil {
		return nil, err
	}

	client.With(log.Logger)

	database := model.GetDB()
	database.Where(model.AzureClusterModel{ClusterModelId: c.modelCluster.ID}).First(&c.modelCluster.Azure)
	//TODO check banzairesponses
	config, err := client.GetClusterConfig(c.modelCluster.Name, c.modelCluster.Azure.ResourceGroup, "clusterUser")
	if err != nil {
		// TODO status code !?
		return nil, err
	}
	log.Info("Get k8s config succeeded")
	decodedConfig, err := base64.StdEncoding.DecodeString(config.Properties.KubeConfig)
	if err != nil {
		return nil, err
	}
	c.k8sConfig = &decodedConfig
	return &decodedConfig, nil
}

//GetName returns the name of the cluster
func (c *AKSCluster) GetName() string {
	return c.modelCluster.Name
}

//GetType returns the cloud type of the cluster
func (c *AKSCluster) GetType() string {
	return c.modelCluster.Cloud
}

//GetStatus gets cluster status
func (c *AKSCluster) GetStatus() (*bTypes.GetClusterStatusResponse, error) {
	log := logger.WithFields(logrus.Fields{"action": constants.TagGetClusterStatus})
	client, err := azureClient.GetAKSClient(nil)
	if err != nil {
		return nil, err
	}

	client.With(log.Logger)

	resp, err := client.GetCluster(c.modelCluster.Name, c.modelCluster.Azure.ResourceGroup)
	if err != nil {
		return nil, errors.New(err)
	}
	log.Info("Get cluster success")
	stage := resp.Value.Properties.ProvisioningState
	log.Info("Cluster stage is", stage)
	if stage == "Succeeded" {
		response := &components.GetClusterStatusResponse{
			Status:           http.StatusOK,
			Name:             c.modelCluster.Name,
			Location:         c.modelCluster.Location,
			Cloud:            c.modelCluster.Cloud,
			NodeInstanceType: c.modelCluster.NodeInstanceType,
			ResourceID:       c.modelCluster.ID,
		}

		return response, nil
	}
	return nil, constants.ErrorClusterNotReady
}

// DeleteCluster deletes cluster from aks
func (c *AKSCluster) DeleteCluster() error {
	log := logger.WithFields(logrus.Fields{"action": constants.TagDeleteCluster})
	client, err := azureClient.GetAKSClient(nil)
	if err != nil {
		return err
	}

	client.With(log.Logger)

	// set azure props
	database := model.GetDB()
	database.Where(model.AzureClusterModel{ClusterModelId: c.modelCluster.ID}).First(&c.modelCluster.Azure)

	err = client.DeleteCluster(c.modelCluster.Name, c.modelCluster.Azure.ResourceGroup)
	if err != nil {
		log.Info("Delete succeeded")
		return nil
	}
	// todo status code !?
	return err
}

// UpdateCluster updates AKS cluster in cloud
func (c *AKSCluster) UpdateCluster(request *bTypes.UpdateClusterRequest) error {
	log := logger.WithFields(logrus.Fields{"action": constants.TagUpdateCluster})
	client, err := azureClient.GetAKSClient(nil)
	if err != nil {
		return err
	}

	client.With(log.Logger)

	ccr := azureCluster.CreateClusterRequest{
		Name:              c.modelCluster.Name,
		Location:          c.modelCluster.Location,
		VMSize:            c.modelCluster.NodeInstanceType,
		ResourceGroup:     c.modelCluster.Azure.ResourceGroup,
		AgentCount:        request.UpdateClusterAzure.AgentCount,
		AgentName:         c.modelCluster.Azure.AgentName,
		KubernetesVersion: c.modelCluster.Azure.KubernetesVersion,
	}

	updatedCluster, err := client.CreateUpdateCluster(ccr)
	if err != nil {
		return err
	}
	log.Info("Cluster update succeeded")
	//Update AWS model
	log.Info("Create updated model")
	updateCluster := &model.ClusterModel{
		Model:            c.modelCluster.Model,
		Name:             c.modelCluster.Name,
		Location:         c.modelCluster.Location,
		NodeInstanceType: c.modelCluster.NodeInstanceType,
		Cloud:            c.modelCluster.Cloud,
		Azure: model.AzureClusterModel{
			ResourceGroup:     c.modelCluster.Azure.ResourceGroup,
			AgentCount:        request.UpdateClusterAzure.AgentCount,
			AgentName:         c.modelCluster.Azure.AgentName,
			KubernetesVersion: c.modelCluster.Azure.KubernetesVersion,
		},
	}
	c.modelCluster = updateCluster
	c.azureCluster = &updatedCluster.Value
	return nil
}

//GetID returns the specified cluster id
func (c *AKSCluster) GetID() uint {
	return c.modelCluster.ID
}

//GetModel returns the whole clusterModel
func (c *AKSCluster) GetModel() *model.ClusterModel {
	return c.modelCluster
}

func (c *AKSCluster) GetAzureCluster() (*banzaiAzureTypes.Value, error) {
	client, err := azureClient.GetAKSClient(nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.GetCluster(c.modelCluster.Name, c.modelCluster.Azure.ResourceGroup)
	if err != nil {
		return nil, err
	}
	c.azureCluster = &resp.Value
	return c.azureCluster, nil
}

//CreateAKSClusterFromModel creates ClusterModel struct from model
func CreateAKSClusterFromModel(clusterModel *model.ClusterModel) (*AKSCluster, error) {
	log := logger.WithFields(logrus.Fields{"action": constants.TagGetCluster})
	log.Debug("Create ClusterModel struct from the request")
	aksCluster := AKSCluster{
		modelCluster: clusterModel,
	}
	return &aksCluster, nil
}

//AddDefaultsToUpdate adds defaults to update request
func (c *AKSCluster) AddDefaultsToUpdate(r *components.UpdateClusterRequest) {

	if r.UpdateClusterAzure == nil {
		log.Info("'azure' field is empty.")
		r.UpdateClusterAzure = &banzaiAzureTypes.UpdateClusterAzure{}
	}

	// ---- [ Node check ] ---- //
	if r.UpdateAzureNode == nil {
		log.Info("'node' field is empty. Load it from stored data.")
		r.UpdateAzureNode = &banzaiAzureTypes.UpdateAzureNode{
			AgentCount: c.modelCluster.Azure.AgentCount,
		}
	}

	// ---- [ Node - Agent count check] ---- //
	if r.AgentCount == 0 {
		def := c.modelCluster.Azure.AgentCount
		log.Info("Node agentCount set to default value: ", def)
		r.AgentCount = def
	}

}

//CheckEqualityToUpdate validates the update request
func (c *AKSCluster) CheckEqualityToUpdate(r *components.UpdateClusterRequest) error {
	// create update request struct with the stored data to check equality
	preCl := &banzaiAzureTypes.UpdateClusterAzure{
		UpdateAzureNode: &banzaiAzureTypes.UpdateAzureNode{
			AgentCount: c.modelCluster.Azure.AgentCount,
		},
	}

	log.Info("Check stored & updated cluster equals")

	// check equality
	return utils.IsDifferent(r.UpdateClusterAzure, preCl)
}

//DeleteFromDatabase deletes model from the database
func (c *AKSCluster) DeleteFromDatabase() error {
	err := c.modelCluster.Delete()
	if err != nil {
		return err
	}
	c.modelCluster = nil
	return nil
}
