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
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/auth/credentials"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/cs"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"
	"github.com/banzaicloud/pipeline/helm"
	"github.com/banzaicloud/pipeline/model"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	"github.com/banzaicloud/pipeline/pkg/cluster/acsk"
	"github.com/banzaicloud/pipeline/pkg/cluster/acsk/action"
	pkgCommon "github.com/banzaicloud/pipeline/pkg/common"
	pkgErrors "github.com/banzaicloud/pipeline/pkg/errors"
	"github.com/banzaicloud/pipeline/secret"
	"github.com/banzaicloud/pipeline/secret/verify"
	"github.com/banzaicloud/pipeline/utils"
	"github.com/jmespath/go-jmespath"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
	"k8s.io/client-go/kubernetes"
)

var _ CommonCluster = (*ACSKCluster)(nil)

type ACSKCluster struct {
	alibabaCluster *acsk.AlibabaDescribeClusterResponse
	modelCluster   *model.ClusterModel
	APIEndpoint    string
	log            logrus.FieldLogger
	CommonClusterBase
}

func (*ACSKCluster) RbacEnabled() bool {
	return true
}

func (*ACSKCluster) RequiresSshPublicKey() bool {
	return true
}

func (c *ACSKCluster) ListNodeNames() (nodes pkgCommon.NodeNames, err error) {
	client, err := c.GetAlibabaCSClient(nil)
	if err != nil {
		return
	}

	r, err := getClusterDetails(client, c.modelCluster.ACSK.ClusterID)
	for _, v := range r.Outputs {
		if v.OutputKey == "NodeInstanceIDs" {
			var result []string
			convertedValue := v.OutputValue.([]interface{})
			for _, v := range convertedValue {
				result = append(result, fmt.Sprint(c.modelCluster.ACSK.RegionID, ".", v.(string)))
			}
			nodes = map[string][]string{
				c.modelCluster.ACSK.NodePools[0].Name: result,
			}
		}
	}
	return
}

// GetAlibabaCSClient creates an Alibaba Container Service client with the credentials
func (c *ACSKCluster) GetAlibabaCSClient(cfg *sdk.Config) (*cs.Client, error) {
	cred, err := c.createAlibabaCredentialsFromSecret()
	if err != nil {
		return nil, err
	}

	return createAlibabaCSClient(cred, c.modelCluster.ACSK.RegionID, cfg)
}

// GetAlibabaECSClient creates an Alibaba Elastic Compute Service client with the credentials
func (c *ACSKCluster) GetAlibabaECSClient(cfg *sdk.Config) (*ecs.Client, error) {
	cred, err := c.createAlibabaCredentialsFromSecret()
	if err != nil {
		return nil, err
	}

	return createAlibabaECSClient(cred, c.modelCluster.ACSK.RegionID, cfg)
}

func createACSKNodePoolsFromRequest(pools acsk.NodePools, userId uint) ([]*model.ACSKNodePoolModel, error) {
	nodePoolsCount := len(pools)
	if nodePoolsCount == 0 {
		return nil, pkgErrors.ErrorNodePoolNotProvided
	}

	var res = make([]*model.ACSKNodePoolModel, len(pools))
	var i int
	for name, pool := range pools {
		res[i] = &model.ACSKNodePoolModel{
			CreatedBy:          userId,
			Name:               name,
			InstanceType:       pool.InstanceType,
			SystemDiskCategory: pool.SystemDiskCategory,
			SystemDiskSize:     pool.SystemDiskSize,
			Count:              pool.Count,
		}
		i++
	}

	return res, nil
}

func (c *ACSKCluster) createACSKNodePoolsModelFromUpdateRequestData(pools acsk.NodePools, userId uint) ([]*model.ACSKNodePoolModel, error) {
	currentNodePoolMap := make(map[string]*model.ACSKNodePoolModel, len(c.modelCluster.ACSK.NodePools))
	for _, nodePool := range c.modelCluster.ACSK.NodePools {
		currentNodePoolMap[nodePool.Name] = nodePool
	}

	updatedNodePools := make([]*model.ACSKNodePoolModel, 0, len(pools))

	for nodePoolName, nodePool := range pools {
		if currentNodePoolMap[nodePoolName] != nil {
			updatedNodePools = append(updatedNodePools, &model.ACSKNodePoolModel{
				ID:             currentNodePoolMap[nodePoolName].ID,
				CreatedBy:      currentNodePoolMap[nodePoolName].CreatedBy,
				CreatedAt:      currentNodePoolMap[nodePoolName].CreatedAt,
				ClusterModelId: currentNodePoolMap[nodePoolName].ClusterModelId,
				Name:           nodePoolName,
				InstanceType:   currentNodePoolMap[nodePoolName].InstanceType,
				Count:          nodePool.Count,
			})
		}
	}
	//TODO add support for new NodePools here
	return updatedNodePools, nil
}

//CreateACSKClusterFromModel creates ClusterModel struct from the Alibaba model
func CreateACSKClusterFromModel(clusterModel *model.ClusterModel) (*ACSKCluster, error) {
	log.Debug("Create ClusterModel struct from the model")
	alibabaCluster := ACSKCluster{
		modelCluster: clusterModel,
		log:          log.WithField("cluster", clusterModel.Name),
	}
	return &alibabaCluster, nil
}

func CreateACSKClusterFromRequest(request *pkgCluster.CreateClusterRequest, orgId, userId uint) (*ACSKCluster, error) {
	log.Debug("Create ClusterModel struct from the request")
	cluster := ACSKCluster{
		log: log.WithField("cluster", request.Name),
	}

	nodePools, err := createACSKNodePoolsFromRequest(request.Properties.CreateClusterACSK.NodePools, userId)
	if err != nil {
		return nil, err
	}

	cluster.modelCluster = &model.ClusterModel{
		Name:           request.Name,
		Location:       request.Location,
		Cloud:          request.Cloud,
		Distribution:   pkgCluster.ACSK,
		OrganizationId: orgId,
		SecretId:       request.SecretId,
		ACSK: model.ACSKClusterModel{
			RegionID:                 request.Properties.CreateClusterACSK.RegionID,
			ZoneID:                   request.Properties.CreateClusterACSK.ZoneID,
			MasterInstanceType:       request.Properties.CreateClusterACSK.MasterInstanceType,
			MasterSystemDiskCategory: request.Properties.CreateClusterACSK.MasterSystemDiskCategory,
			MasterSystemDiskSize:     request.Properties.CreateClusterACSK.MasterSystemDiskSize,
			SNATEntry:                true,
			SSHFlags:                 true,
			NodePools:                nodePools,
		},
		CreatedBy: userId,
	}
	return &cluster, nil
}
func (c *ACSKCluster) CreateCluster() error {
	log.Info("Start create cluster (Alibaba)")

	csClient, err := c.GetAlibabaCSClient(nil)
	if err != nil {
		return err
	}

	ecsClient, err := c.GetAlibabaECSClient(nil)
	if err != nil {
		return err
	}
	creationContext := action.NewACSKClusterCreationContext(
		csClient,
		ecsClient,
		acsk.AlibabaClusterCreateParams{
			ClusterType:              "Kubernetes",
			Name:                     c.modelCluster.Name,
			RegionID:                 c.modelCluster.ACSK.RegionID,                        // "eu-central-1"
			ZoneID:                   c.modelCluster.ACSK.ZoneID,                          // "eu-central-1a"
			MasterInstanceType:       c.modelCluster.ACSK.MasterInstanceType,              // "ecs.sn1.large",
			MasterSystemDiskCategory: c.modelCluster.ACSK.MasterSystemDiskCategory,        // "cloud_efficiency",
			MasterSystemDiskSize:     c.modelCluster.ACSK.MasterSystemDiskSize,            // 40,
			WorkerInstanceType:       c.modelCluster.ACSK.NodePools[0].InstanceType,       // "ecs.sn1.large",
			WorkerSystemDiskCategory: c.modelCluster.ACSK.NodePools[0].SystemDiskCategory, // "cloud_efficiency",
			WorkerSystemDiskSize:     c.modelCluster.ACSK.NodePools[0].SystemDiskSize,     // 40,
			KeyPair:                  c.modelCluster.Name,                                 // uploaded keyPair name
			NumOfNodes:               c.modelCluster.ACSK.NodePools[0].Count,              // 1,
			SNATEntry:                c.modelCluster.ACSK.SNATEntry,                       // true,
			SSHFlags:                 c.modelCluster.ACSK.SSHFlags,                        // true,
		})

	clusterSshSecret, err := c.getSshSecret(c)
	if err != nil {
		return err
	}

	actions := []utils.Action{
		action.NewUploadSSHKeyAction(c.log, creationContext, clusterSshSecret),
		action.NewCreateACSKClusterAction(c.log, creationContext),
	}

	resp, err := utils.NewActionExecutor(c.log).ExecuteActions(actions, nil, true)
	if err != nil {
		errors.Wrap(err, "ACSK cluster create error")
		return err
	}
	castedValue, ok := resp.(*acsk.AlibabaDescribeClusterResponse)
	if !ok {
		return errors.New("could not cast cluster create response")
	}
	c.alibabaCluster = castedValue
	c.modelCluster.ACSK.ClusterID = resp.(*acsk.AlibabaDescribeClusterResponse).ClusterID

	kubeConfig, err := c.DownloadK8sConfig()
	if err != nil {
		return err
	}

	restKubeConfig, err := helm.GetK8sClientConfig(kubeConfig)
	if err != nil {
		return err
	}

	kubeClient, err := kubernetes.NewForConfig(restKubeConfig)
	if err != nil {
		return err
	}

	// create default storage class
	err = createDefaultStorageClass(kubeClient, "alicloud/disk")
	if err != nil {
		return err
	}

	return c.modelCluster.Save()
}

type setSchemeSetDomainer interface {
	SetScheme(string)
	SetDomain(string)
}

func setEndpoint(req setSchemeSetDomainer) {
	req.SetScheme(requests.HTTPS)
	req.SetDomain("cs.aliyuncs.com")
}

func getClusterDetails(client *cs.Client, clusterID string) (r *acsk.AlibabaDescribeClusterResponse, err error) {
	req := cs.CreateDescribeClusterDetailRequest()
	setEndpoint(req)
	req.ClusterId = clusterID
	resp, err := client.DescribeClusterDetail(req)
	if err != nil {
		return
	}
	if !resp.IsSuccess() || resp.GetHttpStatus() < 200 || resp.GetHttpStatus() > 299 {
		err = errors.Wrapf(err, "Unexpected http status code: %d", resp.GetHttpStatus())
		return
	}

	err = json.Unmarshal(resp.GetHttpContentBytes(), &r)
	return
}

type alibabaConnectionInfo struct {
	JumpHost    string
	IntranetURI string
	InternetURI string
}

func getConnectionInfo(client *cs.Client, clusterID string) (inf alibabaConnectionInfo, err error) {
	details, err := getClusterDetails(client, clusterID)
	if err != nil {
		return
	}
	for _, v := range details.Outputs {
		if v.OutputKey == "JumpHost" {
			if jh, ok := v.OutputValue.(string); ok {
				inf.JumpHost = jh
			}
		}
		if v.OutputKey == "APIServerIntranet" {
			if intra, ok := v.OutputValue.(string); ok {
				inf.IntranetURI = intra
			}
		}
		if v.OutputKey == "APIServerInternet" {
			if inter, ok := v.OutputValue.(string); ok {
				inf.InternetURI = inter
			}
		}
	}
	if inf.JumpHost == "" {
		err = errors.New("JumpHost not found")
		return
	}
	if inf.IntranetURI == "" {
		err = errors.New("IntranetURI not found")
		return
	}
	if inf.InternetURI == "" {
		err = errors.New("InternetURI not found")
		return
	}

	return
}

func (c *ACSKCluster) Persist(status, statusMessage string) error {
	log.Infof("Model before save: %v", c.modelCluster)
	return c.modelCluster.UpdateStatus(status, statusMessage)
}

func (c *ACSKCluster) DownloadK8sConfig() ([]byte, error) {
	cfg := sdk.NewConfig()
	cfg.AutoRetry = false
	cfg.Debug = true
	cfg.Timeout = time.Minute

	csClient, err := c.GetAlibabaCSClient(cfg)
	if err != nil {
		return nil, err
	}

	info, err := getConnectionInfo(csClient, c.modelCluster.ACSK.ClusterID)
	if err != nil {
		return nil, err
	}
	sshHost := info.JumpHost

	clusterSshSecret, err := c.getSshSecret(c)
	if err != nil {
		return nil, err
	}
	sshKey := secret.NewSSHKeyPair(clusterSshSecret)

	signer, err := ssh.ParsePrivateKey([]byte(sshKey.PrivateKeyData))
	if err != nil {
		return nil, err
	}
	clientConfig := ssh.ClientConfig{
		User: "root",
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	sshClient, err := ssh.Dial("tcp", fmt.Sprint(sshHost, ":22"), &clientConfig)
	if err != nil {
		return nil, err
	}
	defer sshClient.Close()
	var buff bytes.Buffer
	w := bufio.NewWriter(&buff)
	sshSession, err := sshClient.NewSession()
	if err != nil {
		return nil, err
	}
	defer sshSession.Close()
	sshSession.Stdout = w
	sshSession.Run(fmt.Sprintf("cat %s", "/etc/kubernetes/kube.conf"))
	w.Flush()
	return buff.Bytes(), err

}

// GetCloud returns the cloud type of the cluster
func (c *ACSKCluster) GetCloud() string {
	return c.modelCluster.Cloud
}

// GetDistribution returns the distribution type of the cluster
func (c *ACSKCluster) GetDistribution() string {
	return c.modelCluster.Distribution
}

func (c *ACSKCluster) GetName() string {
	return c.modelCluster.Name
}

func (c *ACSKCluster) GetType() string {
	return c.modelCluster.Cloud
}

func (c *ACSKCluster) GetStatus() (*pkgCluster.GetClusterStatusResponse, error) {
	log.Info("Create cluster status response")

	nodePools := make(map[string]*pkgCluster.NodePoolStatus)
	for _, np := range c.modelCluster.ACSK.NodePools {
		if np != nil {
			nodePools[np.Name] = &pkgCluster.NodePoolStatus{
				Count:        np.Count,
				InstanceType: np.InstanceType,
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
		ResourceID:        c.modelCluster.ID,
		NodePools:         nodePools,
		CreatorBaseFields: *NewCreatorBaseFields(c.modelCluster.CreatedAt, c.modelCluster.CreatedBy),
	}, nil
}

func (c *ACSKCluster) DeleteCluster() error {
	log.Info("Start deleting cluster (alibaba)")

	csClient, err := c.GetAlibabaCSClient(nil)
	if err != nil {
		return err
	}

	ecsClient, err := c.GetAlibabaECSClient(nil)
	if err != nil {
		return err
	}

	deleteContext := action.NewACSKClusterDeletionContext(
		csClient,
		ecsClient,
		c.modelCluster.ACSK.ClusterID)

	actions := []utils.Action{
		action.NewDeleteACSKClusterAction(c.log, deleteContext),
		action.NewDeleteSSHKeyAction(c.log, deleteContext, c.modelCluster.Name, c.modelCluster.ACSK.RegionID),
	}

	_, err = utils.NewActionExecutor(c.log).ExecuteActions(actions, nil, true)
	if err != nil {
		errors.Wrap(err, "ACSK cluster delete error")
		return err
	}

	return nil
}

func (c *ACSKCluster) UpdateCluster(request *pkgCluster.UpdateClusterRequest, userId uint) error {
	log.Info("Start updating cluster (Alibaba)")

	csClient, err := c.GetAlibabaCSClient(nil)
	if err != nil {
		return err
	}

	ecsClient, err := c.GetAlibabaECSClient(nil)
	if err != nil {
		return err
	}

	nodePoolModels, err := c.createACSKNodePoolsModelFromUpdateRequestData(request.ACSK.NodePools, userId)
	if err != nil {
		return err
	}

	context := action.NewACSKClusterContext(csClient, ecsClient, c.modelCluster.ACSK.ClusterID)

	actions := []utils.Action{
		action.NewUpdateACSKClusterAction(c.log, nodePoolModels, context),
	}

	resp, err := utils.NewActionExecutor(c.log).ExecuteActions(actions, nil, true)
	if err != nil {
		errors.Wrap(err, "ACSK cluster create error")
		return err
	}

	castedValue, ok := resp.(*acsk.AlibabaDescribeClusterResponse)
	if !ok {
		return errors.New("could not cast cluster create response")
	}

	updatedNodePools := make([]*model.ACSKNodePoolModel, 0, 1)
	updatedNodePools = append(updatedNodePools, nodePoolModels[0])
	c.modelCluster.ACSK.NodePools = updatedNodePools
	c.alibabaCluster = castedValue

	return nil
}

func (c *ACSKCluster) GetID() uint {
	return c.modelCluster.ID
}

func (c *ACSKCluster) GetUID() string {
	return c.modelCluster.UID
}

func (c *ACSKCluster) GetCreatorID() uint {
	return c.modelCluster.CreatedBy
}

func (c *ACSKCluster) GetSecretId() string {
	return c.modelCluster.SecretId
}

func (c *ACSKCluster) GetSshSecretId() string {
	return c.modelCluster.SshSecretId
}

// GetLocation gets where the cluster is.
func (c *ACSKCluster) GetLocation() string {
	return c.modelCluster.Location
}

func (c *ACSKCluster) SaveSshSecretId(sshSecretId string) error {
	return c.modelCluster.UpdateSshSecret(sshSecretId)
}

func (c *ACSKCluster) GetModel() *model.ClusterModel {
	return c.modelCluster
}

func (c *ACSKCluster) CheckEqualityToUpdate(r *pkgCluster.UpdateClusterRequest) error {
	// create update request struct with the stored data to check equality

	preNodePools := make(map[string]*acsk.NodePool)
	for _, preNp := range c.modelCluster.ACSK.NodePools {
		preNodePools[preNp.Name] = &acsk.NodePool{
			InstanceType:       preNp.InstanceType,
			SystemDiskCategory: preNp.SystemDiskCategory,
			SystemDiskSize:     preNp.SystemDiskSize,
			Count:              preNp.Count,
		}
	}

	preCl := &acsk.UpdateClusterACSK{
		NodePools: preNodePools,
	}

	log.Info("Check stored & updated cluster equals")

	// check equality
	return isDifferent(r.ACSK, preCl)
}

func (c *ACSKCluster) AddDefaultsToUpdate(r *pkgCluster.UpdateClusterRequest) {
}

func (c *ACSKCluster) GetAPIEndpoint() (string, error) {
	if c.APIEndpoint != "" {
		return c.APIEndpoint, nil
	}

	client, err := c.GetAlibabaCSClient(nil)
	if err != nil {
		return "", err
	}
	inf, err := getConnectionInfo(client, c.modelCluster.ACSK.ClusterID)
	if err != nil {
		return "", err
	}
	u, err := url.Parse(inf.InternetURI)
	if err != nil {
		return "", err
	}
	c.APIEndpoint = u.Host
	return c.APIEndpoint, nil
}

func (c *ACSKCluster) DeleteFromDatabase() error {
	err := c.modelCluster.Delete()
	if err != nil {
		return err
	}
	c.modelCluster = nil
	return nil
}

func (c *ACSKCluster) GetOrganizationId() uint {
	return c.modelCluster.OrganizationId
}

func (c *ACSKCluster) UpdateStatus(status, statusMessage string) error {
	return c.modelCluster.UpdateStatus(status, statusMessage)
}

func (c *ACSKCluster) GetClusterDetails() (*pkgCluster.DetailsResponse, error) {
	client, err := c.GetAlibabaCSClient(nil)
	if err != nil {
		return nil, err
	}

	r, err := getClusterDetails(client, c.modelCluster.ACSK.ClusterID)
	if err != nil {
		return nil, err
	}
	if r.State != acsk.AlibabaClusterStateRunning {
		return nil, pkgErrors.ErrorClusterNotReady
	}

	nodePools := make(map[string]*pkgCluster.NodeDetails)
	for _, np := range c.modelCluster.ACSK.NodePools {
		nodePools[np.Name] = &pkgCluster.NodeDetails{
			CreatorBaseFields: *NewCreatorBaseFields(np.CreatedAt, np.CreatedBy),
			Count:             np.Count,
			MinCount:          np.Count,
			MaxCount:          np.Count,
		}
	}

	return &pkgCluster.DetailsResponse{
		CreatorBaseFields: *NewCreatorBaseFields(c.modelCluster.CreatedAt, c.modelCluster.CreatedBy),
		Name:              r.Name,
		Id:                c.modelCluster.ID,
		Location:          c.modelCluster.Location,
		NodePools:         nodePools,
		Status:            c.modelCluster.Status,
	}, nil
}

func interfaceArrayToStringArray(in []interface{}) (out []string) {
	out = make([]string, len(in))
	for i, v := range in {
		out[i] = v.(string)
	}
	return
}

func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

func (c *ACSKCluster) validateRegion(regionID string) error {
	client, err := c.GetAlibabaECSClient(nil)
	if err != nil {
		return err
	}

	req := ecs.CreateDescribeRegionsRequest()
	req.SetScheme(requests.HTTPS)
	resp, err := client.DescribeRegions(req)
	if err != nil {
		return err
	}

	var content interface{}
	err = json.Unmarshal(resp.GetHttpContentBytes(), &content)
	if err != nil {
		return err
	}

	items, err := jmespath.Search("Regions.Region[].RegionId", content)
	if err != nil {
		return err
	}

	var validRegions []string
	if r, ok := items.([]interface{}); ok {
		validRegions = interfaceArrayToStringArray(r)
	}

	if !stringInSlice(regionID, validRegions) {
		return errors.New("Invalid region (" + regionID + ") specified, must be one of: " + strings.Join(validRegions, ", "))
	}

	return nil
}

func (c *ACSKCluster) validateZone(regionID, zoneID string) error {
	client, err := c.GetAlibabaECSClient(nil)
	if err != nil {
		return err
	}

	req := ecs.CreateDescribeZonesRequest()
	req.SetScheme(requests.HTTPS)
	req.RegionId = regionID
	resp, err := client.DescribeZones(req)
	if err != nil {
		return err
	}

	var content interface{}
	err = json.Unmarshal(resp.GetHttpContentBytes(), &content)
	if err != nil {
		return err
	}

	items, err := jmespath.Search("Zones.Zone[].ZoneId", content)
	if err != nil {
		return err
	}

	var validZones []string
	if r, ok := items.([]interface{}); ok {
		validZones = interfaceArrayToStringArray(r)
	}

	if !stringInSlice(zoneID, validZones) {
		return errors.New("Invalid region (" + zoneID + ") specified, must be one of: " + strings.Join(validZones, ", "))
	}

	return nil
}

func (c *ACSKCluster) validateInstanceType(regionID, zoneID, instanceType string) error {
	client, err := c.GetAlibabaECSClient(nil)
	if err != nil {
		return err
	}

	req := ecs.CreateDescribeZonesRequest()
	req.SetScheme(requests.HTTPS)
	req.RegionId = regionID
	resp, err := client.DescribeZones(req)
	if err != nil {
		return err
	}

	var content interface{}
	err = json.Unmarshal(resp.GetHttpContentBytes(), &content)
	if err != nil {
		return err
	}

	items, err := jmespath.Search("Zones.Zone[?ZoneId == '"+zoneID+"'].AvailableResources.ResourcesInfo[].InstanceTypes.supportedInstanceType[]", content)
	if err != nil {
		return err
	}

	var validInstanceTypes []string
	if r, ok := items.([]interface{}); ok {
		validInstanceTypes = interfaceArrayToStringArray(r)
	}

	if !stringInSlice(instanceType, validInstanceTypes) {
		return errors.New("Invalid instance_type (" + instanceType + ") specified, must be one of: " + strings.Join(validInstanceTypes, ", "))
	}

	return nil
}

func (c *ACSKCluster) validateSystemDiskCategories(regionID, zoneID, diskCategory string) error {
	client, err := c.GetAlibabaECSClient(nil)
	if err != nil {
		return err
	}

	req := ecs.CreateDescribeZonesRequest()
	req.SetScheme(requests.HTTPS)
	req.RegionId = regionID
	resp, err := client.DescribeZones(req)
	if err != nil {
		return err
	}

	var content interface{}
	err = json.Unmarshal(resp.GetHttpContentBytes(), &content)
	if err != nil {
		return err
	}

	items, err := jmespath.Search("Zones.Zone[?ZoneId == '"+zoneID+"'].AvailableResources.ResourcesInfo[].SystemDiskCategories.supportedSystemDiskCategory[]", content)
	if err != nil {
		return err
	}

	var validDiskCategory []string
	if r, ok := items.([]interface{}); ok {
		validDiskCategory = interfaceArrayToStringArray(r)
	}

	if !stringInSlice(diskCategory, validDiskCategory) {
		return errors.New("Invalid disk_category (" + diskCategory + ") specified, must be one of: " + strings.Join(validDiskCategory, ", "))
	}

	return nil
}

func (c *ACSKCluster) ValidateCreationFields(r *pkgCluster.CreateClusterRequest) error {
	var (
		region       = r.Properties.CreateClusterACSK.RegionID
		zone         = r.Properties.CreateClusterACSK.ZoneID
		instanceType = r.Properties.CreateClusterACSK.MasterInstanceType
		diskCategory = r.Properties.CreateClusterACSK.MasterSystemDiskCategory
	)
	err := c.validateRegion(region)
	if err != nil {
		return err
	}

	err = c.validateZone(region, zone)
	if err != nil {
		return err
	}

	err = c.validateInstanceType(region, zone, instanceType)
	if err != nil {
		return err
	}

	err = c.validateSystemDiskCategories(region, zone, diskCategory)
	if err != nil {
		return err
	}

	for _, np := range r.Properties.CreateClusterACSK.NodePools {
		var (
			instanceType = np.InstanceType
			diskCategory = np.SystemDiskCategory
		)

		err = c.validateInstanceType(region, zone, instanceType)
		if err != nil {
			return err
		}

		err = c.validateSystemDiskCategories(region, zone, diskCategory)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *ACSKCluster) GetSecretWithValidation() (*secret.SecretItemResponse, error) {
	return c.CommonClusterBase.getSecret(c)
}

func (c *ACSKCluster) SaveConfigSecretId(configSecretId string) error {
	return c.modelCluster.UpdateConfigSecret(configSecretId)
}

func (c *ACSKCluster) GetConfigSecretId() string {
	return c.modelCluster.ConfigSecretId
}

func (c *ACSKCluster) GetK8sConfig() ([]byte, error) {
	return c.CommonClusterBase.getConfig(c)
}

func (c *ACSKCluster) createAlibabaCredentialsFromSecret() (*credentials.AccessKeyCredential, error) {
	clusterSecret, err := c.GetSecretWithValidation()
	if err != nil {
		return nil, err
	}
	return verify.CreateAlibabaCredentials(clusterSecret.Values), nil
}

// NeedAdminRights returns true if rbac is enabled and need to create a cluster role binding to user
func (c *ACSKCluster) NeedAdminRights() bool {
	return false
}

// GetKubernetesUserName returns the user ID which needed to create a cluster role binding which gives admin rights to the user
func (c *ACSKCluster) GetKubernetesUserName() (string, error) {
	return "", nil
}

func createAlibabaConfig() *sdk.Config {
	cfg := sdk.NewConfig()
	cfg.AutoRetry = false
	cfg.Debug = true
	cfg.Timeout = time.Minute
	return cfg
}

func createAlibabaCSClient(auth *credentials.AccessKeyCredential, regionID string, cfg *sdk.Config) (*cs.Client, error) {
	if cfg == nil {
		cfg = createAlibabaConfig()
	}

	cred := credentials.NewAccessKeyCredential(auth.AccessKeyId, auth.AccessKeySecret)
	return cs.NewClientWithOptions(regionID, cfg, cred)
}

func createAlibabaECSClient(auth *credentials.AccessKeyCredential, regionID string, cfg *sdk.Config) (*ecs.Client, error) {
	if cfg == nil {
		cfg = createAlibabaConfig()
	}

	cred := credentials.NewAccessKeyCredential(auth.AccessKeyId, auth.AccessKeySecret)
	return ecs.NewClientWithOptions(regionID, cfg, cred)
}
