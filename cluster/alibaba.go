package cluster

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
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
	"github.com/banzaicloud/pipeline/pkg/cluster/alibaba"
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

var _ CommonCluster = (*AlibabaCluster)(nil)

type AlibabaCluster struct {
	csClient       *cs.Client
	ecsClient      *ecs.Client
	alibabaCluster *alibabaDescribeClusterResponse
	modelCluster   *model.ClusterModel
	APIEndpoint    string
	CommonClusterBase
}

func (*AlibabaCluster) RbacEnabled() bool {
	return true
}

func (*AlibabaCluster) RequiresSshPublicKey() bool {
	return true
}

func (*AlibabaCluster) ListNodeNames() (pkgCommon.NodeNames, error) {
	return nil, nil
}

// GetAlibabaCSClient creates an Alibaba Container Service client with the credentials
func (c *AlibabaCluster) GetAlibabaCSClient(cfg *sdk.Config) (*cs.Client, error) {
	cred, err := c.createAlibabaCredentialsFromSecret()
	if err != nil {
		return nil, err
	}

	return verify.CreateAlibabaCSClient(cred, c.modelCluster.Alibaba.RegionID, cfg)
}

// GetAlibabaECSClient creates an Alibaba Elastic Compute Service client with the credentials
func (c *AlibabaCluster) GetAlibabaECSClient(cfg *sdk.Config) (*ecs.Client, error) {
	cred, err := c.createAlibabaCredentialsFromSecret()
	if err != nil {
		return nil, err
	}

	return verify.CreateAlibabaECSClient(cred, c.modelCluster.Alibaba.RegionID, cfg)
}

func createAlibabaNodePoolsModelFromRequestData(pools alibaba.NodePools) ([]*model.AlibabaNodePoolModel, error) {
	nodePoolsCount := len(pools)
	if nodePoolsCount == 0 {
		return nil, pkgErrors.ErrorNodePoolNotProvided
	}

	var res = make([]*model.AlibabaNodePoolModel, len(pools))
	var i int
	for _, pool := range pools {
		res[i] = &model.AlibabaNodePoolModel{
			WorkerInstanceType:       pool.WorkerInstanceType,
			WorkerSystemDiskCategory: pool.WorkerSystemDiskCategory,
			WorkerSystemDiskSize:     pool.WorkerSystemDiskSize,
			ImageID:                  pool.ImageID,
			NumOfNodes:               pool.NumOfNodes,
		}
		i++
	}

	return res, nil
}

//CreateAlibabaClusterFromModel creates ClusterModel struct from the Alibaba model
func CreateAlibabaClusterFromModel(clusterModel *model.ClusterModel) (*AlibabaCluster, error) {
	log.Debug("Create ClusterModel struct from the model")
	alibabaCluster := AlibabaCluster{
		modelCluster: clusterModel,
	}
	return &alibabaCluster, nil
}

func CreateAlibabaClusterFromRequest(request *pkgCluster.CreateClusterRequest, orgId uint) (*AlibabaCluster, error) {
	log.Debug("Create ClusterModel struct from the request")
	var cluster AlibabaCluster

	nodePools, err := createAlibabaNodePoolsModelFromRequestData(request.Properties.CreateClusterAlibaba.NodePools)
	if err != nil {
		return nil, err
	}

	cluster.modelCluster = &model.ClusterModel{
		Name:           request.Name,
		Location:       request.Location,
		Cloud:          request.Cloud,
		OrganizationId: orgId,
		SecretId:       request.SecretId,
		Alibaba: model.AlibabaClusterModel{
			RegionID:                 request.Properties.CreateClusterAlibaba.RegionID,
			ZoneID:                   request.Properties.CreateClusterAlibaba.ZoneID,
			MasterInstanceType:       request.Properties.CreateClusterAlibaba.MasterInstanceType,
			MasterSystemDiskCategory: request.Properties.CreateClusterAlibaba.MasterSystemDiskCategory,
			MasterSystemDiskSize:     request.Properties.CreateClusterAlibaba.MasterSystemDiskSize,
			LoginPassword:            "",
			SNATEntry:                true,
			SSHFlags:                 true,
			NodePools:                nodePools,
		},
	}
	return &cluster, nil
}

func (c *AlibabaCluster) CreateCluster() error {
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

	clusterSshSecret, err := c.GetSshSecretWithValidation()
	if err != nil {
		return err
	}

	sshKey := secret.NewSSHKeyPair(clusterSshSecret)
	_ = sshKey

	// setup cluster creation request
	params := alibabaClusterCreateParams{
		ClusterType:              "Kubernetes",
		Name:                     c.modelCluster.Name,
		RegionID:                 c.modelCluster.Alibaba.RegionID,                              // "eu-central-1"
		ZoneID:                   c.modelCluster.Alibaba.ZoneID,                                // "eu-central-1a"
		MasterInstanceType:       c.modelCluster.Alibaba.MasterInstanceType,                    // "ecs.sn1.large",
		MasterSystemDiskCategory: c.modelCluster.Alibaba.MasterSystemDiskCategory,              // "cloud_efficiency",
		MasterSystemDiskSize:     c.modelCluster.Alibaba.MasterSystemDiskSize,                  // 40,
		WorkerInstanceType:       c.modelCluster.Alibaba.NodePools[0].WorkerInstanceType,       // "ecs.sn1.large",
		WorkerSystemDiskCategory: c.modelCluster.Alibaba.NodePools[0].WorkerSystemDiskCategory, // "cloud_efficiency",
		WorkerSystemDiskSize:     c.modelCluster.Alibaba.NodePools[0].WorkerSystemDiskSize,     // 40,
		LoginPassword:            c.modelCluster.Alibaba.LoginPassword,                         // TODO: change me to KeyPair
		// KeyPair:                  sshKey.PublicKeyData, // this one should be a keypair name, so keypair should be uploaded
		ImageID:    c.modelCluster.Alibaba.NodePools[0].ImageID,    // "centos_7",
		NumOfNodes: c.modelCluster.Alibaba.NodePools[0].NumOfNodes, // 1,
		SNATEntry:  c.modelCluster.Alibaba.SNATEntry,               // true,
		SSHFlags:   c.modelCluster.Alibaba.SSHFlags,                // true,
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

	c.modelCluster.Alibaba.ClusterID = r.ClusterID
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
		err = errors.New("ssh jump host not found")
		return
	}
	if inf.IntranetURI == "" {
		err = errors.New("ssh jump host not found")
		return
	}
	if inf.InternetURI == "" {
		err = errors.New("ssh jump host not found")
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

func (c *AlibabaCluster) Persist(status, statusMessage string) error {
	log.Infof("Model before save: %v", c.modelCluster)
	return c.modelCluster.UpdateStatus(status, statusMessage)
}

func (c *AlibabaCluster) DownloadK8sConfig() ([]byte, error) {
	cfg := sdk.NewConfig()
	cfg.AutoRetry = false
	cfg.Debug = true
	cfg.Timeout = time.Minute

	ecsClient, err := c.GetAlibabaECSClient(cfg)
	if err != nil {
		return nil, err
	}

	downloadConfigRequest := createObtainClusterConfigRequest(c.modelCluster.Alibaba.ClusterID)
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
func (c *AlibabaCluster) GetCloud() string {
	return c.modelCluster.Cloud
}

// GetDistribution returns the distribution type of the cluster
func (c *AlibabaCluster) GetDistribution() string {
	return c.modelCluster.Distribution
}

func (c *AlibabaCluster) GetName() string {
	return c.modelCluster.Name
}

func (c *AlibabaCluster) GetType() string {
	return c.modelCluster.Cloud
}

func (c *AlibabaCluster) GetStatus() (*pkgCluster.GetClusterStatusResponse, error) {
	log.Info("Create cluster status response")

	nodePools := make(map[string]*pkgCluster.NodePoolStatus)
	for _, np := range c.modelCluster.GKE.NodePools {
		if np != nil {
			nodePools[np.Name] = &pkgCluster.NodePoolStatus{
				Count:        np.NodeCount,
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

func (c *AlibabaCluster) DeleteCluster() error {
	log.Info("Start deleting cluster (alibaba)")

	client, err := c.GetAlibabaCSClient(nil)
	if err != nil {
		return err
	}

	req := cs.CreateDeleteClusterRequest()
	req.ClusterId = c.modelCluster.Alibaba.ClusterID

	setEndpoint(req)
	resp, err := client.DeleteCluster(req)
	if err != nil {
		if sdkErr, ok := err.(*aliErrors.ServerError); ok {
			if strings.Contains(sdkErr.Message(), "ErrorClusterNotFound") {
				// Cluster has been already deleted
				return nil
			}
		}
		return err
	}

	if resp.GetHttpStatus() != http.StatusAccepted {
		return errors.New("Unexpected http status code: " + strconv.Itoa(resp.GetHttpStatus()))
	}

	return nil
}

func (c *AlibabaCluster) UpdateCluster(request *pkgCluster.UpdateClusterRequest, userId uint) error {
	log.Info("Start updating cluster (alibaba)")

	client, err := c.GetAlibabaCSClient(nil)
	if err != nil {
		return err
	}

	req := cs.CreateScaleClusterRequest()

	req.ClusterId = c.modelCluster.Alibaba.ClusterID
	params := alibabaScaleClusterParams{
		WorkerInstanceType:       c.modelCluster.Alibaba.NodePools[0].WorkerInstanceType,
		WorkerSystemDiskCategory: c.modelCluster.Alibaba.NodePools[0].WorkerSystemDiskCategory,
		WorkerSystemDiskSize:     c.modelCluster.Alibaba.NodePools[0].WorkerSystemDiskSize,
		LoginPassword:            c.modelCluster.Alibaba.LoginPassword,
		ImageID:                  c.modelCluster.Alibaba.NodePools[0].ImageID,
		NumOfNodes:               c.modelCluster.Alibaba.NodePools[0].NumOfNodes,
	}
	p, err := json.Marshal(&params)

	setEndpoint(req)
	setJSONContent(req, p)

	resp, err := client.ScaleCluster(req)

	var r alibabaClusterCreateResponse
	err = json.Unmarshal(resp.GetHttpContentBytes(), &r)

	cluster, err := waitForClusterState(client, r.ClusterID)
	if err != nil {
		return err
	}

	c.alibabaCluster = cluster

	return nil
}

func (c *AlibabaCluster) GetID() uint {
	return c.modelCluster.ID
}

func (c *AlibabaCluster) GetSecretId() string {
	return c.modelCluster.SecretId
}

func (c *AlibabaCluster) GetSshSecretId() string {
	return c.modelCluster.SshSecretId
}

func (c *AlibabaCluster) SaveSshSecretId(sshSecretId string) error {
	return c.modelCluster.UpdateSshSecret(sshSecretId)
}

func (c *AlibabaCluster) GetModel() *model.ClusterModel {
	return c.modelCluster
}

func (c *AlibabaCluster) CheckEqualityToUpdate(*pkgCluster.UpdateClusterRequest) error {
	panic("implement me")
}

func (c *AlibabaCluster) AddDefaultsToUpdate(*pkgCluster.UpdateClusterRequest) {
	panic("implement me")
}

func (c *AlibabaCluster) GetAPIEndpoint() (string, error) {
	if c.APIEndpoint != "" {
		return c.APIEndpoint, nil
	}

	client, err := c.GetAlibabaCSClient(nil)
	inf, err := getConnectionInfo(client, c.modelCluster.Alibaba.ClusterID)
	u, err := url.Parse(inf.InternetURI)
	if err != nil {
		return "", err
	}
	c.APIEndpoint = u.Host
	return c.APIEndpoint, nil
}

func (c *AlibabaCluster) DeleteFromDatabase() error {
	err := c.modelCluster.Delete()
	if err != nil {
		return err
	}
	c.modelCluster = nil
	return nil
}

func (c *AlibabaCluster) GetOrganizationId() uint {
	return c.modelCluster.OrganizationId
}

func (c *AlibabaCluster) UpdateStatus(status, statusMessage string) error {
	return c.modelCluster.UpdateStatus(status, statusMessage)
}

func (c *AlibabaCluster) GetClusterDetails() (*pkgCluster.DetailsResponse, error) {
	client, err := c.GetAlibabaCSClient(nil)
	if err != nil {
		return nil, err
	}

	r, err := getClusterDetails(client, c.modelCluster.Alibaba.ClusterID)
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

func (c *AlibabaCluster) validateRegion(regionID string) error {
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

func (c *AlibabaCluster) validateZone(regionID, zoneID string) error {
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

func (c *AlibabaCluster) validateInstanceType(regionID, zoneID, instanceType string) error {
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

func (c *AlibabaCluster) validateSystemDiskCategories(regionID, zoneID, diskCategory string) error {
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

func (c *AlibabaCluster) ValidateCreationFields(r *pkgCluster.CreateClusterRequest) error {
	var (
		region       = r.Properties.CreateClusterAlibaba.RegionID
		zone         = r.Properties.CreateClusterAlibaba.ZoneID
		instanceType = r.Properties.CreateClusterAlibaba.MasterInstanceType
		diskCategory = r.Properties.CreateClusterAlibaba.MasterSystemDiskCategory
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

	for _, np := range r.Properties.CreateClusterAlibaba.NodePools {
		var (
			instanceType = np.WorkerInstanceType
			diskCategory = np.WorkerSystemDiskCategory
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

func (c *AlibabaCluster) GetSecretWithValidation() (*secret.SecretItemResponse, error) {
	return c.CommonClusterBase.getSecret(c)
}

func (c *AlibabaCluster) GetSshSecretWithValidation() (*secret.SecretItemResponse, error) {
	return c.CommonClusterBase.getSshSecret(c)
}

func (c *AlibabaCluster) SaveConfigSecretId(configSecretId string) error {
	return c.modelCluster.UpdateConfigSecret(configSecretId)
}

func (c *AlibabaCluster) GetConfigSecretId() string {
	return c.modelCluster.ConfigSecretId
}

func (c *AlibabaCluster) GetK8sConfig() ([]byte, error) {
	return c.CommonClusterBase.getConfig(c)
}

func (c *AlibabaCluster) ReloadFromDatabase() error {
	return c.modelCluster.ReloadFromDatabase()
}

func (c *AlibabaCluster) createAlibabaCredentialsFromSecret() (*credentials.AccessKeyCredential, error) {
	clusterSecret, err := c.GetSecretWithValidation()
	if err != nil {
		return nil, err
	}
	return verify.CreateAlibabaCredentials(clusterSecret.Values), nil
}
