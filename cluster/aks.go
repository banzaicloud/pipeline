// Copyright © 2018 Banzai Cloud
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cluster

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-10-01/compute"
	"github.com/Azure/azure-sdk-for-go/services/containerservice/mgmt/2018-03-31/containerservice"
	"github.com/Azure/azure-sdk-for-go/services/monitor/mgmt/2017-09-01/insights"
	azureClient "github.com/banzaicloud/azure-aks-client/client"
	azureCluster "github.com/banzaicloud/azure-aks-client/cluster"
	azureErrors "github.com/banzaicloud/azure-aks-client/errors"
	azureType "github.com/banzaicloud/azure-aks-client/types"
	"github.com/banzaicloud/pipeline/config"
	"github.com/banzaicloud/pipeline/internal/providers/azure"
	"github.com/banzaicloud/pipeline/model"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	pkgAzure "github.com/banzaicloud/pipeline/pkg/cluster/aks"
	pkgCommon "github.com/banzaicloud/pipeline/pkg/common"
	pkgErrors "github.com/banzaicloud/pipeline/pkg/errors"
	"github.com/banzaicloud/pipeline/secret"
	"github.com/banzaicloud/pipeline/secret/verify"
	"github.com/banzaicloud/pipeline/utils"
	"github.com/goph/emperror"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	statusSucceeded = "Succeeded"
)

const (
	poolNameKey = "poolName"
)

//CreateAKSClusterFromRequest creates ClusterModel struct from the request
func CreateAKSClusterFromRequest(request *pkgCluster.CreateClusterRequest, orgId, userId uint) (*AKSCluster, error) {
	var cluster AKSCluster

	var nodePools []*model.AKSNodePoolModel
	if request.Properties.CreateClusterAKS.NodePools != nil {
		for name, np := range request.Properties.CreateClusterAKS.NodePools {
			nodePools = append(nodePools, &model.AKSNodePoolModel{
				CreatedBy:        userId,
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
		CreatedBy:      userId,
		SecretId:       request.SecretId,
		Distribution:   pkgCluster.AKS,
		AKS: model.AKSClusterModel{
			ResourceGroup:     request.Properties.CreateClusterAKS.ResourceGroup,
			KubernetesVersion: request.Properties.CreateClusterAKS.KubernetesVersion,
			NodePools:         nodePools,
		},
	}

	log := log.WithField("cluster", request.Name)
	cluster.log = log

	return &cluster, nil
}

//AKSCluster struct for AKS cluster
type AKSCluster struct {
	azureCluster *azureType.Value //Don't use this directly
	modelCluster *model.ClusterModel
	APIEndpoint  string
	CommonClusterBase

	log logrus.FieldLogger
}

type aksClusterCreateUpdateFailedError struct {
	clusterCreateUpdateError error
	failedEventsMsg          []string
}

func (e aksClusterCreateUpdateFailedError) Error() string {
	if len(e.failedEventsMsg) > 0 {
		return e.clusterCreateUpdateError.Error() + "\n" + strings.Join(e.failedEventsMsg, "\n")
	}

	return e.clusterCreateUpdateError.Error()
}

func (e aksClusterCreateUpdateFailedError) Cause() error {
	return e.clusterCreateUpdateError
}

// GetOrganizationId gets org where the cluster belongs
func (c *AKSCluster) GetOrganizationId() uint {
	return c.modelCluster.OrganizationId
}

// GetLocation gets where the cluster is.
func (c *AKSCluster) GetLocation() string {
	return c.modelCluster.Location
}

// GetAKSClient creates an AKS client with the credentials
func (c *AKSCluster) GetAKSClient() (azureClient.ClusterManager, error) {
	clusterSecret, err := c.GetSecretWithValidation()
	if err != nil {
		return nil, emperror.Wrap(err, "couldn't get cluster secret")
	}
	creds := verify.CreateAKSCredentials(clusterSecret.Values)
	aksClient, err := azureClient.GetAKSClient(creds)

	if err != nil {
		return nil, emperror.Wrap(err, "failed to create AKS client")
	}
	aksClient.With(c.log)

	return aksClient, nil
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
		return "", emperror.Wrap(err, "failed to get AKS cluster from Azure")
	}
	c.APIEndpoint = cluster.Properties.Fqdn
	return c.APIEndpoint, nil
}

//CreateCluster creates a new cluster
func (c *AKSCluster) CreateCluster() error {

	// create profiles model for the request
	var profiles []containerservice.ManagedClusterAgentPoolProfile
	c.modelCluster.RbacEnabled = true
	if nodePools := c.modelCluster.AKS.NodePools; nodePools != nil {
		for _, np := range nodePools {
			if np != nil {
				count := int32(np.Count)
				name := np.Name
				profiles = append(profiles, containerservice.ManagedClusterAgentPoolProfile{
					Name:   &name,
					Count:  &count,
					VMSize: containerservice.VMSizeTypes(np.NodeInstanceType),
				})
			}
		}
	}

	clusterSshSecret, err := c.getSshSecret(c)
	if err != nil {
		return emperror.Wrap(err, "failed to retrieve SSH secret")
	}

	sshKey := secret.NewSSHKeyPair(clusterSshSecret)

	r := azureCluster.CreateClusterRequest{
		Name:              c.modelCluster.Name,
		Location:          c.modelCluster.Location,
		ResourceGroup:     c.modelCluster.AKS.ResourceGroup,
		KubernetesVersion: c.modelCluster.AKS.KubernetesVersion,
		SSHPubKey:         sshKey.PublicKeyData,
		Profiles:          profiles,
		EnableRBAC:        c.RbacEnabled(),
	}
	client, err := c.GetAKSClient()
	if err != nil {
		return emperror.Wrap(err, "failed to create AKS client")
	}

	client.With(c.log)

	clusterCreateInitTime := time.Now()
	// call creation
	createdCluster, err := azureClient.CreateUpdateCluster(client, &r)
	if err != nil {
		// initiating cluster create failed
		return emperror.WrapWith(err, "failed to initiate cluster creation", "cluster", c.modelCluster.Name)
	}
	// creation success
	c.log.Info("Cluster create initiated!")

	c.azureCluster = &createdCluster.Value

	// polling cluster
	pollingResult, err := azureClient.PollingCluster(client, r.Name, r.ResourceGroup)
	if err != nil {
		if err == azureErrors.ErrClusterStageFailed {
			// allow some time as there is some latency until all activity logs generated from the cluster create operation become available
			time.Sleep(1 * time.Minute)
			return emperror.Wrap(c.onClusterCreateFailure(err, clusterCreateInitTime, r.ResourceGroup, r.Name), "create cluster failed")
		}

		// polling error
		return emperror.Wrap(err, "failed to poll cluster status")
	}
	c.log.Info("Cluster is ready...")
	c.azureCluster = &pollingResult.Value

	c.log.Info("Assign Storage Account Contributor role for all VM")
	err = azureClient.AssignStorageAccountContributorRole(client, c.modelCluster.AKS.ResourceGroup, c.modelCluster.Name, c.modelCluster.Location)
	if err != nil {
		return emperror.Wrap(err, "failed to assign storage account contributor role to cluster nodes")
	}
	c.log.Info("Role assign succeeded")

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
		return nil, emperror.Wrap(err, "failed to create AKS client")
	}

	client.With(c.log)

	database := config.DB()
	database.Where(model.AKSClusterModel{ID: c.modelCluster.ID}).First(&c.modelCluster.AKS)
	//TODO check banzairesponses
	config, err := azureClient.GetClusterConfig(client, c.modelCluster.Name, c.modelCluster.AKS.ResourceGroup, "clusterUser")
	if err != nil {
		// TODO status code !?
		return nil, err
	}
	c.log.Info("Get k8s config succeeded")
	kubeConfig := []byte(config.Properties.KubeConfig)
	return kubeConfig, nil
}

//GetName returns the name of the cluster
func (c *AKSCluster) GetName() string {
	return c.modelCluster.Name
}

// GetCloud returns the cloud type of the cluster
func (c *AKSCluster) GetCloud() string {
	return c.modelCluster.Cloud
}

// GetDistribution returns the distribution type of the cluster
func (c *AKSCluster) GetDistribution() string {
	return c.modelCluster.Distribution
}

//GetStatus gets cluster status
func (c *AKSCluster) GetStatus() (*pkgCluster.GetClusterStatusResponse, error) {

	c.log.Info("Create cluster status response")

	nodePools := make(map[string]*pkgCluster.NodePoolStatus)
	for _, np := range c.modelCluster.AKS.NodePools {
		if np != nil {
			nodePools[np.Name] = &pkgCluster.NodePoolStatus{
				Autoscaling:       np.Autoscaling,
				Count:             np.Count,
				InstanceType:      np.NodeInstanceType,
				MinCount:          np.NodeMinCount,
				MaxCount:          np.NodeMaxCount,
				CreatorBaseFields: *NewCreatorBaseFields(np.CreatedAt, np.CreatedBy),
			}
		}
	}

	return &pkgCluster.GetClusterStatusResponse{
		Status:            c.modelCluster.Status,
		StatusMessage:     c.modelCluster.StatusMessage,
		Name:              c.modelCluster.Name,
		Location:          c.modelCluster.Location,
		Cloud:             c.modelCluster.Cloud,
		Distribution:      c.modelCluster.Distribution,
		Version:           c.modelCluster.AKS.KubernetesVersion,
		ResourceID:        c.modelCluster.ID,
		Logging:           c.GetLogging(),
		Monitoring:        c.GetMonitoring(),
		ServiceMesh:       c.GetServiceMesh(),
		SecurityScan:      c.GetSecurityScan(),
		CreatorBaseFields: *NewCreatorBaseFields(c.modelCluster.CreatedAt, c.modelCluster.CreatedBy),
		NodePools:         nodePools,
		Region:            c.modelCluster.Location,
	}, nil
}

// DeleteCluster deletes cluster from aks
func (c *AKSCluster) DeleteCluster() error {
	client, err := c.GetAKSClient()
	if err != nil {
		return emperror.Wrap(err, "failed to create AKS client")
	}

	client.With(c.log)

	// set aks props
	database := config.DB()
	database.Where(model.AKSClusterModel{ID: c.modelCluster.ID}).First(&c.modelCluster.AKS)

	err = azureClient.DeleteCluster(client, c.modelCluster.Name, c.modelCluster.AKS.ResourceGroup)
	if err != nil {
		c.log.Info("Delete succeeded")
		return nil
	}
	// todo status code !?
	return err
}

// UpdateCluster updates AKS cluster in cloud
func (c *AKSCluster) UpdateCluster(request *pkgCluster.UpdateClusterRequest, userId uint) error {
	client, err := c.GetAKSClient()
	if err != nil {
		return emperror.Wrap(err, "failed to create AKS client")
	}

	client.With(c.log)

	clusterSshSecret, err := c.getSshSecret(c)
	if err != nil {
		return emperror.Wrap(err, "failed to retrieve SSH secret")
	}

	sshKey := secret.NewSSHKeyPair(clusterSshSecret)

	// send separate requests because Azure not supports multiple nodepool modification
	// Azure not supports adding and deleting nodepools
	var nodePoolAfterUpdate []*model.AKSNodePoolModel
	var updatedCluster *azureType.ResponseWithValue
	if requestNodes := request.AKS.NodePools; requestNodes != nil {
		for name, np := range requestNodes {
			if existNodePool := c.getExistingNodePoolByName(name); np != nil && existNodePool != nil {
				c.log.Infof("NodePool is exists[%s], update...", name)

				count := int32(np.Count)

				// create request model for aks-client
				ccr := azureCluster.CreateClusterRequest{
					Name:              c.modelCluster.Name,
					Location:          c.modelCluster.Location,
					ResourceGroup:     c.modelCluster.AKS.ResourceGroup,
					KubernetesVersion: c.modelCluster.AKS.KubernetesVersion,
					SSHPubKey:         sshKey.PublicKeyData,
					EnableRBAC:        c.RbacEnabled(),
					Profiles: []containerservice.ManagedClusterAgentPoolProfile{
						{
							Name:   &name,
							Count:  &count,
							VMSize: containerservice.VMSizeTypes(existNodePool.NodeInstanceType),
						},
					},
				}

				nodePoolAfterUpdate = append(nodePoolAfterUpdate, &model.AKSNodePoolModel{
					ID:               existNodePool.ID,
					CreatedAt:        time.Now(),
					CreatedBy:        userId,
					ClusterID:        existNodePool.ClusterID,
					Name:             name,
					Autoscaling:      np.Autoscaling,
					NodeMinCount:     np.MinCount,
					NodeMaxCount:     np.MaxCount,
					Count:            np.Count,
					NodeInstanceType: existNodePool.NodeInstanceType,
				})

				updatedCluster, err = c.updateWithPolling(client, &ccr)
				if err != nil {
					return emperror.Wrap(err, "cluster update failed")
				}
			} else {
				c.log.Infof("There's no nodepool with this name[%s]", name)
			}
		}
	}

	if updatedCluster != nil {
		c.modelCluster.AKS.NodePools = nodePoolAfterUpdate
		c.azureCluster = &updatedCluster.Value
	}

	return nil
}

// UpdateNodePools updates nodes pools of a cluster
func (c *AKSCluster) UpdateNodePools(request *pkgCluster.UpdateNodePoolsRequest, userId uint) error {
	return nil
}

// getExistingNodePoolByName returns saved NodePool by name
func (c *AKSCluster) getExistingNodePoolByName(name string) *model.AKSNodePoolModel {

	if nodePools := c.modelCluster.AKS.NodePools; nodePools != nil {
		for _, nodePool := range nodePools {
			if nodePool != nil && nodePool.Name == name {
				return nodePool
			}
		}
	}

	return nil
}

// updateWithPolling sends update request to cloud and polling until it's not ready
func (c *AKSCluster) updateWithPolling(manager azureClient.ClusterManager, ccr *azureCluster.CreateClusterRequest) (*azureType.ResponseWithValue, error) {

	c.log.Info("Send update request to aks")

	clusterUpdateInitTime := time.Now()
	_, err := azureClient.CreateUpdateCluster(manager, ccr)
	if err != nil {
		return nil, emperror.Wrap(err, "failed to initiate cluster update")
	}

	c.log.Info("Polling to check update")
	// polling to check cluster updated
	updatedCluster, err := azureClient.PollingCluster(manager, c.modelCluster.Name, c.modelCluster.AKS.ResourceGroup)
	if err != nil {
		if err == azureErrors.ErrClusterStageFailed {
			// allow some time as there is some latency until all activity logs generated from the cluster create operation become available
			time.Sleep(1 * time.Minute)
			return nil, c.onClusterUpdateFailure(err, clusterUpdateInitTime, ccr.Location, ccr.ResourceGroup, ccr.Name)
		}

		// polling error
		return nil, emperror.Wrap(err, "failed to poll cluster status")
	}

	c.log.Info("Cluster updated successfully")
	return updatedCluster, nil
}

//GetID returns the specified cluster id
func (c *AKSCluster) GetID() uint {
	return c.modelCluster.ID
}

func (c *AKSCluster) GetUID() string {
	return c.modelCluster.UID
}

//GetModel returns the whole clusterModel
func (c *AKSCluster) GetModel() *model.ClusterModel {
	return c.modelCluster
}

// GetAzureCluster returns cluster from cloud
func (c *AKSCluster) GetAzureCluster() (*azureType.Value, error) {
	client, err := c.GetAKSClient()
	if err != nil {
		return nil, emperror.Wrap(err, "failed to create AKS client")
	}
	resp, err := azureClient.GetCluster(client, c.modelCluster.Name, c.modelCluster.AKS.ResourceGroup)
	if err != nil {
		return nil, err
	}
	c.azureCluster = &resp.Value
	return c.azureCluster, nil
}

//CreateAKSClusterFromModel creates ClusterModel struct from model
func CreateAKSClusterFromModel(clusterModel *model.ClusterModel) (*AKSCluster, error) {
	log := log.WithField("cluster", clusterModel.Name)

	aksCluster := AKSCluster{
		modelCluster: clusterModel,
		log:          log,
	}
	return &aksCluster, nil
}

//AddDefaultsToUpdate adds defaults to update request
func (c *AKSCluster) AddDefaultsToUpdate(r *pkgCluster.UpdateClusterRequest) {

	if r.AKS == nil {
		c.log.Info("'aks' field is empty.")
		r.AKS = &pkgAzure.UpdateClusterAzure{}
	}

	if len(r.AKS.NodePools) == 0 {
		storedPools := c.modelCluster.AKS.NodePools
		nodePools := make(map[string]*pkgAzure.NodePoolUpdate)
		for _, np := range storedPools {
			nodePools[np.Name] = &pkgAzure.NodePoolUpdate{
				Autoscaling: np.Autoscaling,
				MinCount:    np.NodeMinCount,
				MaxCount:    np.NodeMaxCount,
				Count:       np.Count,
			}
		}
		r.AKS.NodePools = nodePools
	}

}

//CheckEqualityToUpdate validates the update request
func (c *AKSCluster) CheckEqualityToUpdate(r *pkgCluster.UpdateClusterRequest) error {
	// create update request struct with the stored data to check equality
	preProfiles := make(map[string]*pkgAzure.NodePoolUpdate)

	for _, preP := range c.modelCluster.AKS.NodePools {
		if preP != nil {
			preProfiles[preP.Name] = &pkgAzure.NodePoolUpdate{
				Autoscaling: preP.Autoscaling,
				MinCount:    preP.NodeMinCount,
				MaxCount:    preP.NodeMaxCount,
				Count:       preP.Count,
			}
		}
	}

	preCl := &pkgAzure.UpdateClusterAzure{
		NodePools: preProfiles,
	}

	c.log.Info("Check stored & updated cluster equals")

	// check equality
	return isDifferent(r.AKS, preCl)
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
		return nil, emperror.Wrap(err, "failed to create AKS client")
	}

	return azureClient.GetLocations(client)
}

// GetMachineTypes lists all available virtual machine sizes for a subscription in a location.
func GetMachineTypes(orgId uint, secretId, location string) (response map[string]pkgCluster.MachineType, err error) {
	client, err := getAKSClient(orgId, secretId)
	if err != nil {
		return nil, emperror.Wrap(err, "failed to create AKS client")
	}

	response = make(map[string]pkgCluster.MachineType)
	response[location], err = azureClient.GetVmSizes(client, location)

	return

}

// GetKubernetesVersion returns a list of supported kubernetes version in the specified subscription
func GetKubernetesVersion(orgId uint, secretId, location string) ([]string, error) {
	client, err := getAKSClient(orgId, secretId)
	if err != nil {
		return nil, emperror.Wrap(err, "failed to create AKS client")
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

// NodePoolExists returns true if node pool with nodePoolName exists
func (c *AKSCluster) NodePoolExists(nodePoolName string) bool {
	for _, np := range c.modelCluster.AKS.NodePools {
		if np != nil && np.Name == nodePoolName {
			return true
		}
	}
	return false
}

// IsReady checks if the cluster is running according to the cloud provider.
func (c *AKSCluster) IsReady() (bool, error) {
	client, err := c.GetAKSClient()
	if err != nil {
		return false, emperror.Wrap(err, "failed to create AKS client")
	}

	client.With(c.log)

	resp, err := azureClient.GetCluster(client, c.modelCluster.Name, c.modelCluster.AKS.ResourceGroup)
	if err != nil {
		return false, errors.WithStack(err)
	}

	stage := resp.Value.Properties.ProvisioningState
	c.log.Debug("Cluster stage is", stage)

	return stage == statusSucceeded, nil
}

// ValidateCreationFields validates all field
func (c *AKSCluster) ValidateCreationFields(r *pkgCluster.CreateClusterRequest) error {

	location := r.Location

	// Validate location
	c.log.Info("Validate location")
	if err := c.validateLocation(location); err != nil {
		return err
	}
	c.log.Info("Validate location passed")

	// Validate machine types
	nodePools := r.Properties.CreateClusterAKS.NodePools
	c.log.Info("Validate nodePools")
	if err := c.validateMachineType(nodePools, location); err != nil {
		return err
	}
	c.log.Info("Validate nodePools passed")

	// Validate kubernetes version
	c.log.Info("Validate kubernetesVersion")
	k8sVersion := r.Properties.CreateClusterAKS.KubernetesVersion
	if err := c.validateKubernetesVersion(k8sVersion, location); err != nil {
		return err
	}
	c.log.Info("Validate kubernetesVersion passed")

	return nil

}

// validateLocation validates location
func (c *AKSCluster) validateLocation(location string) error {
	c.log.Infof("Location: %s", location)
	validLocations, err := GetLocations(c.GetOrganizationId(), c.GetSecretId())
	if err != nil {
		return emperror.Wrap(err, "could not get locations from Azure")
	}

	c.log.Infof("Valid locations: %#v", validLocations)

	if isOk := utils.Contains(validLocations, location); !isOk {
		return pkgErrors.ErrorNotValidLocation
	}

	return nil
}

// validateMachineType validates nodeInstanceTypes
func (c *AKSCluster) validateMachineType(nodePools map[string]*pkgAzure.NodePoolCreate, location string) error {

	var machineTypes []string
	for _, nodePool := range nodePools {
		if nodePool != nil {
			machineTypes = append(machineTypes, nodePool.NodeInstanceType)
		}
	}

	c.log.Infof("NodeInstanceTypes: %v", machineTypes)

	validMachineTypes, err := GetMachineTypes(c.GetOrganizationId(), c.GetSecretId(), location)
	if err != nil {
		return emperror.WrapWith(err, "could not get VM types from Azure", "location", location)
	}
	c.log.Infof("Valid NodeInstanceTypes: %v", validMachineTypes[location])

	for _, mt := range machineTypes {
		if isOk := utils.Contains(validMachineTypes[location], mt); !isOk {
			return pkgErrors.ErrorNotValidNodeInstanceType
		}
	}

	return nil
}

// validateKubernetesVersion validates k8s version
func (c *AKSCluster) validateKubernetesVersion(k8sVersion, location string) error {

	c.log.Infof("K8SVersion: %s", k8sVersion)
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
func (c *AKSCluster) GetSecretWithValidation() (*secret.SecretItemResponse, error) {
	return c.CommonClusterBase.getSecret(c)
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

func (c *AKSCluster) GetK8sIpv4Cidrs() (*pkgCluster.Ipv4Cidrs, error) {
	//TODO
	return nil, errors.New("not implemented")
}

// GetK8sConfig returns the Kubernetes config
func (c *AKSCluster) GetK8sConfig() ([]byte, error) {
	return c.CommonClusterBase.getConfig(c)
}

// GetResourceGroup gets the Azure Resoure Group from the model
func (c *AKSCluster) GetResourceGroup() string {
	return c.modelCluster.AKS.ResourceGroup
}

// RequiresSshPublicKey returns true as a public ssh key is needed for bootstrapping
// the cluster
func (c *AKSCluster) RequiresSshPublicKey() bool {
	return true
}

// ListNodeNames returns node names to label them
func (c *AKSCluster) ListNodeNames() (labels pkgCommon.NodeNames, err error) {

	var client azureClient.ClusterManager
	client, err = c.GetAKSClient()
	if err != nil {
		return nil, emperror.Wrap(err, "failed to create AKS client")
	}

	client.With(c.log)

	labels = make(map[string][]string)

	var vms []compute.VirtualMachine
	vms, err = azureClient.ListVirtualMachines(client, c.modelCluster.AKS.ResourceGroup, c.modelCluster.Name, c.modelCluster.Location)
	for _, np := range c.modelCluster.AKS.NodePools {
		if np != nil {
			for _, vm := range vms {
				if vm.OsProfile != nil && vm.OsProfile.ComputerName != nil {
					for key, tag := range vm.Tags {
						if poolNameKey == key && tag != nil && *tag == np.Name {
							labels[np.Name] = append(labels[np.Name], *vm.OsProfile.ComputerName)
						}
					}
				}
			}
		}
	}

	return
}

// RbacEnabled returns true if rbac enabled on the cluster
func (c *AKSCluster) RbacEnabled() bool {
	return c.modelCluster.RbacEnabled
}

// GetSecurityScan returns true if security scan enabled on the cluster
func (c *AKSCluster) GetSecurityScan() bool {
	return c.modelCluster.SecurityScan
}

// SetSecurityScan returns true if security scan enabled on the cluster
func (c *AKSCluster) SetSecurityScan(scan bool) {
	c.modelCluster.SecurityScan = scan
}

// GetLogging returns true if logging enabled on the cluster
func (c *AKSCluster) GetLogging() bool {
	return c.modelCluster.Logging
}

// SetLogging returns true if logging enabled on the cluster
func (c *AKSCluster) SetLogging(l bool) {
	c.modelCluster.Logging = l
}

// GetMonitoring returns true if momnitoring enabled on the cluster
func (c *AKSCluster) GetMonitoring() bool {
	return c.modelCluster.Monitoring
}

// SetMonitoring returns true if monitoring enabled on the cluster
func (c *AKSCluster) SetMonitoring(l bool) {
	c.modelCluster.Monitoring = l
}

// GetServiceMesh returns true if service mesh is enabled on the cluster
func (c *AKSCluster) GetServiceMesh() bool {
	return c.modelCluster.ServiceMesh
}

// SetServiceMesh sets service mesh flag on the cluster
func (c *AKSCluster) SetServiceMesh(m bool) {
	c.modelCluster.ServiceMesh = m
}

// ListResourceGroups returns all resource group
func ListResourceGroups(orgId uint, secretId string) ([]string, error) {

	client, err := getAKSClient(orgId, secretId)
	if err != nil {
		return nil, emperror.Wrap(err, "failed to create AKS client")
	}

	client.With(log)

	groups, err := azureClient.ListGroups(client)
	if err != nil {
		return nil, emperror.Wrap(err, "could not get resource groups from Azure")
	}

	var groupNames []string
	for _, g := range groups {
		if g.Name != nil {
			groupNames = append(groupNames, *g.Name)
		}
	}

	return groupNames, nil
}

// CreateOrUpdateResourceGroup creates or updates a resource group
func CreateOrUpdateResourceGroup(orgId uint, secretId, rgName, location string) error {

	client, err := getAKSClient(orgId, secretId)
	if err != nil {
		return emperror.Wrap(err, "failed to create AKS client")
	}

	client.With(log)

	_, err = azureClient.CreateOrUpdateResourceGroup(client, rgName, location)
	return err

}

// DeleteResourceGroup creates or updates a resource group
func DeleteResourceGroup(orgId uint, secretId, rgName string) error {

	client, err := getAKSClient(orgId, secretId)
	if err != nil {
		return emperror.Wrap(err, "failed to create AKS client")
	}

	client.With(log)

	return azureClient.DeleteResourceGroup(client, rgName)
}

// GetAKSNodePools returns AKS node pools from a common cluster.
func GetAKSNodePools(cluster CommonCluster) ([]*model.AKSNodePoolModel, error) {
	akscluster, ok := cluster.(*AKSCluster)
	if !ok {
		return nil, ErrInvalidClusterInstance
	}

	return akscluster.modelCluster.AKS.NodePools, nil
}

// GetAKSResourceGroup returns AKS resource group from a common cluster.
func GetAKSResourceGroup(cluster CommonCluster) (string, error) {
	akscluster, ok := cluster.(*AKSCluster)
	if !ok {
		return "", ErrInvalidClusterInstance
	}

	return akscluster.modelCluster.AKS.ResourceGroup, nil
}

// NeedAdminRights returns true if rbac is enabled and need to create a cluster role binding to user
func (c *AKSCluster) NeedAdminRights() bool {
	return false
}

// GetKubernetesUserName returns the user ID which needed to create a cluster role binding which gives admin rights to the user
func (c *AKSCluster) GetKubernetesUserName() (string, error) {
	return "", nil
}

// GetCreatedBy returns cluster create userID.
func (c *AKSCluster) GetCreatedBy() uint {
	return c.modelCluster.CreatedBy
}

func (c *AKSCluster) onClusterCreateFailure(createUpdateError error, operationStartTime time.Time, resourceGroup, clusterName string) error {
	// collect error details from activity log
	clusterSecret, err := c.GetSecretWithValidation()
	if err != nil {
		return emperror.Wrap(err, "failed to get cluster secret")
	}
	creds := verify.CreateAKSCredentials(clusterSecret.Values)

	clusterResourceUri := fmt.Sprintf("/subscriptions/%s/resourcegroups/%s/providers/Microsoft.ContainerService/managedClusters/%s",
		creds.SubscriptionId,
		resourceGroup,
		clusterName,
	)

	toTimeStamp := time.Now()

	filter := fmt.Sprintf("eventTimestamp ge '%s' and eventTimestamp le '%s' and resourceUri eq '%s'",
		operationStartTime.UTC().Format(time.RFC3339Nano),
		toTimeStamp.UTC().Format(time.RFC3339Nano),
		clusterResourceUri,
	)

	errorEvents, err := c.collectActivityLogsWithErrors(filter)
	if err != nil {
		c.log.Errorln("retrieving activity logs failed: ", err.Error())
		return createUpdateError
	}

	return createClusterCreateUpdateFailedError(createUpdateError, errorEvents)
}

func (c *AKSCluster) onClusterUpdateFailure(createUpdateError error, operationStartTime time.Time, location, resourceGroup, clusterName string) error {
	// collect error details from activity log
	toTimeStamp := time.Now()

	filter := fmt.Sprintf("eventTimestamp ge '%s' and eventTimestamp le '%s' and resourceGroupName eq '%s'",
		operationStartTime.UTC().Format(time.RFC3339Nano),
		toTimeStamp.UTC().Format(time.RFC3339Nano),
		fmt.Sprintf("MC_%s_%s_%s", resourceGroup, clusterName, location),
	)

	errorEvents, err := c.collectActivityLogsWithErrors(filter)
	if err != nil {
		c.log.Errorln("retrieving activity logs failed: ", err.Error())
		return createUpdateError
	}

	return createClusterCreateUpdateFailedError(createUpdateError, errorEvents)
}

func createClusterCreateUpdateFailedError(createUpdateError error, errorEvents []insights.EventData) error {
	if len(errorEvents) > 0 {
		var failedEventsMsg []string

		for _, event := range errorEvents {
			if msg, ok := event.Properties["statusMessage"]; ok {
				failedEventsMsg = append(failedEventsMsg, *msg)
			}
		}

		return aksClusterCreateUpdateFailedError{
			clusterCreateUpdateError: createUpdateError,
			failedEventsMsg:          failedEventsMsg,
		}

	}

	return createUpdateError
}

// collectActivityLogsWithErrors collects cluster activity logs that denotes errors and matches the passed filter
func (c *AKSCluster) collectActivityLogsWithErrors(filter string) ([]insights.EventData, error) {
	clusterSecret, err := c.GetSecretWithValidation()
	if err != nil {
		return nil, emperror.Wrap(err, "failed to get cluster secret")
	}
	creds := verify.CreateAKSCredentials(clusterSecret.Values)

	activityLogClient, err := azure.NewActivityLogsClient(creds)

	if err != nil {
		return nil, emperror.Wrap(err, "failed to create activity log client")
	}

	c.log.Debug("query activity log with filter: ", filter)

	result, err := activityLogClient.List(
		context.TODO(),
		filter,
		"")

	if err != nil {
		return nil, emperror.Wrap(err, "failed to query activity log")
	}

	var errEvents []insights.EventData

	for result.NotDone() {
		events := result.Values()
		for _, event := range events {
			if (event.Level == insights.Critical || event.Level == insights.Error) && *event.Status.LocalizedValue == "Failed" {
				errEvents = append(errEvents, event)
			}
		}

		err := result.Next()
		if err != nil {
			return nil, emperror.Wrap(err, "failed to get next activity log page")
		}
	}

	return errEvents, nil
}
