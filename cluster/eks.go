package cluster

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/eks"
	"github.com/banzaicloud/pipeline/helm"
	"github.com/banzaicloud/pipeline/model"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	"github.com/banzaicloud/pipeline/pkg/cluster/eks/action"
	pkgCommon "github.com/banzaicloud/pipeline/pkg/common"
	"github.com/banzaicloud/pipeline/secret"
	"github.com/banzaicloud/pipeline/secret/verify"
	"github.com/banzaicloud/pipeline/utils"
	"github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api/v1"
)

const authConfigMapTemplate = `- rolearn: %s
  username: system:node:{{EC2PrivateDNSName}}
  groups:
  - system:bootstrappers
  - system:nodes
`

//CreateEKSClusterFromRequest creates ClusterModel struct from the request
func CreateEKSClusterFromRequest(request *pkgCluster.CreateClusterRequest, orgId uint, userId uint) (*EKSCluster, error) {
	log.Debug("Create ClusterModel struct from the request")
	var cluster EKSCluster

	modelNodePools := createNodePoolsFromRequest(request.Properties.CreateClusterEks.NodePools, userId)

	cluster.modelCluster = &model.ClusterModel{
		Name:           request.Name,
		Location:       request.Location,
		Cloud:          request.Cloud,
		OrganizationId: orgId,
		SecretId:       request.SecretId,

		Eks: model.AmazonEksClusterModel{
			Version:   request.Properties.CreateClusterEks.Version,
			NodePools: modelNodePools,
		},
	}
	return &cluster, nil
}

//EKSCluster struct for EKS cluster
type EKSCluster struct {
	eksCluster               *eks.Cluster //Don't use this directly
	modelCluster             *model.ClusterModel
	APIEndpoint              string
	CertificateAuthorityData []byte
	CommonClusterBase
}

// GetOrganizationId gets org where the cluster belongs
func (e *EKSCluster) GetOrganizationId() uint {
	return e.modelCluster.OrganizationId
}

// GetSecretId retrieves the secret id
func (e *EKSCluster) GetSecretId() string {
	return e.modelCluster.SecretId
}

// GetSshSecretId retrieves the secret id
func (e *EKSCluster) GetSshSecretId() string {
	return e.modelCluster.SshSecretId
}

// SaveSshSecretId saves the ssh secret id to database
func (e *EKSCluster) SaveSshSecretId(sshSecretId string) error {
	return e.modelCluster.UpdateSshSecret(sshSecretId)
}

//GetAPIEndpoint returns the Kubernetes Api endpoint
func (e *EKSCluster) GetAPIEndpoint() (string, error) {
	return e.APIEndpoint, nil
}

//CreateEKSClusterFromModel creates ClusterModel struct from the model
func CreateEKSClusterFromModel(clusterModel *model.ClusterModel) (*EKSCluster, error) {
	log.Debug("Create ClusterModel struct from the request")
	eksCluster := EKSCluster{
		modelCluster: clusterModel,
	}
	return &eksCluster, nil
}

func (e *EKSCluster) createAWSCredentialsFromSecret() (*credentials.Credentials, error) {
	clusterSecret, err := e.GetSecretWithValidation()
	if err != nil {
		return nil, err
	}
	return verify.CreateAWSCredentials(clusterSecret.Values), nil
}

// CreateCluster creates an EKS cluster with cloudformation templates.
func (e *EKSCluster) CreateCluster() error {

	log.Info("Start creating EKS cluster")

	awsCred, err := e.createAWSCredentialsFromSecret()
	if err != nil {
		return err
	}

	session, err := session.NewSession(&aws.Config{
		Region:      aws.String(e.modelCluster.Location),
		Credentials: awsCred,
	})
	if err != nil {
		return err
	}

	//ez a role mondja majd meg, hogy mikhez lesz jogunk a tovabbiakban, szoval adnunk kell magunknak jogot eks inditashoz, stb
	//ehhez is vagy role kell vagy aws access/secret key.

	roleName := e.generateIAMRoleNameForCluster()
	eksStackName := e.generateStackNameForCluster()
	sshKeyName := e.generateSSHKeyNameForCluster()

	creationContext := action.NewEksClusterCreationContext(
		session,
		e.modelCluster.Name,
		sshKeyName,
	)

	sshSecret, err := e.GetSshSecretWithValidation()
	if err != nil {
		return err
	}

	actions := []utils.Action{
		action.NewEnsureIAMRoleAction(creationContext, roleName),
		action.NewCreateVPCAction(creationContext, eksStackName),
		action.NewUploadSSHKeyAction(creationContext, sshSecret),
		action.NewGenerateVPCConfigRequestAction(creationContext, eksStackName),
		action.NewCreateEksClusterAction(creationContext, e.modelCluster.Eks.Version),
		action.NewLoadEksSettingsAction(creationContext),
	}

	for _, nodePool := range e.modelCluster.Eks.NodePools {
		nodePoolStackName := e.generateNodePoolStackName(nodePool.Name)
		createNodePoolAction := action.NewCreateNodePoolStackAction(creationContext, nodePoolStackName, nodePool)
		actions = append(actions, createNodePoolAction)
	}

	_, err = utils.NewActionExecutor(log).ExecuteActions(actions, nil, true)
	if err != nil {
		log.Errorln("EKS cluster create error:", err.Error())
		return err
	}

	e.APIEndpoint = *creationContext.APIEndpoint
	e.CertificateAuthorityData, err = base64.StdEncoding.DecodeString(aws.StringValue(creationContext.CertificateAuthorityData))

	if err != nil {
		log.Errorf("Decoding base64 format EKS K8S certificate authority data failed: %s", err.Error())
		return err
	}

	// Create the aws-auth ConfigMap for letting other nodes join
	// See: https://docs.aws.amazon.com/eks/latest/userguide/add-user-role.html
	kubeConfig, err := e.DownloadK8sConfig()
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

	mapRoles := ""
	for _, roleArn := range creationContext.NodeInstanceRoles {
		mapRoles += fmt.Sprintf(authConfigMapTemplate, roleArn)
	}
	awsAuthConfigMap := v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: "aws-auth"},
		Data:       map[string]string{"mapRoles": mapRoles},
	}

	_, err = kubeClient.CoreV1().ConfigMaps("kube-system").Create(&awsAuthConfigMap)
	if err != nil {
		return err
	}

	log.Infoln("EKS cluster created:", e.modelCluster.Name)

	return nil
}

func (e *EKSCluster) generateSSHKeyNameForCluster() string {
	sshKeyName := "ssh-key-for-cluster-" + e.modelCluster.Name
	return sshKeyName
}

func (e *EKSCluster) generateNodePoolStackName(nodePoolName string) string {
	return e.modelCluster.Name + "-pipeline-eks-nodepool-" + nodePoolName
}

func (e *EKSCluster) generateStackNameForCluster() string {
	eksStackName := e.modelCluster.Name + "-pipeline-eks"
	return eksStackName
}

func (e *EKSCluster) generateIAMRoleNameForCluster() string {
	roleName := (e.modelCluster.Name) + "-pipeline-eks"
	return roleName
}

// Persist saves the cluster model
func (e *EKSCluster) Persist(status, statusMessage string) error {
	log.Infof("Model before save: %v", e.modelCluster)
	return e.modelCluster.UpdateStatus(status, statusMessage)
}

// GetName returns the name of the cluster
func (e *EKSCluster) GetName() string {
	return e.modelCluster.Name
}

// GetType returns the cloud type of the cluster
func (e *EKSCluster) GetType() string {
	return e.modelCluster.Cloud
}

// DeleteCluster deletes cluster from google
func (e *EKSCluster) DeleteCluster() error {
	log.Info("Start delete EKS cluster")

	awsCred, err := e.createAWSCredentialsFromSecret()
	if err != nil {
		return err
	}

	session, err := session.NewSession(&aws.Config{
		Region:      aws.String(e.modelCluster.Location),
		Credentials: awsCred,
	})
	if err != nil {
		return err
	}

	deleteContext := action.NewEksClusterDeleteContext(
		session,
		e.modelCluster.Name,
	)
	actions := []utils.Action{
		action.NewDeleteClusterAction(deleteContext, e.modelCluster.Name),
		action.NewDeleteSSHKeyAction(deleteContext, e.generateSSHKeyNameForCluster()),
		action.NewDeleteStackAction(deleteContext, e.generateStackNameForCluster()),
		action.NewDeleteIAMRoleAction(deleteContext, e.generateIAMRoleNameForCluster()),
	}

	for _, nodePool := range e.modelCluster.Eks.NodePools {
		nodePoolStackName := e.generateNodePoolStackName(nodePool.Name)
		createStackAction := action.NewDeleteStackAction(deleteContext, nodePoolStackName)
		actions = append(actions, createStackAction)
	}

	_, err = utils.NewActionExecutor(logrus.New()).ExecuteActions(actions, nil, false)
	if err != nil {
		log.Errorln("EKS cluster delete error:", err.Error())
		return err
	}

	return nil
}

// UpdateCluster updates EKS cluster in cloud
func (e *EKSCluster) UpdateCluster(updateRequest *pkgCluster.UpdateClusterRequest, updatedBy uint) error {
	// TODO missing implementation
	log.Info("Start updating EKS cluster")
	return nil
}

// GenerateK8sConfig generates kube config for this EKS cluster which authenticates through the aws-iam-authenticator,
// you have to install with: go get github.com/kubernetes-sigs/aws-iam-authenticator/cmd/aws-iam-authenticator
func (e *EKSCluster) GenerateK8sConfig() *clientcmdapi.Config {
	return &clientcmdapi.Config{
		APIVersion: "v1",
		Clusters: []clientcmdapi.NamedCluster{
			{
				Name: e.modelCluster.Name,
				Cluster: clientcmdapi.Cluster{
					Server: e.APIEndpoint,
					CertificateAuthorityData: e.CertificateAuthorityData,
				},
			},
		},
		Contexts: []clientcmdapi.NamedContext{
			{
				Name: e.modelCluster.Name,
				Context: clientcmdapi.Context{
					AuthInfo: "eks",
					Cluster:  e.modelCluster.Name,
				},
			},
		},
		AuthInfos: []clientcmdapi.NamedAuthInfo{
			{
				Name: "eks",
				AuthInfo: clientcmdapi.AuthInfo{
					Exec: &clientcmdapi.ExecConfig{
						APIVersion: "client.authentication.k8s.io/v1alpha1",
						Command:    "aws-iam-authenticator",
						Args:       []string{"token", "-i", e.modelCluster.Name},
						// Env:        []clientcmdapi.ExecEnvVar{clientcmdapi.ExecEnvVar{Name: "AWS_PROFILE", Value: "your_aws_profile_name"}},
					},
				},
			},
		},
		Kind:           "Config",
		CurrentContext: e.modelCluster.Name,
	}
}

// DownloadK8sConfig generates and marshalls the kube config for this cluster.
func (e *EKSCluster) DownloadK8sConfig() ([]byte, error) {
	config := e.GenerateK8sConfig()
	return json.Marshal(config)
}

// GetStatus describes the status of this EKS cluster.
func (e *EKSCluster) GetStatus() (*pkgCluster.GetClusterStatusResponse, error) {

	nodePools := make(map[string]*pkgCluster.NodePoolStatus)
	for _, np := range e.modelCluster.Eks.NodePools {
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
		Status:        e.modelCluster.Status,
		StatusMessage: e.modelCluster.StatusMessage,
		Name:          e.modelCluster.Name,
		Location:      e.modelCluster.Location,
		Cloud:         e.modelCluster.Cloud,
		ResourceID:    e.modelCluster.ID,
		NodePools:     nodePools,
	}, nil
}

// GetID returns the DB ID of this cluster
func (e *EKSCluster) GetID() uint {
	return e.modelCluster.ID
}

// GetModel returns the DB model of this cluster
func (e *EKSCluster) GetModel() *model.ClusterModel {
	return e.modelCluster
}

// CheckEqualityToUpdate validates the update request
func (e *EKSCluster) CheckEqualityToUpdate(*pkgCluster.UpdateClusterRequest) error {
	return nil //TODO missing
}

// AddDefaultsToUpdate adds defaults to update request
func (e *EKSCluster) AddDefaultsToUpdate(*pkgCluster.UpdateClusterRequest) {
	//TODO missing
}

// DeleteFromDatabase deletes model from the database
func (e *EKSCluster) DeleteFromDatabase() error {
	err := e.modelCluster.Delete()
	if err != nil {
		return err
	}
	e.modelCluster = nil
	return nil
}

// ListNodeNames returns node names to label them
func (e *EKSCluster) ListNodeNames() (nodeNames pkgCommon.NodeNames, err error) {
	// TODO missing
	return pkgCommon.NodeNames{}, nil
}

// UpdateStatus updates cluster status in database
func (e *EKSCluster) UpdateStatus(status string, statusMessage string) error {
	return e.modelCluster.UpdateStatus(status, statusMessage)
}

// GetClusterDetails gets cluster details from cloud
func (e *EKSCluster) GetClusterDetails() (*pkgCluster.DetailsResponse, error) {
	log.Info("Start getting cluster details")

	return &pkgCluster.DetailsResponse{
		Name:     e.modelCluster.Name,
		Id:       e.modelCluster.ID,
		Endpoint: e.APIEndpoint,
	}, nil
}

// ValidateCreationFields validates all fields
func (e *EKSCluster) ValidateCreationFields(r *pkgCluster.CreateClusterRequest) error {
	//TODO validate location, node AMIs
	/*
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

	*/
	return nil
}

// GetSecretWithValidation returns secret from vault
func (e *EKSCluster) GetSecretWithValidation() (*secret.SecretItemResponse, error) {
	return e.CommonClusterBase.getSecret(e)
}

// GetSshSecretWithValidation returns ssh secret from vault
func (e *EKSCluster) GetSshSecretWithValidation() (*secret.SecretItemResponse, error) {
	return e.CommonClusterBase.getSshSecret(e)
}

// SaveConfigSecretId saves the config secret id in database
func (e *EKSCluster) SaveConfigSecretId(configSecretId string) error {
	return e.modelCluster.UpdateConfigSecret(configSecretId)
}

// GetConfigSecretId return config secret id
func (e *EKSCluster) GetConfigSecretId() string {
	return e.modelCluster.ConfigSecretId
}

// GetK8sConfig returns the Kubernetes config
func (e *EKSCluster) GetK8sConfig() ([]byte, error) {
	return e.CommonClusterBase.getConfig(e)
}

// RequiresSshPublicKey returns true as a public ssh key is needed for bootstrapping
// the cluster
func (e *EKSCluster) RequiresSshPublicKey() bool {
	return true
}

// ReloadFromDatabase load cluster from DB
func (e *EKSCluster) ReloadFromDatabase() error {
	return e.modelCluster.ReloadFromDatabase()
}
