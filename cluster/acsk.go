package cluster

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/auth/credentials"
	aliErrors "github.com/aliyun/alibaba-cloud-sdk-go/sdk/errors"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/responses"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/cs"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"
	"github.com/banzaicloud/pipeline/model"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	"github.com/banzaicloud/pipeline/pkg/cluster/acsk"
	pkgCommon "github.com/banzaicloud/pipeline/pkg/common"
	pkgErrors "github.com/banzaicloud/pipeline/pkg/errors"
	"github.com/banzaicloud/pipeline/secret"
	"github.com/banzaicloud/pipeline/secret/verify"
	"github.com/jmespath/go-jmespath"
	"github.com/pkg/errors"
)

const (
	AlibabaClusterStateRunning = "running"
	AlibabaClusterStateFailed  = "failed"
)

type alibabaClusterCreateParams struct {
	ClusterType              string `json:"cluster_type"`                  // Network type. Always set to: kubernetes.
	DisableRollback          bool   `json:"disable_rollback,omitempty"`    // Whether the failure is rolled back, true means that the failure does not roll back, and false fails to roll back. If you choose to fail back, it will release the resources produced during the creation process. It is not recommended to use false.
	Name                     string `json:"name"`                          // Cluster name, cluster name can use uppercase and lowercase English letters, Chinese, numbers, and dash.
	TimeoutMins              int    `json:"timeout_mins,omitempty"`        // Cluster resource stack creation timeout in minutes, default value 60.
	RegionID                 string `json:"region_id"`                     // Domain ID of the cluster.
	ZoneID                   string `json:"zoneid"`                        // Regional Availability Zone.
	VPCID                    string `json:"vpcid,omitempty"`               // VPCID, can be empty. If it is not set, the system will automatically create a VPC. The network segment created by the system is 192.168.0.0/16. VpcId and vswitchid can only be empty at the same time or set the corresponding value at the same time.
	VSwitchID                string `json:"vswitchid,omitempty"`           // Switch ID, can be empty. If it is not set, the system automatically creates the switch. The network segment of the switch created by the system is 192.168.0.0/16..
	ContainerCIDR            string `json:"container_cidr,omitempty"`      // The container network segment cannot conflict with the VPC network segment. When the system is selected to automatically create a VPC, the network segment 172.16.0.0/16 is used by default.
	ServiceCIDR              string `json:"service_cidr,omitempty"`        // The service network segment cannot conflict with the VPC segment and container segment. When the system is selected to create a VPC automatically, the network segment 172.19.0.0/20 is used by default.
	MasterInstanceType       string `json:"master_instance_type"`          // Master node ECS specification type code.
	MasterSystemDiskCategory string `json:"master_system_disk_category"`   // Master node system disk type.
	MasterSystemDiskSize     int    `json:"master_system_disk_size"`       // Master node system disk size.
	WorkerInstanceType       string `json:"worker_instance_type"`          // Worker node ECS specification type code.
	WorkerSystemDiskCategory string `json:"worker_system_disk_category"`   // Worker node system disk type.
	WorkerSystemDiskSize     int    `json:"worker_system_disk_size"`       // Worker node system disk size.
	LoginPassword            string `json:"login_password"`                // SSH login password. The password rule is 8 - 30 characters and contains three items (uppercase, lowercase, numbers, and special symbols). Select one of the key_pair.
	KeyPair                  string `json:"key_pair"`                      // Keypair name. Choose one with login_password
	ImageID                  string `json:"image_id"`                      // Image ID, currently only supports the centos system. It is recommended to use centos_7.
	NumOfNodes               int    `json:"num_of_nodes"`                  // Worker node number. The range is [0,300].
	SNATEntry                bool   `json:"snat_entry"`                    // Whether to configure SNAT for the network. If it is automatically created VPC must be set to true. If you are using an existing VPC, set it according to whether you have network access capability
	SSHFlags                 bool   `json:"ssh_flags,omitempty"`           // Whether to open public network SSH login.
	CloudMonitorFlags        bool   `json:"cloud_monitor_flags,omitempty"` // Whether to install cloud monitoring plug-in.
}

type alibabaClusterCreateResponse struct {
	ClusterID string `json:"cluster_id"`
	RequestID string `json:"request_id"`
	TaskID    string `json:"task_id"`
}

// alibabaDescribeClusterResponse docs: https://www.alibabacloud.com/help/doc-detail/69344.htm
type alibabaDescribeClusterResponse struct {
	AgentVersion           string       `json:"agent_version"`            // The Agent version.
	ClusterID              string       `json:"cluster_id"`               // The cluster ID, which is the unique identifier of the cluster.
	Created                time.Time    `json:"created"`                  // The created time of the cluster.
	ExternalLoadbalancerID string       `json:"external_loadbalancer_id"` // The Server Load Balancer instance ID of the cluster.
	MasterURL              string       `json:"master_url"`               // The master address of the cluster, which is used to connect to the cluster to perform operations.
	Name                   string       `json:"name"`                     // The cluster name, which is specified when you create the cluster and is unique for each account.
	NetworkMode            string       `json:"network_mode"`             // The network mode of the cluster (Classic or Virtual Private Cloud (VPC)).
	RegionID               string       `json:"region_id"`                // The ID of the region in which the cluster resides.
	SecurityGroupID        string       `json:"security_group_id"`        // The security group ID.
	Size                   int          `json:"size"`                     // The number of nodes.
	State                  string       `json:"state"`                    // The cluster status.
	Updated                time.Time    `json:"updated"`                  // Last updated time.
	VPCID                  string       `json:"vpc_id"`                   // VPC ID.
	VSwitchID              string       `json:"vswitch_id"`               // VSwitch ID.
	ZoneID                 string       `json:"zone_id"`                  // Zone ID.
	Outputs                []outputItem `json:"outputs,omitempty"`
}

type outputItem struct {
	Description string
	OutputKey   string
	OutputValue interface{}
}

type alibabaScaleClusterParams struct {
	DisableRollback          bool   `json:"disable_rollback,omitempty"`  // Whether the failure is rolled back, true means that the failure does not roll back, and false fails to roll back. If you choose to fail back, it will release the resources produced during the creation process. It is not recommended to use false.
	TimeoutMins              int    `json:"timeout_mins,omitempty"`      // Cluster resource stack creation timeout in minutes, default value 60.
	WorkerInstanceType       string `json:"worker_instance_type"`        // Worker node ECS specification type code.
	WorkerSystemDiskCategory string `json:"worker_system_disk_category"` // Worker node system disk type.
	WorkerSystemDiskSize     int    `json:"worker_system_disk_size"`     // Worker node system disk size.
	LoginPassword            string `json:"login_password"`              // SSH login password. The password rule is 8 - 30 characters and contains three items (uppercase, lowercase, numbers, and special symbols). Select one of the key_pair.
	ImageID                  string `json:"image_id"`                    // Image ID, currently only supports the centos system. It is recommended to use centos_7.
	NumOfNodes               int    `json:"num_of_nodes"`                // Worker node number. The range is [0,300].
}

type obtainClusterConfigRequest struct {
	*requests.RoaRequest
	ClusterId string `position:"Path" name:"ClusterId"`
}

type obtainClusterConfigResponse struct {
	*responses.BaseResponse
}

func createObtainClusterConfigRequest(clusterID string) (request *obtainClusterConfigRequest) {
	request = &obtainClusterConfigRequest{
		RoaRequest: &requests.RoaRequest{},
		ClusterId:  clusterID,
	}
	request.InitWithApiInfo("CS", "2015-12-15", "UserConfig", "/k8s/[ClusterId]/user_config", "", "")
	request.Method = requests.GET
	return
}

func createObtainClusterConfigResponse() (response *obtainClusterConfigResponse) {
	response = &obtainClusterConfigResponse{
		BaseResponse: &responses.BaseResponse{},
	}
	return
}

type clusterConfigResponse struct {
	Config string `json:"config"`
}

var _ CommonCluster = (*ACSKCluster)(nil)

type ACSKCluster struct {
	alibabaCluster *alibabaDescribeClusterResponse
	modelCluster   *model.ClusterModel
	APIEndpoint    string
	CommonClusterBase
}

func (*ACSKCluster) RbacEnabled() bool {
	return true
}

func (*ACSKCluster) RequiresSshPublicKey() bool {
	return true
}

func (*ACSKCluster) ListNodeNames() (pkgCommon.NodeNames, error) {
	return nil, nil
}

// GetAlibabaCSClient creates an Alibaba Container Service client with the credentials
func (c *ACSKCluster) GetAlibabaCSClient(cfg *sdk.Config) (*cs.Client, error) {
	cred, err := c.createAlibabaCredentialsFromSecret()
	if err != nil {
		return nil, err
	}

	return verify.CreateAlibabaCSClient(cred, c.modelCluster.ACSK.RegionID, cfg)
}

// GetAlibabaECSClient creates an Alibaba Elastic Compute Service client with the credentials
func (c *ACSKCluster) GetAlibabaECSClient(cfg *sdk.Config) (*ecs.Client, error) {
	cred, err := c.createAlibabaCredentialsFromSecret()
	if err != nil {
		return nil, err
	}

	return verify.CreateAlibabaECSClient(cred, c.modelCluster.ACSK.RegionID, cfg)
}

func createACSKNodePoolsModelFromRequestData(pools acsk.NodePools, userId uint) ([]*model.ACSKNodePoolModel, error) {
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
			Image:              pool.Image,
			Count:              pool.Count,
		}
		i++
	}

	return res, nil
}

//CreateACSKClusterFromModel creates ClusterModel struct from the Alibaba model
func CreateACSKClusterFromModel(clusterModel *model.ClusterModel) (*ACSKCluster, error) {
	log.Debug("Create ClusterModel struct from the model")
	alibabaCluster := ACSKCluster{
		modelCluster: clusterModel,
	}
	return &alibabaCluster, nil
}

func CreateACSKClusterFromRequest(request *pkgCluster.CreateClusterRequest, orgId, userId uint) (*ACSKCluster, error) {
	log.Debug("Create ClusterModel struct from the request")
	var cluster ACSKCluster

	nodePools, err := createACSKNodePoolsModelFromRequestData(request.Properties.CreateClusterACSK.NodePools, userId)
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
			LoginPassword:            "",
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

	// TODO: create method for this
	cfg := sdk.NewConfig()
	cfg.AutoRetry = false
	cfg.Debug = true
	cfg.Timeout = time.Minute
	client, err := c.GetAlibabaCSClient(cfg)
	if err != nil {
		return err
	}

	clusterSshSecret, err := c.getSshSecret(c)
	if err != nil {
		return err
	}

	sshKey := secret.NewSSHKeyPair(clusterSshSecret)
	_ = sshKey

	// setup cluster creation request
	params := alibabaClusterCreateParams{
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
		LoginPassword:            c.modelCluster.ACSK.LoginPassword,                   // TODO: change me to KeyPair
		// KeyPair:                  sshKey.PublicKeyData, // this one should be a keypair name, so keypair should be uploaded
		ImageID:    c.modelCluster.ACSK.NodePools[0].Image, // "centos_7",
		NumOfNodes: c.modelCluster.ACSK.NodePools[0].Count, // 1,
		SNATEntry:  c.modelCluster.ACSK.SNATEntry,          // true,
		SSHFlags:   c.modelCluster.ACSK.SSHFlags,           // true,
	}
	p, err := json.Marshal(&params)
	if err != nil {
		return err
	}

	req := cs.CreateCreateClusterRequest()
	setEndpoint(req)
	setJSONContent(req, p)

	// do a cluster creation
	resp, err := client.CreateCluster(req)
	if err != nil {
		return err
	}
	if !resp.IsSuccess() || resp.GetHttpStatus() < 200 || resp.GetHttpStatus() > 299 {
		// TODO: create error message
		return errors.New("TODO error")
	}

	// parse response
	var r alibabaClusterCreateResponse
	err = json.Unmarshal(resp.GetHttpContentBytes(), &r)
	if err != nil {
		return err
	}

	log.Infof("Alibaba cluster creating with id %s", r.ClusterID)

	// wait for cluster created
	log.Info("Waiting for cluster...")
	aliCluster, err := waitForClusterState(client, r.ClusterID)
	if err != nil {
		return err
	}
	c.alibabaCluster = aliCluster

	c.modelCluster.ACSK.ClusterID = r.ClusterID
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

type setContentSetContentTyper interface {
	SetContent([]byte)
	SetContentType(string)
}

func setJSONContent(req setContentSetContentTyper, p []byte) {
	req.SetContent(p)
	req.SetContentType("application/json")
}

func getClusterDetails(client *cs.Client, clusterID string) (r *alibabaDescribeClusterResponse, err error) {
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

// waitForClusterState docs: https://www.alibabacloud.com/help/doc-detail/26005.htm
func waitForClusterState(client *cs.Client, clusterID string) (*alibabaDescribeClusterResponse, error) {
	var (
		r     *alibabaDescribeClusterResponse
		state string
		err   error
	)
	for {
		r, err = getClusterDetails(client, clusterID)
		if err != nil {
			return nil, err
		}

		if r.State != state {
			log.Infof("%s cluster %s", r.State, clusterID)
			state = r.State
		}

		switch r.State {
		case AlibabaClusterStateRunning:
			return r, nil
		case AlibabaClusterStateFailed:
			return nil, errors.New("The cluster creation failed")
		default:
			time.Sleep(time.Second * 5)
		}
	}
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

	ecsClient, err := c.GetAlibabaECSClient(cfg)
	if err != nil {
		return nil, err
	}

	downloadConfigRequest := createObtainClusterConfigRequest(c.modelCluster.ACSK.ClusterID)
	setEndpoint(downloadConfigRequest)

	downloadConfigResponse := createObtainClusterConfigResponse()

	err = ecsClient.DoAction(downloadConfigRequest, downloadConfigResponse)
	if err != nil {
		return nil, err
	}

	var config clusterConfigResponse
	err = json.Unmarshal(downloadConfigResponse.GetHttpContentBytes(), &config)
	if err != nil {
		return nil, err
	}

	return []byte(config.Config), err
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

	client, err := c.GetAlibabaCSClient(nil)
	if err != nil {
		return err
	}

	req := cs.CreateDeleteClusterRequest()
	req.ClusterId = c.modelCluster.ACSK.ClusterID

	setEndpoint(req)
	resp, err := client.DeleteCluster(req)
	if err != nil {
		if sdkErr, ok := err.(*aliErrors.ServerError); ok {
			if strings.Contains(sdkErr.Message(), "ErrorClusterNotFound") {
				// Cluster has been already deleted
				return nil
			}
		}
		log.Errorf("DeleteClusterResponse: %#v\n", resp)
		return err
	}

	if resp.GetHttpStatus() != http.StatusAccepted {
		return fmt.Errorf("Unexpected http status code: %d", resp.GetHttpStatus())
	}

	return nil
}

func (c *ACSKCluster) UpdateCluster(request *pkgCluster.UpdateClusterRequest, userId uint) error {
	log.Info("Start updating cluster (alibaba)")

	client, err := c.GetAlibabaCSClient(nil)
	if err != nil {
		return err
	}

	req := cs.CreateScaleClusterRequest()
	req.ClusterId = c.modelCluster.ACSK.ClusterID

	nodePoolModels, err := createACSKNodePoolsModelFromRequestData(request.ACSK.NodePools, userId)
	if err != nil {
		return err
	}
	params := alibabaScaleClusterParams{
		DisableRollback:          true,
		TimeoutMins:              60,
		WorkerInstanceType:       nodePoolModels[0].InstanceType,
		WorkerSystemDiskCategory: nodePoolModels[0].SystemDiskCategory,
		WorkerSystemDiskSize:     nodePoolModels[0].SystemDiskSize,
		LoginPassword:            c.modelCluster.ACSK.LoginPassword,
		ImageID:                  nodePoolModels[0].Image,
		NumOfNodes:               nodePoolModels[0].Count,
	}

	p, err := json.Marshal(&params)
	if err != nil {
		return err
	}

	setEndpoint(req)
	setJSONContent(req, p)

	resp, err := client.ScaleCluster(req)
	if err != nil {
		return err
	}

	var r alibabaClusterCreateResponse
	err = json.Unmarshal(resp.GetHttpContentBytes(), &r)
	if err != nil {
		return err
	}

	cluster, err := waitForClusterState(client, r.ClusterID)
	if err != nil {
		return err
	}

	updatedNodePools := make([]*model.ACSKNodePoolModel, 0, 1)
	updatedNodePools = append(updatedNodePools, nodePoolModels[0])
	c.modelCluster.ACSK.NodePools = updatedNodePools
	c.alibabaCluster = cluster

	return nil
}

func (c *ACSKCluster) GetID() uint {
	return c.modelCluster.ID
}

func (c *ACSKCluster) GetUID() string {
	return c.modelCluster.UID
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
			Image:              preNp.Image,
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
	for _, np := range r.ACSK.NodePools {
		if np.Image == "" {
			np.Image = acsk.DefaultImage
		}
	}
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
	if r.State != AlibabaClusterStateRunning {
		return nil, pkgErrors.ErrorClusterNotReady
	}

	return &pkgCluster.DetailsResponse{
		Name: r.Name,
		Id:   c.modelCluster.ID,
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
