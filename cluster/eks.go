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
	"github.com/banzaicloud/pipeline/model/defaults"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	"github.com/banzaicloud/pipeline/pkg/cluster/eks/action"
	pkgCommon "github.com/banzaicloud/pipeline/pkg/common"
	"github.com/banzaicloud/pipeline/secret"
	"github.com/banzaicloud/pipeline/secret/verify"
	"github.com/banzaicloud/pipeline/utils"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
	"k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api/v1"
)

//CreateEKSClusterFromRequest creates ClusterModel struct from the request
func CreateEKSClusterFromRequest(request *pkgCluster.CreateClusterRequest, orgId uint) (*EKSCluster, error) {
	log.Debug("Create ClusterModel struct from the request")
	var cluster EKSCluster

	cluster.modelCluster = &model.ClusterModel{
		Name:           request.Name,
		Location:       request.Location,
		Cloud:          request.Cloud,
		OrganizationId: orgId,
		SecretId:       request.SecretId,

		Eks: model.AmazonEksClusterModel{
			NodeImageId:      request.Properties.CreateClusterEks.NodeImageId,
			NodeInstanceType: request.Properties.CreateClusterEks.NodeInstanceType,
			Version:          request.Properties.CreateClusterEks.Version,
			NodeMinCount:     request.Properties.CreateClusterEks.NodeMinCount,
			NodeMaxCount:     request.Properties.CreateClusterEks.NodeMaxCount,
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

func (e *EKSCluster) CreateCluster() error {
	//TODO jelenleg nem lehet kulsoleg megadott role ARN-t vagy sajat SSH secret azonositot megadni, ehelyett ezeket mind legyartja ez a fuggveny
	//TODO ha ezekre is szukseg van, akkor itt tobb atalakitas is kell, hogy pl a rollback(=undo) ne torolje ki a kulsoleg megadott role-t vagy ssh kulcsot

	log.Info("Start create cluster (Eks)")

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
	eksStackName := e.generateEksStackNameForCluster()
	eksWorkerStackName := e.generateEksWorkerStackNameForCluster()
	sshKeyName := e.generateSshKeyNameForCluster()

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
		action.NewCreateEksClusterAction(creationContext),
		action.NewLoadEksSettingsAction(creationContext),
		action.NewCreateWorkersAction(creationContext, eksWorkerStackName, e.modelCluster.Eks.NodeMinCount, e.modelCluster.Eks.NodeMaxCount, e.modelCluster.Eks.NodeInstanceType, e.modelCluster.Eks.NodeImageId),
		//action.NewDelayAction(10 * time.Minute), //pl ezzel lehet szimulalni ket lepes kozott egy kis varakozast, vagy varkakoztatni a kovetkezo lepest
		//action.NewRevertStepsAction(), //ez fixen hibat dob, igy minden elozo lepest megprobal visszavonni az executor
	}

	_, err = utils.NewActionExecutor(logrus.New()).ExecuteActions(actions, nil, true)
	if err != nil {
		fmt.Printf("EKS cluster create error: %v\n", err)
		return err
	}

	e.APIEndpoint = *creationContext.APIEndpoint
	e.CertificateAuthorityData, _ = base64.StdEncoding.DecodeString(*creationContext.CertificateAuthorityData)

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

	awsAuthConfigMap := v1.ConfigMap{}
	awsAuthConfigMap.Name = "aws-auth"
	awsAuthConfigMap.Data = map[string]string{"mapRoles": fmt.Sprintf(
		`- rolearn: %s
	username: system:node:{{EC2PrivateDNSName}}
	groups:
	- system:bootstrappers
	- system:nodes`, *creationContext.Role.Arn)}

	_, err = kubeClient.CoreV1().ConfigMaps("kube-system").Create(&awsAuthConfigMap)
	if err != nil {
		return err
	}

	createdCluster, err := e.GetCreatedClusterModel(creationContext)
	if err != nil {
		return err
	}
	fmt.Printf("EKS cluster created: %s\n", createdCluster.Name)

	return nil
}

func (e *EKSCluster) generateSshKeyNameForCluster() string {
	sshKeyName := "ssh-key-for-cluster-" + e.modelCluster.Name
	return sshKeyName
}

func (e *EKSCluster) generateEksWorkerStackNameForCluster() string {
	eksWorkerStackName := e.modelCluster.Name + "-pipeline-eks-worker-stack"
	return eksWorkerStackName
}

func (e *EKSCluster) generateEksStackNameForCluster() string {
	eksStackName := e.modelCluster.Name + "-pipeline-eks-stack"
	return eksStackName
}

func (e *EKSCluster) generateIAMRoleNameForCluster() string {
	roleName := (e.modelCluster.Name) + "-pipeline-iam-role"
	return roleName
}

//Persist save the cluster model
func (e *EKSCluster) Persist(status, statusMessage string) error {
	log.Infof("Model before save: %v", e.modelCluster)
	//TODO meg a region field erteke helytelen (ures) a db-ben
	return e.modelCluster.UpdateStatus(status, statusMessage)
}

//GetName returns the name of the cluster
func (e *EKSCluster) GetName() string {
	return e.modelCluster.Name
}

//GetType returns the cloud type of the cluster
func (e *EKSCluster) GetType() string {
	return e.modelCluster.Cloud
}

// DeleteCluster deletes cluster from google
func (e *EKSCluster) DeleteCluster() error {
	//TODO not tested

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
		action.NewDeleteStackAction(deleteContext, e.generateEksWorkerStackNameForCluster()),
		action.NewDeleteEksClusterAction(deleteContext, e.modelCluster.Name),
		action.NewDeleteSSHKeyAction(deleteContext, e.generateSshKeyNameForCluster()),
		action.NewDeleteStackAction(deleteContext, e.generateEksStackNameForCluster()),
		action.NewDeleteIAMRoleAction(deleteContext, e.generateIAMRoleNameForCluster()),
	}
	_, err = utils.NewActionExecutor(logrus.New()).ExecuteActions(actions, nil, false)
	if err != nil {
		fmt.Printf("EKS cluster delete error: %v\n", err)
		return err
	}

	return nil
}

// UpdateCluster updates EKS cluster in cloud
func (e *EKSCluster) UpdateCluster(updateRequest *pkgCluster.UpdateClusterRequest, updatedBy uint) error {
	//TODO missing implementation
	log.Info("Start updating cluster (eks)")
	return nil
}

func (e *EKSCluster) GenerateK8sConfig() *clientcmdapi.Config {
	//TODO install
	// go get github.com/kubernetes-sigs/aws-iam-authenticator/cmd/aws-iam-authenticator
	cfg := clientcmdapi.Config{
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
	return &cfg
}

func (e *EKSCluster) DownloadK8sConfig() ([]byte, error) { //YAML data bytes
	//user := "ec2-user"
	// defined here: https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/AccessingInstancesLinux.html

	config := e.GenerateK8sConfig()
	bytes, err := json.Marshal(config)
	return bytes, err
}

func (e *EKSCluster) GetStatus() (*pkgCluster.GetClusterStatusResponse, error) {

	nodePools := make(map[string]*pkgCluster.NodePoolStatus)
	//TODO missing implementation..itt a nodepoolrol semmi infot nem adunk egyelore vissza,
	//TODO mert egyelore nem poolokat, hanem egy stacket hozunk csak letre az ekshez.
	//TODO ezt a stacket lehetne egy pool elemkent jelenleg visszaadni, vagy atalakitani, hogy
	//TODO tobbet is tamogassunk

	//for _, np := range e.modelCluster.Eks.NodePools {
	//	if np != nil {
	//		nodePools[np.Name] = &pkgCluster.NodePoolStatus{
	//			Count:          np.NodeCount,
	//			InstanceType:   np.NodeInstanceType,
	//			ServiceAccount: np.ServiceAccount,
	//		}
	//	}
	//}

	return &pkgCluster.GetClusterStatusResponse{
		Status:        e.modelCluster.Status,
		StatusMessage: e.modelCluster.StatusMessage,
		Name:          e.modelCluster.Name,
		Location:      e.modelCluster.Location,
		Cloud:         e.modelCluster.Cloud,
		ResourceID:    e.modelCluster.ID,
		NodePools:     nodePools, //TODO not supported yet
	}, nil
}

func (e *EKSCluster) GetID() uint {
	return e.modelCluster.ID
}

func (e *EKSCluster) GetModel() *model.ClusterModel {
	return e.modelCluster
}

func (e *EKSCluster) CheckEqualityToUpdate(*pkgCluster.UpdateClusterRequest) error {
	return nil //TODO missing
}

func (e *EKSCluster) AddDefaultsToUpdate(*pkgCluster.UpdateClusterRequest) {
	//TODO missing
}

func (e *EKSCluster) DeleteFromDatabase() error {
	err := e.modelCluster.Delete()
	if err != nil {
		return err
	}
	e.modelCluster = nil
	return nil
}

func (e *EKSCluster) ListNodeNames() (nodeNames pkgCommon.NodeNames, err error) {
	panic("not implemented")
}

func (e *EKSCluster) UpdateStatus(status string, statusMessage string) error {
	return e.modelCluster.UpdateStatus(status, statusMessage)
}

func (e *EKSCluster) GetClusterDetails() (*pkgCluster.DetailsResponse, error) {
	log.Info("Start getting cluster details")
	//TODO not tested
	e.GetK8sConfig()
	e.GetAPIEndpoint()

	return &pkgCluster.DetailsResponse{
		Name: e.modelCluster.Name,
		Id:   e.modelCluster.ID,
	}, nil
}

func (e *EKSCluster) ValidateCreationFields(r *pkgCluster.CreateClusterRequest) error {
	//TODO itt hianyzik az osszes input validalas
	return nil
}

func (e *EKSCluster) GetSecretWithValidation() (*secret.SecretItemResponse, error) {
	return e.CommonClusterBase.getSecret(e)
}

func (e *EKSCluster) GetSshSecretWithValidation() (*secret.SecretItemResponse, error) {
	return e.CommonClusterBase.getSshSecret(e)
}

func (e *EKSCluster) SaveConfigSecretId(configSecretId string) error {
	return e.modelCluster.UpdateConfigSecret(configSecretId)
}

func (e *EKSCluster) GetConfigSecretId() string {
	return e.modelCluster.ConfigSecretId
}

func (e *EKSCluster) GetK8sConfig() ([]byte, error) {
	return e.CommonClusterBase.getConfig(e)
}

func (e *EKSCluster) RequiresSshPublicKey() bool {
	return true
}

func (e *EKSCluster) ReloadFromDatabase() error {
	return e.modelCluster.ReloadFromDatabase()
}
func (e *EKSCluster) GetCreatedClusterModel(context *action.EksClusterCreationContext) (*defaults.EksCluster, error) {

	configBytes, err := yaml.Marshal(e.GenerateK8sConfig())
	if err != nil {
		return nil, err
	}
	return &defaults.EksCluster{
		Name:             e.GetName(),
		Cloud:            pkgCluster.Eks,
		Region:           e.modelCluster.Location,
		K8SVersion:       e.modelCluster.Eks.Version,
		SSHPrivateKey:    []byte{},
		NodeImageId:      e.modelCluster.Eks.NodeImageId,
		NodeInstanceType: e.modelCluster.Eks.NodeInstanceType,
		ApiEndpoint:      e.APIEndpoint,
		MinNodes:         e.modelCluster.Eks.NodeMinCount,
		MaxNodes:         e.modelCluster.Eks.NodeMaxCount,
		KubeConfig:       string(configBytes),
	}, nil
}
