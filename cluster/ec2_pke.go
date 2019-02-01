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
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	pipConfig "github.com/banzaicloud/pipeline/config"
	"github.com/banzaicloud/pipeline/internal/backoff"
	"github.com/banzaicloud/pipeline/internal/cluster"
	banzaicloudDB "github.com/banzaicloud/pipeline/internal/providers/banzaicloud"
	"github.com/banzaicloud/pipeline/model"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	"github.com/banzaicloud/pipeline/pkg/cluster/pke"
	"github.com/banzaicloud/pipeline/pkg/common"
	pkgError "github.com/banzaicloud/pipeline/pkg/errors"
	pkgSecret "github.com/banzaicloud/pipeline/pkg/secret"
	"github.com/banzaicloud/pipeline/secret"
	"github.com/banzaicloud/pipeline/secret/verify"
	"github.com/goph/emperror"
	"github.com/jinzhu/gorm"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

var _ CommonCluster = (*EC2ClusterPKE)(nil)

type EC2ClusterPKE struct {
	db    *gorm.DB
	model *banzaicloudDB.EC2PKEClusterModel
	//amazonCluster *ec2.EC2 //Don't use this directly
	APIEndpoint string
	log         logrus.FieldLogger
	session     *session.Session
	CommonClusterBase
}

func (c *EC2ClusterPKE) GetSecurityScan() bool {
	return c.model.Cluster.SecurityScan
}

func (c *EC2ClusterPKE) SetSecurityScan(scan bool) {
	c.model.Cluster.SecurityScan = scan
}

func (c *EC2ClusterPKE) GetLogging() bool {
	return c.model.Cluster.Logging
}

func (c *EC2ClusterPKE) SetLogging(l bool) {
	c.model.Cluster.Logging = l
}

func (c *EC2ClusterPKE) GetMonitoring() bool {
	return c.model.Cluster.Monitoring
}

func (c *EC2ClusterPKE) SetMonitoring(m bool) {
	c.model.Cluster.Monitoring = m
}

// GetScaleOptions returns scale options for the cluster
func (c *EC2ClusterPKE) GetScaleOptions() *pkgCluster.ScaleOptions {
	return getScaleOptionsFromModel(c.model.Cluster.ScaleOptions)
}

// SetScaleOptions sets scale options for the cluster
func (c *EC2ClusterPKE) SetScaleOptions(scaleOptions *pkgCluster.ScaleOptions) {
	updateScaleOptions(&c.model.Cluster.ScaleOptions, scaleOptions)
}

func (c *EC2ClusterPKE) GetServiceMesh() bool {
	return c.model.Cluster.ServiceMesh
}

func (c *EC2ClusterPKE) SetServiceMesh(m bool) {
	c.model.Cluster.ServiceMesh = m
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

func (c *EC2ClusterPKE) GetCreatedBy() uint {
	return c.model.Cluster.CreatedBy
}

func (c *EC2ClusterPKE) GetSecretId() string {
	return c.model.Cluster.SecretID
}

func (c *EC2ClusterPKE) GetSshSecretId() string {
	return c.model.Cluster.SSHSecretID
}

func (c *EC2ClusterPKE) SaveSshSecretId(sshSecretId string) error {
	c.model.Cluster.SSHSecretID = sshSecretId

	err := c.db.Save(&c.model).Error
	if err != nil {
		return emperror.WrapWith(err, "failed to save ssh secret", "secret", sshSecretId)
	}

	return nil
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
	return c.model.Cluster.ConfigSecretID
}

func (c *EC2ClusterPKE) GetSecretWithValidation() (*secret.SecretItemResponse, error) {
	return c.CommonClusterBase.getSecret(c)
}

func (c *EC2ClusterPKE) Persist(string, string) error {
	err := c.db.Save(c.model).Error
	return err
}

func (c *EC2ClusterPKE) UpdateStatus(status, statusMessage string) error {
	originalStatus := c.model.Cluster.Status
	originalStatusMessage := c.model.Cluster.StatusMessage

	c.model.Cluster.Status = status
	c.model.Cluster.StatusMessage = statusMessage

	err := c.db.Save(&c.model).Error
	if err != nil {
		return errors.Wrap(err, "failed to update status")
	}

	if originalStatus != status {
		statusHistory := &cluster.StatusHistoryModel{
			ClusterID:   c.model.Cluster.ID,
			ClusterName: c.model.Cluster.Name,

			FromStatus:        originalStatus,
			FromStatusMessage: originalStatusMessage,
			ToStatus:          status,
			ToStatusMessage:   statusMessage,
		}

		err := c.db.Save(&statusHistory).Error
		if err != nil {
			return errors.Wrap(err, "failed to update cluster status history")
		}
	}

	return nil
}

// DeleteFromDatabase deletes the distribution related entities from the database
func (c *EC2ClusterPKE) DeleteFromDatabase() error {

	// dependencies are deleted using a GORM hook!
	if e := c.db.Delete(c.model).Error; e != nil {
		return emperror.WrapWith(e, "failed to delete EC2BanzaiCloudCluster", "distro", c.model.ID)
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
	awsCred := verify.CreateAWSCredentials(secret.Values)
	return session.NewSession(&aws.Config{
		Region:      aws.String(c.model.Cluster.Location),
		Credentials: awsCred,
	})
}

func (c *EC2ClusterPKE) CreatePKECluster(tokenGenerator TokenGenerator, externalBaseURL string) error {
	// Fetch

	// Generate certificates
	clusterUidTag := fmt.Sprintf("clusterUID:%s", c.GetUID())
	req := &secret.CreateSecretRequest{
		Name: fmt.Sprintf("cluster-%d-ca", c.GetID()),
		Values: map[string]string{
			pkgSecret.ClusterUID: fmt.Sprintf("%d-%d", c.GetOrganizationId(), c.GetID()),
		},
		Type: pkgSecret.PKESecretType,
		Tags: []string{
			clusterUidTag,
			pkgSecret.TagBanzaiReadonly,
			pkgSecret.TagBanzaiHidden,
		},
	}
	_, err := secret.Store.GetOrCreate(c.GetOrganizationId(), req)
	if err != nil {
		return err
	}
	// prepare input for real AWS flow
	_, err = c.GetPipelineToken(tokenGenerator)
	if err != nil {
		return emperror.Wrap(err, "can't generate Pipeline token")
	}
	//client, err := c.GetAWSClient()
	//if err != nil {
	//	return err
	//}
	//cloudformationSrv := cloudformation.New(client)
	//err = CreateMasterCF(cloudformationSrv)
	//if err != nil {
	//	return emperror.Wrap(err, "can't create master CF template")
	//}
	token := "XXX"
	for _, nodePool := range c.model.NodePools {
		cmd := c.GetBootstrapCommand(nodePool.Name, externalBaseURL, token)
		c.log.Debugf("TODO: start ASG with command %s", cmd)
	}

	clusters := cluster.NewClusters(pipConfig.DB()) // TODO get it from non-global context

	err = backoff.Retry(func() error {
		id, err := clusters.GetConfigSecretIDByClusterID(c.GetOrganizationId(), c.GetID())
		if err != nil {
			return err
		} else if id == "" {
			return errors.New("no Kubeconfig received from master")
		}
		c.model.Cluster.ConfigSecretID = id
		return nil
	}, backoff.NewConstantBackoffPolicy(&backoff.ConstantBackoffConfig{
		Delay:          20 * time.Second,
		MaxElapsedTime: 30 * time.Minute}))

	if err != nil {
		return emperror.Wrap(err, "timeout")
	}
	return nil
}

// RegisterNode adds a Node to the DB
func (c *EC2ClusterPKE) RegisterNode(name, nodePoolName, ip string, master, worker bool) error {

	db := pipConfig.DB()
	nodePool := banzaicloudDB.NodePool{
		Name:      nodePoolName,
		ClusterID: c.GetID(),
	}

	roles := banzaicloudDB.Roles{}
	if master {
		roles = append(roles, banzaicloudDB.RoleMaster)
	}
	if worker {
		roles = append(roles, banzaicloudDB.RoleWorker)
	}

	if err := db.Where(nodePool).Attrs(banzaicloudDB.NodePool{
		Roles: roles,
	}).FirstOrCreate(&nodePool).Error; err != nil {
		return emperror.Wrap(err, "failed to register nodepool")
	}

	node := banzaicloudDB.Host{
		NodePoolID: nodePool.NodePoolID,
		Name:       name,
	}

	if err := db.Where(node).Attrs(banzaicloudDB.Host{
		Labels:    make(banzaicloudDB.Labels),
		PrivateIP: ip,
	}).FirstOrCreate(&node).Error; err != nil {
		return emperror.Wrap(err, "failed to register node")
	}
	c.log.WithField("node", name).Info("node registered")

	return nil
}

// Create master CF template
func CreateMasterCF(formation *cloudformation.CloudFormation) error {
	return nil
}

func (c *EC2ClusterPKE) ValidateCreationFields(r *pkgCluster.CreateClusterRequest) error {
	// TODO(Ecsy): implement me
	return nil
}

func (c *EC2ClusterPKE) UpdateCluster(*pkgCluster.UpdateClusterRequest, uint) error {
	panic("implement me")
}

func (c *EC2ClusterPKE) UpdateNodePools(*pkgCluster.UpdateNodePoolsRequest, uint) error {
	panic("implement me")
}

func (c *EC2ClusterPKE) CheckEqualityToUpdate(*pkgCluster.UpdateClusterRequest) error {
	panic("implement me")
}

func (c *EC2ClusterPKE) AddDefaultsToUpdate(*pkgCluster.UpdateClusterRequest) {
	panic("implement me")
}

func (c *EC2ClusterPKE) DeleteCluster() error {
	// do nothing (the cluster should be left on the provider for now
	return nil
}

func (c *EC2ClusterPKE) DownloadK8sConfig() ([]byte, error) {
	return nil, pkgError.ErrorFunctionShouldNotBeCalled
}

func (c *EC2ClusterPKE) GetAPIEndpoint() (string, error) {
	if c.APIEndpoint != "" {
		return c.APIEndpoint, nil
	}

	config, err := c.GetK8sConfig()
	if err != nil {
		return "", emperror.Wrap(err, "failed to get cluster's Kubeconfig")
	}

	kubeConf := kubeConfig{}
	err = yaml.Unmarshal(config, &kubeConf)
	if err != nil {
		return "", emperror.Wrap(err, "failed to parse cluster's Kubeconfig")
	}

	c.APIEndpoint = kubeConf.Clusters[0].Cluster.Server
	return c.APIEndpoint, nil
}

func (c *EC2ClusterPKE) GetK8sIpv4Cidrs() (*pkgCluster.Ipv4Cidrs, error) {
	//TODO
	return nil, errors.New("not implemented")
}

func (c *EC2ClusterPKE) GetK8sConfig() ([]byte, error) {
	return c.CommonClusterBase.getConfig(c)
}

func (c *EC2ClusterPKE) RbacEnabled() bool {
	return c.model.Kubernetes.RBACEnabled
}

func (c *EC2ClusterPKE) NeedAdminRights() bool {
	return false
}

func (c *EC2ClusterPKE) GetKubernetesUserName() (string, error) {
	return "", nil
}

func (c *EC2ClusterPKE) GetStatus() (*pkgCluster.GetClusterStatusResponse, error) {
	log.Info("Create cluster status response")

	hasSpotNodePool := false
	nodePools := make(map[string]*pkgCluster.NodePoolStatus)
	for _, np := range c.model.NodePools {
		providerConfig := banzaicloudDB.NodePoolProviderConfigAmazon{}
		err := mapstructure.Decode(np.ProviderConfig, &providerConfig)
		if err != nil {
			return nil, emperror.WrapWith(err, "failed to decode providerconfig", "cluster", c.model.Cluster.Name)
		}
		nodePools[np.Name] = &pkgCluster.NodePoolStatus{
			Count:             len(np.Hosts),
			InstanceType:      providerConfig.AutoScalingGroup.InstanceType,
			SpotPrice:         providerConfig.AutoScalingGroup.SpotPrice,
			CreatorBaseFields: *NewCreatorBaseFields(np.CreatedAt, np.CreatedBy),
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
		Logging:           c.GetLogging(),
		Monitoring:        c.GetMonitoring(),
		ServiceMesh:       c.GetServiceMesh(),
		SecurityScan:      c.GetSecurityScan(),
		NodePools:         nodePools,
		Version:           c.model.Kubernetes.Version,
		CreatorBaseFields: *NewCreatorBaseFields(c.model.Cluster.CreatedAt, c.model.Cluster.CreatedBy),
		Region:            c.model.Cluster.Location,
	}, nil
}

// IsReady checks if the cluster is running according to the cloud provider.
func (c *EC2ClusterPKE) IsReady() (bool, error) {
	// TODO: is this a correct implementation?
	return true, nil
}

// ListNodeNames returns node names to label them
func (c *EC2ClusterPKE) ListNodeNames() (common.NodeNames, error) {
	var nodes = make(map[string][]string)
	for _, nodepool := range c.model.NodePools {
		nodes[nodepool.Name] = []string{}
		for _, host := range nodepool.Hosts {
			nodes[nodepool.Name] = append(nodes[nodepool.Name], host.Name)
		}
	}
	return nodes, nil
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
	crt := secret.Values[pkgSecret.KubernetesCACert]
	block, _ := pem.Decode([]byte(crt))
	if block == nil {
		return "", errors.New("failed to parse certificate")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return "", emperror.Wrapf(err, "failed to parse certificate")
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
func (c *EC2ClusterPKE) GetBootstrapCommand(nodePoolName, url, token string) string {
	roles := ""
	for _, np := range c.model.NodePools {
		if np.Name == nodePoolName {
			for _, role := range np.Roles {
				roles += fmt.Sprintf(" --role=%s", role)
			}
		}
	}
	return fmt.Sprintf("pke-installer install --pipeline-url=%q --pipeline-token=%q --pipeline-org-id=%d --pipeline-cluster-id=%d --node-pool=%q%s",
		url, token, c.model.Cluster.OrganizationID, c.model.Cluster.ID, nodePoolName, roles)
}

func CreateEC2ClusterPKEFromRequest(request *pkgCluster.CreateClusterRequest, orgId uint, userId uint) (*EC2ClusterPKE, error) {
	c := &EC2ClusterPKE{
		log: log.WithField("cluster", request.Name).WithField("organization", orgId),
	}

	c.db = pipConfig.DB()

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

	c.model = &banzaicloudDB.EC2PKEClusterModel{
		Cluster: cluster.ClusterModel{
			Name:           request.Name,
			Location:       request.Location,
			Cloud:          request.Cloud,
			Distribution:   pkgCluster.BanzaiCloud,
			OrganizationID: orgId,
			RbacEnabled:    kubernetes.RBAC.Enabled,
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

	db := pipConfig.DB()

	m := banzaicloudDB.EC2PKEClusterModel{
		ClusterID: modelCluster.ID,
	}

	log.Debug("Load EC2 props from database")
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

func createEC2ClusterPKENodePoolsFromRequest(pools pke.NodePools, userId uint) banzaicloudDB.NodePools {
	var nps banzaicloudDB.NodePools

	for _, pool := range pools {
		np := banzaicloudDB.NodePool{
			Name:           pool.Name,
			Roles:          convertRoles(pool.Roles),
			Hosts:          convertHosts(pool.Hosts),
			Provider:       convertNodePoolProvider(pool.Provider),
			ProviderConfig: pool.ProviderConfig,
		}
		np.CreatedBy = userId
		nps = append(nps, np)
	}
	return nps
}

func convertRoles(roles pke.Roles) (result banzaicloudDB.Roles) {
	for _, role := range roles {
		result = append(result, banzaicloudDB.Role(role))
	}
	return
}

func convertHosts(hosts pke.Hosts) (result banzaicloudDB.Hosts) {
	for _, host := range hosts {
		result = append(result, banzaicloudDB.Host{
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

func convertNodePoolProvider(provider pke.NodePoolProvider) (result banzaicloudDB.NodePoolProvider) {
	return banzaicloudDB.NodePoolProvider(provider)
}

func convertLabels(labels pke.Labels) banzaicloudDB.Labels {
	res := make(banzaicloudDB.Labels, len(labels))
	for k, v := range labels {
		res[k] = v
	}
	return res
}

func convertTaints(taints pke.Taints) (result banzaicloudDB.Taints) {
	for _, taint := range taints {
		result = append(result, banzaicloudDB.Taint(taint))
	}
	return
}

func createEC2PKENetworkFromRequest(network pke.Network, userId uint) banzaicloudDB.Network {
	n := banzaicloudDB.Network{
		ServiceCIDR:      network.ServiceCIDR,
		PodCIDR:          network.PodCIDR,
		Provider:         convertNetworkProvider(network.Provider),
		APIServerAddress: network.APIServerAddress,
	}
	n.CreatedBy = userId
	return n
}

func convertNetworkProvider(provider pke.NetworkProvider) (result banzaicloudDB.NetworkProvider) {
	return banzaicloudDB.NetworkProvider(provider)
}

func createEC2ClusterPKEFromRequest(kubernetes pke.Kubernetes, userId uint) banzaicloudDB.Kubernetes {
	k := banzaicloudDB.Kubernetes{
		Version: kubernetes.Version,
		RBAC:    banzaicloudDB.RBAC{Enabled: kubernetes.RBAC.Enabled},
	}
	k.CreatedBy = userId
	return k
}

func createEC2ClusterPKEKubeADMFromRequest(kubernetes pke.KubeADM, userId uint) banzaicloudDB.KubeADM {
	a := banzaicloudDB.KubeADM{
		ExtraArgs: convertExtraArgs(kubernetes.ExtraArgs),
	}
	a.CreatedBy = userId
	return a
}

func convertExtraArgs(extraArgs pke.ExtraArgs) banzaicloudDB.ExtraArgs {
	res := make(banzaicloudDB.ExtraArgs, len(extraArgs))
	for k, v := range extraArgs {
		res[k] = banzaicloudDB.ExtraArg(v)
	}
	return res
}

func createEC2ClusterPKECRIFromRequest(cri pke.CRI, userId uint) banzaicloudDB.CRI {
	c := banzaicloudDB.CRI{
		Runtime:       banzaicloudDB.Runtime(cri.Runtime),
		RuntimeConfig: cri.RuntimeConfig,
	}
	c.CreatedBy = userId
	return c
}

func getMasterInstanceTypeAndImageFromNodePools(nodepools banzaicloudDB.NodePools) (masterInstanceType string, masterImage string, err error) {
	for _, nodepool := range nodepools {
		for _, role := range nodepool.Roles {
			if role == banzaicloudDB.RoleMaster {
				switch nodepool.Provider {
				case banzaicloudDB.NPPAmazon:
					providerConfig := banzaicloudDB.NodePoolProviderConfigAmazon{}
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
