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
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"net"
	"strconv"
	"time"

	"emperror.dev/errors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/jinzhu/gorm"
	"github.com/mitchellh/mapstructure"
	"github.com/sirupsen/logrus"
	"go.uber.org/cadence/client"

	"github.com/banzaicloud/pipeline/internal/cluster/clusteradapter"
	"github.com/banzaicloud/pipeline/internal/cluster/clusteradapter/clustermodel"
	"github.com/banzaicloud/pipeline/internal/global"
	internalPke "github.com/banzaicloud/pipeline/internal/providers/pke"
	"github.com/banzaicloud/pipeline/internal/providers/pke/pkeworkflow"
	"github.com/banzaicloud/pipeline/internal/secret/secrettype"
	"github.com/banzaicloud/pipeline/internal/secret/ssh/sshadapter"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	"github.com/banzaicloud/pipeline/pkg/cluster/pke"
	"github.com/banzaicloud/pipeline/pkg/common"
	"github.com/banzaicloud/pipeline/pkg/providers/amazon"
	pkgEC2 "github.com/banzaicloud/pipeline/pkg/providers/amazon/ec2"
	"github.com/banzaicloud/pipeline/src/model"
	"github.com/banzaicloud/pipeline/src/secret"
)

const defaultK8sVersion = "1.15.3"

var _ CommonCluster = (*EC2ClusterPKE)(nil)

type EC2ClusterPKE struct {
	db    *gorm.DB
	model *internalPke.EC2PKEClusterModel
	// amazonCluster *ec2.EC2 //Don't use this directly
	APIEndpoint string
	log         logrus.FieldLogger
	session     *session.Session
	CommonClusterBase
}

// GetScaleOptions returns scale options for the cluster
func (c *EC2ClusterPKE) GetScaleOptions() *pkgCluster.ScaleOptions {
	return getScaleOptionsFromModel(c.model.Cluster.ScaleOptions)
}

// SetScaleOptions sets scale options for the cluster
func (c *EC2ClusterPKE) SetScaleOptions(scaleOptions *pkgCluster.ScaleOptions) {
	updateScaleOptions(&c.model.Cluster.ScaleOptions, scaleOptions)
}

func (c *EC2ClusterPKE) GetID() uint {
	return c.model.Cluster.ID
}

func (c *EC2ClusterPKE) GetUID() string {
	return c.model.Cluster.UID
}

func (c *EC2ClusterPKE) GetOrganizationId() uint {
	return c.model.Cluster.OrganizationID
}

func (c *EC2ClusterPKE) GetName() string {
	return c.model.Cluster.Name
}

func (c *EC2ClusterPKE) GetCloud() string {
	return c.model.Cluster.Cloud
}

func (c *EC2ClusterPKE) GetDistribution() string {
	return c.model.Cluster.Distribution
}

func (c *EC2ClusterPKE) GetLocation() string {
	return c.model.Cluster.Location
}

func (c *EC2ClusterPKE) GetSecretId() string {
	return c.model.Cluster.SecretID
}

func (c *EC2ClusterPKE) GetSshSecretId() string {
	return c.model.Cluster.SSHSecretID
}

// RequiresSshPublicKey returns true as a public ssh key is needed for bootstrapping
// the cluster
func (c *EC2ClusterPKE) RequiresSshPublicKey() bool {
	return true
}

func (c *EC2ClusterPKE) SaveSshSecretId(sshSecretId string) error {
	c.model.Cluster.SSHSecretID = sshSecretId

	err := c.db.Save(&c.model).Error
	if err != nil {
		return errors.WrapIfWithDetails(err, "failed to save ssh secret", "secret", sshSecretId)
	}

	return nil
}

func (c *EC2ClusterPKE) GetSshPublicKey() (string, error) {
	sshSecret, err := c.getSshSecret(c)
	if err != nil {
		return "", err
	}
	sshKey := sshadapter.KeyPairFromSecret(sshSecret)
	return sshKey.PublicKeyData, nil
}

func (c *EC2ClusterPKE) SaveConfigSecretId(configSecretId string) error {
	c.model.Cluster.ConfigSecretID = configSecretId

	err := c.db.Save(&c.model).Error
	if err != nil {
		return errors.Wrap(err, "failed to save config secret id")
	}

	return nil
}

func (c *EC2ClusterPKE) GetConfigSecretId() string {
	clusters := clusteradapter.NewClusters(global.DB()) // TODO get it from non-global context
	id, err := clusters.GetConfigSecretIDByClusterID(c.GetOrganizationId(), c.GetID())
	if err == nil {
		c.model.Cluster.ConfigSecretID = id
	}
	return c.model.Cluster.ConfigSecretID
}

func (c *EC2ClusterPKE) GetSecretWithValidation() (*secret.SecretItemResponse, error) {
	return c.CommonClusterBase.getSecret(c)
}

func (c *EC2ClusterPKE) Persist() error {
	return errors.WrapIf(c.db.Save(c.model).Error, "failed to persist cluster")
}

// SetStatus sets the cluster's status
func (c *EC2ClusterPKE) SetStatus(status, statusMessage string) error {
	if c.model.Cluster.Status == status && c.model.Cluster.StatusMessage == statusMessage {
		return nil
	}

	if c.model.Cluster.ID != 0 {
		// Record status change to history before modifying the actual status.
		// If setting/saving the actual status doesn't succeed somehow, at least we can reconstruct it from history (i.e. event sourcing).
		statusHistory := clustermodel.StatusHistoryModel{
			ClusterID:   c.model.Cluster.ID,
			ClusterName: c.model.Cluster.Name,

			FromStatus:        c.model.Cluster.Status,
			FromStatusMessage: c.model.Cluster.StatusMessage,
			ToStatus:          status,
			ToStatusMessage:   statusMessage,
		}

		if err := c.db.Save(&statusHistory).Error; err != nil {
			return errors.Wrap(err, "failed to record cluster status change to history")
		}
	}

	now := time.Now()
	justStarted := c.model.Cluster.Status == pkgCluster.Creating && (status == pkgCluster.Running || status == pkgCluster.Warning)

	updateFields := map[string]interface{}{"status": status, "status_message": statusMessage}
	if justStarted {
		updateFields["started_at"] = &now
	}

	if err := c.db.Model(&c.model.Cluster).Updates(updateFields).Error; err != nil {
		return errors.Wrap(err, "failed to update cluster status")
	}

	c.model.Cluster.Status = status
	c.model.Cluster.StatusMessage = statusMessage
	if justStarted {
		c.model.Cluster.StartedAt = &now
	}

	return nil
}

// DeleteFromDatabase deletes the distribution related entities from the database
func (c *EC2ClusterPKE) DeleteFromDatabase() error {
	// dependencies are deleted using a GORM hook!
	if e := c.db.Delete(c.model).Error; e != nil {
		return errors.WrapIfWithDetails(e, "failed to delete EC2BanzaiCloudCluster", "distro", c.model.ID)
	}

	return nil
}

func (c *EC2ClusterPKE) CreateCluster() error {
	return errors.New("not implemented")
}

func (c *EC2ClusterPKE) GetAWSClient() (*session.Session, error) {
	if c.session != nil {
		return c.session, nil
	}
	secret, err := c.getSecret(c)
	if err != nil {
		return nil, err
	}
	awsCred := amazon.CreateAWSCredentials(secret.Values)
	return session.NewSession(&aws.Config{
		Region:      aws.String(c.model.Cluster.Location),
		Credentials: awsCred,
	})
}

func (c *EC2ClusterPKE) GetCurrentWorkflowID() string {
	return c.model.CurrentWorkflowID
}

func (c *EC2ClusterPKE) SetCurrentWorkflowID(workflowID string) error {
	c.model.CurrentWorkflowID = workflowID

	err := c.db.Save(&c.model).Error
	if err != nil {
		return errors.WrapIfWithDetails(err, "failed to save workflow id", "workflowId", workflowID)
	}

	return nil
}

func (c *EC2ClusterPKE) CreatePKECluster(tokenGenerator TokenGenerator, externalBaseURL string) error {
	return errors.New("unused method")
}

// HasK8sConfig returns true if the cluster's k8s config is available
func (c *EC2ClusterPKE) HasK8sConfig() (bool, error) {
	cfg, err := c.GetK8sConfig()
	if err == ErrConfigNotExists {
		return false, nil
	}
	return len(cfg) > 0, errors.WrapIf(err, "failed to check if k8s config is available")
}

// IsMasterReady returns true when the master node has been reported as ready
func (c *EC2ClusterPKE) IsMasterReady() (bool, error) {
	return c.HasK8sConfig()
}

// RegisterNode adds a Node to the DB
func (c *EC2ClusterPKE) RegisterNode(name, nodePoolName, ip string, master, worker bool) error {
	/* TODO: decide if we need this on AWS
	db := pipConfig.DB()
	nodePool := internalPke.NodePool{
		Name:      nodePoolName,
		ClusterID: c.GetID(),
	}

	roles := internalPke.Roles{}
	if master {
		roles = append(roles, internalPke.RoleMaster)
	}
	if worker {
		roles = append(roles, internalPke.RoleWorker)
	}

	if err := db.Where(nodePool).Attrs(internalPke.NodePool{
		Roles: roles,
	}).FirstOrCreate(&nodePool).Error; err != nil {
		return errors.WrapIf(err, "failed to register nodepool")
	}

	node := internalPke.Host{
		NodePoolID: nodePool.NodePoolID,
		Name:       name,
	}

	if err := db.Where(node).Attrs(internalPke.Host{
		CustomLabels:    make(internalPke.CustomLabels),
		PrivateIP: ip,
	}).FirstOrCreate(&node).Error; err != nil {
		return errors.WrapIf(err, "failed to register node")
	}
	c.log.WithField("node", name).Info("node registered")
	*/

	return nil
}

// Create master CF template
func CreateMasterCF(formation *cloudformation.CloudFormation) error {
	return nil
}

func (c *EC2ClusterPKE) ValidateCreationFields(r *pkgCluster.CreateClusterRequest) error {
	// TODO(Ecsy): implement me

	for _, np := range r.Properties.CreateClusterPKE.NodePools {
		if err := common.ValidateNodePoolLabels(np.Name, np.Labels); err != nil {
			return err
		}
	}

	return nil
}

func (c *EC2ClusterPKE) UpdateCluster(*pkgCluster.UpdateClusterRequest, uint) error {
	return errors.New("not implemented")
}

func createNodePoolsFromPKERequest(nodePools pke.UpdateNodePools) []pkeworkflow.NodePool {
	var out = make([]pkeworkflow.NodePool, len(nodePools))
	i := 0
	for nodePoolName, nodePool := range nodePools {
		count := nodePool.Count

		if !nodePool.Autoscaling {
			// if no cluster autoscaler enabled than min, max and desired count
			// should be same
			nodePool.MinCount = count
			nodePool.MaxCount = count
		} else {
			if count < nodePool.MinCount {
				count = nodePool.MinCount
			} else if count > nodePool.MaxCount {
				count = nodePool.MaxCount
			}
		}

		out[i] = pkeworkflow.NodePool{
			Name:         nodePoolName,
			Worker:       true,
			MinCount:     nodePool.MinCount,
			MaxCount:     nodePool.MaxCount,
			Count:        count,
			Autoscaling:  nodePool.Autoscaling,
			InstanceType: nodePool.InstanceType,
			SpotPrice:    nodePool.SpotPrice,
		}
		for _, subnet := range nodePool.Subnets {
			out[i].Subnets = append(out[i].Subnets, string(subnet))
		}
		i++
	}
	return out
}

func createNodePoolsFromPKENodePools(pkeNodePools []PKENodePool) []pkeworkflow.NodePool {
	var nodePools []pkeworkflow.NodePool

	for _, np := range pkeNodePools {
		nodePools = append(nodePools,
			pkeworkflow.NodePool{
				Name:              np.Name,
				MinCount:          np.MinCount,
				MaxCount:          np.MaxCount,
				Count:             np.Count,
				Autoscaling:       np.Autoscaling,
				Master:            np.Master,
				Worker:            np.Worker,
				InstanceType:      np.InstanceType,
				AvailabilityZones: np.AvailabilityZones,
				ImageID:           np.ImageID,
				SpotPrice:         np.SpotPrice,
			})
	}

	return nodePools
}

func (c *EC2ClusterPKE) UpdatePKECluster(ctx context.Context, request *pkgCluster.UpdateClusterRequest, userID uint, workflowClient client.Client, externalBaseURL string, externalBaseURLInsecure bool) error {
	vpcid, ok := c.model.Network.CloudProviderConfig["vpcID"].(string)
	if !ok {
		return errors.New("VPC ID not found")
	}

	subnets := []string{}
	if subnetIfaces, ok := c.model.Network.CloudProviderConfig["subnets"].([]interface{}); ok {
		for _, subnet := range subnetIfaces {
			if str, ok := subnet.(string); ok {
				subnets = append(subnets, str)
			} else {
				c.log.Errorf("Subnet ID is not a string (%v %T)", subnet, subnet)
			}
		}
	} else {
		return errors.New(fmt.Sprintf("Subnet IDs not found (%v %T)", c.model.Network.CloudProviderConfig["subnets"], c.model.Network.CloudProviderConfig["subnets"]))
	}

	if len(subnets) == 0 {
		return errors.New("subnet IDs not found in cluster network configuration")
	}

	reqNodePools := createNodePoolsFromPKERequest(request.PKE.NodePools)
	reqNodePoolsMap := map[string]pkeworkflow.NodePool{}
	for _, np := range reqNodePools {
		reqNodePoolsMap[np.Name] = np
	}

	clusterNodePools := createNodePoolsFromPKENodePools(c.GetNodePools())
	clusterNodePoolsMap := map[string]pkeworkflow.NodePool{}
	for _, np := range clusterNodePools {
		clusterNodePoolsMap[np.Name] = np
	}

	var nodePoolsToAdd []pkeworkflow.NodePool
	var nodePoolsToUpdate []pkeworkflow.NodePool
	var nodePoolsToDelete []pkeworkflow.NodePool

	for _, np := range clusterNodePools {
		if np.Master || np.Name == "master" {
			continue
		}

		if reqNodePool, ok := reqNodePoolsMap[np.Name]; ok {
			reqNodePool.Master = np.Master
			reqNodePool.Worker = np.Worker
			reqNodePool.AvailabilityZones = np.AvailabilityZones

			nodePoolsToUpdate = append(nodePoolsToUpdate, reqNodePool)
		} else {
			nodePoolsToDelete = append(nodePoolsToDelete, np)
		}
	}

	for _, np := range reqNodePools {
		if np.Master || np.Name == "master" {
			continue
		}

		if _, ok := clusterNodePoolsMap[np.Name]; !ok {
			nodePoolsToAdd = append(nodePoolsToAdd, np)
		}
	}

	// update or delete existing pools in DB
	newModelNodePools := internalPke.NodePools{}
	deletedModelNodePools := internalPke.NodePools{}
	for _, np := range c.model.NodePools {
		if reqNodePool, ok := reqNodePoolsMap[np.Name]; ok { // update
			np.Autoscaling = reqNodePool.Autoscaling

			providerConfig := internalPke.NodePoolProviderConfigAmazon{}
			if err := mapstructure.Decode(np.ProviderConfig, &providerConfig); err != nil {
				return errors.WrapIff(err, "decoding nodepool %q config", np.Name)
			}
			providerConfig.AutoScalingGroup.Size.Min = reqNodePool.MinCount
			providerConfig.AutoScalingGroup.Size.Max = reqNodePool.MaxCount
			providerConfig.AutoScalingGroup.Size.Desired = reqNodePool.Count
			np.ProviderConfig["autoScalingGroup"] = providerConfig.AutoScalingGroup

			newModelNodePools = append(newModelNodePools, np)
		} else {
			deletedModelNodePools = append(deletedModelNodePools, np)
		}
	}

	// add new pools
	for _, np := range reqNodePools {
		if _, ok := clusterNodePoolsMap[np.Name]; !ok {
			providerConfig := internalPke.NodePoolProviderConfigAmazon{}
			providerConfig.AutoScalingGroup.Name = np.Name
			providerConfig.AutoScalingGroup.InstanceType = np.InstanceType
			providerConfig.AutoScalingGroup.LaunchConfigurationName = np.Name
			providerConfig.AutoScalingGroup.Image = np.ImageID
			providerConfig.AutoScalingGroup.Size.Min = np.MinCount
			providerConfig.AutoScalingGroup.Size.Max = np.MaxCount
			providerConfig.AutoScalingGroup.Size.Desired = np.Count
			providerConfig.AutoScalingGroup.SpotPrice = np.SpotPrice
			for _, subnet := range np.Subnets {
				providerConfig.AutoScalingGroup.Subnets = append(providerConfig.AutoScalingGroup.Subnets, internalPke.Subnet(subnet))
			}

			modelNodepool := internalPke.NodePool{
				Name:        np.Name,
				CreatedBy:   userID,
				Roles:       internalPke.Roles{"worker"},
				Autoscaling: np.Autoscaling,
				Provider:    internalPke.NPPAmazon,
				ProviderConfig: internalPke.Config{
					"autoScalingGroup": providerConfig.AutoScalingGroup},
			}
			newModelNodePools = append(newModelNodePools, modelNodepool)
		}
	}

	c.model.NodePools = newModelNodePools
	if err := c.db.Save(&c.model).Error; err != nil {
		return errors.WrapIf(err, "failed to save cluster")
	}

	for _, np := range deletedModelNodePools {
		if np.NodePoolID == 0 {
			panic("prevented deleting all nodepools")
		}

		c.log.WithField("nodepool", np.Name).Info("deleting nodepool")
		if err := c.db.Delete(&np).Error; err != nil {
			return errors.WrapIf(err, "failed to delete nodepool")
		}
	}

	input := pkeworkflow.UpdateClusterWorkflowInput{
		ClusterID:                   c.GetID(),
		NodePoolsToAdd:              nodePoolsToAdd,
		NodePoolsToUpdate:           nodePoolsToUpdate,
		NodePoolsToDelete:           nodePoolsToDelete,
		OrganizationID:              c.GetOrganizationId(),
		ClusterUID:                  c.GetUID(),
		ClusterName:                 c.GetName(),
		SecretID:                    c.GetSecretId(),
		Region:                      c.GetLocation(),
		PipelineExternalURL:         externalBaseURL,
		PipelineExternalURLInsecure: externalBaseURLInsecure,
		VPCID:                       vpcid,
		SubnetIDs:                   subnets,
	}

	workflowOptions := client.StartWorkflowOptions{
		TaskList:                     "pipeline",
		ExecutionStartToCloseTimeout: 40 * time.Minute, // TODO: lower timeout
	}
	exec, err := workflowClient.ExecuteWorkflow(ctx, workflowOptions, pkeworkflow.UpdateClusterWorkflowName, input)
	if err != nil {
		return err
	}

	err = c.SetCurrentWorkflowID(exec.GetID())
	if err != nil {
		return err
	}

	workflowError := exec.Get(ctx, nil)

	return workflowError
}

func (c *EC2ClusterPKE) UpdateNodePools(*pkgCluster.UpdateNodePoolsRequest, uint) error {
	panic("implement me")
}

func (c *EC2ClusterPKE) CheckEqualityToUpdate(*pkgCluster.UpdateClusterRequest) error {
	return nil
}

func (c *EC2ClusterPKE) AddDefaultsToUpdate(*pkgCluster.UpdateClusterRequest) {
}

func (c *EC2ClusterPKE) DeleteCluster() error {
	return errors.New("not implemented")
}

func (c *EC2ClusterPKE) DeletePKECluster(ctx context.Context, workflowClient client.Client) error {
	input := pkeworkflow.DeleteClusterWorkflowInput{
		ClusterID: c.GetID(),
	}
	workflowOptions := client.StartWorkflowOptions{
		TaskList:                     "pipeline",
		ExecutionStartToCloseTimeout: 40 * time.Minute, // TODO: lower timeout
	}
	exec, err := workflowClient.ExecuteWorkflow(ctx, workflowOptions, pkeworkflow.DeleteClusterWorkflowName, input)
	if err != nil {
		return err
	}

	err = c.SetCurrentWorkflowID(exec.GetID())
	if err != nil {
		return err
	}

	err = exec.Get(ctx, nil)
	if err != nil {
		return err
	}

	return nil
}

func (c *EC2ClusterPKE) GetAPIEndpoint() (string, error) {
	if c.APIEndpoint != "" {
		return c.APIEndpoint, nil
	}

	config, err := c.GetK8sConfig()
	if err != nil {
		return "", errors.WrapIf(err, "failed to get cluster's Kubeconfig")
	}

	return pkgCluster.GetAPIEndpointFromKubeconfig(config)
}

func (c *EC2ClusterPKE) GetK8sConfig() ([]byte, error) {
	return c.CommonClusterBase.getConfig(c)
}

func (c *EC2ClusterPKE) GetK8sUserConfig() ([]byte, error) {
	return c.GetK8sConfig()
}

func (c *EC2ClusterPKE) RbacEnabled() bool {
	return c.model.Kubernetes.RBACEnabled
}

func (c *EC2ClusterPKE) GetStatus() (*pkgCluster.GetClusterStatusResponse, error) {
	// log.Info("Create cluster status response")
	hasSpotNodePool := false
	nodePools := make(map[string]*pkgCluster.NodePoolStatus)
	for _, np := range c.model.NodePools {
		providerConfig := internalPke.NodePoolProviderConfigAmazon{}
		err := mapstructure.Decode(np.ProviderConfig, &providerConfig)
		if err != nil {
			return nil, errors.WrapIfWithDetails(err, "failed to decode providerconfig", "cluster", c.model.Cluster.Name)
		}

		nodePools[np.Name] = &pkgCluster.NodePoolStatus{
			Autoscaling:       np.Autoscaling,
			Count:             providerConfig.AutoScalingGroup.Size.Desired,
			MaxCount:          providerConfig.AutoScalingGroup.Size.Max,
			MinCount:          providerConfig.AutoScalingGroup.Size.Min,
			InstanceType:      providerConfig.AutoScalingGroup.InstanceType,
			SpotPrice:         providerConfig.AutoScalingGroup.SpotPrice,
			CreatorBaseFields: *NewCreatorBaseFields(np.CreatedAt, np.CreatedBy),
			Labels:            np.Labels,
		}

		if p, err := strconv.ParseFloat(providerConfig.AutoScalingGroup.SpotPrice, 64); err == nil && p > 0.0 {
			hasSpotNodePool = true
		}
	}

	return &pkgCluster.GetClusterStatusResponse{
		Status:            c.model.Cluster.Status,
		StatusMessage:     c.model.Cluster.StatusMessage,
		Name:              c.model.Cluster.Name,
		Location:          c.model.Cluster.Location,
		Cloud:             c.model.Cluster.Cloud,
		Distribution:      c.model.Cluster.Distribution,
		Spot:              hasSpotNodePool,
		ResourceID:        c.model.Cluster.ID,
		NodePools:         nodePools,
		Version:           c.model.Kubernetes.Version,
		OIDCEnabled:       c.model.Cluster.OidcEnabled,
		CreatorBaseFields: *NewCreatorBaseFields(c.model.Cluster.CreatedAt, c.model.Cluster.CreatedBy),
		Region:            c.model.Cluster.Location,
		StartedAt:         c.model.Cluster.StartedAt,
	}, nil
}

// IsReady checks if the cluster is running according to the cloud provider.
func (c *EC2ClusterPKE) IsReady() (bool, error) {
	// cluster is not ready in case there's no config secret yet
	if c.GetConfigSecretId() == "" {
		return false, nil
	}
	return true, nil
}

type PKENodePool struct {
	Name              string
	MinCount          int
	MaxCount          int
	Count             int
	Autoscaling       bool
	Master            bool
	Worker            bool
	InstanceType      string
	AvailabilityZones []string
	ImageID           string
	SpotPrice         string
	Subnets           []string
}

func (c *EC2ClusterPKE) GetNodePools() []PKENodePool {
	pools := make([]PKENodePool, len(c.model.NodePools), len(c.model.NodePools))
	for i, np := range c.model.NodePools {
		var amazonPool internalPke.NodePoolProviderConfigAmazon
		_ = mapstructure.Decode(np.ProviderConfig, &amazonPool)

		var azs []string
		for _, az := range amazonPool.AutoScalingGroup.Zones {
			azs = append(azs, string(az))
		}

		var subnets []string
		for _, subnet := range amazonPool.AutoScalingGroup.Subnets {
			subnets = append(subnets, string(subnet))
		}

		pools[i] = PKENodePool{
			Name:              np.Name,
			MinCount:          amazonPool.AutoScalingGroup.Size.Min,
			MaxCount:          amazonPool.AutoScalingGroup.Size.Max,
			Count:             amazonPool.AutoScalingGroup.Size.Desired,
			InstanceType:      amazonPool.AutoScalingGroup.InstanceType,
			AvailabilityZones: azs,
			ImageID:           amazonPool.AutoScalingGroup.Image,
			SpotPrice:         amazonPool.AutoScalingGroup.SpotPrice,
			Autoscaling:       np.Autoscaling,
			Subnets:           subnets,
		}
		for _, role := range np.Roles {
			if role == "master" {
				pools[i].Master = true
			}
			if role == "worker" {
				pools[i].Worker = true
			}
		}
	}
	return pools
}

// ListNodePools returns node pool names.
func (c *EC2ClusterPKE) ListNodePools() ([]string, error) {
	var nodePools = make([]string, 0, len(c.model.NodePools))

	for _, nodePool := range c.model.NodePools {
		nodePools = append(nodePools, nodePool.Name)
	}

	return nodePools, nil
}

func (c *EC2ClusterPKE) NodePoolExists(nodePoolName string) bool {
	for _, np := range c.model.NodePools {
		if np.Name == nodePoolName {
			return true
		}
	}
	return false
}

func (c *EC2ClusterPKE) GetCAHash() (string, error) {
	secret, err := secret.Store.GetByName(c.GetOrganizationId(), fmt.Sprintf("cluster-%d-ca", c.GetID()))
	if err != nil {
		return "", err
	}
	crt := secret.Values[secrettype.KubernetesCACert]
	block, _ := pem.Decode([]byte(crt))
	if block == nil {
		return "", errors.New("failed to parse certificate")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return "", errors.WrapIff(err, "failed to parse certificate")
	}
	h := sha256.Sum256(cert.RawSubjectPublicKeyInfo)
	return fmt.Sprintf("sha256:%s", hex.EncodeToString(h[:])), nil
}

// GetPipelineToken returns a lazily generated token for Pipeline
func (c *EC2ClusterPKE) GetPipelineToken(tokenGenerator interface{}) (string, error) {
	generator, ok := tokenGenerator.(TokenGenerator)
	if !ok {
		return "", errors.New(fmt.Sprintf("failed to use %T as TokenGenerator", tokenGenerator))
	}
	_, token, err := generator.GenerateClusterToken(c.model.Cluster.OrganizationID, c.model.Cluster.ID)
	return token, err
}

// GetBootstrapCommand returns a command line to use to install a node in the given nodepool
func (c *EC2ClusterPKE) GetBootstrapCommand(nodePoolName, url string, urlInsecure bool, token string) (string, error) {
	subcommand := "worker"
	var np *internalPke.NodePool
	for _, nodePool := range c.model.NodePools {
		if nodePool.Name == nodePoolName {
			np = &nodePool
			break
		}
	}

	if np == nil {
		return "", errors.New(fmt.Sprintf("can't find nodepool %q", nodePoolName))
	}

	for _, role := range np.Roles {
		if role == internalPke.RoleMaster {
			subcommand = "master"
			break
		}
	}
	if nodePoolName == "master" {
		subcommand = "master" // TODO remove this if not needed anymore
	}

	nodePoolAmazonConfig := internalPke.NodePoolProviderConfigAmazon{}
	err := mapstructure.Decode(np.ProviderConfig, &nodePoolAmazonConfig)
	if err != nil {
		return "", errors.WrapIfWithDetails(err, "failed to decode providerconfig", "cluster", c.model.Cluster.Name)
	}

	version := c.model.Kubernetes.Version
	if version == "" {
		version = defaultK8sVersion
	}
	if version[0] == 'v' {
		version = version[1:]
	}
	infrastructureCIDR := ""

	kubernetesNetworkProvider, err := c.GetKubernetesNetworkProvider()
	if err != nil {
		return "", errors.WrapIf(err, "couldn't get Kubernetes network provider config")
	}

	// determine the CIDR of the subnet of the node pool
	subnetId := ""
	if len(nodePoolAmazonConfig.AutoScalingGroup.Subnets) > 0 {
		subnetId = string(nodePoolAmazonConfig.AutoScalingGroup.Subnets[0])
	} else {
		// subnet not provided for nodepool. fall back to global provider network config
		_, _, subnets, err := c.GetNetworkCloudProvider()
		if err != nil {
			return "", errors.WrapIf(err, "couldn't get cloud provider network config")
		}

		if len(subnets) > 0 {
			subnetId = subnets[0]
		}
	}

	if subnetId != "" {
		// query subnet CIDR from amazon
		awsClient, err := c.GetAWSClient()
		if err != nil {
			return "", err
		}

		netSvc := pkgEC2.NewNetworkSvc(ec2.New(awsClient), NewLogurLogger(c.log))
		infrastructureCIDR, err = netSvc.GetSubnetCidr(subnetId)
		if err != nil {
			return "", errors.WrapIff(err, "couldn't get CIDR for subnet %q", subnetId)
		}
	}

	if infrastructureCIDR == "" {
		return "", errors.WrapIff(err, "couldn't get CIDR for subnet %q", subnetId)
	}

	apiAddress, _, err := c.GetNetworkApiServerAddress()
	if err != nil {
		return "", err
	}

	// master
	if subcommand == "master" {
		masterMode := "default"
		if nodePoolAmazonConfig.AutoScalingGroup.Size.Max > 1 {
			masterMode = "ha"
		}

		command := fmt.Sprintf("pke install %s "+
			"--pipeline-url=%q "+
			"--pipeline-insecure=%q "+
			"--pipeline-token=%q "+
			"--pipeline-org-id=%d "+
			"--pipeline-cluster-id=%d "+
			"--pipeline-nodepool=%q "+
			"--kubernetes-cloud-provider=aws "+
			"--kubernetes-version=%q "+
			"--kubernetes-network-provider=%q "+
			"--kubernetes-service-cidr=10.10.0.0/16 "+
			"--kubernetes-pod-network-cidr=10.20.0.0/16 "+
			"--kubernetes-infrastructure-cidr=%q "+
			"--kubernetes-api-server=%q "+
			"--kubernetes-cluster-name=%q "+
			"--kubernetes-master-mode=%q "+
			"--kubernetes-advertise-address=0.0.0.0:6443",
			subcommand,
			url,
			strconv.FormatBool(urlInsecure),
			token,
			c.model.Cluster.OrganizationID,
			c.model.Cluster.ID,
			nodePoolName,
			version,
			kubernetesNetworkProvider,
			infrastructureCIDR,
			apiAddress,
			c.GetName(),
			masterMode,
		)

		if c.model.Cluster.OidcEnabled {
			oidcClientID := c.GetUID()

			command = fmt.Sprintf("%s "+
				"--kubernetes-oidc-issuer-url=%q "+
				"--kubernetes-oidc-client-id=%q",
				command,
				global.Config.Auth.OIDC.Issuer, // TODO this should be configurable as well
				oidcClientID,
			)
		}

		return command, nil
	}

	// worker
	return fmt.Sprintf("pke install %s "+
		"--pipeline-url=%q "+
		"--pipeline-insecure=%q "+
		"--pipeline-token=%q "+
		"--pipeline-org-id=%d "+
		"--pipeline-cluster-id=%d "+
		"--pipeline-nodepool=%q "+
		"--kubernetes-cloud-provider=aws "+
		"--kubernetes-version=%q "+
		"--kubernetes-infrastructure-cidr=%q",
		subcommand,
		url,
		strconv.FormatBool(urlInsecure),
		token,
		c.model.Cluster.OrganizationID,
		c.model.Cluster.ID,
		nodePoolName,
		version,
		infrastructureCIDR,
	), nil
}

func (c *EC2ClusterPKE) GetKubernetesVersion() (string, error) {
	return c.model.Kubernetes.Version, nil
}

func (c *EC2ClusterPKE) GetKubernetesNetworkProvider() (string, error) {
	return string(c.model.Network.Provider), nil
}

// GetNetworkCloudProvider return cloud provider specific network information.
func (c *EC2ClusterPKE) GetNetworkCloudProvider() (cloudProvider, vpcID string, subnets []string, err error) {
	cp := c.model.Network.CloudProvider
	cloudProvider = string(cp)
	switch cp {
	case internalPke.CNPAmazon:
		cpc := &internalPke.NetworkCloudProviderConfigAmazon{}
		err = mapstructure.Decode(c.model.Network.CloudProviderConfig, &cpc)
		if err != nil {
			return
		}
		vpcID = cpc.VPCID
		for _, subnet := range cpc.Subnets {
			subnets = append(subnets, string(subnet))
		}
	}

	return
}

// SaveNetworkCloudProvider saves cloud provider specific network information.
func (c *EC2ClusterPKE) SaveNetworkCloudProvider(cloudProvider, vpcID string, subnets []string) error {
	if cloudProvider != string(internalPke.CNPAmazon) {
		return errors.New("unsupported cloud network provider")
	}

	c.model.Network.CloudProvider = internalPke.CNPAmazon
	c.model.Network.CloudProviderConfig = make(internalPke.Config)
	c.model.Network.CloudProviderConfig["vpcID"] = vpcID
	c.model.Network.CloudProviderConfig["subnets"] = subnets

	err := c.db.Save(&c.model).Error
	if err != nil {
		return errors.WrapIfWithDetails(err, "failed to save network cloud provider", "cloudProvider", cloudProvider)
	}

	return nil
}

// GetNetworkApiServerAddress returns Kubernetes API Server host and port.
func (c *EC2ClusterPKE) GetNetworkApiServerAddress() (host, port string, err error) {
	return net.SplitHostPort(c.model.Network.APIServerAddress)
}

// SaveNetworkApiServerAddress stores Kubernetes API Server host and port.
func (c *EC2ClusterPKE) SaveNetworkApiServerAddress(host, port string) error {
	if port == "" {
		// default port
		port = "6443"
	}
	c.model.Network.APIServerAddress = net.JoinHostPort(host, port)

	err := c.db.Save(&c.model).Error
	if err != nil {
		return errors.WrapIfWithDetails(err, "failed to save network api server address", "address", c.model.Network.APIServerAddress)
	}

	return nil
}

func CreateEC2ClusterPKEFromRequest(request *pkgCluster.CreateClusterRequest, orgId uint, userId uint) (*EC2ClusterPKE, error) {
	c := &EC2ClusterPKE{
		log: log.WithField("cluster", request.Name).WithField("organization", orgId),
	}

	c.db = global.DB()

	var (
		network    = createEC2PKENetworkFromRequest(request.Properties.CreateClusterPKE.Network, userId)
		nodepools  = createEC2ClusterPKENodePoolsFromRequest(request.Properties.CreateClusterPKE.NodePools, userId)
		kubernetes = createEC2ClusterPKEFromRequest(request.Properties.CreateClusterPKE.Kubernetes, userId)
		kubeADM    = createEC2ClusterPKEKubeADMFromRequest(request.Properties.CreateClusterPKE.KubeADM, userId)
		cri        = createEC2ClusterPKECRIFromRequest(request.Properties.CreateClusterPKE.CRI, userId)
	)

	instanceType, image, err := getMasterInstanceTypeAndImageFromNodePools(nodepools)
	if err != nil {
		return nil, err
	}

	c.model = &internalPke.EC2PKEClusterModel{
		Cluster: clustermodel.ClusterModel{
			Name:           request.Name,
			Location:       request.Location,
			Cloud:          request.Cloud,
			Distribution:   pkgCluster.PKE,
			OrganizationID: orgId,
			RbacEnabled:    kubernetes.RBAC.Enabled,
			OidcEnabled:    request.Properties.CreateClusterPKE.Kubernetes.OIDC.Enabled,
			CreatedBy:      userId,
		},
		MasterInstanceType: instanceType,
		MasterImage:        image,
		Network:            network,
		NodePools:          nodepools,
		Kubernetes:         kubernetes,
		KubeADM:            kubeADM,
		CRI:                cri,
	}

	return c, nil
}

func CreateEC2ClusterPKEFromModel(modelCluster *model.ClusterModel) (*EC2ClusterPKE, error) {
	log := log.WithField("cluster", modelCluster.Name).WithField("organization", modelCluster.OrganizationId)

	db := global.DB()

	m := internalPke.EC2PKEClusterModel{
		ClusterID: modelCluster.ID,
	}

	// log.Debug("Load EC2 props from database")
	err := db.Where(m).
		Preload("Cluster").
		Preload("Network").
		Preload("NodePools").
		Preload("Kubernetes").
		Preload("KubeADM").
		Preload("CRI").
		First(&m).
		Error
	if err != nil {
		return nil, err
	}

	c := &EC2ClusterPKE{
		db:    db,
		model: &m,
		log:   log,
	}
	return c, nil
}

func createEC2ClusterPKENodePoolsFromRequest(pools pke.NodePools, userId uint) internalPke.NodePools {
	var nps internalPke.NodePools

	for _, pool := range pools {
		np := internalPke.NodePool{
			Name:           pool.Name,
			Roles:          convertRoles(pool.Roles),
			Hosts:          convertHosts(pool.Hosts),
			Provider:       convertNodePoolProvider(pool.Provider),
			ProviderConfig: pool.ProviderConfig,
			Labels:         pool.Labels,
			Autoscaling:    pool.Autoscaling,
		}
		np.CreatedBy = userId
		nps = append(nps, np)
	}
	return nps
}

func convertRoles(roles pke.Roles) (result internalPke.Roles) {
	for _, role := range roles {
		result = append(result, internalPke.Role(role))
	}
	return
}

func convertHosts(hosts pke.Hosts) (result internalPke.Hosts) {
	for _, host := range hosts {
		result = append(result, internalPke.Host{
			Name:             host.Name,
			PrivateIP:        host.PrivateIP,
			NetworkInterface: host.NetworkInterface,
			Roles:            convertRoles(host.Roles),
			Labels:           convertLabels(host.Labels),
			Taints:           convertTaints(host.Taints),
		})
	}

	return
}

func convertNodePoolProvider(provider pke.NodePoolProvider) (result internalPke.NodePoolProvider) {
	return internalPke.NodePoolProvider(provider)
}

func convertLabels(labels pke.Labels) internalPke.Labels {
	res := make(internalPke.Labels, len(labels))
	for k, v := range labels {
		res[k] = v
	}
	return res
}

func convertTaints(taints pke.Taints) (result internalPke.Taints) {
	for _, taint := range taints {
		result = append(result, internalPke.Taint(taint))
	}
	return
}

func createEC2PKENetworkFromRequest(network pke.Network, userId uint) internalPke.Network {
	n := internalPke.Network{
		ServiceCIDR:      network.ServiceCIDR,
		PodCIDR:          network.PodCIDR,
		Provider:         convertNetworkProvider(network.Provider),
		APIServerAddress: network.APIServerAddress,
	}
	n.CreatedBy = userId
	return n
}

func convertNetworkProvider(provider pke.NetworkProvider) (result internalPke.NetworkProvider) {
	return internalPke.NetworkProvider(provider)
}

func createEC2ClusterPKEFromRequest(kubernetes pke.Kubernetes, userId uint) internalPke.Kubernetes {
	k := internalPke.Kubernetes{
		Version: kubernetes.Version,
		RBAC:    internalPke.RBAC{Enabled: kubernetes.RBAC.Enabled},
	}
	k.CreatedBy = userId
	return k
}

func createEC2ClusterPKEKubeADMFromRequest(kubernetes pke.KubeADM, userId uint) internalPke.KubeADM {
	a := internalPke.KubeADM{
		ExtraArgs: convertExtraArgs(kubernetes.ExtraArgs),
	}
	a.CreatedBy = userId
	return a
}

func convertExtraArgs(extraArgs pke.ExtraArgs) internalPke.ExtraArgs {
	res := make(internalPke.ExtraArgs, len(extraArgs))
	for k, v := range extraArgs {
		res[k] = internalPke.ExtraArg(v)
	}
	return res
}

func createEC2ClusterPKECRIFromRequest(cri pke.CRI, userId uint) internalPke.CRI {
	c := internalPke.CRI{
		Runtime:       internalPke.Runtime(cri.Runtime),
		RuntimeConfig: cri.RuntimeConfig,
	}
	c.CreatedBy = userId
	return c
}

func getMasterInstanceTypeAndImageFromNodePools(nodepools internalPke.NodePools) (masterInstanceType string, masterImage string, err error) {
	for _, nodepool := range nodepools {
		for _, role := range nodepool.Roles {
			if role == internalPke.RoleMaster {
				switch nodepool.Provider {
				case internalPke.NPPAmazon:
					providerConfig := internalPke.NodePoolProviderConfigAmazon{}
					err = mapstructure.Decode(nodepool.ProviderConfig, &providerConfig)
					if err != nil {
						return
					}
					masterInstanceType = providerConfig.AutoScalingGroup.InstanceType
					masterImage = providerConfig.AutoScalingGroup.Image
					return
				}
			}
		}
	}
	return
}
