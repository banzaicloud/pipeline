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
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/banzaicloud/pipeline/model"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	pkgEC2 "github.com/banzaicloud/pipeline/pkg/cluster/ec2"
	pkgCommon "github.com/banzaicloud/pipeline/pkg/common"
	pkgErrors "github.com/banzaicloud/pipeline/pkg/errors"
	"github.com/banzaicloud/pipeline/secret"
	"github.com/banzaicloud/pipeline/secret/verify"
	kcluster "github.com/kubicorn/kubicorn/apis/cluster"
	"github.com/kubicorn/kubicorn/pkg"
	"github.com/kubicorn/kubicorn/pkg/kubeadm"
	kubicornLogger "github.com/kubicorn/kubicorn/pkg/logger"
	"github.com/kubicorn/kubicorn/pkg/uuid"
	"github.com/kubicorn/kubicorn/state"
	"github.com/kubicorn/kubicorn/state/fs"
	"github.com/pkg/errors"
	"github.com/pkg/sftp"
	"github.com/spf13/viper"
	"golang.org/x/crypto/ssh"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	vpcIdKey   = "vpc-id"
	k8sCluster = "KubernetesCluster"
)

const (
	dependencyViolation = "DependencyViolation"
	notFoundGroup       = "InvalidGroup.NotFound"
)

const (
	filterForNodeName = "key-name"
)

// SetCredentials sets AWS credentials in session options
func SetCredentials(awscred *credentials.Credentials) func(*session.Options) error {
	return func(opts *session.Options) error {
		opts.Config.Credentials = awscred
		return nil
	}
}

//EC2Cluster struct for EC2 cluster
type EC2Cluster struct {
	kubicornCluster *kcluster.Cluster //Don't use this directly
	modelCluster    *model.ClusterModel
	APIEndpoint     string
	CommonClusterBase
}

// GetOrganizationId gets org where the cluster belongs
func (c *EC2Cluster) GetOrganizationId() uint {
	return c.modelCluster.OrganizationId
}

// GetLocation gets where the cluster is.
func (c *EC2Cluster) GetLocation() string {
	return c.modelCluster.Location
}

// GetSecretId retrieves the secret id
func (c *EC2Cluster) GetSecretId() string {
	return c.modelCluster.SecretId
}

// GetSshSecretId retrieves the ssh secret id
func (c *EC2Cluster) GetSshSecretId() string {
	return c.modelCluster.SshSecretId
}

// SaveSshSecretId saves the ssh secret id to database
func (c *EC2Cluster) SaveSshSecretId(sshSecretId string) error {
	return c.modelCluster.UpdateSshSecret(sshSecretId)
}

//GetID returns the specified cluster id
func (c *EC2Cluster) GetID() uint {
	return c.modelCluster.ID
}

func (c *EC2Cluster) GetUID() string {
	return c.modelCluster.UID
}

func (c *EC2Cluster) GetCreatorID() uint {
	return c.modelCluster.CreatedBy
}

//GetAPIEndpoint returns the Kubernetes Api endpoint
func (c *EC2Cluster) GetAPIEndpoint() (string, error) {
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
func (c *EC2Cluster) GetName() string {
	return c.modelCluster.Name
}

// GetCloud returns the cloud type of the cluster
func (c *EC2Cluster) GetCloud() string {
	return c.modelCluster.Cloud
}

// GetDistribution returns the distribution type of the cluster
func (c *EC2Cluster) GetDistribution() string {
	return c.modelCluster.Distribution
}

//GetModel returns the whole clusterModel
func (c *EC2Cluster) GetModel() *model.ClusterModel {
	return c.modelCluster
}

//CreateEC2ClusterFromModel creates ClusterModel struct from the kubicorn model
func CreateEC2ClusterFromModel(clusterModel *model.ClusterModel) (*EC2Cluster, error) {
	log.Debug("Create ClusterModel struct from the request")
	ec2Cluster := EC2Cluster{
		modelCluster: clusterModel,
	}
	if ec2Cluster.modelCluster.Status == pkgCluster.Running {
		_, err := ec2Cluster.GetKubicornCluster()
		if err != nil {
			return nil, err
		}
	}
	return &ec2Cluster, nil
}

//CreateEC2ClusterFromRequest creates ClusterModel struct from the request
func CreateEC2ClusterFromRequest(request *pkgCluster.CreateClusterRequest, orgId, userId uint) (*EC2Cluster, error) {
	log.Debug("Create ClusterModel struct from the request")
	var cluster EC2Cluster

	modelNodePools := createNodePoolsFromRequest(request.Properties.CreateClusterEC2.NodePools, userId)

	cluster.modelCluster = &model.ClusterModel{
		Name:           request.Name,
		Location:       request.Location,
		Cloud:          request.Cloud,
		SecretId:       request.SecretId,
		Distribution:   pkgCluster.EC2,
		OrganizationId: orgId,
		CreatedBy:      userId,
		EC2: model.EC2ClusterModel{
			MasterInstanceType: request.Properties.CreateClusterEC2.Master.InstanceType,
			MasterImage:        request.Properties.CreateClusterEC2.Master.Image,
			NodePools:          modelNodePools,
		},
	}
	return &cluster, nil
}

func createNodePoolsFromRequest(nodePools map[string]*pkgEC2.NodePool, userId uint) []*model.AmazonNodePoolsModel {
	var modelNodePools = make([]*model.AmazonNodePoolsModel, len(nodePools))
	i := 0
	for nodePoolName, nodePool := range nodePools {
		modelNodePools[i] = &model.AmazonNodePoolsModel{
			CreatedBy:        userId,
			Name:             nodePoolName,
			NodeSpotPrice:    nodePool.SpotPrice,
			Autoscaling:      nodePool.Autoscaling,
			NodeMinCount:     nodePool.MinCount,
			NodeMaxCount:     nodePool.MaxCount,
			Count:            nodePool.Count,
			NodeImage:        nodePool.Image,
			NodeInstanceType: nodePool.InstanceType,
			Delete:           false,
		}
		i++
	}
	return modelNodePools
}

//Persist save the cluster model
func (c *EC2Cluster) Persist(status, statusMessage string) error {
	return c.modelCluster.UpdateStatus(status, statusMessage)
}

//CreateCluster creates a new cluster
func (c *EC2Cluster) CreateCluster() error {

	// Set up credentials TODO simplify
	runtimeParam := pkg.RuntimeParameters{
		AwsProfile: "",
	}

	awsCred, err := c.createAWSCredentialsFromSecret()
	if err != nil {
		return err
	}

	clusterSshSecret, err := c.getSshSecret(c)
	if err != nil {
		return err
	}

	sshKey := secret.NewSSHKeyPair(clusterSshSecret)

	runtimeParam.AwsOptions = append(runtimeParam.AwsOptions, SetCredentials(awsCred))

	kubicornLogger.Level = getKubicornLogLevel()

	//TODO check if this should be private
	c.kubicornCluster = GetKubicornProfile(c.modelCluster)

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
		return pkgErrors.ErrorReconcile
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

// RequiresSshPublicKey returns true as a public ssh key is needed for bootstrapping
// the cluster
func (c *EC2Cluster) RequiresSshPublicKey() bool {
	return true
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
		Image:    cs.EC2.MasterImage, //"ami-835b4efa"
		Size:     cs.EC2.MasterInstanceType,
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
	var nodeServerPool = make([]*kcluster.ServerPool, len(cs.EC2.NodePools))
	for i, nodePool := range cs.EC2.NodePools {
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
func (c *EC2Cluster) GetStatus() (*pkgCluster.GetClusterStatusResponse, error) {
	log.Info("Start get cluster status (ec2)")

	nodePools := make(map[string]*pkgCluster.NodePoolStatus)
	for _, np := range c.modelCluster.EC2.NodePools {
		if np != nil {
			nodePools[np.Name] = &pkgCluster.NodePoolStatus{
				Autoscaling:  np.Autoscaling,
				Count:        np.Count,
				InstanceType: np.NodeInstanceType,
				SpotPrice:    np.NodeSpotPrice,
				MinCount:     np.NodeMinCount,
				MaxCount:     np.NodeMaxCount,
				Image:        np.NodeImage,
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
		CreatorBaseFields: *NewCreatorBaseFields(c.modelCluster.CreatedAt, c.modelCluster.CreatedBy),
		NodePools:         nodePools,
	}, nil
}

// getExistingNodePoolByName returns existing NodePool from model nodepools by name
func (c *EC2Cluster) getExistingNodePoolByName(name string) *model.AmazonNodePoolsModel {
	for _, np := range c.modelCluster.EC2.NodePools {
		if np != nil && np.Name == name {
			return np
		}
	}
	return nil
}

// UpdateCluster updates Amazon cluster in cloud
func (c *EC2Cluster) UpdateCluster(request *pkgCluster.UpdateClusterRequest, userId uint) error {

	kubicornLogger.Level = getKubicornLogLevel()

	log.Info("Start updating cluster (ec2)")

	if request == nil {
		return pkgErrors.ErrorEmptyUpdateRequest
	}

	existingNodePools := map[string]*model.AmazonNodePoolsModel{}
	for _, nodePool := range c.modelCluster.EC2.NodePools {
		existingNodePools[nodePool.Name] = nodePool
	}

	existingAsgs := map[string]*kcluster.ServerPool{}
	for _, asg := range c.kubicornCluster.ServerPools {
		poolName := getAsgNodePoolName(asg.Name)
		existingAsgs[poolName] = asg
	}

	var updatedNodePools []*model.AmazonNodePoolsModel
	for name, np := range request.EC2.NodePools {
		if np != nil {

			existsNode := c.getExistingNodePoolByName(name)
			var id uint
			if existsNode != nil {
				id = existsNode.ID
			}
			nodePoolModel := &model.AmazonNodePoolsModel{
				ID:               id,
				CreatedAt:        time.Now(),
				CreatedBy:        userId,
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

	updatedNodePools = addMarkedForDeletePools(c.modelCluster.EC2.NodePools, updatedNodePools)

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
		ConfigSecretId: c.modelCluster.ConfigSecretId,
		SshSecretId:    c.modelCluster.SshSecretId,
		Status:         c.modelCluster.Status,
		EC2: model.EC2ClusterModel{
			MasterInstanceType: c.modelCluster.EC2.MasterInstanceType,
			MasterImage:        c.modelCluster.EC2.MasterImage,
			NodePools:          updatedNodePools,
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

	awsCred, err := c.createAWSCredentialsFromSecret()
	if err != nil {
		return err
	}

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

	//Update EC2 model
	c.modelCluster = updateCluster
	c.kubicornCluster = kubicornCluster //This is redundant TODO check if it's ok

	// TODO check statestore usage
	statestore := getStateStoreForCluster(updateCluster)
	log.Info("Save cluster to the statestore")
	statestore.Commit(updated)

	// mark for deletion the node pool model entries that has no corresponding node pool in the cluster
	for _, np := range c.modelCluster.EC2.NodePools {
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
	return 0, pkgErrors.ErrorNodePoolNotFoundByName
}

//GetKubicornCluster returns a Kubicorn cluster
func (c *EC2Cluster) GetKubicornCluster() (*kcluster.Cluster, error) {
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

// DeleteCluster deletes cluster from ec2
func (c *EC2Cluster) DeleteCluster() error {

	kubicornLogger.Level = getKubicornLogLevel()

	log.Info("Start delete ec2 cluster")
	kubicornCluster, err := c.GetKubicornCluster()
	if err != nil {
		return err
	}

	log.Info("Start deleting security group for Kubernetes ELB")
	if err := c.revokeELBDependency(kubicornCluster.Network.Identifier); err != nil {
		return err
	}

	statestore := getStateStoreForCluster(c.modelCluster)
	log.Debug("Get reconciler")

	// Set up credentials TODO simplify
	runtimeParam := pkg.RuntimeParameters{
		AwsProfile: "",
	}

	awsCred, err := c.createAWSCredentialsFromSecret()
	if err != nil {
		return err
	}

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

// DownloadK8sConfig downloads the kubeconfig file from cloud
func (c *EC2Cluster) DownloadK8sConfig() ([]byte, error) {
	kubicornCluster, err := c.GetKubicornCluster()
	if err != nil {
		err = errors.Wrap(err, "error getting kubicorn cluster")
		return nil, err
	}

	sshSecret, err := c.getSshSecret(c)
	if err != nil {
		return nil, err
	}

	return DownloadK8sConfig(kubicornCluster, c.GetOrganizationId(), secret.NewSSHKeyPair(sshSecret))
}

//DownloadK8sConfig downloads the Kubernetes config from the cluster
// Todo check first if config is locally available
func DownloadK8sConfig(kubicornCluster *kcluster.Cluster, organizationID uint, key *secret.SSHKeyPair) ([]byte, error) {

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
	BootstrapScriptMasterDefault = "https://raw.githubusercontent.com/banzaicloud/banzai-charts/master/pipeline/bootstrap/amazon_k8s_ubuntu_16.04_master_pipeline.sh"
	BootstrapScriptNodeDefault   = "https://raw.githubusercontent.com/banzaicloud/banzai-charts/master/pipeline/bootstrap/amazon_k8s_ubuntu_16.04_node_pipeline.sh"
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
func (c *EC2Cluster) AddDefaultsToUpdate(r *pkgCluster.UpdateClusterRequest) {

	// add default node image(s) if needed
	if r != nil && r.EC2 != nil && r.EC2.NodePools != nil {
		for name, np := range r.EC2.NodePools {
			if len(np.Image) == 0 {
				np.Image = c.getImageFromNodePool(name)
			}
		}
	}

}

func (c *EC2Cluster) getImageFromNodePool(nodePoolName string) string {
	for _, np := range c.modelCluster.EC2.NodePools {
		if np != nil && np.Name == nodePoolName {
			return np.NodeImage
		}
	}
	return pkgEC2.DefaultImages[c.modelCluster.Location]
}

//CheckEqualityToUpdate validates the update request
func (c *EC2Cluster) CheckEqualityToUpdate(r *pkgCluster.UpdateClusterRequest) error {
	return CheckEqualityToUpdate(r, c.modelCluster.EC2.NodePools)
}

//CheckEqualityToUpdate validates the update request
func CheckEqualityToUpdate(r *pkgCluster.UpdateClusterRequest, nodePools []*model.AmazonNodePoolsModel) error {
	// create update request struct with the stored data to check equality
	preNodePools := make(map[string]*pkgEC2.NodePool)
	for _, preNp := range nodePools {
		preNodePools[preNp.Name] = &pkgEC2.NodePool{
			InstanceType: preNp.NodeInstanceType,
			SpotPrice:    preNp.NodeSpotPrice,
			Autoscaling:  preNp.Autoscaling,
			MinCount:     preNp.NodeMinCount,
			MaxCount:     preNp.NodeMaxCount,
			Count:        preNp.Count,
			Image:        preNp.NodeImage,
		}
	}

	preCl := &pkgEC2.UpdateClusterAmazon{
		NodePools: preNodePools,
	}

	log.Info("Check stored & updated cluster equals")

	// check equality
	return isDifferent(r.EC2, preCl)
}

//DeleteFromDatabase deletes model from the database
func (c *EC2Cluster) DeleteFromDatabase() error {
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
	c := &EC2Cluster{
		modelCluster: &model.ClusterModel{
			OrganizationId: orgId,
			SecretId:       secretId,
			Cloud:          pkgCluster.Amazon,
		},
	}
	return c.ListRegions(region)
}

// ListRegions lists supported regions
func (c *EC2Cluster) ListRegions(region string) ([]*ec2.Region, error) {

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
	c := &EC2Cluster{
		modelCluster: &model.ClusterModel{
			OrganizationId: orgId,
			SecretId:       secretId,
			Cloud:          pkgCluster.Amazon,
		},
	}
	return c.ListAMIs(region, tags)
}

// ListAMIs returns supported AMIs by region and tags
func (c *EC2Cluster) ListAMIs(region string, tags []*string) ([]*ec2.Image, error) {

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
func (c *EC2Cluster) newEC2Client(region string) (*ec2.EC2, error) {

	log.Info("create new ec2 client")

	awsCred, err := c.createAWSCredentialsFromSecret()
	if err != nil {
		return nil, err
	}

	return verify.CreateEC2Client(awsCred, region)
}

// UpdateStatus updates cluster status in database
func (c *EC2Cluster) UpdateStatus(status, statusMessage string) error {
	return c.modelCluster.UpdateStatus(status, statusMessage)
}

// GetClusterDetails gets cluster details from cloud
func (c *EC2Cluster) GetClusterDetails() (*pkgCluster.DetailsResponse, error) {

	log.Info("Start getting cluster details")

	c.GetK8sConfig()
	c.GetAPIEndpoint()
	kubicornCluster, err := c.GetKubicornCluster()
	if err != nil {
		return nil, err
	}

	nodePools := make(map[string]*pkgCluster.NodeDetails, 0)
	for _, np := range c.modelCluster.EC2.NodePools {
		if np != nil {

			nodePools[np.Name] = &pkgCluster.NodeDetails{
				CreatorBaseFields: *NewCreatorBaseFields(np.CreatedAt, np.CreatedBy),
				Count:             np.Count,
				MinCount:          np.NodeMinCount,
				MaxCount:          np.NodeMaxCount,
			}
		}
	}

	return &pkgCluster.DetailsResponse{
		CreatorBaseFields: *NewCreatorBaseFields(c.modelCluster.CreatedAt, c.modelCluster.CreatedBy),
		Name:              kubicornCluster.Name,
		Id:                c.modelCluster.ID,
		Location:          c.modelCluster.Location,
		NodePools:         nodePools,
		Status:            c.modelCluster.Status,
	}, nil
}

// ValidateCreationFields validates all fields
func (c *EC2Cluster) ValidateCreationFields(r *pkgCluster.CreateClusterRequest) error {
	location := r.Location

	// Validate location
	log.Info("Validate location")
	if err := c.validateLocation(location); err != nil {
		return err
	}
	log.Info("Validate location passed")

	// Validate images
	log.Info("Validate images")
	masterImage := r.Properties.CreateClusterEC2.Master.Image
	if err := c.validateAMIs(masterImage, r.Properties.CreateClusterEC2.NodePools, location); err != nil {
		return err
	}
	log.Info("Validate images passed")

	return nil

}

// validateLocation validates location
func (c *EC2Cluster) validateLocation(location string) error {
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
		return pkgErrors.ErrorNotValidLocation
	}

	return nil
}

// validateAMIs validates AMIs
func (c *EC2Cluster) validateAMIs(masterAMI string, nodePools map[string]*pkgEC2.NodePool, location string) error {

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
		return pkgErrors.ErrorNotValidMasterImage
	}

	for _, node := range nodePools {
		if validImageMap[node.Image] == nil {
			return pkgErrors.ErrorNotValidNodeImage
		}
	}

	return nil

}

// GetSecretWithValidation returns secret from vault
func (c *EC2Cluster) GetSecretWithValidation() (*secret.SecretItemResponse, error) {
	return c.CommonClusterBase.getSecret(c)
}

// SaveConfigSecretId saves the config secret id in database
func (c *EC2Cluster) SaveConfigSecretId(configSecretId string) error {
	return c.modelCluster.UpdateConfigSecret(configSecretId)
}

// GetConfigSecretId return config secret id
func (c *EC2Cluster) GetConfigSecretId() string {
	return c.modelCluster.ConfigSecretId
}

// GetK8sConfig returns the Kubernetes config
func (c *EC2Cluster) GetK8sConfig() ([]byte, error) {
	return c.CommonClusterBase.getConfig(c)
}

// listSecurityGroups listing security groups by VPC id
func (c *EC2Cluster) listSecurityGroups(svc *ec2.EC2, vpcId string) ([]*ec2.SecurityGroup, error) {

	output, err := svc.DescribeSecurityGroups(&ec2.DescribeSecurityGroupsInput{
		Filters: []*ec2.Filter{
			{
				Name:   &vpcIdKey,
				Values: []*string{&vpcId},
			},
		},
	})

	if err != nil {
		return nil, err
	}

	return output.SecurityGroups, nil
}

// deleteSecurityGroup deletes a security group by group params
func (c *EC2Cluster) deleteSecurityGroup(svc *ec2.EC2, group *ec2.SecurityGroup) error {

	log.Infof("Delete security group [%s]", *group.GroupId)

	for {
		if _, err := svc.DeleteSecurityGroup(&ec2.DeleteSecurityGroupInput{
			GroupId: group.GroupId,
		}); err != nil {
			if strings.Contains(err.Error(), dependencyViolation) {
				log.Infof("retry delete: %s", err.Error())
				time.Sleep(time.Duration(10) * time.Second)
				continue
			} else if strings.Contains(err.Error(), notFoundGroup) {
				return nil
			}
			return err
		}
		break
	}

	return nil
}

// revokeELBDependency remove all ELB dependency
func (c *EC2Cluster) revokeELBDependency(vpcId string) error {

	log.Info("Revoke all ELB security group dependency")

	log.Info("Create new EC2 client")
	svc, err := c.newEC2Client(c.modelCluster.Location)
	if err != nil {
		return err
	}

	log.Info("List security groups")
	groups, err := c.listSecurityGroups(svc, vpcId)
	if err != nil {
		return err
	}

	log.Info("Find ELB security group(s)")
	sourceGroups := c.findELBSecurityGroups(groups)
	if len(sourceGroups) == 0 {
		return nil
	}
	for _, group := range sourceGroups {
		log.Debugf("ELB security group id: %s", *group.GroupId)
		log.Info("Revoke ELB security group dependency")
		if err := c.revokeELBSecurityGroupDependency(svc, groups, group); err != nil {
			return err
		}

		log.Info("Delete ELB security group")
		if err := c.deleteSecurityGroup(svc, group); err != nil {
			return err
		}

	}

	return nil
}

// revokeELBSecurityGroupDependency removes all ELB dependency ingress rules from security groups
func (c *EC2Cluster) revokeELBSecurityGroupDependency(svc *ec2.EC2, groups []*ec2.SecurityGroup, sourceGroup *ec2.SecurityGroup) error {

	for _, group := range groups {
		if group != nil && len(group.IpPermissions) != 0 {
			for _, p := range group.IpPermissions {
				if p != nil && len(p.UserIdGroupPairs) != 0 {
					for _, gp := range p.UserIdGroupPairs {
						if gp != nil && *gp.GroupId == *sourceGroup.GroupId {
							log.Infof("ELB security group dependency found [%s]", *gp.GroupId)
							if err := c.revokeSecurityGroupIngress(svc, group, sourceGroup, p); err != nil {
								return err
							}
						}
					}
				}
			}
		}
	}

	return nil
}

// revokeSecurityGroupIngress removes an ingress rules from a security group
func (c *EC2Cluster) revokeSecurityGroupIngress(svc *ec2.EC2, sg, sourceGroup *ec2.SecurityGroup, permission *ec2.IpPermission) error {

	log.Infof("Revoke security group ingress [ %s // %s ]", *sg.GroupId, *sourceGroup.GroupId)

	_, err := svc.RevokeSecurityGroupIngress(&ec2.RevokeSecurityGroupIngressInput{
		GroupId:       sg.GroupId,
		IpPermissions: []*ec2.IpPermission{permission},
	})

	log.Info("Revoke security group succeeded")

	return err
}

// findELBSecurityGroups looks for ELB security group(s)
func (c *EC2Cluster) findELBSecurityGroups(groups []*ec2.SecurityGroup) []*ec2.SecurityGroup {

	var elbs []*ec2.SecurityGroup

	for _, g := range groups {
		if g != nil && g.Tags != nil {
			for _, t := range g.Tags {
				if t != nil {
					if t.Key != nil && t.Value != nil &&
						*t.Key == k8sCluster && *t.Value == c.modelCluster.Name {
						elbs = append(elbs, g)
					}
				}
			}
		}
	}

	return elbs
}

func (c *EC2Cluster) createAWSCredentialsFromSecret() (*credentials.Credentials, error) {
	clusterSecret, err := c.GetSecretWithValidation()
	if err != nil {
		return nil, err
	}
	return verify.CreateAWSCredentials(clusterSecret.Values), nil
}

// ListNodeNames returns node names to label them
func (c *EC2Cluster) ListNodeNames() (labels pkgCommon.NodeNames, err error) {

	var svc *ec2.EC2
	svc, err = c.newEC2Client(c.modelCluster.Location)
	if err != nil {
		return
	}

	filterName := filterForNodeName

	var out *ec2.DescribeInstancesOutput
	out, err = svc.DescribeInstances(&ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			{
				Name:   &filterName,
				Values: []*string{&c.modelCluster.Name},
			},
		},
	})
	if err != nil {
		return
	}

	labels = make(map[string][]string)

	for _, np := range c.modelCluster.EC2.NodePools {
		if np != nil {
			for _, r := range out.Reservations {
				for _, i := range r.Instances {
					if i != nil &&
						hasTagWithNodeName(i.Tags, fmt.Sprintf("%s.node.%s", c.modelCluster.Name, np.Name)) &&
						i.PrivateDnsName != nil {
						labels[np.Name] = append(labels[np.Name], *i.PrivateDnsName)
					}
				}
			}

		}
	}

	return
}

func hasTagWithNodeName(tags []*ec2.Tag, nodeName string) bool {
	for _, tag := range tags {
		if tag != nil && tag.Key != nil && tag.Value != nil && *tag.Key == "Name" && *tag.Value == nodeName {
			return true
		}
	}
	return false
}

// RbacEnabled returns true if rbac enabled on the cluster
func (c *EC2Cluster) RbacEnabled() bool {
	return c.modelCluster.RbacEnabled
}

// GetEC2NodePools returns EC2 node pools from a common cluster.
func GetEC2NodePools(cluster CommonCluster) ([]*model.AmazonNodePoolsModel, error) {
	ec2cluster, ok := cluster.(*EC2Cluster)
	if !ok {
		return nil, ErrInvalidClusterInstance
	}

	return ec2cluster.modelCluster.EC2.NodePools, nil
}

// NeedAdminRights returns true if rbac is enabled and need to create a cluster role binding to user
func (c *EC2Cluster) NeedAdminRights() bool {
	return false
}

// GetKubernetesUserName returns the user ID which needed to create a cluster role binding which gives admin rights to the user
func (c *EC2Cluster) GetKubernetesUserName() (string, error) {
	return "", nil
}
