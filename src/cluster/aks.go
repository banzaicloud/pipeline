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
	"regexp"
	"strings"
	"time"

	"emperror.dev/errors"
	"github.com/Azure/azure-sdk-for-go/services/containerservice/mgmt/2020-02-01/containerservice"
	"github.com/Azure/azure-sdk-for-go/services/preview/monitor/mgmt/2018-09-01/insights"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-02-01/resources"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/sirupsen/logrus"

	"github.com/banzaicloud/pipeline/internal/global"
	internalAzure "github.com/banzaicloud/pipeline/internal/providers/azure"
	"github.com/banzaicloud/pipeline/internal/providers/azure/azureadapter"
	"github.com/banzaicloud/pipeline/internal/secret/ssh"
	"github.com/banzaicloud/pipeline/internal/secret/ssh/sshadapter"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	pkgClusterAzure "github.com/banzaicloud/pipeline/pkg/cluster/aks"
	pkgCommon "github.com/banzaicloud/pipeline/pkg/common"
	pkgErrors "github.com/banzaicloud/pipeline/pkg/errors"
	pkgAzure "github.com/banzaicloud/pipeline/pkg/providers/azure"
	"github.com/banzaicloud/pipeline/src/model"
	"github.com/banzaicloud/pipeline/src/secret"
	"github.com/banzaicloud/pipeline/src/utils"
)

// nolint: gochecknoglobals
var (
	ErrNoInfrastructureRG = errors.New("no infrastructure resource group found")
)

// AKSCluster represents an AKS cluster
type AKSCluster struct {
	CommonClusterBase
	modelCluster *model.ClusterModel
	log          logrus.FieldLogger
}

// CreateAKSClusterFromRequest returns an AKS cluster instance created from the specified request
func CreateAKSClusterFromRequest(request *pkgCluster.CreateClusterRequest, orgID uint, userID uint) (*AKSCluster, error) {
	var nodePools = make([]*azureadapter.AKSNodePoolModel, 0, len(request.Properties.CreateClusterAKS.NodePools))
	for name, np := range request.Properties.CreateClusterAKS.NodePools {
		nodePools = append(nodePools, &azureadapter.AKSNodePoolModel{
			CreatedBy:        userID,
			Name:             name,
			Autoscaling:      np.Autoscaling,
			NodeMinCount:     np.MinCount,
			NodeMaxCount:     np.MaxCount,
			Count:            np.Count,
			NodeInstanceType: np.NodeInstanceType,
			VNetSubnetID:     np.VNetSubnetID,
			Labels:           np.Labels,
		})
	}

	var cluster AKSCluster

	cluster.modelCluster = &model.ClusterModel{
		Name:           request.Name,
		Location:       request.Location,
		Cloud:          request.Cloud,
		OrganizationId: orgID,
		CreatedBy:      userID,
		SecretId:       request.SecretId,
		Distribution:   pkgCluster.AKS,
		AKS: azureadapter.AKSClusterModel{
			ResourceGroup:     request.Properties.CreateClusterAKS.ResourceGroup,
			KubernetesVersion: request.Properties.CreateClusterAKS.KubernetesVersion,
			NodePools:         nodePools,
		},
	}

	cluster.log = log.WithField("cluster", request.Name)

	updateScaleOptions(&cluster.modelCluster.ScaleOptions, request.ScaleOptions)
	return &cluster, nil
}

type aksClusterCreateOrUpdateFailedError struct {
	clusterCreateUpdateError error
	failedEventsMsg          []string
}

func (e aksClusterCreateOrUpdateFailedError) Error() string {
	if len(e.failedEventsMsg) > 0 {
		return e.clusterCreateUpdateError.Error() + "\n" + strings.Join(e.failedEventsMsg, "\n")
	}

	return e.clusterCreateUpdateError.Error()
}

func (e aksClusterCreateOrUpdateFailedError) Cause() error {
	return e.clusterCreateUpdateError
}

func createClusterCreateOrUpdateFailedError(createOrUpdateError error, errorEvents []insights.EventData) error {
	if len(errorEvents) > 0 {
		var failedEventsMsg []string

		for _, event := range errorEvents {
			if msg, ok := event.Properties["statusMessage"]; ok {
				failedEventsMsg = append(failedEventsMsg, *msg)
			}
		}

		return aksClusterCreateOrUpdateFailedError{
			clusterCreateUpdateError: createOrUpdateError,
			failedEventsMsg:          failedEventsMsg,
		}
	}

	return createOrUpdateError
}

func (*AKSCluster) getEnvironment() *azure.Environment {
	return &azure.PublicCloud // TODO: this should come from the cluster model
}

// GetLocation returns the location of the cluster
func (c *AKSCluster) GetLocation() string {
	return c.modelCluster.Location
}

// GetOrganizationId returns the ID of the organization where the cluster belongs to
func (c *AKSCluster) GetOrganizationId() uint {
	return c.modelCluster.OrganizationId
}

// GetSecretId returns the cluster secret's ID
func (c *AKSCluster) GetSecretId() string {
	return c.modelCluster.SecretId
}

func (c *AKSCluster) getCloudConnection() (*pkgAzure.CloudConnection, error) {
	creds, err := c.getCredentials()
	if err != nil {
		return nil, errors.WrapIf(err, "failed to retrieve AKS credentials")
	}
	return pkgAzure.NewCloudConnection(c.getEnvironment(), creds)
}

// GetAPIEndpoint returns the Kubernetes API endpoint
func (c *AKSCluster) GetAPIEndpoint() (string, error) {
	cluster, err := c.getAzureCluster()
	if err != nil {
		return "", errors.WrapIf(err, "failed to get AKS cluster from Azure")
	}
	return *cluster.Fqdn, nil
}

func (c *AKSCluster) getNewSSHKeyPair() (ssh.KeyPair, error) {
	clusterSSHSecret, err := c.getSshSecret(c)
	if err != nil {
		return ssh.KeyPair{}, errors.WrapIf(err, "failed to retrieve SSH secret")
	}
	return sshadapter.KeyPairFromSecret(clusterSSHSecret), nil
}

func isProvisioningSuccessful(cluster *containerservice.ManagedCluster) bool {
	return *cluster.ProvisioningState == "Succeeded"
}

func getVNetSubnetID(np *azureadapter.AKSNodePoolModel) *string {
	if len(np.VNetSubnetID) == 0 {
		return nil
	}
	return &np.VNetSubnetID
}

// CreateCluster creates a new cluster
func (c *AKSCluster) CreateCluster() error {
	c.log.Info("Creating cluster...")

	cc, err := c.getCloudConnection()
	if err != nil {
		return errors.WrapIf(err, "failed to get cloud connection")
	}

	var profiles []containerservice.ManagedClusterAgentPoolProfile
	for _, np := range c.modelCluster.AKS.NodePools {
		if np != nil {
			if err := validateVNetSubnet(cc, c.GetResourceGroupName(), np.VNetSubnetID); err != nil {
				return errors.WrapIf(err, "virtual network subnet validation failed")
			}
			count := int32(np.Count)
			name := np.Name
			profiles = append(profiles, containerservice.ManagedClusterAgentPoolProfile{
				Name:         &name,
				Count:        &count,
				VMSize:       containerservice.VMSizeTypes(np.NodeInstanceType),
				VnetSubnetID: getVNetSubnetID(np),
				NodeLabels: map[string]*string{
					pkgCommon.LabelKey: &name,
				},
			})
		}
	}

	c.modelCluster.RbacEnabled = true

	sshKey, err := c.getNewSSHKeyPair()
	if err != nil {
		return errors.WrapIf(err, "failed to get new SSH key pair")
	}
	c.log.Debug("successfully created new SSH keys")
	creds, err := c.getCredentials()
	if err != nil {
		return errors.WrapIf(err, "failed to retrieve AKS credentials")
	}
	c.log.Debug("successfully retrieved credentials")
	dnsPrefix := "dnsprefix"
	adminUsername := "pipeline"
	params := &containerservice.ManagedCluster{
		Name:     &c.modelCluster.Name,
		Location: &c.modelCluster.Location,
		ManagedClusterProperties: &containerservice.ManagedClusterProperties{
			DNSPrefix:         &dnsPrefix,
			KubernetesVersion: &c.modelCluster.AKS.KubernetesVersion,
			EnableRBAC:        &c.modelCluster.RbacEnabled,
			AgentPoolProfiles: &profiles,
			LinuxProfile: &containerservice.LinuxProfile{
				AdminUsername: &adminUsername,
				SSH: &containerservice.SSHConfiguration{
					PublicKeys: &[]containerservice.SSHPublicKey{
						{
							KeyData: &sshKey.PublicKeyData,
						},
					},
				},
			},
			ServicePrincipalProfile: &containerservice.ManagedClusterServicePrincipalProfile{
				ClientID: &creds.ClientID,
				Secret:   &creds.ClientSecret,
			},
		},
		Tags: internalAzure.PipelineTags(),
	}

	c.log.Info("Sending cluster creation request to AKS and waiting for completion")
	clusterCreateInitTime := time.Now()
	cluster, err := cc.GetManagedClustersClient().CreateOrUpdateAndWaitForIt(context.TODO(), c.GetResourceGroupName(), c.GetName(), params)
	if err != nil {
		return errors.WrapIf(err, "failed to create cluster")
	}
	if !isProvisioningSuccessful(cluster) {
		return c.onClusterCreateFailure(err, clusterCreateInitTime)
	}
	c.log.Info("Cluster ready")

	return nil
}

// Storage Account Contributor role constant
const (
	StorageAccountContributor = "Storage Account Contributor"
)

func (c *AKSCluster) getInfrastructureResourceGroupName() string {
	return makeInfrastructureResourceGroupName(c.GetResourceGroupName(), c.GetName(), c.GetLocation())
}

func makeInfrastructureResourceGroupName(resourceGroupName, clusterName, location string) string {
	return fmt.Sprintf("MC_%s_%s_%s", resourceGroupName, clusterName, location)
}

// Persist saves the cluster model
// Deprecated: Do not use.
func (c *AKSCluster) Persist() error {
	return errors.WrapIf(c.modelCluster.Save(), "failed to persist cluster")
}

// GetResourceGroupName return the resource group's name the cluster belongs in
func (c *AKSCluster) GetResourceGroupName() string {
	return c.modelCluster.AKS.ResourceGroup
}

func (c *AKSCluster) loadAKSClusterModelFromDB() {
	database := global.DB()
	database.Where(azureadapter.AKSClusterModel{ID: c.GetID()}).First(&c.modelCluster.AKS)
}

// DownloadK8sConfig returns the kubeconfig file's contents from AKS
func (c *AKSCluster) DownloadK8sConfig() ([]byte, error) {
	cc, err := c.getCloudConnection()
	if err != nil {
		return nil, errors.WrapIf(err, "failed to get cloud connection")
	}

	c.loadAKSClusterModelFromDB()

	// TODO check banzairesponses
	roleName := "clusterUser"
	c.log.Infof("Get %s cluster's config in %s, role name: %s", c.GetName(), c.GetResourceGroupName(), roleName)
	profile, err := cc.GetManagedClustersClient().GetAccessProfile(context.TODO(), c.GetResourceGroupName(), c.GetName(), roleName)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to get access profile")
	}
	if profile.KubeConfig == nil {
		c.log.Debug("K8s config not set in access profile")
		return nil, nil
	}
	c.log.Info("Successfully retrieved k8s config")
	return *profile.KubeConfig, nil
}

// GetName returns the name of the cluster
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

// GetStatus returns the cluster's status
func (c *AKSCluster) GetStatus() (*pkgCluster.GetClusterStatusResponse, error) {
	// c.log.Info("Create cluster status response")

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
				Labels:            np.Labels,
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
		CreatorBaseFields: *NewCreatorBaseFields(c.modelCluster.CreatedAt, c.modelCluster.CreatedBy),
		NodePools:         nodePools,
		Region:            c.modelCluster.Location,
		StartedAt:         c.modelCluster.StartedAt,
	}, nil
}

// DeleteCluster deletes the cluster from AKS
func (c *AKSCluster) DeleteCluster() error {
	cc, err := c.getCloudConnection()
	if err != nil {
		return errors.WrapIf(err, "failed to get cloud connection")
	}

	c.loadAKSClusterModelFromDB()

	err = cc.GetManagedClustersClient().DeleteAndWaitForIt(context.TODO(), c.GetResourceGroupName(), c.GetName())
	if err != nil {
		return errors.WrapIf(err, "cluster deletion request failed")
	}
	c.log.Info("Delete succeeded")
	return nil
}

func getAgentPoolProfileByName(cluster *containerservice.ManagedCluster, name string) *containerservice.ManagedClusterAgentPoolProfile {
	if cluster.AgentPoolProfiles != nil {
		for idx, app := range *cluster.AgentPoolProfiles {
			if *app.Name == name {
				return &(*cluster.AgentPoolProfiles)[idx]
			}
		}
	}
	return nil
}

// UpdateCluster updates the cluster in AKS
func (c *AKSCluster) UpdateCluster(request *pkgCluster.UpdateClusterRequest, userID uint) error {
	cc, err := c.getCloudConnection()
	if err != nil {
		return errors.WrapIf(err, "failed to get cloud connection")
	}
	client := cc.GetManagedClustersClient()

	cluster, err := c.getAzureCluster()
	if err != nil {
		return errors.WrapIf(err, "failed to retrieve AKS cluster")
	}

	for name, np := range request.AKS.NodePools {
		log := c.log.WithField("nodePool", name)
		// Azure does not allow to create or delete pools when updating, only existing pools' properties can be changed
		if app := getAgentPoolProfileByName(cluster, name); np != nil && app != nil {
			count := int32(np.Count)
			app.Count = &count
		} else {
			log.Warning("No such nodepool found")
		}
	}

	c.log.Info("Sending cluster update request to AKS and waiting for completion")
	clusterUpdateInitTime := time.Now()
	cluster, err = client.CreateOrUpdateAndWaitForIt(context.TODO(), c.GetResourceGroupName(), c.GetName(), cluster)
	if err != nil {
		return errors.WrapIf(err, "cluster update request failed")
	}
	if !isProvisioningSuccessful(cluster) {
		return c.onClusterUpdateFailure(err, clusterUpdateInitTime)
	}

	for name, np := range request.AKS.NodePools {
		if np != nil {
			app := getAgentPoolProfileByName(cluster, name)
			npm := c.getNodePoolByName(name)
			if app != nil && npm != nil {
				npm.CreatedAt = clusterUpdateInitTime
				npm.CreatedBy = userID
				npm.Autoscaling = np.Autoscaling
				npm.NodeMinCount = np.MinCount
				npm.NodeMaxCount = np.MaxCount
				npm.Count = int(*app.Count)
			}
		}
	}

	return nil
}

// UpdateNodePools updates nodes pools of a cluster
func (c *AKSCluster) UpdateNodePools(request *pkgCluster.UpdateNodePoolsRequest, userID uint) error {
	cc, err := c.getCloudConnection()
	if err != nil {
		return errors.WrapIf(err, "failed to get cloud connection")
	}
	client := cc.GetManagedClustersClient()

	cluster, err := c.getAzureCluster()
	if err != nil {
		return errors.WrapIf(err, "failed to retrieve AKS cluster")
	}

	for name, np := range request.NodePools {
		log := c.log.WithField("nodePool", name)
		// Azure does not allow to create or delete pools when updating, only existing pools' properties can be changed
		if app := getAgentPoolProfileByName(cluster, name); np != nil && app != nil {
			count := int32(np.Count)
			app.Count = &count
		} else {
			log.Warning("No such nodepool found")
		}
	}

	c.log.Info("Sending cluster update request to AKS and waiting for completion")
	clusterUpdateInitTime := time.Now()
	cluster, err = client.CreateOrUpdateAndWaitForIt(context.TODO(), c.GetResourceGroupName(), c.GetName(), cluster)
	if err != nil {
		return errors.WrapIf(err, "cluster update request failed")
	}
	if !isProvisioningSuccessful(cluster) {
		return c.onClusterUpdateFailure(err, clusterUpdateInitTime)
	}

	for name, np := range request.NodePools {
		if np != nil {
			app := getAgentPoolProfileByName(cluster, name)
			npm := c.getNodePoolByName(name)
			if app != nil && npm != nil {
				npm.CreatedAt = clusterUpdateInitTime
				npm.CreatedBy = userID
				npm.Count = int(*app.Count)
			}
		}
	}

	return nil
}

// getNodePoolByName returns saved NodePool by name
func (c *AKSCluster) getNodePoolByName(name string) *azureadapter.AKSNodePoolModel {
	for _, nodePool := range c.modelCluster.AKS.NodePools {
		if nodePool != nil && nodePool.Name == name {
			return nodePool
		}
	}
	return nil
}

// GetID returns the cluster's ID
func (c *AKSCluster) GetID() uint {
	return c.modelCluster.ID
}

// GetUID returns the cluster's UID
func (c *AKSCluster) GetUID() string {
	return c.modelCluster.UID
}

// GetModel returns the cluster's model
func (c *AKSCluster) GetModel() *model.ClusterModel {
	return c.modelCluster
}

// getAzureCluster returns cluster from cloud
func (c *AKSCluster) getAzureCluster() (*containerservice.ManagedCluster, error) {
	cc, err := c.getCloudConnection()
	if err != nil {
		return nil, errors.WrapIf(err, "failed to create cloud connection")
	}
	cluster, err := cc.GetManagedClustersClient().Get(context.TODO(), c.GetResourceGroupName(), c.GetName())
	if err != nil {
		return nil, errors.WrapIf(err, "failed to get managed cluster")
	}
	return &cluster, nil
}

// CreateAKSClusterFromModel creates ClusterModel struct from model
func CreateAKSClusterFromModel(clusterModel *model.ClusterModel) *AKSCluster {
	return &AKSCluster{
		modelCluster: clusterModel,
		log:          log.WithField("cluster", clusterModel.Name),
	}
}

// AddDefaultsToUpdate adds defaults to update request
func (c *AKSCluster) AddDefaultsToUpdate(r *pkgCluster.UpdateClusterRequest) {
	if r.AKS == nil {
		c.log.Warn("'aks' field is empty.")
		r.AKS = &pkgClusterAzure.UpdateClusterAzure{}
	}

	if len(r.AKS.NodePools) == 0 {
		nodePools := make(map[string]*pkgClusterAzure.NodePoolUpdate)
		for _, np := range c.modelCluster.AKS.NodePools {
			if np != nil {
				nodePools[np.Name] = &pkgClusterAzure.NodePoolUpdate{
					Autoscaling: np.Autoscaling,
					MinCount:    np.NodeMinCount,
					MaxCount:    np.NodeMaxCount,
					Count:       np.Count,
				}
			}
		}
		r.AKS.NodePools = nodePools
	}
}

// CheckEqualityToUpdate validates the update request
func (c *AKSCluster) CheckEqualityToUpdate(r *pkgCluster.UpdateClusterRequest) error {
	// create update request struct with the stored data to check equality
	preProfiles := make(map[string]*pkgClusterAzure.NodePoolUpdate)

	for _, preP := range c.modelCluster.AKS.NodePools {
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

	c.log.Debug("Check stored & updated cluster equals")

	// check equality
	return isDifferent(r.AKS, preCl)
}

// DeleteFromDatabase deletes model from the database
func (c *AKSCluster) DeleteFromDatabase() error {
	err := c.modelCluster.Delete()
	if err != nil {
		return errors.WrapIf(err, "failed to delete cluster model")
	}
	c.modelCluster = nil
	return nil
}

func (c *AKSCluster) getCredentials() (*pkgAzure.Credentials, error) {
	clusterSecret, err := c.GetSecretWithValidation()
	if err != nil {
		return nil, errors.WrapIf(err, "failed to retrieve AKS secret")
	}
	return pkgAzure.NewCredentials(clusterSecret.Values), nil
}

func getAzureCredentials(orgID uint, secretID string) (*pkgAzure.Credentials, error) {
	sir, err := getSecret(orgID, secretID)
	if err != nil {
		return nil, errors.WrapIfWithDetails(err, "failed to retrieve secret", "orgID", orgID, "secretID", secretID)
	}

	if err := secret.ValidateSecretType(sir, pkgCluster.Azure); err != nil {
		return nil, errors.WrapIf(err, "failed to validate secret type")
	}
	return pkgAzure.NewCredentials(sir.Values), nil
}

func getDefaultCloudConnection(orgID uint, secretID string) (*pkgAzure.CloudConnection, error) {
	creds, err := getAzureCredentials(orgID, secretID)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to retrieve Azure credentials")
	}
	return pkgAzure.NewCloudConnection(&azure.PublicCloud, creds)
}

// GetLocations returns all the locations that are available for resource providers
func GetLocations(orgID uint, secretID string) (locations []string, err error) {
	cc, err := getDefaultCloudConnection(orgID, secretID)
	if err != nil {
		return
	}
	res, err := cc.GetSubscriptionsClient().ListLocations(context.TODO(), cc.GetSubscriptionID())
	if err != nil {
		return
	}
	if res.Value == nil {
		return
	}
	locations = make([]string, 0, len(*res.Value))
	for _, loc := range *res.Value {
		locations = append(locations, *loc.Name)
	}
	return
}

// GetMachineTypes lists all available virtual machine sizes for a subscription in a location
func GetMachineTypes(orgID uint, secretID string, location string) (pkgCluster.MachineTypes, error) {
	cc, err := getDefaultCloudConnection(orgID, secretID)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to get cloud connection")
	}
	return cc.GetVirtualMachineSizesClient().ListMachineTypes(context.TODO(), location)
}

// GetKubernetesVersion returns a list of supported kubernetes version in the specified subscription
func GetKubernetesVersion(orgID uint, secretID string, location string) ([]string, error) {
	cc, err := getDefaultCloudConnection(orgID, secretID)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to get cloud connection")
	}
	return cc.GetContainerServicesClient().ListKubernetesVersions(context.TODO(), location)
}

// SetStatus sets the cluster's status
func (c *AKSCluster) SetStatus(status, statusMessage string) error {
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

// IsReady checks if the cluster is running according to the cloud provider
func (c *AKSCluster) IsReady() (bool, error) {
	cluster, err := c.getAzureCluster()
	if err != nil {
		return false, errors.WrapIf(err, "failed to retrieve AKS cluster")
	}

	c.log.Debugf("Cluster provisioning state is: %s", *cluster.ProvisioningState)

	return isProvisioningSuccessful(cluster), nil
}

// ValidateCreationFields validates all field
func (c *AKSCluster) ValidateCreationFields(r *pkgCluster.CreateClusterRequest) error {
	location := r.Location

	// Validate location
	c.log.Debug("Validate location")
	if err := c.validateLocation(location); err != nil {
		return errors.WrapIf(err, "failed to validate location")
	}
	c.log.Debug("Validate location passed")

	// Validate machine types
	nodePools := r.Properties.CreateClusterAKS.NodePools
	c.log.Debug("Validate nodePools")
	if err := c.validateMachineType(nodePools, location); err != nil {
		return errors.WrapIf(err, "failed to validate machine type")
	}
	c.log.Debug("Validate nodePools passed")

	// Validate kubernetes version
	c.log.Debug("Validate kubernetesVersion")
	k8sVersion := r.Properties.CreateClusterAKS.KubernetesVersion
	if err := c.validateKubernetesVersion(k8sVersion, location); err != nil {
		return errors.WrapIf(err, "failed to validate k8s version")
	}
	c.log.Debug("Validate kubernetesVersion passed")

	return nil
}

// validateLocation validates location
func (c *AKSCluster) validateLocation(location string) error {
	c.log.Debugln("Location:", location)
	validLocations, err := GetLocations(c.GetOrganizationId(), c.GetSecretId())
	if err != nil {
		return errors.WrapIf(err, "could not get locations from Azure")
	}

	c.log.Debugf("Valid locations: %#v", validLocations)

	if isOk := utils.Contains(validLocations, location); !isOk {
		return pkgErrors.ErrorNotValidLocation
	}

	return nil
}

// validateMachineType validates nodeInstanceTypes
func (c *AKSCluster) validateMachineType(nodePools map[string]*pkgClusterAzure.NodePoolCreate, location string) error {
	validMachineTypes, err := GetMachineTypes(c.GetOrganizationId(), c.GetSecretId(), location)
	if err != nil {
		return errors.WrapIfWithDetails(err, "could not get VM types from Azure", "location", location)
	}
	c.log.Debugf("Valid NodeInstanceTypes: %v", validMachineTypes)

	validMachineTypeLT := make(map[string]bool)
	for _, mt := range validMachineTypes {
		validMachineTypeLT[mt] = true
	}

	for _, nodePool := range nodePools {
		if nodePool != nil && !validMachineTypeLT[nodePool.NodeInstanceType] {
			return pkgErrors.ErrorNotValidNodeInstanceType // TODO should include the invalid type in the error
		}
	}

	return nil
}

// validateKubernetesVersion validates k8s version
func (c *AKSCluster) validateKubernetesVersion(k8sVersion, location string) error {
	c.log.Debugln("K8SVersion:", k8sVersion)
	validVersions, err := GetKubernetesVersion(c.GetOrganizationId(), c.GetSecretId(), location)
	if err != nil {
		return errors.WrapIf(err, "failed to get k8s version")
	}
	c.log.Debugln("Valid K8SVersions:", validVersions)

	if isOk := utils.Contains(validVersions, k8sVersion); !isOk {
		return pkgErrors.ErrorNotValidKubernetesVersion
	}

	return nil
}

// GetSecretWithValidation returns secret from vault
func (c *AKSCluster) GetSecretWithValidation() (*secret.SecretItemResponse, error) {
	return c.CommonClusterBase.getSecret(c)
}

// SaveConfigSecretId saves the config secret ID in database
func (c *AKSCluster) SaveConfigSecretId(configSecretID string) error {
	return c.modelCluster.UpdateConfigSecret(configSecretID)
}

// GetConfigSecretId returns the cluster's config secret ID
func (c *AKSCluster) GetConfigSecretId() string {
	return c.modelCluster.ConfigSecretId
}

// GetSSHSecretID returns the cluster's SSH secret ID
func (c *AKSCluster) GetSshSecretId() string {
	return c.modelCluster.SshSecretId
}

// SaveSshSecretId saves the SSH secret ID to database
func (c *AKSCluster) SaveSshSecretId(sshSecretID string) error {
	c.log.Debugf("Saving SSH secret ID [%s]", sshSecretID)
	return c.modelCluster.UpdateSshSecret(sshSecretID)
}

// GetK8sConfig returns the Kubernetes config
func (c *AKSCluster) GetK8sConfig() ([]byte, error) {
	return c.CommonClusterBase.getConfig(c)
}

// GetK8sUserConfig returns the Kubernetes config for external users
func (c *AKSCluster) GetK8sUserConfig() ([]byte, error) {
	return c.GetK8sConfig()
}

// RequiresSshPublicKey returns true if a public SSH key is needed for bootstrapping the cluster
func (c *AKSCluster) RequiresSshPublicKey() bool {
	return true
}

// RbacEnabled returns true if rbac enabled on the cluster
func (c *AKSCluster) RbacEnabled() bool {
	return c.modelCluster.RbacEnabled
}

// GetScaleOptions returns scale options for the cluster
func (c *AKSCluster) GetScaleOptions() *pkgCluster.ScaleOptions {
	return getScaleOptionsFromModel(c.modelCluster.ScaleOptions)
}

// SetScaleOptions sets scale options for the cluster
func (c *AKSCluster) SetScaleOptions(scaleOptions *pkgCluster.ScaleOptions) {
	updateScaleOptions(&c.modelCluster.ScaleOptions, scaleOptions)
}

// ListResourceGroups returns all resource group
func ListResourceGroups(orgID uint, secretID string) (res []string, err error) {
	cc, err := getDefaultCloudConnection(orgID, secretID)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to get cloud connection")
	}
	groups, err := cc.GetGroupsClient().ListAll(context.TODO(), "", nil)
	if err != nil {
		return nil, errors.WrapIf(err, "could not get resource groups from Azure")
	}
	res = make([]string, 0, len(groups))
	for _, g := range groups {
		if g.Name != nil {
			res = append(res, *g.Name)
		}
	}
	return
}

// CreateOrUpdateResourceGroup creates or updates a resource group
func CreateOrUpdateResourceGroup(orgID uint, secretID string, resourceGroupName, location string) error {
	cc, err := getDefaultCloudConnection(orgID, secretID)
	if err != nil {
		return errors.WrapIf(err, "failed to get cloud connection")
	}
	_, err = cc.GetGroupsClient().CreateOrUpdate(context.TODO(), resourceGroupName, resources.Group{
		Location: &location,
	}) // TODO should we wait for it?
	return errors.WrapIf(err, "failed to create or update resource group")
}

// GetAKSNodePools returns AKS node pools from a common cluster.
func GetAKSNodePools(cluster CommonCluster) ([]*azureadapter.AKSNodePoolModel, error) {
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

func (c *AKSCluster) onClusterCreateFailure(createError error, operationStartTime time.Time) error {
	// collect error details from activity log

	creds, err := c.getCredentials()
	if err != nil {
		return errors.WrapIf(err, "failed to retrieve AKS credentials")
	}

	clusterResourceURI := fmt.Sprintf("/subscriptions/%s/resourcegroups/%s/providers/Microsoft.ContainerService/managedClusters/%s",
		creds.SubscriptionID,
		c.GetResourceGroupName(),
		c.GetName(),
	)

	toTimeStamp := time.Now()

	filter := fmt.Sprintf("eventTimestamp ge '%s' and eventTimestamp le '%s' and resourceUri eq '%s'",
		operationStartTime.UTC().Format(time.RFC3339Nano),
		toTimeStamp.UTC().Format(time.RFC3339Nano),
		clusterResourceURI,
	)

	errorEvents, err := c.collectActivityLogsWithErrors(filter)
	if err != nil {
		c.log.Errorln("retrieving activity logs failed: ", err.Error())
		return createError
	}

	return createClusterCreateOrUpdateFailedError(createError, errorEvents)
}

func (c *AKSCluster) onClusterUpdateFailure(createUpdateError error, operationStartTime time.Time) error {
	// collect error details from activity log
	toTimeStamp := time.Now()

	filter := fmt.Sprintf("eventTimestamp ge '%s' and eventTimestamp le '%s' and resourceGroupName eq '%s'",
		operationStartTime.UTC().Format(time.RFC3339Nano),
		toTimeStamp.UTC().Format(time.RFC3339Nano),
		c.getInfrastructureResourceGroupName(),
	)

	errorEvents, err := c.collectActivityLogsWithErrors(filter)
	if err != nil {
		c.log.Errorln("retrieving activity logs failed: ", err.Error())
		return createUpdateError
	}

	return createClusterCreateOrUpdateFailedError(createUpdateError, errorEvents)
}

// collectActivityLogsWithErrors collects cluster activity logs that denotes errors and matches the passed filter
func (c *AKSCluster) collectActivityLogsWithErrors(filter string) ([]insights.EventData, error) {
	cc, err := c.getCloudConnection()
	if err != nil {
		return nil, errors.WrapIf(err, "failed to create cloud connection")
	}

	activityLogClient := cc.GetActivityLogsClient()

	c.log.Debugln("query activity log with filter:", filter)

	result, err := activityLogClient.List(
		context.TODO(),
		filter,
		"")

	if err != nil {
		return nil, errors.WrapIf(err, "failed to query activity log")
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
			return nil, errors.WrapIf(err, "failed to get next activity log page")
		}
	}

	return errEvents, nil
}

// nolint: gochecknoglobals
var vnetSubnetIDRegexp = regexp.MustCompile("/subscriptions/([^/]+)/resourceGroups/([^/]+)/providers/Microsoft.Network/virtualNetworks/([^/]+)/subnets/([^/]+)")

func validateVNetSubnet(cc *pkgAzure.CloudConnection, resourceGroupName, vnetSubnetID string) error {
	if vnetSubnetID != "" {
		matches := vnetSubnetIDRegexp.FindStringSubmatch(vnetSubnetID)
		if matches == nil {
			return errors.New("virtual network subnet ID format is invalid")
		}
		if matches[1] != cc.GetSubscriptionID() {
			return errors.New("virtual network subnet is not from same subscription")
		}
		if matches[2] != resourceGroupName {
			return errors.New("virtual network subnet is not from same resource group")
		}
		_, err := cc.GetSubnetsClient().Get(context.TODO(), matches[2], matches[3], matches[4], "")
		if err != nil {
			return errors.WrapIf(err, "request to retrieve subnet failed")
		}
	}
	return nil
}

func (c *AKSCluster) GetKubernetesVersion() (string, error) {
	return c.modelCluster.AKS.KubernetesVersion, nil
}
