package cluster

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/banzaicloud/banzai-types/components"
	"github.com/banzaicloud/banzai-types/components/amazon"
	"github.com/banzaicloud/banzai-types/constants"
	"github.com/banzaicloud/pipeline/config"
	"github.com/banzaicloud/pipeline/model"
	"github.com/banzaicloud/pipeline/secret"
	kcluster "github.com/kubicorn/kubicorn/apis/cluster"
	"github.com/kubicorn/kubicorn/pkg"
	"github.com/kubicorn/kubicorn/pkg/initapi"
	"github.com/kubicorn/kubicorn/pkg/kubeadm"
	kubicornLogger "github.com/kubicorn/kubicorn/pkg/logger"
	"github.com/kubicorn/kubicorn/pkg/uuid"
	"github.com/kubicorn/kubicorn/state"
	"github.com/kubicorn/kubicorn/state/fs"
	"github.com/pkg/errors"
	"github.com/pkg/sftp"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"golang.org/x/crypto/ssh"
	"io/ioutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

//AWSCluster struct for AWS cluster
type AWSCluster struct {
	kubicornCluster *kcluster.Cluster //Don't use this directly
	modelCluster    *model.ClusterModel
	k8sConfig       []byte
	APIEndpoint     string
}

func (c *AWSCluster) GetOrg() uint {
	return c.modelCluster.OrganizationId
}

func (c *AWSCluster) GetSecretID() string {
	return c.modelCluster.SecretId
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
	if awsCluster.modelCluster.Status == constants.Running {
		_, err := awsCluster.GetKubicornCluster()
		if err != nil {
			return nil, err
		}
	}
	return &awsCluster, nil
}

//CreateAWSClusterFromRequest creates ClusterModel struct from the request
func CreateAWSClusterFromRequest(request *components.CreateClusterRequest, orgId uint) (*AWSCluster, error) {
	log := logger.WithFields(logrus.Fields{"action": constants.TagCreateCluster})
	log.Debug("Create ClusterModel struct from the request")
	var cluster AWSCluster

	modelNodePools := createNodePoolsFromRequest(request.Properties.CreateClusterAmazon.NodePools)

	cluster.modelCluster = &model.ClusterModel{
		Name:           request.Name,
		Location:       request.Location,
		Cloud:          request.Cloud,
		SecretId:       request.SecretId,
		OrganizationId: orgId,
		Amazon: model.AmazonClusterModel{
			MasterInstanceType: request.Properties.CreateClusterAmazon.Master.InstanceType,
			MasterImage:        request.Properties.CreateClusterAmazon.Master.Image,
			NodePools:          modelNodePools,
		},
	}
	return &cluster, nil
}

func createNodePoolsFromRequest(nodePools map[string]*amazon.AmazonNodePool) []*model.AmazonNodePoolsModel {
	var modelNodePools = make([]*model.AmazonNodePoolsModel, len(nodePools))
	i := 0
	for nodePoolName, nodePool := range nodePools {
		modelNodePools[i] = &model.AmazonNodePoolsModel{
			Name:             nodePoolName,
			NodeInstanceType: nodePool.InstanceType,
			NodeSpotPrice:    nodePool.SpotPrice,
			NodeImage:        nodePool.Image,
			NodeMinCount:     nodePool.MinCount,
			NodeMaxCount:     nodePool.MaxCount,
		}
		i++
	}
	return modelNodePools
}

//Persist save the cluster model
func (c *AWSCluster) Persist(status string) error {
	return c.modelCluster.UpdateStatus(status)
}

//CreateCluster creates a new cluster
func (c *AWSCluster) CreateCluster() error {
	log := logger.WithFields(logrus.Fields{"action": constants.TagCreateCluster})

	// Set up credentials TODO simplify
	runtimeParam := pkg.RuntimeParameters{
		AwsProfile: "",
	}
	clusterSecret, err := GetSecret(c)
	if err != nil {
		return err
	}
	if clusterSecret.SecretType != constants.Amazon {
		return errors.Errorf("missmatch secret type %s versus %s", clusterSecret.SecretType, constants.Amazon)
	}
	awsCred := credentials.NewStaticCredentials(
		clusterSecret.Values[secret.AwsAccessKeyId],
		clusterSecret.Values[secret.AwsSecretAccessKey],
		"",
	)
	runtimeParam.AwsOptions = append(runtimeParam.AwsOptions, SetCredentials(awsCred))

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
	reconciler, err := pkg.GetReconciler(newCluster, &runtimeParam)

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

func getMasterServerPool(cs *model.ClusterModel, nodeServerPool []*kcluster.ServerPool, uuidSuffix string) *kcluster.ServerPool {
	var ingressRules = make([]*kcluster.IngressRule, 0, 2+len(nodeServerPool))
	ingressRules = append(ingressRules, &kcluster.IngressRule{
		IngressFromPort: "22",
		IngressToPort:   "22",
		IngressSource:   "0.0.0.0/0",
		IngressProtocol: "tcp",
	})
	ingressRules = append(ingressRules, &kcluster.IngressRule{
		IngressFromPort: "443",
		IngressToPort:   "443",
		IngressSource:   "0.0.0.0/0",
		IngressProtocol: "tcp",
	})

	for _, node := range nodeServerPool {
		ingressRules = append(ingressRules, &kcluster.IngressRule{
			IngressFromPort: "0",
			IngressToPort:   "65535",
			IngressSource:   node.Subnets[0].CIDR,
			IngressProtocol: "-1",
		})
	}

	return &kcluster.ServerPool{
		Type:     kcluster.ServerPoolTypeMaster,
		Name:     fmt.Sprintf("%s.master", cs.Name),
		MinCount: 1,
		MaxCount: 1,
		Image:    cs.Amazon.MasterImage, //"ami-835b4efa"
		Size:     cs.Amazon.MasterInstanceType,
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
													 "autoscaling:DescribeAutoScalingInstances",
													 "autoscaling:DescribeTags",
													 "autoscaling:DescribeLaunchConfigurations",
													 "autoscaling:SetDesiredCapacity",
													 "autoscaling:TerminateInstanceInAutoScalingGroup",
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
				Name:         fmt.Sprintf("%s.master-external-%s", cs.Name, uuidSuffix),
				IngressRules: ingressRules,
			},
		},
	}
}

func getAsgNodePoolName(asgName string) string {
	if strings.HasSuffix(asgName, "master") {
		return "master"
	}
	asgNameSplit := strings.Split(asgName, ".node.")
	if len(asgNameSplit) > 1 {
		return asgNameSplit[1]
	}
	return asgName
}

func getNodeServerPool(clusterName string, location string, nodePool *model.AmazonNodePoolsModel,
	cidr string, uuidSuffix string) *kcluster.ServerPool {

	return &kcluster.ServerPool{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				"k8s.io/cluster-autoscaler/enabled":    "true",
				"kubernetes.io/cluster/" + clusterName: "true",
			},
		},
		Type:     kcluster.ServerPoolTypeNode,
		Name:     fmt.Sprintf("%s.node.%s", clusterName, nodePool.Name),
		MinCount: nodePool.NodeMinCount,
		MaxCount: nodePool.NodeMaxCount,
		Image:    nodePool.NodeImage, //"ami-835b4efa"
		Size:     nodePool.NodeInstanceType,
		AwsConfiguration: &kcluster.AwsConfiguration{
			SpotPrice: nodePool.NodeSpotPrice,
		},
		BootstrapScripts: []string{
			getBootstrapScriptFromEnv(false),
		},
		InstanceProfile: &kcluster.IAMInstanceProfile{
			Name: fmt.Sprintf("%s-KubicornNodeInstanceProfile-%s", clusterName, nodePool.Name),
			Role: &kcluster.IAMRole{
				Name: fmt.Sprintf("%s-KubicornNodeRole-%s", clusterName, nodePool.Name),
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
													"s3:DeleteObject",
													"autoscaling:DescribeAutoScalingGroups",
													"autoscaling:UpdateAutoScalingGroup",
													"autoscaling:DescribeAutoScalingInstances",
													"autoscaling:DescribeTags",
													"autoscaling:DescribeLaunchConfigurations",
													"autoscaling:SetDesiredCapacity",
													"autoscaling:TerminateInstanceInAutoScalingGroup"
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
				Name:     fmt.Sprintf("%s.node.%s", clusterName, nodePool.Name),
				CIDR:     cidr,
				Location: location,
			},
		},
		Firewalls: []*kcluster.Firewall{
			{
				Name: fmt.Sprintf("%s.node.%s-external-%s", clusterName, nodePool.Name, uuidSuffix),
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
	}
}

// GetKubicornProfile creates *cluster.Cluster from ClusterModel struct
func GetKubicornProfile(cs *model.ClusterModel) *kcluster.Cluster {

	uuidSuffix := uuid.TimeOrderedUUID()
	var nodeServerPool = make([]*kcluster.ServerPool, len(cs.Amazon.NodePools))
	for i, nodePool := range cs.Amazon.NodePools {
		nodeServerPool[i] = getNodeServerPool(cs.Name, cs.Location, nodePool, fmt.Sprintf("10.0.%d.0/24", 100+i), uuidSuffix)
	}
	var masterServerPool = []*kcluster.ServerPool{
		getMasterServerPool(cs, nodeServerPool, uuidSuffix),
	}
	nodeServerPool = append(masterServerPool, nodeServerPool...)

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
		ServerPools: nodeServerPool,
	}
}

//GetStatus gets cluster status
func (c *AWSCluster) GetStatus() (*components.GetClusterStatusResponse, error) {
	log.Info("Start get cluster status (amazon)")

	nodePools := make(map[string]*components.StatusNodePool)
	for _, np := range c.modelCluster.Amazon.NodePools {
		nodePools[np.Name] = &components.StatusNodePool{
			InstanceType: np.NodeInstanceType,
			SpotPrice:    np.NodeSpotPrice,
			MinCount:     np.NodeMinCount,
			MaxCount:     np.NodeMaxCount,
			Image:        np.NodeImage,
		}
	}

	return &components.GetClusterStatusResponse{
		Status:     c.modelCluster.Status,
		Name:       c.modelCluster.Name,
		Location:   c.modelCluster.Location,
		Cloud:      c.modelCluster.Cloud,
		ResourceID: c.modelCluster.ID,
		NodePools:  nodePools,
	}, nil
}

// UpdateCluster updates Amazon cluster in cloud
func (c *AWSCluster) UpdateCluster(request *components.UpdateClusterRequest) error {

	log := logger.WithFields(logrus.Fields{"action": constants.TagUpdateCluster})
	kubicornLogger.Level = getKubicornLogLevel()

	log.Info("Start updating cluster (amazon)")

	if request == nil {
		return constants.ErrorEmptyUpdateRequest
	}

	existingNodePools := map[string]*model.AmazonNodePoolsModel{}
	for _, nodePool := range c.modelCluster.Amazon.NodePools {
		existingNodePools[nodePool.Name] = nodePool
	}

	updatedNodePools := make([]*model.AmazonNodePoolsModel, len(c.modelCluster.Amazon.NodePools))

	existingAsgs := map[string]*kcluster.ServerPool{}
	for _, asg := range c.kubicornCluster.ServerPools {
		poolName := getAsgNodePoolName(asg.Name)
		existingAsgs[poolName] = asg
	}

	//updatedAsgs := make([]*kcluster.ServerPool, len(c.modelCluster.Amazon.NodePools))

	log.Info("Create updated model")
	updateCluster := &model.ClusterModel{
		ID:               c.modelCluster.ID,
		CreatedAt:        c.modelCluster.CreatedAt,
		UpdatedAt:        c.modelCluster.UpdatedAt,
		DeletedAt:        c.modelCluster.DeletedAt,
		Name:             c.modelCluster.Name,
		Location:         c.modelCluster.Location,
		Cloud:            request.Cloud,
		OrganizationId:   c.modelCluster.OrganizationId,
		SecretId:         c.modelCluster.SecretId,
		Status:           c.modelCluster.Status,
		Amazon: model.AmazonClusterModel{
			MasterInstanceType: c.modelCluster.Amazon.MasterInstanceType,
			MasterImage:        c.modelCluster.Amazon.MasterImage,
			NodePools:          updatedNodePools,
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
	//log.Debugf("Worker pool min size from %d to %d", kubicornCluster.ServerPools[1].MinCount, updateCluster.Amazon.NodeMinCount)
	//kubicornCluster.ServerPools[1].MinCount = updateCluster.Amazon.NodeMinCount
	//log.Debugf("Worker pool max size from %d to %d", kubicornCluster.ServerPools[1].MaxCount, updateCluster.Amazon.NodeMaxCount)
	//kubicornCluster.ServerPools[1].MaxCount = updateCluster.Amazon.NodeMaxCount

	log.Debug("Get reconciler")

	// Set up credentials TODO simplify
	runtimeParam := pkg.RuntimeParameters{
		AwsProfile: "",
	}
	clusterSecret, err := GetSecret(c)
	if err != nil {
		return err
	}
	if clusterSecret.SecretType != constants.Amazon {
		return errors.Errorf("missmatch secret type %s versus %s", clusterSecret.SecretType, constants.Amazon)
	}
	awsCred := credentials.NewStaticCredentials(
		clusterSecret.Values[secret.AwsAccessKeyId],
		clusterSecret.Values[secret.AwsSecretAccessKey],
		"",
	)
	runtimeParam.AwsOptions = append(runtimeParam.AwsOptions, SetCredentials(awsCred))

	reconciler, err := pkg.GetReconciler(kubicornCluster, &runtimeParam)
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

	// Set up credentials TODO simplify
	runtimeParam := pkg.RuntimeParameters{
		AwsProfile: "",
	}
	clusterSecret, err := GetSecret(c)
	if err != nil {
		return err
	}
	if clusterSecret.SecretType != constants.Amazon {
		return errors.Errorf("missmatch secret type %s versus %s", clusterSecret.SecretType, constants.Amazon)
	}
	awsCred := credentials.NewStaticCredentials(
		clusterSecret.Values[secret.AwsAccessKeyId],
		clusterSecret.Values[secret.AwsSecretAccessKey],
		"",
	)
	runtimeParam.AwsOptions = append(runtimeParam.AwsOptions, SetCredentials(awsCred))

	reconciler, err := pkg.GetReconciler(kubicornCluster, &runtimeParam)
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
func (c *AWSCluster) GetK8sConfig() ([]byte, error) {
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
func DownloadK8sConfig(kubicornCluster *kcluster.Cluster) ([]byte, error) {

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
	return config, nil
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
	log.Info("TODO")
	// ---- [ Node check ] ---- //
	//if r.UpdateAmazonNode == nil {
	//	log.Info("'node' field is empty. Fill from stored data")
	//	r.UpdateAmazonNode = &amazon.UpdateAmazonNode{
	//		MinCount: c.modelCluster.Amazon.NodeMinCount,
	//		MaxCount: c.modelCluster.Amazon.NodeMaxCount,
	//	}
	//}
	//
	//// ---- [ Node min count check ] ---- //
	//if r.UpdateAmazonNode.MinCount == 0 {
	//	defMinCount := c.modelCluster.Amazon.NodeMinCount
	//	log.Info(constants.TagValidateUpdateCluster, "Node minCount set to default value: ", defMinCount)
	//	r.UpdateAmazonNode.MinCount = defMinCount
	//}
	//
	//// ---- [ Node max count check ] ---- //
	//if r.UpdateAmazonNode.MaxCount == 0 {
	//	defMaxCount := c.modelCluster.Amazon.NodeMaxCount
	//	log.Info(constants.TagValidateUpdateCluster, "Node maxCount set to default value: ", defMaxCount)
	//	r.UpdateAmazonNode.MaxCount = defMaxCount
	//}

}

//CheckEqualityToUpdate validates the update request
func (c *AWSCluster) CheckEqualityToUpdate(r *components.UpdateClusterRequest) error {
	// create update request struct with the stored data to check equality
	//preCl := &amazon.UpdateClusterAmazon{
	//	UpdateAmazonNode: &amazon.UpdateAmazonNode{
	//		MinCount: c.modelCluster.Amazon.NodeMinCount,
	//		MaxCount: c.modelCluster.Amazon.NodeMaxCount,
	//	},
	//}

	log.Info("Check stored & updated cluster equals")

	// check equality
	return nil //utils.IsDifferent(r.UpdateClusterAmazon, preCl)
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

// ListRegions lists supported regions
func ListRegions(region string) ([]*ec2.Region, error) {

	svc := newEC2Client(&aws.Config{
		Region: &region,
	})

	resultRegions, err := svc.DescribeRegions(nil)
	if err != nil {
		return nil, err
	}

	return resultRegions.Regions, nil
}

// ListAMIs returns supported AMIs by region and tags
func ListAMIs(region string, tags []*string) ([]*ec2.Image, error) {

	svc := newEC2Client(&aws.Config{
		Region: &region,
	})

	var input *ec2.DescribeImagesInput
	if tags != nil {
		tagKey := "tag:Name"
		input = &ec2.DescribeImagesInput{
			Filters: []*ec2.Filter{
				{
					Name:   &tagKey,
					Values: tags,
				},
			},
		}
	}

	images, err := svc.DescribeImages(input)
	if err != nil {
		return nil, err
	}

	return images.Images, nil
}

// newEC2Client creates new EC2 client
func newEC2Client(config *aws.Config) *ec2.EC2 {

	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	return ec2.New(sess, config)
}

// UpdateStatus updates cluster status in database
func (c *AWSCluster) UpdateStatus(status string) error {
	return c.modelCluster.UpdateStatus(status)
}

// GetClusterDetails gets cluster details from cloud
func (c *AWSCluster) GetClusterDetails() (*components.ClusterDetailsResponse, error) {

	log := logger.WithFields(logrus.Fields{"tag": "GetClusterDetails"})
	log.Info("Start getting cluster details")

	c.GetK8sConfig()
	c.GetAPIEndpoint()
	kubicornCluster, err := c.GetKubicornCluster()
	if err != nil {
		return nil, err
	}

	return &components.ClusterDetailsResponse{
		Name: kubicornCluster.Name,
		Id:   c.modelCluster.ID,
	}, nil
}

// ValidateCreationFields validates all field
func (c *AWSCluster) ValidateCreationFields(r *components.CreateClusterRequest) error {
	location := r.Location

	// Validate location
	log.Info("Validate location")
	if err := c.validateLocation(location); err != nil {
		return err
	}
	log.Info("Validate location passed")

	// Validate images
	log.Info("Validate images")
	masterImage := r.Properties.CreateClusterAmazon.Master.Image
	if err := c.validateAMIs(masterImage, r.Properties.CreateClusterAmazon.NodePools, location); err != nil {
		return err
	}
	log.Info("Validate images passed")

	return nil

}

// validateLocation validates location
func (c *AWSCluster) validateLocation(location string) error {
	log.Infof("Location: %s", location)
	validRegions, err := ListRegions(location)
	if err != nil {
		return err
	}

	log.Infof("Valid locations: %#v", validRegions)
	isContains := false
	for _, r := range validRegions {
		if location == *r.RegionName {
			isContains = true
			break
		}
	}

	if !isContains {
		return constants.ErrorNotValidLocation
	}

	return nil
}

// validateAMIs validates AMIs
func (c *AWSCluster) validateAMIs(masterAMI string, nodePools map[string]*amazon.AmazonNodePool, location string) error {

	log.Infof("Master image: %s", masterAMI)
	for nodePoolName, node := range nodePools {
		log.Infof("Node pool %s image: %s", nodePoolName, node.Image)
	}

	validImages, err := ListAMIs(location, nil)
	if err != nil {
		return err
	}

	var validImageMap = make(map[string]*ec2.Image, len(validImages))
	for _, image := range validImages {
		if image.ImageId != nil {
			validImageMap[*image.ImageId] = image
		}
	}

	if validImageMap[masterAMI] == nil {
		return constants.ErrorNotValidMasterImage
	}

	for _, node := range nodePools {
		if validImageMap[node.Image] == nil {
			return constants.ErrorNotValidNodeImage
		}
	}

	return nil

}
