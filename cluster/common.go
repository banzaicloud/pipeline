package cluster

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"syscall"

	"encoding/base64"
	bTypes "github.com/banzaicloud/banzai-types/components"
	"github.com/banzaicloud/banzai-types/constants"
	"github.com/banzaicloud/pipeline/config"
	pipConstants "github.com/banzaicloud/pipeline/constants"
	"github.com/banzaicloud/pipeline/model"
	"github.com/banzaicloud/pipeline/secret"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/terminal"
)

// TODO se who will win
var logger *logrus.Logger
var log *logrus.Entry

//CommonCluster interface for clusters
type CommonCluster interface {
	CreateCluster() error
	Persist(string, string) error
	DownloadK8sConfig() ([]byte, error)
	GetName() string
	GetType() string
	GetStatus() (*bTypes.GetClusterStatusResponse, error)
	DeleteCluster() error
	UpdateCluster(*bTypes.UpdateClusterRequest) error
	GetID() uint
	GetSecretId() string
	GetSshSecretId() string
	SaveSshSecretId(string) error
	GetModel() *model.ClusterModel
	CheckEqualityToUpdate(*bTypes.UpdateClusterRequest) error
	AddDefaultsToUpdate(*bTypes.UpdateClusterRequest)
	GetAPIEndpoint() (string, error)
	DeleteFromDatabase() error
	GetOrganizationId() uint
	UpdateStatus(string, string) error
	GetClusterDetails() (*bTypes.ClusterDetailsResponse, error)
	ValidateCreationFields(r *bTypes.CreateClusterRequest) error
	GetSecretWithValidation() (*secret.SecretsItemResponse, error)
	GetSshSecretWithValidation() (*secret.SecretsItemResponse, error)
	SaveConfigSecretId(string) error
	GetConfigSecretId() string
	GetK8sConfig() ([]byte, error)
	RequiresSshPublicKey() bool
}

type CommonClusterBase struct {
	secret    *secret.SecretsItemResponse
	sshSecret *secret.SecretsItemResponse

	config []byte
}

// RequiresSshPublicKey returns true if an ssh public key is needed for the cluster for bootstrapping it.
// The default is false.
func (c *CommonClusterBase) RequiresSshPublicKey() bool {
	return false
}

func (c *CommonClusterBase) getSecret(cluster CommonCluster) (*secret.SecretsItemResponse, error) {
	if c.secret == nil {
		log.Info("Secret is nil.. load from vault")
		s, err := getSecret(cluster.GetOrganizationId(), cluster.GetSecretId())
		if err != nil {
			return nil, err
		}
		c.secret = s
	} else {
		log.Info("Secret is loaded before")
	}

	err := c.secret.ValidateSecretType(cluster.GetType())
	if err != nil {
		return nil, err
	}

	return c.secret, err
}

func (c *CommonClusterBase) getSshSecret(cluster CommonCluster) (*secret.SecretsItemResponse, error) {
	if c.sshSecret == nil {
		log.Info("Ssh secret is nil.. load from vault")
		s, err := getSecret(cluster.GetOrganizationId(), cluster.GetSshSecretId())
		if err != nil {
			log.Errorf("Get ssh key failed OrganizationID: %q, SshSecretID: %q  reason: %s", cluster.GetOrganizationId(), cluster.GetSshSecretId, err.Error())
			return nil, err
		}
		c.sshSecret = s
	} else {
		log.Info("Secret is loaded before")
	}

	err := c.sshSecret.ValidateSecretType(pipConstants.SshSecretType)
	if err != nil {
		return nil, err
	}

	return c.sshSecret, err
}

func (c *CommonClusterBase) getConfig(cluster CommonCluster) ([]byte, error) {
	if c.config == nil {
		log.Info("config is nil.. load from vault")
		configSecret, err := getSecret(cluster.GetOrganizationId(), cluster.GetConfigSecretId())
		if err != nil {
			return nil, err
		}

		configStr, err := base64.StdEncoding.DecodeString(configSecret.GetValue(pipConstants.K8SConfig))
		if err != nil {
			return nil, err
		}

		c.config = []byte(configStr)
	} else {
		log.Info("Config is loaded before")
	}
	return c.config, nil
}

func getSecret(organizationId uint, secretId string) (*secret.SecretsItemResponse, error) {
	org := strconv.FormatUint(uint64(organizationId), 10)
	return secret.Store.Get(org, secretId)
}

//GetCommonClusterFromModel extracts CommonCluster from a ClusterModel
func GetCommonClusterFromModel(modelCluster *model.ClusterModel) (CommonCluster, error) {

	database := model.GetDB()
	log := logger.WithFields(logrus.Fields{"tag": "GetCommonClusterFromModel"})

	cloudType := modelCluster.Cloud
	switch cloudType {
	case constants.Amazon:
		//Create Amazon struct
		awsCluster, err := CreateAWSClusterFromModel(modelCluster)
		if err != nil {
			return nil, err
		}

		log.Debug("Load Amazon props from database")
		database.Where(model.AmazonClusterModel{ClusterModelId: awsCluster.modelCluster.ID}).First(&awsCluster.modelCluster.Amazon)
		database.Model(&awsCluster.modelCluster.Amazon).Related(&awsCluster.modelCluster.Amazon.NodePools, "NodePools")

		return awsCluster, nil

	case constants.Azure:
		// Create Azure struct
		aksCluster, err := CreateAKSClusterFromModel(modelCluster)
		if err != nil {
			return nil, err
		}

		log.Info("Load Azure props from database")
		database.Where(model.AzureClusterModel{ClusterModelId: aksCluster.modelCluster.ID}).First(&aksCluster.modelCluster.Azure)
		database.Model(&aksCluster.modelCluster.Azure).Related(&aksCluster.modelCluster.Azure.NodePools, "NodePools")

		return aksCluster, nil

	case constants.Google:
		// Create Google struct
		gkeCluster, err := CreateGKEClusterFromModel(modelCluster)
		if err != nil {
			return nil, err
		}

		log.Info("Load Google props from database")
		database.Where(model.GoogleClusterModel{ClusterModelId: gkeCluster.modelCluster.ID}).First(&gkeCluster.modelCluster.Google)
		database.Model(&gkeCluster.modelCluster.Google).Related(&gkeCluster.modelCluster.Google.NodePools, "NodePools")

		return gkeCluster, nil

	case constants.Dummy:
		dummyCluster, err := CreateDummyClusterFromModel(modelCluster)
		if err != nil {
			return nil, err
		}
		log.Info("Load Dummy props from database")
		database.Where(model.DummyClusterModel{ClusterModelId: dummyCluster.modelCluster.ID}).First(&dummyCluster.modelCluster.Dummy)

		return dummyCluster, nil

	case constants.Kubernetes:
		// Create Kubernetes struct
		kubernetesCluster, err := CreateKubernetesClusterFromModel(modelCluster)
		if err != nil {
			return nil, err
		}

		log.Info("Load Kubernetes props from database")
		database.Where(model.KubernetesClusterModel{ClusterModelId: kubernetesCluster.modelCluster.ID}).First(&kubernetesCluster.modelCluster.Kubernetes)

		return kubernetesCluster, nil
	}

	return nil, constants.ErrorNotSupportedCloudType
}

//CreateCommonClusterFromRequest creates a CommonCluster from a request
func CreateCommonClusterFromRequest(createClusterRequest *bTypes.CreateClusterRequest, orgId uint) (CommonCluster, error) {

	// validate request
	if err := createClusterRequest.Validate(); err != nil {
		return nil, err
	}

	cloudType := createClusterRequest.Cloud
	switch cloudType {
	case constants.Amazon:
		//Create Amazon struct
		awsCluster, err := CreateAWSClusterFromRequest(createClusterRequest, orgId)
		if err != nil {
			return nil, err
		}
		return awsCluster, nil

	case constants.Azure:
		// Create Azure struct
		aksCluster, err := CreateAKSClusterFromRequest(createClusterRequest, orgId)
		if err != nil {
			return nil, err
		}
		return aksCluster, nil

	case constants.Google:
		// Create Google struct
		gkeCluster, err := CreateGKEClusterFromRequest(createClusterRequest, orgId)
		if err != nil {
			return nil, err
		}
		return gkeCluster, nil

	case constants.Dummy:
		// Create Dummy struct
		dummy, err := CreateDummyClusterFromRequest(createClusterRequest, orgId)
		if err != nil {
			return nil, err
		}

		return dummy, nil

	case constants.Kubernetes:
		// Create Kubernetes struct
		kubeCluster, err := CreateKubernetesClusterFromRequest(createClusterRequest, orgId)
		if err != nil {
			return nil, err
		}
		return kubeCluster, nil
	}

	return nil, constants.ErrorNotSupportedCloudType
}

func home() string {
	home := os.Getenv("HOME")
	return home
}

func expand(path string) string {
	if strings.Contains(path, "~") {
		return strings.Replace(path, "~", home(), 1)
	}
	return path
}

func getSigner(pemBytes []byte) (ssh.Signer, error) {
	signerwithoutpassphrase, err := ssh.ParsePrivateKey(pemBytes)
	if err != nil {
		logger.Debug(err.Error())
		fmt.Print("SSH Key Passphrase [none]: ")
		passPhrase, err := terminal.ReadPassword(int(syscall.Stdin))
		fmt.Println("")
		if err != nil {
			return nil, err
		}
		signerwithpassphrase, err := ssh.ParsePrivateKeyWithPassphrase(pemBytes, passPhrase)
		if err != nil {
			return nil, err
		}

		return signerwithpassphrase, err
	}

	return signerwithoutpassphrase, err
}

// CleanStateStore deletes state store folder by cluster name
func CleanStateStore(path string) error {
	if len(path) != 0 {
		stateStorePath := config.GetStateStorePath(path)
		return os.RemoveAll(stateStorePath)
	}
	return constants.ErrStateStorePathEmpty
}
