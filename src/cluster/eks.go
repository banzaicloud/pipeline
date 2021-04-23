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
	"time"

	"emperror.dev/errors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/eks"
	"github.com/jinzhu/gorm"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"

	"github.com/banzaicloud/pipeline/internal/cluster/clusteradapter/clustermodel"
	eks2 "github.com/banzaicloud/pipeline/internal/cluster/distribution/eks"
	pkgEks "github.com/banzaicloud/pipeline/internal/cluster/distribution/eks/ekscluster"
	"github.com/banzaicloud/pipeline/internal/cluster/distribution/eks/ekscluster/nodepools"
	"github.com/banzaicloud/pipeline/internal/cluster/distribution/eks/eksmodel"
	"github.com/banzaicloud/pipeline/internal/global"
	"github.com/banzaicloud/pipeline/internal/global/globaleks"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	"github.com/banzaicloud/pipeline/pkg/k8sutil"
	"github.com/banzaicloud/pipeline/pkg/providers/amazon"
	"github.com/banzaicloud/pipeline/src/model"
	"github.com/banzaicloud/pipeline/src/secret"
)

const (
	asgWaitLoopSleepSeconds = 5
	asgFulfillmentTimeout   = 10 * time.Minute
)

// CreateEKSClusterFromRequest creates ClusterModel struct from the request
func CreateEKSClusterFromRequest(request *pkgCluster.CreateClusterRequest, orgId uint, userId uint) (*EKSCluster, error) {
	cluster := EKSCluster{
		log: log.WithField("cluster", request.Name),
	}

	cluster.EncryptionConfig = request.Properties.CreateClusterEKS.EncryptionConfig

	var err error
	cluster.repository, err = NewDBEKSClusterRepository(global.DB())
	if err != nil {
		return nil, errors.WrapIf(err, "failed to create EKS cluster repository")
	}

	modelNodePools := createNodePoolsFromRequest(request.Properties.CreateClusterEKS.NodePools, userId)

	authConfigMap, err := request.Properties.CreateClusterEKS.AuthConfig.ConvertToString()
	if err != nil {
		return nil, errors.WrapIf(err, "failed to convert config map to string")
	}
	cluster.model = &eksmodel.EKSClusterModel{
		Cluster: clustermodel.ClusterModel{
			Name:           request.Name,
			Location:       request.Location,
			Cloud:          request.Cloud,
			OrganizationID: orgId,
			SecretID:       request.SecretId,
			Distribution:   pkgCluster.EKS,
			RbacEnabled:    true,
			CreatedBy:      userId,
		},
		Version:               request.Properties.CreateClusterEKS.Version,
		LogTypes:              request.Properties.CreateClusterEKS.LogTypes,
		NodePools:             modelNodePools,
		VpcId:                 &request.Properties.CreateClusterEKS.Vpc.VpcId,
		VpcCidr:               &request.Properties.CreateClusterEKS.Vpc.Cidr,
		RouteTableId:          &request.Properties.CreateClusterEKS.RouteTableId,
		Subnets:               createSubnetsFromRequest(request.Properties.CreateClusterEKS),
		DefaultUser:           request.Properties.CreateClusterEKS.IAM.DefaultUser,
		ClusterRoleId:         request.Properties.CreateClusterEKS.IAM.ClusterRoleID,
		NodeInstanceRoleId:    request.Properties.CreateClusterEKS.IAM.NodeInstanceRoleID,
		APIServerAccessPoints: createAPIServerAccessPointsFromRequest(request),
		AuthConfigMap:         authConfigMap,
	}

	if request.Properties.CreateClusterEKS.Tags != nil {
		cluster.model.Cluster.Tags = request.Properties.CreateClusterEKS.Tags
	}

	// subnet mapping
	cluster.SubnetMapping = createSubnetMappingFromRequest(request.Properties.CreateClusterEKS)

	return &cluster, nil
}

func createAPIServerAccessPointsFromRequest(request *pkgCluster.CreateClusterRequest) []string {
	if len(request.Properties.CreateClusterEKS.APIServerAccessPoints) != 0 {
		return request.Properties.CreateClusterEKS.APIServerAccessPoints
	}
	return []string{"public"}
}

func createNodePoolsFromRequest(nodePools map[string]*pkgEks.NodePool, userId uint) []*eksmodel.AmazonNodePoolsModel {
	modelNodePools := make([]*eksmodel.AmazonNodePoolsModel, len(nodePools))
	i := 0
	for nodePoolName, nodePool := range nodePools {
		modelNodePools[i] = &eksmodel.AmazonNodePoolsModel{
			CreatedBy:        userId,
			Name:             nodePoolName,
			StackID:          "",
			NodeSpotPrice:    nodePool.SpotPrice,
			Autoscaling:      nodePool.Autoscaling,
			NodeMinCount:     nodePool.MinCount,
			NodeMaxCount:     nodePool.MaxCount,
			Count:            nodePool.Count,
			NodeVolumeSize:   nodePool.VolumeSize,
			NodeImage:        nodePool.Image,
			NodeInstanceType: nodePool.InstanceType,
			Status:           eks2.NodePoolStatusCreating,
			StatusMessage:    "",
			Labels:           nodePool.Labels,
			Delete:           false,
		}
		i++
	}
	return modelNodePools
}

// createSubnetsFromRequest collects distinct existing (subnetid !=0) and to be created subnets from the request
func createSubnetsFromRequest(eksRequest *pkgEks.CreateClusterEKS) []*eksmodel.EKSSubnetModel {
	if eksRequest == nil {
		return nil
	}

	var subnetsFromRequest []*pkgEks.Subnet
	for _, subnet := range eksRequest.Subnets {
		if subnet != nil {
			subnetsFromRequest = append(subnetsFromRequest, subnet)
		}
	}

	for _, np := range eksRequest.NodePools {
		if np != nil && np.Subnet != nil {
			subnetsFromRequest = append(subnetsFromRequest, np.Subnet)
		}
	}

	uniqueSubnets := make(map[string]*eksmodel.EKSSubnetModel, 0)
	for _, subnet := range subnetsFromRequest {
		if subnet != nil {
			if subnet.SubnetId != "" {
				if _, ok := uniqueSubnets[subnet.SubnetId]; !ok {
					uniqueSubnets[subnet.SubnetId] = &eksmodel.EKSSubnetModel{SubnetId: &subnet.SubnetId}
				}
			} else if subnet.Cidr != "" {
				if _, ok := uniqueSubnets[subnet.Cidr]; !ok {
					uniqueSubnets[subnet.Cidr] = &eksmodel.EKSSubnetModel{
						Cidr:             &subnet.Cidr,
						AvailabilityZone: &subnet.AvailabilityZone,
					}
				}
			}
		}
	}

	var modelSubnets []*eksmodel.EKSSubnetModel
	for _, subnet := range uniqueSubnets {
		modelSubnets = append(modelSubnets, subnet)
	}

	return modelSubnets
}

// createSubnetMappingFromRequest maps node pools to subnets provided in request.Properties.CreateClusterEKS.
// The subnets identified by the "default" key in the returned map represent the subnets provided in
// request.Properties.CreateClusterEKS.Subnets
func createSubnetMappingFromRequest(eksRequest *pkgEks.CreateClusterEKS) map[string][]*pkgEks.Subnet {
	subnetMapping := make(map[string][]*pkgEks.Subnet, len(eksRequest.NodePools)+1)

	subnetMapping["default"] = eksRequest.Subnets

	for nodePoolName, nodePool := range eksRequest.NodePools {
		subnetMapping[nodePoolName] = []*pkgEks.Subnet{nodePool.Subnet}
	}

	return subnetMapping
}

type dbEKSClusterRepository struct {
	db *gorm.DB
}

func (r dbEKSClusterRepository) DeleteClusterModel(model *clustermodel.ClusterModel) error {
	return r.db.Delete(model).Error
}

func (r dbEKSClusterRepository) DeleteModel(model *eksmodel.EKSClusterModel) error {
	return r.db.Delete(model).Error
}

func (r dbEKSClusterRepository) DeleteNodePool(model *eksmodel.AmazonNodePoolsModel) error {
	return r.db.Delete(model).Error
}

func (r dbEKSClusterRepository) DeleteSubnet(model *eksmodel.EKSSubnetModel) error {
	return r.db.Delete(model).Error
}

func (r dbEKSClusterRepository) SaveModel(model *eksmodel.EKSClusterModel) error {
	return r.db.Save(model).Error
}

func (r dbEKSClusterRepository) SaveStatusHistory(model *clustermodel.StatusHistoryModel) error {
	return r.db.Save(model).Error
}

// NewDBEKSClusterRepository returns a new EKSClusterRepository backed by a GORM DB
func NewDBEKSClusterRepository(db *gorm.DB) (EKSClusterRepository, error) {
	if db == nil {
		return nil, errors.New("db parameter cannot be nil")
	}
	return dbEKSClusterRepository{
		db: db,
	}, nil
}

// EKSClusterRepository describes a EKS cluster's persistent storage repository
type EKSClusterRepository interface {
	DeleteClusterModel(model *clustermodel.ClusterModel) error
	DeleteModel(model *eksmodel.EKSClusterModel) error
	DeleteNodePool(model *eksmodel.AmazonNodePoolsModel) error
	DeleteSubnet(model *eksmodel.EKSSubnetModel) error
	SaveModel(model *eksmodel.EKSClusterModel) error
	SaveStatusHistory(model *clustermodel.StatusHistoryModel) error
}

// EKSCluster struct for EKS cluster
type EKSCluster struct {
	EncryptionConfig []pkgEks.EncryptionConfig
	repository       EKSClusterRepository
	model            *eksmodel.EKSClusterModel

	// maps node pools to subnets. The subnets identified by the "default" key represent the subnets provided in
	// request.Properties.CreateClusterEKS.Subnets
	SubnetMapping map[string][]*pkgEks.Subnet
	log           logrus.FieldLogger
	CommonClusterBase
}

func (c *EKSCluster) ValidateCreationFields(*pkgCluster.CreateClusterRequest) error {
	return errors.New("not implemented")
}

func (c *EKSCluster) CreateCluster() error {
	return errors.New("not implemented")
}

func (c *EKSCluster) DeleteCluster() error {
	return errors.New("not implemented")
}

// Deprecated: UpdateCluster updates EKS cluster in cloud
func (c *EKSCluster) UpdateCluster(*pkgCluster.UpdateClusterRequest, uint) error {
	return errors.New("not implemented")
}

func (c *EKSCluster) GetSubnetMapping() map[string][]*pkgEks.Subnet {
	return c.SubnetMapping
}

// GetOrganizationId gets org where the cluster belongs
func (c *EKSCluster) GetOrganizationId() uint {
	return c.model.Cluster.OrganizationID
}

// GetLocation gets where the cluster is.
func (c *EKSCluster) GetLocation() string {
	return c.model.Cluster.Location
}

// GetSecretId retrieves the secret id
func (c *EKSCluster) GetSecretId() string {
	return c.model.Cluster.SecretID
}

// GetSshSecretId retrieves the secret id
func (c *EKSCluster) GetSshSecretId() string {
	return c.model.Cluster.SSHSecretID
}

// SaveSshSecretId saves the ssh secret id to database
func (c *EKSCluster) SaveSshSecretId(sshSecretId string) error {
	c.model.Cluster.SSHSecretID = sshSecretId

	err := c.repository.SaveModel(c.model)
	if err != nil {
		return errors.Wrap(err, "failed to save ssh secret id")
	}

	return nil
}

// GetAPIEndpoint returns the Kubernetes Api endpoint
func (c *EKSCluster) GetAPIEndpoint() (string, error) {
	config, err := c.GetK8sConfig()
	if err != nil {
		return "", errors.WrapIf(err, "failed to get cluster's Kubeconfig")
	}

	return pkgCluster.GetAPIEndpointFromKubeconfig(config)
}

// CreateEKSClusterFromModel creates ClusterModel struct from the model
func CreateEKSClusterFromModel(clusterModel *model.ClusterModel) (*EKSCluster, error) {
	db := global.DB()

	m := eksmodel.EKSClusterModel{
		ClusterID: clusterModel.ID,
	}

	err := db.Where(m).Preload("Cluster").Preload("NodePools").Preload("Subnets").First(&m).Error
	if err != nil {
		return nil, err
	}

	repository, err := NewDBEKSClusterRepository(db)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to create DB EKS cluster repository")
	}

	log := log.WithFields(logrus.Fields{"cluster": clusterModel.Name, "clusterID": m.Cluster.ID})

	return &EKSCluster{
		repository: repository,
		model:      &m,
		log:        log.WithField("cluster", clusterModel.Name),
	}, nil
}

func (c *EKSCluster) createAWSCredentialsFromSecret() (*credentials.Credentials, error) {
	clusterSecret, err := c.GetSecretWithValidation()
	if err != nil {
		return nil, err
	}
	return amazon.CreateAWSCredentials(clusterSecret.Values), nil
}

func (c *EKSCluster) SetCurrentWorkflowID(workflowID string) error {
	return c.model.SetCurrentWorkflowID(workflowID)
}

func (c *EKSCluster) PersistSSHGenerate(sshGenerated bool) error {
	return c.model.PersistSSHGenerate(sshGenerated)
}

func (c *EKSCluster) IsSSHGenerated() bool {
	return c.model.IsSSHGenerated()
}

// Persist saves the cluster model
// Deprecated: Do not use.
func (c *EKSCluster) Persist() error {
	return errors.WrapIf(c.repository.SaveModel(c.model), "failed to persist cluster")
}

// GetName returns the name of the cluster
func (c *EKSCluster) GetName() string {
	return c.model.Cluster.Name
}

// GetCloud returns the cloud type of the cluster
func (c *EKSCluster) GetCloud() string {
	return c.model.Cluster.Cloud
}

// GetDistribution returns the distribution type of the cluster
func (c *EKSCluster) GetDistribution() string {
	return c.model.Cluster.Distribution
}

// UpdateNodePools updates nodes pools of a cluster
// 		This will become obsolete once we have the Node Pool API ready
func (c *EKSCluster) UpdateNodePools(request *pkgCluster.UpdateNodePoolsRequest, userId uint) error {
	c.log.Info("start updating node pools")

	awsCred, err := c.createAWSCredentialsFromSecret()
	if err != nil {
		return errors.WrapIf(err, "error retrieving AWS credentials")
	}

	awsSession, err := session.NewSession(&aws.Config{
		Region:      aws.String(c.model.Cluster.Location),
		Credentials: awsCred,
	})
	if err != nil {
		return errors.WrapIf(err, "error creating AWS awsSession")
	}

	autoscalingSrv := autoscaling.New(awsSession)
	cloudformationSrv := cloudformation.New(awsSession)

	waitRoutines := 0
	waitChan := make(chan error)
	defer close(waitChan)

	var caughtErrors []error
	ASGWaitLoopCount := int(asgFulfillmentTimeout.Seconds() / asgWaitLoopSleepSeconds)

	for poolName, nodePool := range request.NodePools {
		asgName, err := c.getAutoScalingGroupName(cloudformationSrv, autoscalingSrv, poolName)
		if err != nil {
			c.log.Errorf("ASG not found for node pool %v. %v", poolName, err.Error())
			continue
		}
		params := &autoscaling.SetDesiredCapacityInput{
			AutoScalingGroupName: aws.String(*asgName),
			DesiredCapacity:      aws.Int64(int64(nodePool.Count)),
			HonorCooldown:        aws.Bool(false),
		}
		c.log.Infof("setting node pool %s size to %d", poolName, nodePool.Count)
		_, err = autoscalingSrv.SetDesiredCapacity(params)
		if err != nil {
			caughtErrors = append(caughtErrors, errors.Wrapf(err, "failed to set size for node pool %s", poolName))
			continue
		}
		c.setNodePoolSize(poolName, nodePool.Count)

		waitRoutines++
		go func(poolName string) {
			waitChan <- nodepools.WaitForASGToBeFulfilled(context.Background(), awsSession, c.log, c.model.Cluster.Name,
				poolName, ASGWaitLoopCount, asgWaitLoopSleepSeconds*time.Second)
		}(poolName)
	}

	// wait for goroutines to finish
	for i := 0; i < waitRoutines; i++ {
		waitErr := <-waitChan
		if waitErr != nil {
			caughtErrors = append(caughtErrors, waitErr)
		}
	}

	return errors.Combine(caughtErrors...)
}

func (c *EKSCluster) getAutoScalingGroupName(cloudformationSrv *cloudformation.CloudFormation, autoscalingSrv *autoscaling.AutoScaling, nodePoolName string) (*string, error) {
	logResourceId := "NodeGroup"
	stackName := nodepools.GenerateNodePoolStackName(c.model.Cluster.Name, nodePoolName)
	describeStackResourceInput := &cloudformation.DescribeStackResourceInput{
		LogicalResourceId: &logResourceId,
		StackName:         aws.String(stackName),
	}
	describeStacksOutput, err := cloudformationSrv.DescribeStackResource(describeStackResourceInput)
	if err != nil {
		return nil, err
	}

	return describeStacksOutput.StackResourceDetail.PhysicalResourceId, nil
}

// GetStatus describes the status of this EKS cluster.
func (c *EKSCluster) GetStatus() (*pkgCluster.GetClusterStatusResponse, error) {
	var hasSpotNodePool bool

	nodePools := make(map[string]*pkgCluster.NodePoolStatus)
	for _, np := range c.model.NodePools {
		if np != nil {
			nodePools[np.Name] = &pkgCluster.NodePoolStatus{
				Autoscaling:       np.Autoscaling,
				Count:             np.Count,
				InstanceType:      np.NodeInstanceType,
				SpotPrice:         np.NodeSpotPrice,
				MinCount:          np.NodeMinCount,
				MaxCount:          np.NodeMaxCount,
				Image:             np.NodeImage,
				CreatorBaseFields: *NewCreatorBaseFields(np.CreatedAt, np.CreatedBy),
				Labels:            np.Labels,
			}
			if np.NodeSpotPrice != "" && np.NodeSpotPrice != "0" {
				hasSpotNodePool = true
			}
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
		Version:           c.model.Version,
		CreatorBaseFields: *NewCreatorBaseFields(c.model.Cluster.CreatedAt, c.model.Cluster.CreatedBy),
		Region:            c.model.Cluster.Location,
		StartedAt:         c.model.Cluster.StartedAt,
	}, nil
}

// GetID returns the DB ID of this cluster
func (c *EKSCluster) GetID() uint {
	return c.model.Cluster.ID
}

func (c *EKSCluster) GetUID() string {
	return c.model.Cluster.UID
}

// GetModel returns the DB model of this cluster
func (c *EKSCluster) GetModel() *eksmodel.EKSClusterModel {
	return c.model
}

// CheckEqualityToUpdate validates the update request
func (c *EKSCluster) CheckEqualityToUpdate(r *pkgCluster.UpdateClusterRequest) error {
	// create update request struct with the stored data to check equality
	preNodePools := make(map[string]*pkgEks.NodePool)
	for _, preNp := range c.model.NodePools {
		preNodePools[preNp.Name] = &pkgEks.NodePool{
			InstanceType: preNp.NodeInstanceType,
			SpotPrice:    preNp.NodeSpotPrice,
			Autoscaling:  preNp.Autoscaling,
			MinCount:     preNp.NodeMinCount,
			MaxCount:     preNp.NodeMaxCount,
			Count:        preNp.Count,
			Image:        preNp.NodeImage,
		}
	}

	preCl := &pkgEks.UpdateClusterAmazonEKS{
		NodePools: preNodePools,
	}

	log.Info("Check stored & updated cluster equals")

	// check equality
	return isDifferent(r.EKS, preCl)
}

// AddDefaultsToUpdate adds defaults to update request
func (c *EKSCluster) AddDefaultsToUpdate(r *pkgCluster.UpdateClusterRequest) {
	// add default node image(s) if needed
	if r != nil && r.EKS != nil && r.EKS.NodePools != nil {
		for _, np := range r.EKS.NodePools {
			if np.Image == "" {
				criteria := eks2.ImageSelectionCriteria{
					Region:            c.model.Cluster.Location,
					InstanceType:      np.InstanceType,
					KubernetesVersion: c.model.Version,
				}

				// TODO: need to return an error
				image, _ := globaleks.ImageSelector().SelectImage(context.Background(), criteria)

				np.Image = image
			}
		}
	}
}

// DeleteFromDatabase deletes model from the database
func (c *EKSCluster) DeleteFromDatabase() error {
	for _, nodePool := range c.model.NodePools {
		if err := c.repository.DeleteNodePool(nodePool); err != nil {
			return err
		}
	}

	for _, subnet := range c.model.Subnets {
		if err := c.repository.DeleteSubnet(subnet); err != nil {
			return err
		}
	}

	if err := c.repository.DeleteModel(c.model); err != nil {
		return err
	}

	if err := c.repository.DeleteClusterModel(&c.model.Cluster); err != nil {
		return err
	}

	c.model = nil

	return nil
}

// SetStatus sets the cluster's status
func (c *EKSCluster) SetStatus(status string, statusMessage string) error {
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

		if err := c.repository.SaveStatusHistory(&statusHistory); err != nil {
			return errors.Wrap(err, "failed to record cluster status change to history")
		}
	}

	if c.model.Cluster.Status == pkgCluster.Creating && (status == pkgCluster.Running || status == pkgCluster.Warning) {
		now := time.Now()
		c.model.Cluster.StartedAt = &now
	}
	c.model.Cluster.Status = status
	c.model.Cluster.StatusMessage = statusMessage

	if err := c.repository.SaveModel(c.model); err != nil {
		return errors.Wrap(err, "failed to update cluster status")
	}

	return nil
}

// NodePoolExists returns true if node pool with nodePoolName exists
func (c *EKSCluster) NodePoolExists(nodePoolName string) bool {
	for _, np := range c.model.NodePools {
		if np != nil && np.Name == nodePoolName {
			return true
		}
	}
	return false
}

func (c *EKSCluster) setNodePoolSize(nodePoolName string, count int) bool {
	for _, np := range c.model.NodePools {
		if np != nil && np.Name == nodePoolName {
			np.Count = count
		}
	}
	return false
}

// IsReady checks if the cluster is running according to the cloud provider.
func (c *EKSCluster) IsReady() (bool, error) {
	awsCred, err := c.createAWSCredentialsFromSecret()
	if err != nil {
		return false, err
	}

	awsSession, err := session.NewSession(&aws.Config{
		Region:      aws.String(c.model.Cluster.Location),
		Credentials: awsCred,
	})
	if err != nil {
		return false, err
	}

	eksSvc := eks.New(awsSession)
	describeCluster := &eks.DescribeClusterInput{Name: aws.String(c.GetName())}
	clusterDesc, err := eksSvc.DescribeCluster(describeCluster)
	if err != nil {
		return false, err
	}

	return aws.StringValue(clusterDesc.Cluster.Status) == eks.ClusterStatusActive, nil
}

// GetSecretWithValidation returns secret from vault
func (c *EKSCluster) GetSecretWithValidation() (*secret.SecretItemResponse, error) {
	return c.CommonClusterBase.getSecret(c)
}

// SaveConfigSecretId saves the config secret id in database
func (c *EKSCluster) SaveConfigSecretId(configSecretId string) error {
	c.model.Cluster.ConfigSecretID = configSecretId

	err := c.repository.SaveModel(c.model)
	if err != nil {
		return errors.Wrap(err, "failed to save config secret id")
	}

	return nil
}

// GetConfigSecretId returns config secret id
func (c *EKSCluster) GetConfigSecretId() string {
	return c.model.Cluster.ConfigSecretID
}

// GetK8sConfig returns the Kubernetes config for internal use
func (c *EKSCluster) GetK8sConfig() ([]byte, error) {
	return c.CommonClusterBase.getConfig(c)
}

// GetK8sUserConfig returns the Kubernetes config for external users
func (c *EKSCluster) GetK8sUserConfig() ([]byte, error) {
	adminConfig, err := c.CommonClusterBase.getConfig(c)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to get raw kubernetes config")
	}

	if global.Config.Distribution.EKS.ExposeAdminKubeconfig {
		return adminConfig, nil
	}

	parsedAdminConfig, err := clientcmd.Load(adminConfig)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to load kubernetes API config")
	}

	userConfig := k8sutil.ExtractConfigBase(parsedAdminConfig).CreateConfigFromTemplate(
		k8sutil.CreateAuthInfoFunc(func(clusterName string) *clientcmdapi.AuthInfo {
			return &clientcmdapi.AuthInfo{
				Exec: &clientcmdapi.ExecConfig{
					APIVersion: "client.authentication.k8s.io/v1alpha1",
					Command:    "aws-iam-authenticator",
					Args:       []string{"token", "-i", clusterName},
				},
			}
		}))

	out, err := clientcmd.Write(*userConfig)
	if err != nil {
		return nil, errors.Wrap(err, "failed to serialize generated user config")
	}

	return out, nil
}

// RequiresSshPublicKey returns true as a public ssh key is needed for bootstrapping
// the cluster
func (c *EKSCluster) RequiresSshPublicKey() bool {
	return true
}

// RbacEnabled returns true if rbac enabled on the cluster
func (c *EKSCluster) RbacEnabled() bool {
	return c.model.Cluster.RbacEnabled
}

// GetEKSNodePools returns EKS node pools from a common cluster.
func GetEKSNodePools(cluster CommonCluster) ([]*eksmodel.AmazonNodePoolsModel, error) {
	ekscluster, ok := cluster.(*EKSCluster)
	if !ok {
		return nil, ErrInvalidClusterInstance
	}

	return ekscluster.model.NodePools, nil
}

func (c *EKSCluster) GetKubernetesVersion() (string, error) {
	return c.model.Version, nil
}
