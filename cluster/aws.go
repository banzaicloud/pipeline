package cluster

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/banzaicloud/banzai-types/components"
	"github.com/banzaicloud/banzai-types/components/amazon"
	"github.com/banzaicloud/banzai-types/constants"
	"github.com/banzaicloud/pipeline/config"
	"github.com/banzaicloud/pipeline/model"
	"github.com/banzaicloud/pipeline/utils"
	kcluster "github.com/kris-nova/kubicorn/apis/cluster"
	"github.com/kris-nova/kubicorn/cutil"
	"github.com/kris-nova/kubicorn/cutil/initapi"
	"github.com/kris-nova/kubicorn/cutil/kubeadm"
	kubicornLogger "github.com/kris-nova/kubicorn/cutil/logger"
	"github.com/kris-nova/kubicorn/cutil/uuid"
	"github.com/kris-nova/kubicorn/state"
	"github.com/kris-nova/kubicorn/state/fs"
	"github.com/pkg/errors"
	"github.com/pkg/sftp"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"golang.org/x/crypto/ssh"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
)

// Simple init for logging
func init() {
	logger = config.Logger()
	log = logger.WithFields(logrus.Fields{"action": "Cluster"})
}

func SetCredentials(awscred *credentials.Credentials) func(*session.Options) error {
	return func(opts *session.Options) error {
		opts.Config.Credentials = awscred
		return nil
	}
}

var runtimeParam = cutil.RuntimeParameters{
	AwsProfile: "",
}

//AWSCluster struct for AWS cluster
type AWSCluster struct {
	kubicornCluster *kcluster.Cluster //Don't use this directly
	modelCluster    *model.ClusterModel
	k8sConfig       *[]byte
	APIEndpoint     string
}

func (c *AWSCluster) GetOrg() uint {
	return c.modelCluster.OrganizationId
}

//GetID returns the specified cluster id
func (c *AWSCluster) GetID() uint {
	return c.modelCluster.ID
}

//GetAPIEndpoint returns the Kubernetes Api endpoint
func (c *AWSCluster) GetAPIEndpoint() (string, error) {
	if c.APIEndpoint != "" {
		return c.APIEndpoint, nil
	}
	kubicornCluster, err := c.GetKubicornCluster()
	if err != nil {
		return "", err
	}
	return kubicornCluster.KubernetesAPI.Endpoint, nil
}

//GetName returns the name of the cluster
func (c *AWSCluster) GetName() string {
	return c.modelCluster.Name
}

//GetType returns the cloud type of the cluster
func (c *AWSCluster) GetType() string {
	return c.modelCluster.Cloud
}

//GetModel returns the whole clusterModel
func (c *AWSCluster) GetModel() *model.ClusterModel {
	return c.modelCluster
}

//CreateAWSClusterFromModel creates ClusterModel struct from the kubicorn model
func CreateAWSClusterFromModel(clusterModel *model.ClusterModel) (*AWSCluster, error) {
	log := logger.WithFields(logrus.Fields{"action": constants.TagGetCluster})
	log.Debug("Create ClusterModel struct from the request")
	awsCluster := AWSCluster{
		modelCluster: clusterModel,
	}
	//
	_, err := awsCluster.GetKubicornCluster()
	if err != nil {
		return nil, err
	}
	return &awsCluster, nil
}

//CreateAWSClusterFromRequest creates ClusterModel struct from the request
func CreateAWSClusterFromRequest(request *components.CreateClusterRequest, orgId uint) (*AWSCluster, error) {
	log := logger.WithFields(logrus.Fields{"action": constants.TagCreateCluster})
	log.Debug("Create ClusterModel struct from the request")
	var cluster AWSCluster

	cluster.modelCluster = &model.ClusterModel{
		Name:             request.Name,
		Location:         request.Location,
		NodeInstanceType: request.NodeInstanceType,
		Cloud:            request.Cloud,
		SecretId:         request.SecretId,
		OrganizationId:   orgId,
		Amazon: model.AmazonClusterModel{
			NodeSpotPrice:      request.Properties.CreateClusterAmazon.Node.SpotPrice,
			NodeMinCount:       request.Properties.CreateClusterAmazon.Node.MinCount,
			NodeMaxCount:       request.Properties.CreateClusterAmazon.Node.MaxCount,
			NodeImage:          request.Properties.CreateClusterAmazon.Node.Image,
			MasterInstanceType: request.Properties.CreateClusterAmazon.Master.InstanceType,
			MasterImage:        request.Properties.CreateClusterAmazon.Master.Image,
		},
	}
	return &cluster, nil
}

//Persist save the cluster model
func (c *AWSCluster) Persist() error {
	return c.modelCluster.Save()
}

//CreateCluster creates a new cluster
func (c *AWSCluster) CreateCluster() error {
	log := logger.WithFields(logrus.Fields{"action": constants.TagCreateCluster})

	//uid := c.GetModel().OrganizationId
	//awsCred := credentials.NewStaticCredentials("", "", "")
	//runtimeParam.AwsOptions = append(runtimeParam.AwsOptions, SetCredentials(awsCred))

	kubicornLogger.Level = getKubicornLogLevel()

	//TODO check if this should be private
	c.kubicornCluster = GetKubicornProfile(c.modelCluster)
	sshKeyPath := viper.GetString("cloud.keypath")
	//TODO move to the profile section
	if sshKeyPath != "" {
		log.Debug("Overwriting default SSH key path to:", sshKeyPath)
		c.kubicornCluster.SSH.PublicKeyPath = sshKeyPath
	}

	log.Info("Init cluster")
	newCluster, err := initapi.InitCluster(c.kubicornCluster)

	if err != nil {
		return err
	}

	log.Info("Get reconciler")
	reconciler, err := cutil.GetReconciler(newCluster, &runtimeParam)

	if err != nil {
		return err
	}

	log.Info("Get expected")
	expected, err := reconciler.Expected(newCluster)
	if err != nil {
		return err
	}
	log.Info("Get expected state succeeded")

	// ---- [ Get actual state ] ---- //
	actual, err := reconciler.Actual(newCluster)
	if err != nil {
		return err
	}

	// ---- [ Reconcile ] ---- //
	created, err := reconciler.Reconcile(actual, expected)
	if err != nil {
		return err
	}

	if created == nil {
		return constants.ErrorReconcile
	}

	log.Debug("Created cluster:", created.Name)

	log.Info("Get state store")
	stateStore := getStateStoreForCluster(c.modelCluster)
	if stateStore.Exists() {
		return fmt.Errorf("state store [%s] exists, will not overwrite", c.kubicornCluster.Name)
	}
	stateStore.Commit(created)

	return nil
}

//We return stateStore so update can use it.
func getStateStoreForCluster(clusterType *model.ClusterModel) (stateStore state.ClusterStorer) {
	stateStore = fs.NewFileSystemStore(&fs.FileSystemStoreOptions{
		BasePath:    "statestore",
		ClusterName: clusterType.Name,
	})
	return stateStore
}

// GetKubicornProfile creates *cluster.Cluster from ClusterModel struct
func GetKubicornProfile(cs *model.ClusterModel) *kcluster.Cluster {
	uuidSuffix := uuid.TimeOrderedUUID()
	return &kcluster.Cluster{
		Name:     cs.Name,
		Cloud:    kcluster.CloudAmazon,
		Location: cs.Location,
		SSH: &kcluster.SSH{
			Name:          cs.Name + "-" + uuidSuffix,
			PublicKeyPath: "~/.ssh/id_rsa.pub",
			User:          "ubuntu",
		},
		KubernetesAPI: &kcluster.KubernetesAPI{
			Port: "443",
		},
		Network: &kcluster.Network{
			Type:       kcluster.NetworkTypePublic,
			CIDR:       "10.0.0.0/16",
			InternetGW: &kcluster.InternetGW{},
		},
		Values: &kcluster.Values{
			ItemMap: map[string]string{
				"INJECTEDTOKEN": kubeadm.GetRandomToken(),
			},
		},
		ServerPools: []*kcluster.ServerPool{
			{
				Type:     kcluster.ServerPoolTypeMaster,
				Name:     fmt.Sprintf("%s.master", cs.Name),
				MinCount: 1,
				MaxCount: 1,
				Image:    cs.Amazon.MasterImage, //"ami-835b4efa"
				Size:     cs.NodeInstanceType,
				BootstrapScripts: []string{
					getBootstrapScriptFromEnv(true),
				},
				InstanceProfile: &kcluster.IAMInstanceProfile{
					Name: fmt.Sprintf("%s-KubicornMasterInstanceProfile", cs.Name),
					Role: &kcluster.IAMRole{
						Name: fmt.Sprintf("%s-KubicornMasterRole", cs.Name),
						Policies: []*kcluster.IAMPolicy{
							{
								Name: "MasterPolicy",
								Document: `{
                  "Version": "2012-10-17",
                  "Statement": [
                     {
                        "Effect": "Allow",
                        "Action": [
                           "ec2:*",
                           "elasticloadbalancing:*",
                           "ecr:GetAuthorizationToken",
                           "ecr:BatchCheckLayerAvailability",
                           "ecr:GetDownloadUrlForLayer",
                           "ecr:GetRepositoryPolicy",
                           "ecr:DescribeRepositories",
                           "ecr:ListImages",
                           "ecr:BatchGetImage",
                           "autoscaling:DescribeAutoScalingGroups",
                           "autoscaling:UpdateAutoScalingGroup",
													 "s3:ListBucket",
													 "s3:GetObject",
													 "s3:PutObject",
													 "s3:ListObjects",
													 "s3:DeleteObject"
                        ],
                        "Resource": "*"
                     }
                  ]
								}`,
							},
						},
					},
				},
				Subnets: []*kcluster.Subnet{
					{
						Name:     fmt.Sprintf("%s.master", cs.Name),
						CIDR:     "10.0.0.0/24",
						Location: cs.Location,
					},
				},

				Firewalls: []*kcluster.Firewall{
					{
						Name: fmt.Sprintf("%s.master-external-%s", cs.Name, uuidSuffix),
						IngressRules: []*kcluster.IngressRule{
							{
								IngressFromPort: "22",
								IngressToPort:   "22",
								IngressSource:   "0.0.0.0/0",
								IngressProtocol: "tcp",
							},
							{
								IngressFromPort: "443",
								IngressToPort:   "443",
								IngressSource:   "0.0.0.0/0",
								IngressProtocol: "tcp",
							},
							{
								IngressFromPort: "0",
								IngressToPort:   "65535",
								IngressSource:   "10.0.100.0/24",
								IngressProtocol: "-1",
							},
						},
					},
				},
			},
			{
				Type:     kcluster.ServerPoolTypeNode,
				Name:     fmt.Sprintf("%s.node", cs.Name),
				MinCount: cs.Amazon.NodeMinCount,
				MaxCount: cs.Amazon.NodeMaxCount,
				Image:    cs.Amazon.NodeImage, //"ami-835b4efa"
				Size:     cs.NodeInstanceType,
				AwsConfiguration: &kcluster.AwsConfiguration{
					SpotPrice: cs.Amazon.NodeSpotPrice,
				},
				BootstrapScripts: []string{
					getBootstrapScriptFromEnv(false),
				},
				InstanceProfile: &kcluster.IAMInstanceProfile{
					Name: fmt.Sprintf("%s-KubicornNodeInstanceProfile", cs.Name),
					Role: &kcluster.IAMRole{
						Name: fmt.Sprintf("%s-KubicornNodeRole", cs.Name),
						Policies: []*kcluster.IAMPolicy{
							{
								Name: "NodePolicy",
								Document: `{
                  "Version": "2012-10-17",
                  "Statement": [
                     {
                        "Effect": "Allow",
                        "Action": [
            							"ec2:Describe*",
            							"ecr:GetAuthorizationToken",
            							"ecr:BatchCheckLayerAvailability",
            							"ecr:GetDownloadUrlForLayer",
            							"ecr:GetRepositoryPolicy",
            							"ecr:DescribeRepositories",
            							"ecr:ListImages",
            							"ecr:BatchGetImage",
													"s3:ListBucket",
													"s3:GetObject",
													"s3:PutObject",
													"s3:ListObjects",
													"s3:DeleteObject"
                        ],
                        "Resource": "*"
                     }
                  ]
								}`,
							},
						},
					},
				},
				Subnets: []*kcluster.Subnet{
					{
						Name:     fmt.Sprintf("%s.node", cs.Name),
						CIDR:     "10.0.100.0/24",
						Location: cs.Location,
					},
				},
				Firewalls: []*kcluster.Firewall{
					{
						Name: fmt.Sprintf("%s.node-external-%s", cs.Name, uuidSuffix),
						IngressRules: []*kcluster.IngressRule{
							{
								IngressFromPort: "22",
								IngressToPort:   "22",
								IngressSource:   "0.0.0.0/0",
								IngressProtocol: "tcp",
							},
							{
								IngressFromPort: "0",
								IngressToPort:   "65535",
								IngressSource:   "10.0.0.0/24",
								IngressProtocol: "-1",
							},
						},
					},
				},
			},
		},
	}
}

//GetStatus gets cluster status
func (c *AWSCluster) GetStatus() (*components.GetClusterStatusResponse, error) {
	log.Info("Start get cluster status (amazon)")

	c.GetK8sConfig()
	c.GetAPIEndpoint()
	kubicornCluster, err := c.GetKubicornCluster()
	if err != nil {
		return nil, err
	}

	response := &components.GetClusterStatusResponse{
		Status:           http.StatusOK,
		Name:             kubicornCluster.Name,
		Location:         kubicornCluster.Location,
		Cloud:            kubicornCluster.Cloud,
		NodeInstanceType: c.modelCluster.NodeInstanceType,
		ResourceID:       c.modelCluster.ID,
	}
	return response, nil
}

// UpdateCluster updates Amazon cluster in cloud
func (c *AWSCluster) UpdateCluster(request *components.UpdateClusterRequest) error {

	log := logger.WithFields(logrus.Fields{"action": constants.TagUpdateCluster})
	kubicornLogger.Level = getKubicornLogLevel()

	log.Info("Start updating cluster (amazon)")

	if request == nil {
		return constants.ErrorEmptyUpdateRequest
	}

	log.Info("Create updated model")
	updateCluster := &model.ClusterModel{
		Model:            c.modelCluster.Model,
		Name:             c.modelCluster.Name,
		Location:         c.modelCluster.Location,
		NodeInstanceType: c.modelCluster.NodeInstanceType,
		Cloud:            request.Cloud,
		Amazon: model.AmazonClusterModel{
			NodeSpotPrice:      c.modelCluster.Amazon.NodeSpotPrice,
			NodeMinCount:       request.UpdateClusterAmazon.MinCount,
			NodeMaxCount:       request.UpdateClusterAmazon.MaxCount,
			NodeImage:          c.modelCluster.Amazon.NodeImage,
			MasterInstanceType: c.modelCluster.Amazon.MasterInstanceType,
			MasterImage:        c.modelCluster.Amazon.MasterImage,
		},
	}

	log.Debug("Resizing cluster: ", c.GetName())
	kubicornCluster, err := c.GetKubicornCluster()
	if err != nil {
		return err
	}
	log.Debug("Resizing cluster: ", c.GetName())
	kubicornCluster.ServerPools[0].MinCount = 1
	kubicornCluster.ServerPools[0].MaxCount = 1
	log.Debugf("Worker pool min size from %d to %d", kubicornCluster.ServerPools[1].MinCount, updateCluster.Amazon.NodeMinCount)
	kubicornCluster.ServerPools[1].MinCount = updateCluster.Amazon.NodeMinCount
	log.Debugf("Worker pool max size from %d to %d", kubicornCluster.ServerPools[1].MaxCount, updateCluster.Amazon.NodeMaxCount)
	kubicornCluster.ServerPools[1].MaxCount = updateCluster.Amazon.NodeMaxCount

	log.Debug("Get reconciler")
	reconciler, err := cutil.GetReconciler(kubicornCluster, &runtimeParam)
	if err != nil {
		err = errors.Wrap(err, "error getting reconciler")
		return err
	}

	log.Debug("Get expected cluster")
	expected, err := reconciler.Expected(kubicornCluster)
	if err != nil {
		err = errors.Wrap(err, "error getting expected")
		return err
	}

	log.Debug("Reconcile")
	updated, err := reconciler.Reconcile(kubicornCluster, expected)
	if err != nil {
		err = errors.Wrap(err, "error during reconcile")
		return err
	}

	//Update AWS model
	c.modelCluster = updateCluster
	c.kubicornCluster = kubicornCluster //This is redundant TODO check if it's ok

	// TODO check statestore usage
	statestore := getStateStoreForCluster(updateCluster)
	log.Info("Save cluster to the statestore")
	statestore.Commit(updated)

	return nil
}

//GetKubicornCluster returns a Kubicorn cluster
func (c *AWSCluster) GetKubicornCluster() (*kcluster.Cluster, error) {
	if c.kubicornCluster != nil {
		return c.kubicornCluster, nil
	}
	kubicornCluster, err := ReadCluster(c.modelCluster)
	if err != nil {
		return nil, err
	}
	c.kubicornCluster = kubicornCluster
	return c.kubicornCluster, nil
}

// ReadCluster reads a persisted cluster from the statestore
func ReadCluster(modelCluster *model.ClusterModel) (*kcluster.Cluster, error) {
	stateStore := getStateStoreForCluster(modelCluster)
	readCluster, err := stateStore.GetCluster()
	if err != nil {
		return nil, err
	}
	return readCluster, nil
}

// DeleteCluster deletes cluster from amazon
func (c *AWSCluster) DeleteCluster() error {

	log := logger.WithFields(logrus.Fields{"action": constants.TagDeleteCluster})
	kubicornLogger.Level = getKubicornLogLevel()

	log.Info("Start delete amazon cluster")
	kubicornCluster, err := c.GetKubicornCluster()
	if err != nil {
		return err
	}
	statestore := getStateStoreForCluster(c.modelCluster)
	log.Debug("Get reconciler")
	reconciler, err := cutil.GetReconciler(kubicornCluster, &runtimeParam)
	if err != nil {
		err = errors.Wrap(err, "error getting reconciler")
		return err
	}
	log.Info("Delete cluster with kubicorn")
	_, err = reconciler.Destroy()
	if err != nil {
		err = errors.Wrap(err, "error destroying cluster")
		return err
	}

	log.Info("Destroy cluster from statestore")
	err = statestore.Destroy()
	if err != nil {
		err = errors.Wrap(err, "error destroying stetestore")
		return err
	}
	c.kubicornCluster = nil
	return nil
}

//GetK8sConfig returns the Kubernetes config
func (c *AWSCluster) GetK8sConfig() (*[]byte, error) {
	if c.k8sConfig != nil {
		return c.k8sConfig, nil
	}
	kubicornCluster, err := c.GetKubicornCluster()
	if err != nil {
		err = errors.Wrap(err, "error getting kubicorn cluster")
		return nil, err
	}
	kubeConfig, err := DownloadK8sConfig(kubicornCluster)
	if err != nil {
		err = errors.Wrap(err, "error downloading kubernetes config")
		return nil, err
	}
	c.k8sConfig = kubeConfig
	return c.k8sConfig, nil
}

//DownloadK8sConfig downloads the Kubernetes config from the cluster
// Todo check first if config is locally available
func DownloadK8sConfig(kubicornCluster *kcluster.Cluster) (*[]byte, error) {

	user := kubicornCluster.SSH.User
	pubKeyPath := expand(kubicornCluster.SSH.PublicKeyPath)
	privKeyPath := strings.Replace(pubKeyPath, ".pub", "", 1)
	address := fmt.Sprintf("%s:%s", kubicornCluster.KubernetesAPI.Endpoint, "22")

	sshConfig := &ssh.ClientConfig{
		User:            user,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	remotePath := ""
	if user == "root" {
		remotePath = "/root/.kube/config"
	} else {
		remotePath = fmt.Sprintf("/home/%s/.kube/config", user)
	}

	pemBytes, err := ioutil.ReadFile(privKeyPath)
	if err != nil {

		return nil, err
	}

	signer, err := getSigner(pemBytes)
	if err != nil {
		return nil, err
	}

	auths := []ssh.AuthMethod{
		ssh.PublicKeys(signer),
	}
	sshConfig.Auth = auths

	sshConfig.SetDefaults()

	connection, err := ssh.Dial("tcp", address, sshConfig)
	if err != nil {
		return nil, err
	}
	defer connection.Close()
	sftpClient, err := sftp.NewClient(connection)
	if err != nil {
		return nil, err
	}
	defer sftpClient.Close()
	sftpConnection, err := sftpClient.Open(remotePath)
	if err != nil {
		return nil, err
	}
	defer sftpConnection.Close()
	config, err := ioutil.ReadAll(sftpConnection)
	if err != nil {
		return nil, err
	}
	return &config, nil
}

// const's of BootstrapScript values
const (
	BootstrapScriptMasterKey     = "BOOTSTRAP_SCRIPT_MASTER"
	BootstrapScriptNodeKey       = "BOOTSTRAP_SCRIPT_NODE"
	BootstrapScriptMasterDefault = "https://raw.githubusercontent.com/banzaicloud/banzai-charts/master/stable/pipeline/bootstrap/amazon_k8s_ubuntu_16.04_master_pipeline.sh"
	BootstrapScriptNodeDefault   = "https://raw.githubusercontent.com/banzaicloud/banzai-charts/master/stable/pipeline/bootstrap/amazon_k8s_ubuntu_16.04_node_pipeline.sh"
)

func getBootstrapScriptFromEnv(isMaster bool) string {

	var s string
	if isMaster {
		s = os.Getenv(BootstrapScriptMasterKey)
	} else {
		s = os.Getenv(BootstrapScriptNodeKey)
	}

	if len(s) == 0 {
		if isMaster {
			return BootstrapScriptMasterDefault
		}
		return BootstrapScriptNodeDefault
	}
	return s

}

//AddDefaultsToUpdate adds defaults to update request
func (c *AWSCluster) AddDefaultsToUpdate(r *components.UpdateClusterRequest) {

	// ---- [ Node check ] ---- //
	if r.UpdateAmazonNode == nil {
		log.Info("'node' field is empty. Fill from stored data")
		r.UpdateAmazonNode = &amazon.UpdateAmazonNode{
			MinCount: c.modelCluster.Amazon.NodeMinCount,
			MaxCount: c.modelCluster.Amazon.NodeMaxCount,
		}
	}

	// ---- [ Node min count check ] ---- //
	if r.UpdateAmazonNode.MinCount == 0 {
		defMinCount := c.modelCluster.Amazon.NodeMinCount
		log.Info(constants.TagValidateUpdateCluster, "Node minCount set to default value: ", defMinCount)
		r.UpdateAmazonNode.MinCount = defMinCount
	}

	// ---- [ Node max count check ] ---- //
	if r.UpdateAmazonNode.MaxCount == 0 {
		defMaxCount := c.modelCluster.Amazon.NodeMaxCount
		log.Info(constants.TagValidateUpdateCluster, "Node maxCount set to default value: ", defMaxCount)
		r.UpdateAmazonNode.MaxCount = defMaxCount
	}

}

//CheckEqualityToUpdate validates the update request
func (c *AWSCluster) CheckEqualityToUpdate(r *components.UpdateClusterRequest) error {
	// create update request struct with the stored data to check equality
	preCl := &amazon.UpdateClusterAmazon{
		UpdateAmazonNode: &amazon.UpdateAmazonNode{
			MinCount: c.modelCluster.Amazon.NodeMinCount,
			MaxCount: c.modelCluster.Amazon.NodeMaxCount,
		},
	}

	log.Info("Check stored & updated cluster equals")

	// check equality
	return utils.IsDifferent(r.UpdateClusterAmazon, preCl)
}

//DeleteFromDatabase deletes model from the database
func (c *AWSCluster) DeleteFromDatabase() error {
	err := c.modelCluster.Delete()
	if err != nil {
		return err
	}
	c.modelCluster = nil
	return nil
}

func getKubicornLogLevel() int {
	lvl := viper.GetString("logging.kubicornloglevel")
	switch lvl {
	case "critical":
		return 1
	case "warn":
		return 2
	case "info":
		return 3
	case "debug":
		return 4
	default:
		return 4
	}
}
