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
	pipelineSsh "github.com/banzaicloud/pipeline/ssh"
	"github.com/banzaicloud/pipeline/utils"
	kcluster "github.com/kubicorn/kubicorn/apis/cluster"
	"github.com/kubicorn/kubicorn/pkg"
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

// SetCredentials sets AWS credentials in session options
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
	commonSecret
}

// GetOrganizationId gets org where the cluster belongs
func (c *AWSCluster) GetOrganizationId() uint {
	return c.modelCluster.OrganizationId
}

// GetSecretID retrieves the secret id
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

func createNodePoolsFromRequest(nodePools map[string]*amazon.NodePool) []*model.AmazonNodePoolsModel {
	var modelNodePools = make([]*model.AmazonNodePoolsModel, len(nodePools))
	i := 0
	for nodePoolName, nodePool := range nodePools {
		modelNodePools[i] = &model.AmazonNodePoolsModel{
			Name:             nodePoolName,
			NodeInstanceType: nodePool.InstanceType,
			NodeSpotPrice:    nodePool.SpotPrice,
			NodeImage:        nodePool.Image,
			Autoscaling:      nodePool.Autoscaling,
			NodeMinCount:     nodePool.MinCount,
			NodeMaxCount:     nodePool.MaxCount,
			Count:            nodePool.Count,
		}
		i++
	}
	return modelNodePools
}

//Persist save the cluster model
func (c *AWSCluster) Persist(status, statusMessage string) error {
	return c.modelCluster.UpdateStatus(status, statusMessage)
}

//CreateCluster creates a new cluster
func (c *AWSCluster) CreateCluster() error {
	log := logger.WithFields(logrus.Fields{"action": constants.TagCreateCluster})

	// Set up credentials TODO simplify
	runtimeParam := pkg.RuntimeParameters{
		AwsProfile: "",
	}
	clusterSecret, err := c.GetSecretWithValidation()
	if err != nil {
		return err
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

	sshSecretID, sshKey, err := pipelineSsh.KeyAdd(c.GetModel().OrganizationId, c.GetModel().ID)
	c.GetModel().Amazon.SshSecretID = sshSecretID

	c.kubicornCluster.SSH.PublicKeyData = []byte(sshKey.PublicKeyData)
	c.kubicornCluster.SSH.PublicKeyFingerprint = sshKey.PublicKeyFingerprint

	newCluster := c.kubicornCluster
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
		Name:     getNodeName(clusterName, nodePool.Name),
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
				Name:     getNodeName(clusterName, nodePool.Name),
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
						IngressSource:   "10.0.0.0/16",
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

	nodePools := make(map[string]*components.NodePoolStatus)
	for _, np := range c.modelCluster.Amazon.NodePools {
		if np != nil {
			nodePools[np.Name] = &components.NodePoolStatus{
				InstanceType: np.NodeInstanceType,
				SpotPrice:    np.NodeSpotPrice,
				MinCount:     np.NodeMinCount,
				MaxCount:     np.NodeMaxCount,
				Image:        np.NodeImage,
			}
		}
	}

	return &components.GetClusterStatusResponse{
		Status:        c.modelCluster.Status,
		StatusMessage: c.modelCluster.StatusMessage,
		Name:          c.modelCluster.Name,
		Location:      c.modelCluster.Location,
		Cloud:         c.modelCluster.Cloud,
		ResourceID:    c.modelCluster.ID,
		NodePools:     nodePools,
	}, nil
}

// getExistingNodePoolByName returns existing NodePool from model nodepools by name
func (c *AWSCluster) getExistingNodePoolByName(name string) *model.AmazonNodePoolsModel {
	for _, np := range c.modelCluster.Amazon.NodePools {
		if np != nil && np.Name == name {
			return np
		}
	}
	return nil
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

	existingAsgs := map[string]*kcluster.ServerPool{}
	for _, asg := range c.kubicornCluster.ServerPools {
		poolName := getAsgNodePoolName(asg.Name)
		existingAsgs[poolName] = asg
	}

	var updatedNodePools []*model.AmazonNodePoolsModel
	for name, np := range request.Amazon.NodePools {
		if np != nil {

			existsNode := c.getExistingNodePoolByName(name)
			var id uint
			if existsNode != nil {
				id = existsNode.ID
			}
			nodePoolModel := &model.AmazonNodePoolsModel{
				ID:               id,
				Name:             name,
				NodeSpotPrice:    np.SpotPrice,
				Autoscaling:      np.Autoscaling,
				NodeMinCount:     np.MinCount,
				NodeMaxCount:     np.MaxCount,
				Count:            np.Count,
				NodeImage:        np.Image,
				NodeInstanceType: np.InstanceType,
				Delete:           false,
			}
			updatedNodePools = append(updatedNodePools, nodePoolModel)

		}
	}

	updatedNodePools = addMarkedForDeletePools(c.modelCluster.Amazon.NodePools, updatedNodePools)

	log.Info("Create updated model")
	updateCluster := &model.ClusterModel{
		ID:             c.modelCluster.ID,
		CreatedAt:      c.modelCluster.CreatedAt,
		UpdatedAt:      c.modelCluster.UpdatedAt,
		DeletedAt:      c.modelCluster.DeletedAt,
		Name:           c.modelCluster.Name,
		Location:       c.modelCluster.Location,
		Cloud:          request.Cloud,
		OrganizationId: c.modelCluster.OrganizationId,
		SecretId:       c.modelCluster.SecretId,
		Status:         c.modelCluster.Status,
		Amazon: model.AmazonClusterModel{
			MasterInstanceType: c.modelCluster.Amazon.MasterInstanceType,
			MasterImage:        c.modelCluster.Amazon.MasterImage,
			NodePools:          updatedNodePools,
			SshSecretID:        c.modelCluster.Amazon.SshSecretID,
		},
	}

	log.Debug("Resizing cluster: ", c.GetName())
	kubicornCluster, err := c.GetKubicornCluster()
	if err != nil {
		return err
	}

	kubicornCluster.ServerPools[0].MinCount = 1
	kubicornCluster.ServerPools[0].MaxCount = 1

	uuidSuffix := uuid.TimeOrderedUUID()
	var missingIds []int
	globalI := len(kubicornCluster.ServerPools) + 1
	for i, np := range updatedNodePools {
		id, err := findServerPool(kubicornCluster.ServerPools, c.GetName(), np.Name)
		if !np.Delete {
			if err != nil {
				missingIds = append(missingIds, i)
			} else {
				log.Infof("[%d] Update existing nodepool: %s, min count: %d, max count: %d", id, np.Name, np.NodeMinCount, np.NodeMaxCount)
				kubicornCluster.ServerPools[id].MinCount = np.NodeMinCount
				kubicornCluster.ServerPools[id].MaxCount = np.NodeMaxCount
			}
		} else {
			log.Infof("Nodepool mark for delete: %s", np.Name)
			if err == nil {
				// remove from kubicorn server pool
				kubicornCluster.ServerPools[id].MinCount = 0
				kubicornCluster.ServerPools[id].MaxCount = 0
			}
		}
	}

	// add new pools
	for _, id := range missingIds {
		log.Infof("Add nodepool: %s", updatedNodePools[id].Name)
		sp := getNodeServerPool(c.modelCluster.Name, c.modelCluster.Location, updatedNodePools[id], fmt.Sprintf("10.0.%d.0/24", 100+globalI), uuidSuffix)
		kubicornCluster.ServerPools = append(kubicornCluster.ServerPools, sp)
		globalI += 1
	}

	log.Debug("Get reconciler")

	// Set up credentials TODO simplify
	runtimeParam := pkg.RuntimeParameters{
		AwsProfile: "",
	}
	clusterSecret, err := c.GetSecretWithValidation()
	if err != nil {
		return err
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

	// mark for deletion the node pool model entries that has no corresponding node pool in the cluster
	for _, np := range c.modelCluster.Amazon.NodePools {
		found := false

		for _, kubicornNodePool := range kubicornCluster.ServerPools {
			if kubicornNodePool != nil {
				if getNodeName(c.modelCluster.Name, np.Name) == kubicornNodePool.Name {
					found = true
					break
				}
			}
		}

		if !found {
			np.Delete = true
		}

	}

	return nil
}

// addMarkedForDeletePools adds delete "flag" for the proper pools
func addMarkedForDeletePools(storedNodePools []*model.AmazonNodePoolsModel, updatedNodePools []*model.AmazonNodePoolsModel) []*model.AmazonNodePoolsModel {
	for _, storedPool := range storedNodePools {
		found := false
		for _, updatedPool := range updatedNodePools {
			if storedPool.Name == updatedPool.Name {
				found = true
				break
			}
		}
		if !found {
			storedPool.Delete = true
			updatedNodePools = append(updatedNodePools, storedPool)
		}
	}
	return updatedNodePools
}

// getNodeName returns node name
func getNodeName(clusterName, name string) string {
	return fmt.Sprintf("%s.node.%s", clusterName, name)
}

// findServerPool search serverPool in kubernetes pools by name
func findServerPool(pools []*kcluster.ServerPool, clusterName, name string) (int, error) {
	for i, pool := range pools {
		if pool != nil && pool.Name == getNodeName(clusterName, name) {
			return i, nil
		}
	}
	return 0, constants.ErrorNodePoolNotFoundByName
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
	clusterSecret, err := c.GetSecretWithValidation()
	if err != nil {
		return err
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
	kubeConfig, err := DownloadK8sConfig(kubicornCluster, fmt.Sprint(c.GetModel().OrganizationId), c.GetModel().ID)
	if err != nil {
		err = errors.Wrap(err, "error downloading kubernetes config")
		return nil, err
	}
	c.k8sConfig = kubeConfig
	return c.k8sConfig, nil
}

//DownloadK8sConfig downloads the Kubernetes config from the cluster
// Todo check first if config is locally available
func DownloadK8sConfig(kubicornCluster *kcluster.Cluster, organizationID string, ClusterID uint) ([]byte, error) {

	user := kubicornCluster.SSH.User
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
	key, err := pipelineSsh.KeyGet(organizationID, ClusterID)
	if err != nil {
		return nil, err
	}

	pemBytes := []byte(key.PrivateKeyData)

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
	// no needed this time, validate failed if there's missing field(s)
}

//CheckEqualityToUpdate validates the update request
func (c *AWSCluster) CheckEqualityToUpdate(r *components.UpdateClusterRequest) error {
	// create update request struct with the stored data to check equality

	preNodePools := make(map[string]*amazon.NodePool)
	for _, preNp := range c.modelCluster.Amazon.NodePools {
		preNodePools[preNp.Name] = &amazon.NodePool{
			InstanceType: preNp.NodeInstanceType,
			SpotPrice:    preNp.NodeSpotPrice,
			Autoscaling:  preNp.Autoscaling,
			MinCount:     preNp.NodeMinCount,
			MaxCount:     preNp.NodeMaxCount,
			Count:        preNp.Count,
			Image:        preNp.NodeImage,
		}
	}

	preCl := &amazon.UpdateClusterAmazon{
		NodePools: preNodePools,
	}

	log.Info("Check stored & updated cluster equals")

	// check equality
	return utils.IsDifferent(r.Amazon, preCl)
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
func ListRegions(orgId uint, secretId, region string) ([]*ec2.Region, error) {
	c := &AWSCluster{
		modelCluster: &model.ClusterModel{
			OrganizationId: orgId,
			SecretId:       secretId,
			Cloud:          constants.Amazon,
		},
	}
	return c.ListRegions(region)
}

// ListRegions lists supported regions
func (c *AWSCluster) ListRegions(region string) ([]*ec2.Region, error) {

	svc, err := c.newEC2Client(region)
	if err != nil {
		return nil, err
	}

	resultRegions, err := svc.DescribeRegions(nil)
	if err != nil {
		return nil, err
	}

	return resultRegions.Regions, nil
}

// ListAMIs returns supported AMIs by region and tags
func ListAMIs(orgId uint, secretId, region string, tags []*string) ([]*ec2.Image, error) {
	c := &AWSCluster{
		modelCluster: &model.ClusterModel{
			OrganizationId: orgId,
			SecretId:       secretId,
			Cloud:          constants.Amazon,
		},
	}
	return c.ListAMIs(region, tags)
}

// ListAMIs returns supported AMIs by region and tags
func (c *AWSCluster) ListAMIs(region string, tags []*string) ([]*ec2.Image, error) {

	svc, err := c.newEC2Client(region)
	if err != nil {
		return nil, err
	}

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
func (c *AWSCluster) newEC2Client(region string) (*ec2.EC2, error) {

	log.Info("create new ec2 client")

	clusterSecret, err := c.GetSecretWithValidation()
	if err != nil {
		return nil, err
	}

	awsCred := credentials.NewStaticCredentials(
		clusterSecret.Values[secret.AwsAccessKeyId],
		clusterSecret.Values[secret.AwsSecretAccessKey],
		"",
	)

	// set aws log level
	var lv aws.LogLevelType
	if logger.Level == logrus.DebugLevel {
		log.Info("set aws log level to debug")
		lv = aws.LogDebug
	} else {
		log.Info("set aws log off")
		lv = aws.LogOff
	}

	sess, err := session.NewSession(&aws.Config{
		Credentials: awsCred,
		Region:      &region,
		LogLevel:    &lv,
	})
	if err != nil {
		return nil, err
	}

	return ec2.New(sess), nil
}

// UpdateStatus updates cluster status in database
func (c *AWSCluster) UpdateStatus(status, statusMessage string) error {
	return c.modelCluster.UpdateStatus(status, statusMessage)
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
	validRegions, err := c.ListRegions(location)
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
func (c *AWSCluster) validateAMIs(masterAMI string, nodePools map[string]*amazon.NodePool, location string) error {

	log.Infof("Master image: %s", masterAMI)
	for nodePoolName, node := range nodePools {
		log.Infof("Node pool %s image: %s", nodePoolName, node.Image)
	}

	validImages, err := c.ListAMIs(location, nil)
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

// GetSecretWithValidation returns secret from vault
func (c *AWSCluster) GetSecretWithValidation() (*secret.SecretsItemResponse, error) {
	return c.commonSecret.get(c)
}
