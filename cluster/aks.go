package cluster

import (
	azureClient "github.com/banzaicloud/azure-aks-client/client"
	azureCluster "github.com/banzaicloud/azure-aks-client/cluster"
	"github.com/banzaicloud/banzai-types/components"
	bTypes "github.com/banzaicloud/banzai-types/components"
	banzaiAzureTypes "github.com/banzaicloud/banzai-types/components/azure"
	"github.com/banzaicloud/banzai-types/constants"
	"github.com/banzaicloud/pipeline/model"
	"github.com/banzaicloud/pipeline/secret"
	"github.com/banzaicloud/pipeline/utils"
	"github.com/go-errors/errors"
	"github.com/sirupsen/logrus"
	"github.com/Azure/azure-sdk-for-go/services/containerservice/mgmt/2017-09-30/containerservice"
)

//CreateAKSClusterFromRequest creates ClusterModel struct from the request
func CreateAKSClusterFromRequest(request *components.CreateClusterRequest, orgId uint) (*AKSCluster, error) {
	log := logger.WithFields(logrus.Fields{"action": constants.TagCreateCluster})
	log.Debug("Create ClusterModel struct from the request")
	var cluster AKSCluster

	var nodePools []*model.AzureNodePoolModel
	if request.Properties.CreateClusterAzure.NodePools != nil {
		for name, np := range *request.Properties.CreateClusterAzure.NodePools {
			nodePools = append(nodePools, &model.AzureNodePoolModel{
				Name:      name,
				NodeCount: np.AgentCount,
				VmSize:    np.VmSize,
			})
		}
	}

	cluster.modelCluster = &model.ClusterModel{
		Name:             request.Name,
		Location:         request.Location,
		NodeInstanceType: request.NodeInstanceType, // todo make it optional
		Cloud:            request.Cloud,
		OrganizationId:   orgId,
		SecretId:         request.SecretId,
		Azure: model.AzureClusterModel{
			ResourceGroup:     request.Properties.CreateClusterAzure.ResourceGroup,
			KubernetesVersion: request.Properties.CreateClusterAzure.KubernetesVersion,
			NodePools:         nodePools, // todo profiles
		},
	}
	return &cluster, nil
}

//AKSCluster struct for AKS cluster
type AKSCluster struct {
	azureCluster *banzaiAzureTypes.Value //Don't use this directly
	modelCluster *model.ClusterModel
	k8sConfig    []byte
	APIEndpoint  string
}

func (c *AKSCluster) GetOrg() uint {
	return c.modelCluster.OrganizationId
}

func (c *AKSCluster) GetAKSClient() (*azureClient.AKSClient, error) {
	clusterSecret, err := GetSecret(c)
	if err != nil {
		return nil, err
	}
	if clusterSecret.SecretType != secret.Azure {
		return nil, errors.Errorf("missmatch secret type %s versus %s", clusterSecret.SecretType, secret.Azure)
	}
	creds := &azureCluster.AKSCredential{
		ClientId:       clusterSecret.Values[secret.AzureClientId],
		ClientSecret:   clusterSecret.Values[secret.AzureClientSecret],
		SubscriptionId: clusterSecret.Values[secret.AzureSubscriptionId],
		TenantId:       clusterSecret.Values[secret.AzureTenantId],
	}
	client, err := azureClient.GetAKSClient(creds)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func (c *AKSCluster) GetSecretID() string {
	return c.modelCluster.SecretId
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

	var profiles []containerservice.AgentPoolProfile
	if nodePools := c.modelCluster.Azure.NodePools; nodePools != nil {
		for _, np := range nodePools {
			if np != nil {
				count := int32(np.NodeCount)
				name := np.Name
				profiles = append(profiles, containerservice.AgentPoolProfile{
					Name:   &name,
					Count:  &count,
					VMSize: containerservice.VMSizeTypes(np.VmSize),
				})
			}
		}
	}

	r := azureCluster.CreateClusterRequest{
		Name:              c.modelCluster.Name,
		Location:          c.modelCluster.Location,
		ResourceGroup:     c.modelCluster.Azure.ResourceGroup,
		KubernetesVersion: c.modelCluster.Azure.KubernetesVersion,
		Profiles:          profiles,
	}
	client, err := c.GetAKSClient()
	if err != nil {
		return err
	}

	client.With(log.Logger)

	// call creation
	createdCluster, err := azureClient.CreateUpdateCluster(client, &r)
	if err != nil {
		// creation failed
		// todo status code!??
		return err
	}
	// creation success
	log.Info("Cluster created successfully!")

	c.azureCluster = &createdCluster.Value

	// polling cluster
	pollingResult, err := azureClient.PollingCluster(client, r.Name, r.ResourceGroup)
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
func (c *AKSCluster) Persist(status string) error {
	return c.modelCluster.UpdateStatus(status)
}

//GetK8sConfig returns the Kubernetes config
func (c *AKSCluster) GetK8sConfig() ([]byte, error) {
	if c.k8sConfig != nil {
		return c.k8sConfig, nil
	}
	client, err := c.GetAKSClient()
	if err != nil {
		return nil, err
	}

	client.With(log.Logger)

	database := model.GetDB()
	database.Where(model.AzureClusterModel{ClusterModelId: c.modelCluster.ID}).First(&c.modelCluster.Azure)
	//TODO check banzairesponses
	config, err := azureClient.GetClusterConfig(client, c.modelCluster.Name, c.modelCluster.Azure.ResourceGroup, "clusterUser")
	if err != nil {
		// TODO status code !?
		return nil, err
	}
	log.Info("Get k8s config succeeded")
	c.k8sConfig = []byte(config.Properties.KubeConfig)
	return c.k8sConfig, nil
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
	log.Info("Create cluster status response")

	return &components.GetClusterStatusResponse{
		Status:           c.modelCluster.Status,
		Name:             c.modelCluster.Name,
		Location:         c.modelCluster.Location,
		Cloud:            c.modelCluster.Cloud,
		NodeInstanceType: c.modelCluster.NodeInstanceType,
		ResourceID:       c.modelCluster.ID,
	}, nil
}

// DeleteCluster deletes cluster from aks
func (c *AKSCluster) DeleteCluster() error {
	log := logger.WithFields(logrus.Fields{"action": constants.TagDeleteCluster})
	client, err := c.GetAKSClient()
	if err != nil {
		return err
	}

	client.With(log.Logger)

	// set azure props
	database := model.GetDB()
	database.Where(model.AzureClusterModel{ClusterModelId: c.modelCluster.ID}).First(&c.modelCluster.Azure)

	err = azureClient.DeleteCluster(client, c.modelCluster.Name, c.modelCluster.Azure.ResourceGroup)
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
	client, err := c.GetAKSClient()
	if err != nil {
		return err
	}

	client.With(log.Logger)

	ccr := azureCluster.CreateClusterRequest{
		Name:     c.modelCluster.Name,
		Location: c.modelCluster.Location,
		//VMSize:            c.modelCluster.NodeInstanceType,
		ResourceGroup: c.modelCluster.Azure.ResourceGroup,
		//AgentCount:        request.UpdateClusterAzure.AgentCount,
		//AgentName:         c.modelCluster.Azure.AgentName,
		KubernetesVersion: c.modelCluster.Azure.KubernetesVersion,
		Profiles:          nil, // todo profiles
	}

	updatedCluster, err := azureClient.CreateUpdateCluster(client, &ccr)
	if err != nil {
		return err
	}
	log.Info("Cluster update succeeded")
	//Update AKS model
	log.Info("Create updated model")

	c.azureCluster = &updatedCluster.Value
	return nil
}

func (c *AKSCluster) UpdateClusterModelFromRequest(request *bTypes.UpdateClusterRequest) {
	updatedModel := &model.ClusterModel{// todo make it testable
		Model: c.modelCluster.Model,
		Name: c.modelCluster.Name,
		Location: c.modelCluster.Location,
		NodeInstanceType: c.modelCluster.NodeInstanceType,
		Cloud: c.modelCluster.Cloud,
		OrganizationId: c.modelCluster.OrganizationId,
		SecretId: c.modelCluster.SecretId,
		Status: c.modelCluster.Status,
		Azure: model.AzureClusterModel{
			ResourceGroup: c.modelCluster.Azure.ResourceGroup,
			//AgentCount:        request.UpdateClusterAzure.AgentCount,
			//AgentName:         c.modelCluster.Azure.AgentName,
			KubernetesVersion: c.modelCluster.Azure.KubernetesVersion,
			// todo profiles
		},
	}
	c.modelCluster = updatedModel
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
	client, err := c.GetAKSClient()
	if err != nil {
		return nil, err
	}
	resp, err := azureClient.GetCluster(client, c.modelCluster.Name, c.modelCluster.Azure.ResourceGroup)
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

	// todo profiles
	//// ---- [ Node check ] ---- //
	//if r.UpdateAzureNode == nil {
	//	log.Info("'node' field is empty. Load it from stored data.")
	//	r.UpdateAzureNode = &banzaiAzureTypes.UpdateAzureNode{
	//		AgentCount: c.modelCluster.Azure.AgentCount,
	//	}
	//}
	//
	//// ---- [ Node - Agent count check] ---- //
	//if r.AgentCount == 0 {
	//	def := c.modelCluster.Azure.AgentCount
	//	log.Info("Node agentCount set to default value: ", def)
	//	r.AgentCount = def
	//}

}

//CheckEqualityToUpdate validates the update request
func (c *AKSCluster) CheckEqualityToUpdate(r *components.UpdateClusterRequest) error {
	// create update request struct with the stored data to check equality
	preCl := &banzaiAzureTypes.UpdateClusterAzure{
		UpdateAzureNode: &banzaiAzureTypes.UpdateAzureNode{
			// AgentCount: c.modelCluster.Azure.AgentCount,// todo profiles
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

// GetLocations returns all the locations that are available for resource providers
func GetLocations(orgId uint, secretId string) ([]string, error) {
	client, err := getAKSClient(orgId, secretId)
	if err != nil {
		return nil, err
	}

	return azureClient.GetLocations(client)
}

// GetMachineTypes lists all available virtual machine sizes for a subscription in a location.
func GetMachineTypes(orgId uint, secretId, location string) (response map[string]components.MachineType, err error) {
	client, err := getAKSClient(orgId, secretId)
	if err != nil {
		return nil, err
	}

	response = make(map[string]components.MachineType)
	response[location], err = azureClient.GetVmSizes(client, location)

	return

}

// GetKubernetesVersion returns a list of supported kubernetes version in the specified subscription
func GetKubernetesVersion(orgId uint, secretId, location string) ([]string, error) {
	client, err := getAKSClient(orgId, secretId)
	if err != nil {
		return nil, err
	}

	return azureClient.GetKubernetesVersions(client, location)
}

// getAKSClient create AKSClient with the given organization id and secret id
func getAKSClient(orgId uint, secretId string) (*azureClient.AKSClient, error) {
	c := AKSCluster{
		modelCluster: &model.ClusterModel{
			OrganizationId: orgId,
			SecretId:       secretId,
		},
	}

	return c.GetAKSClient()
}

// UpdateStatus updates cluster status in database
func (c *AKSCluster) UpdateStatus(status string) error {
	return c.modelCluster.UpdateStatus(status)
}

// GetClusterDetails gets cluster details from cloud
func (c *AKSCluster) GetClusterDetails() (*components.ClusterDetailsResponse, error) {

	log := logger.WithFields(logrus.Fields{"action": "GetClusterDetails"})
	client, err := c.GetAKSClient()
	if err != nil {
		return nil, err
	}

	client.With(log.Logger)

	resp, err := azureClient.GetCluster(client, c.modelCluster.Name, c.modelCluster.Azure.ResourceGroup)
	if err != nil {
		return nil, errors.New(err)
	}
	log.Info("Get cluster success")
	stage := resp.Value.Properties.ProvisioningState
	log.Info("Cluster stage is", stage)
	if stage == "Succeeded" {
		return &components.ClusterDetailsResponse{
			Name: c.modelCluster.Name,
			Id:   c.modelCluster.ID,
		}, nil

	}
	return nil, constants.ErrorClusterNotReady
}
