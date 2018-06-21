package cluster

import (
	"github.com/Azure/azure-sdk-for-go/services/containerservice/mgmt/2017-09-30/containerservice"
	azureClient "github.com/banzaicloud/azure-aks-client/client"
	azureCluster "github.com/banzaicloud/azure-aks-client/cluster"
	"github.com/banzaicloud/pipeline/model"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	pkgClusterAzure "github.com/banzaicloud/pipeline/pkg/cluster/azure"
	pkgErrors "github.com/banzaicloud/pipeline/pkg/errors"
	"github.com/banzaicloud/pipeline/secret"
	"github.com/banzaicloud/pipeline/secret/verify"
	pipelineSsh "github.com/banzaicloud/pipeline/ssh"
	"github.com/banzaicloud/pipeline/utils"
	"github.com/go-errors/errors"
)

//CreateAKSClusterFromRequest creates ClusterModel struct from the request
func CreateAKSClusterFromRequest(request *pkgCluster.CreateClusterRequest, orgId uint) (*AKSCluster, error) {
	log.Debug("Create ClusterModel struct from the request")
	var cluster AKSCluster

	var nodePools []*model.AzureNodePoolModel
	if request.Properties.CreateClusterAzure.NodePools != nil {
		for name, np := range request.Properties.CreateClusterAzure.NodePools {
			nodePools = append(nodePools, &model.AzureNodePoolModel{
				Name:             name,
				Autoscaling:      np.Autoscaling,
				NodeMinCount:     np.MinCount,
				NodeMaxCount:     np.MaxCount,
				Count:            np.Count,
				NodeInstanceType: np.NodeInstanceType,
			})
		}
	}

	cluster.modelCluster = &model.ClusterModel{
		Name:           request.Name,
		Location:       request.Location,
		Cloud:          request.Cloud,
		OrganizationId: orgId,
		SecretId:       request.SecretId,
		Azure: model.AzureClusterModel{
			ResourceGroup:     request.Properties.CreateClusterAzure.ResourceGroup,
			KubernetesVersion: request.Properties.CreateClusterAzure.KubernetesVersion,
			NodePools:         nodePools,
		},
	}
	return &cluster, nil
}

//AKSCluster struct for AKS cluster
type AKSCluster struct {
	azureCluster *pkgClusterAzure.Value //Don't use this directly
	modelCluster *model.ClusterModel
	APIEndpoint  string
	CommonClusterBase
}

// GetOrganizationId gets org where the cluster belongs
func (c *AKSCluster) GetOrganizationId() uint {
	return c.modelCluster.OrganizationId
}

// GetAKSClient creates an AKS client with the credentials
func (c *AKSCluster) GetAKSClient() (azureClient.ClusterManager, error) {
	clusterSecret, err := c.GetSecretWithValidation()
	if err != nil {
		return nil, err
	}
	creds := verify.CreateAKSCredentials(clusterSecret.Values)
	return azureClient.GetAKSClient(creds)
}

// GetSecretId retrieves the secret id
func (c *AKSCluster) GetSecretId() string {
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

	// create profiles model for the request
	var profiles []containerservice.AgentPoolProfile
	if nodePools := c.modelCluster.Azure.NodePools; nodePools != nil {
		for _, np := range nodePools {
			if np != nil {
				count := int32(np.Count)
				name := np.Name
				profiles = append(profiles, containerservice.AgentPoolProfile{
					Name:   &name,
					Count:  &count,
					VMSize: containerservice.VMSizeTypes(np.NodeInstanceType),
				})
			}
		}
	}

	clusterSshSecret, err := c.GetSshSecretWithValidation()
	if err != nil {
		return err
	}

	sshKey := pipelineSsh.NewKey(clusterSshSecret)

	r := azureCluster.CreateClusterRequest{
		Name:              c.modelCluster.Name,
		Location:          c.modelCluster.Location,
		ResourceGroup:     c.modelCluster.Azure.ResourceGroup,
		KubernetesVersion: c.modelCluster.Azure.KubernetesVersion,
		SSHPubKey:         sshKey.PublicKeyData,
		Profiles:          profiles,
	}
	client, err := c.GetAKSClient()
	if err != nil {
		return err
	}

	client.With(log)

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

	log.Info("Assign Storage Account Contributor role for all VM")
	err = azureClient.AssignStorageAccountContributorRole(client, c.modelCluster.Azure.ResourceGroup, c.modelCluster.Name, c.modelCluster.Location)
	if err != nil {
		return err
	}
	log.Info("Role assign succeeded")

	return nil
}

//Persist save the cluster model
func (c *AKSCluster) Persist(status, statusMessage string) error {
	return c.modelCluster.UpdateStatus(status, statusMessage)
}

// DownloadK8sConfig downloads the kubeconfig file from cloud
func (c *AKSCluster) DownloadK8sConfig() ([]byte, error) {
	client, err := c.GetAKSClient()
	if err != nil {
		return nil, err
	}

	client.With(log)

	database := model.GetDB()
	database.Where(model.AzureClusterModel{ClusterModelId: c.modelCluster.ID}).First(&c.modelCluster.Azure)
	//TODO check banzairesponses
	config, err := azureClient.GetClusterConfig(client, c.modelCluster.Name, c.modelCluster.Azure.ResourceGroup, "clusterUser")
	if err != nil {
		// TODO status code !?
		return nil, err
	}
	log.Info("Get k8s config succeeded")
	kubeConfig := []byte(config.Properties.KubeConfig)
	return kubeConfig, nil
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
func (c *AKSCluster) GetStatus() (*pkgCluster.GetClusterStatusResponse, error) {

	log.Info("Create cluster status response")

	nodePools := make(map[string]*pkgCluster.NodePoolStatus)
	for _, np := range c.modelCluster.Azure.NodePools {
		if np != nil {
			nodePools[np.Name] = &pkgCluster.NodePoolStatus{
				Count:        np.Count,
				InstanceType: np.NodeInstanceType,
			}
		}
	}

	return &pkgCluster.GetClusterStatusResponse{
		Status:        c.modelCluster.Status,
		StatusMessage: c.modelCluster.StatusMessage,
		Name:          c.modelCluster.Name,
		Location:      c.modelCluster.Location,
		Cloud:         c.modelCluster.Cloud,
		ResourceID:    c.modelCluster.ID,
		NodePools:     nodePools,
	}, nil
}

// DeleteCluster deletes cluster from aks
func (c *AKSCluster) DeleteCluster() error {
	client, err := c.GetAKSClient()
	if err != nil {
		return err
	}

	client.With(log)

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
func (c *AKSCluster) UpdateCluster(request *pkgCluster.UpdateClusterRequest) error {
	client, err := c.GetAKSClient()
	if err != nil {
		return err
	}

	client.With(log)

	clusterSshSecret, err := c.GetSshSecretWithValidation()
	if err != nil {
		return err
	}

	sshKey := pipelineSsh.NewKey(clusterSshSecret)

	// send separate requests because Azure not supports multiple nodepool modification
	// Azure not supports adding and deleting nodepools
	var nodePoolAfterUpdate []*model.AzureNodePoolModel
	var updatedCluster *pkgClusterAzure.ResponseWithValue
	if requestNodes := request.Azure.NodePools; requestNodes != nil {
		for name, np := range requestNodes {
			if existNodePool := c.getExistingNodePoolByName(name); np != nil && existNodePool != nil {
				log.Infof("NodePool is exists[%s], update...", name)

				count := int32(np.Count)

				// create request model for aks-client
				ccr := azureCluster.CreateClusterRequest{
					Name:              c.modelCluster.Name,
					Location:          c.modelCluster.Location,
					ResourceGroup:     c.modelCluster.Azure.ResourceGroup,
					KubernetesVersion: c.modelCluster.Azure.KubernetesVersion,
					SSHPubKey:         sshKey.PublicKeyData,
					Profiles: []containerservice.AgentPoolProfile{
						{
							Name:   &name,
							Count:  &count,
							VMSize: containerservice.VMSizeTypes(existNodePool.NodeInstanceType),
						},
					},
				}

				nodePoolAfterUpdate = append(nodePoolAfterUpdate, &model.AzureNodePoolModel{
					ID:               existNodePool.ID,
					ClusterModelId:   existNodePool.ClusterModelId,
					Name:             name,
					Autoscaling:      np.Autoscaling,
					NodeMinCount:     np.MinCount,
					NodeMaxCount:     np.MaxCount,
					Count:            np.Count,
					NodeInstanceType: existNodePool.NodeInstanceType,
				})

				updatedCluster, err = c.updateWithPolling(client, &ccr)
				if err != nil {
					return err
				}
			} else {
				log.Infof("There's no nodepool with this name[%s]", name)
			}
		}
	}

	if updatedCluster != nil {
		updateCluster := &model.ClusterModel{
			ID:             c.modelCluster.ID,
			CreatedAt:      c.modelCluster.CreatedAt,
			UpdatedAt:      c.modelCluster.UpdatedAt,
			DeletedAt:      c.modelCluster.DeletedAt,
			Name:           c.modelCluster.Name,
			Location:       c.modelCluster.Location,
			Cloud:          c.modelCluster.Cloud,
			OrganizationId: c.modelCluster.OrganizationId,
			SecretId:       c.modelCluster.SecretId,
			ConfigSecretId: c.modelCluster.ConfigSecretId,
			SshSecretId:    c.modelCluster.SshSecretId,
			Status:         c.modelCluster.Status,
			Azure: model.AzureClusterModel{
				ResourceGroup:     c.modelCluster.Azure.ResourceGroup,
				KubernetesVersion: c.modelCluster.Azure.KubernetesVersion,
				NodePools:         nodePoolAfterUpdate,
			},
		}
		c.modelCluster = updateCluster
		c.azureCluster = &updatedCluster.Value
	}

	return nil
}

// getExistingNodePoolByName returns saved NodePool by name
func (c *AKSCluster) getExistingNodePoolByName(name string) *model.AzureNodePoolModel {

	if nodePools := c.modelCluster.Azure.NodePools; nodePools != nil {
		for _, nodePool := range nodePools {
			if nodePool != nil && nodePool.Name == name {
				return nodePool
			}
		}
	}

	return nil
}

// updateWithPolling sends update request to cloud and polling until it's not ready
func (c *AKSCluster) updateWithPolling(manager azureClient.ClusterManager, ccr *azureCluster.CreateClusterRequest) (*pkgClusterAzure.ResponseWithValue, error) {

	log.Info("Send update request to azure")
	_, err := azureClient.CreateUpdateCluster(manager, ccr)
	if err != nil {
		return nil, err
	}

	log.Info("Polling to check update")
	// polling to check cluster updated
	updatedCluster, err := azureClient.PollingCluster(manager, c.modelCluster.Name, c.modelCluster.Azure.ResourceGroup)
	if err != nil {
		return nil, err
	}

	log.Info("Cluster updated successfully")
	return updatedCluster, nil
}

//GetID returns the specified cluster id
func (c *AKSCluster) GetID() uint {
	return c.modelCluster.ID
}

//GetModel returns the whole clusterModel
func (c *AKSCluster) GetModel() *model.ClusterModel {
	return c.modelCluster
}

// GetAzureCluster returns cluster from cloud
func (c *AKSCluster) GetAzureCluster() (*pkgClusterAzure.Value, error) {
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
	log.Debug("Create ClusterModel struct from the request")
	aksCluster := AKSCluster{
		modelCluster: clusterModel,
	}
	return &aksCluster, nil
}

//AddDefaultsToUpdate adds defaults to update request
func (c *AKSCluster) AddDefaultsToUpdate(r *pkgCluster.UpdateClusterRequest) {

	if r.Azure == nil {
		log.Info("'azure' field is empty.")
		r.Azure = &pkgClusterAzure.UpdateClusterAzure{}
	}

	if len(r.Azure.NodePools) == 0 {
		storedPools := c.modelCluster.Azure.NodePools
		nodePools := make(map[string]*pkgClusterAzure.NodePoolUpdate)
		for _, np := range storedPools {
			nodePools[np.Name] = &pkgClusterAzure.NodePoolUpdate{
				Autoscaling: np.Autoscaling,
				MinCount:    np.NodeMinCount,
				MaxCount:    np.NodeMaxCount,
				Count:       np.Count,
			}
		}
		r.Azure.NodePools = nodePools
	}

}

//CheckEqualityToUpdate validates the update request
func (c *AKSCluster) CheckEqualityToUpdate(r *pkgCluster.UpdateClusterRequest) error {
	// create update request struct with the stored data to check equality
	preProfiles := make(map[string]*pkgClusterAzure.NodePoolUpdate)

	for _, preP := range c.modelCluster.Azure.NodePools {
		if preP != nil {
			preProfiles[preP.Name] = &pkgClusterAzure.NodePoolUpdate{
				Autoscaling: preP.Autoscaling,
				MinCount:    preP.NodeMinCount,
				MaxCount:    preP.NodeMaxCount,
				Count:       preP.Count,
			}
		}
	}

	preCl := &pkgClusterAzure.UpdateClusterAzure{
		NodePools: preProfiles,
	}

	log.Info("Check stored & updated cluster equals")

	// check equality
	return utils.IsDifferent(r.Azure, preCl)
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
func GetMachineTypes(orgId uint, secretId, location string) (response map[string]pkgCluster.MachineType, err error) {
	client, err := getAKSClient(orgId, secretId)
	if err != nil {
		return nil, err
	}

	response = make(map[string]pkgCluster.MachineType)
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
func getAKSClient(orgId uint, secretId string) (azureClient.ClusterManager, error) {
	c := AKSCluster{
		modelCluster: &model.ClusterModel{
			OrganizationId: orgId,
			SecretId:       secretId,
			Cloud:          pkgCluster.Azure,
		},
	}

	return c.GetAKSClient()
}

// UpdateStatus updates cluster status in database
func (c *AKSCluster) UpdateStatus(status, statusMessage string) error {
	return c.modelCluster.UpdateStatus(status, statusMessage)
}

// GetClusterDetails gets cluster details from cloud
func (c *AKSCluster) GetClusterDetails() (*pkgCluster.ClusterDetailsResponse, error) {

	client, err := c.GetAKSClient()
	if err != nil {
		return nil, err
	}

	client.With(log)

	resp, err := azureClient.GetCluster(client, c.modelCluster.Name, c.modelCluster.Azure.ResourceGroup)
	if err != nil {
		return nil, errors.New(err)
	}
	log.Info("Get cluster success")
	stage := resp.Value.Properties.ProvisioningState
	log.Info("Cluster stage is", stage)
	if stage == "Succeeded" {
		return &pkgCluster.ClusterDetailsResponse{
			Name: c.modelCluster.Name,
			Id:   c.modelCluster.ID,
		}, nil

	}
	return nil, pkgErrors.ErrorClusterNotReady
}

// ValidateCreationFields validates all field
func (c *AKSCluster) ValidateCreationFields(r *pkgCluster.CreateClusterRequest) error {

	location := r.Location

	// Validate location
	log.Info("Validate location")
	if err := c.validateLocation(location); err != nil {
		return err
	}
	log.Info("Validate location passed")

	// Validate machine types
	nodePools := r.Properties.CreateClusterAzure.NodePools
	log.Info("Validate nodePools")
	if err := c.validateMachineType(nodePools, location); err != nil {
		return err
	}
	log.Info("Validate nodePools passed")

	// Validate kubernetes version
	log.Info("Validate kubernetesVersion")
	k8sVersion := r.Properties.CreateClusterAzure.KubernetesVersion
	if err := c.validateKubernetesVersion(k8sVersion, location); err != nil {
		return err
	}
	log.Info("Validate kubernetesVersion passed")

	return nil

}

// validateLocation validates location
func (c *AKSCluster) validateLocation(location string) error {
	log.Infof("Location: %s", location)
	validLocations, err := GetLocations(c.GetOrganizationId(), c.GetSecretId())
	if err != nil {
		return err
	}

	log.Infof("Valid locations: %#v", validLocations)

	if isOk := utils.Contains(validLocations, location); !isOk {
		return pkgErrors.ErrorNotValidLocation
	}

	return nil
}

// validateMachineType validates nodeInstanceTypes
func (c *AKSCluster) validateMachineType(nodePools map[string]*pkgClusterAzure.NodePoolCreate, location string) error {

	var machineTypes []string
	for _, nodePool := range nodePools {
		if nodePool != nil {
			machineTypes = append(machineTypes, nodePool.NodeInstanceType)
		}
	}

	log.Infof("NodeInstanceTypes: %v", machineTypes)

	validMachineTypes, err := GetMachineTypes(c.GetOrganizationId(), c.GetSecretId(), location)
	if err != nil {
		return err
	}
	log.Infof("Valid NodeInstanceTypes: %v", validMachineTypes[location])

	for _, mt := range machineTypes {
		if isOk := utils.Contains(validMachineTypes[location], mt); !isOk {
			return pkgErrors.ErrorNotValidNodeInstanceType
		}
	}

	return nil
}

// validateKubernetesVersion validates k8s version
func (c *AKSCluster) validateKubernetesVersion(k8sVersion, location string) error {

	log.Infof("K8SVersion: %s", k8sVersion)
	validVersions, err := GetKubernetesVersion(c.GetOrganizationId(), c.GetSecretId(), location)
	if err != nil {
		return err
	}
	log.Infof("Valid K8SVersions: %s", validVersions)

	if isOk := utils.Contains(validVersions, k8sVersion); !isOk {
		return pkgErrors.ErrorNotValidKubernetesVersion
	}

	return nil

}

// GetSecretWithValidation returns secret from vault
func (c *AKSCluster) GetSecretWithValidation() (*secret.SecretsItemResponse, error) {
	return c.CommonClusterBase.getSecret(c)
}

// GetSshSecretWithValidation returns ssh secret from vault
func (c *AKSCluster) GetSshSecretWithValidation() (*secret.SecretsItemResponse, error) {
	return c.CommonClusterBase.getSshSecret(c)
}

// SaveConfigSecretId saves the config secret id in database
func (c *AKSCluster) SaveConfigSecretId(configSecretId string) error {
	return c.modelCluster.UpdateConfigSecret(configSecretId)
}

// GetConfigSecretId return config secret id
func (c *AKSCluster) GetConfigSecretId() string {
	return c.modelCluster.ConfigSecretId
}

// GetSshSecretId return ssh secret id
func (c *AKSCluster) GetSshSecretId() string {
	return c.modelCluster.SshSecretId
}

// SaveSshSecretId saves the ssh secret id to database
func (c *AKSCluster) SaveSshSecretId(sshSecretId string) error {
	return c.modelCluster.UpdateSshSecret(sshSecretId)
}

// GetK8sConfig returns the Kubernetes config
func (c *AKSCluster) GetK8sConfig() ([]byte, error) {
	return c.CommonClusterBase.getConfig(c)
}

// RequiresSshPublicKey returns true as a public ssh key is needed for bootstrapping
// the cluster
func (c *AKSCluster) RequiresSshPublicKey() bool {
	return true
}

// ReloadFromDatabase load cluster from DB
func (c *AKSCluster) ReloadFromDatabase() error {
	return c.modelCluster.ReloadFromDatabase()
}
