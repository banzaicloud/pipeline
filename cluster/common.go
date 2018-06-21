package cluster

import (
	"fmt"
	"os"
	"strings"
	"syscall"

	"encoding/base64"

	"github.com/banzaicloud/pipeline/config"
	"github.com/banzaicloud/pipeline/model"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	pkgErrors "github.com/banzaicloud/pipeline/pkg/errors"
	pkgSecret "github.com/banzaicloud/pipeline/pkg/secret"
	"github.com/banzaicloud/pipeline/secret"
	"github.com/banzaicloud/pipeline/utils"
	"github.com/kubicorn/kubicorn/pkg/logger"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/terminal"
)

var log *logrus.Logger

// Simple init for logging
func init() {
	log = config.Logger()
}

//CommonCluster interface for clusters
type CommonCluster interface {
	CreateCluster() error
	Persist(string, string) error
	DownloadK8sConfig() ([]byte, error)
	GetName() string
	GetType() string
	GetStatus() (*pkgCluster.GetClusterStatusResponse, error)
	DeleteCluster() error
	UpdateCluster(*pkgCluster.UpdateClusterRequest) error
	GetID() uint
	GetSecretId() string
	GetSshSecretId() string
	SaveSshSecretId(string) error
	GetModel() *model.ClusterModel
	CheckEqualityToUpdate(*pkgCluster.UpdateClusterRequest) error
	AddDefaultsToUpdate(*pkgCluster.UpdateClusterRequest)
	GetAPIEndpoint() (string, error)
	DeleteFromDatabase() error
	GetOrganizationId() uint
	UpdateStatus(string, string) error
	GetClusterDetails() (*pkgCluster.ClusterDetailsResponse, error)
	ValidateCreationFields(r *pkgCluster.CreateClusterRequest) error
	GetSecretWithValidation() (*secret.SecretsItemResponse, error)
	GetSshSecretWithValidation() (*secret.SecretsItemResponse, error)
	SaveConfigSecretId(string) error
	GetConfigSecretId() string
	GetK8sConfig() ([]byte, error)
	RequiresSshPublicKey() bool
	ReloadFromDatabase() error
}

// CommonClusterBase holds the fields that is common to all cluster types
// also provides default implementation for common interface methods.
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

	err := c.sshSecret.ValidateSecretType(pkgSecret.SSHSecretType)
	if err != nil {
		return nil, err
	}

	return c.sshSecret, err
}

func (c *CommonClusterBase) getConfig(cluster CommonCluster) ([]byte, error) {
	if c.config == nil {
		log.Info("config is nil.. load from vault")
		var loadedConfig []byte
		configSecret, err := getSecret(cluster.GetOrganizationId(), cluster.GetConfigSecretId())
		if err != nil {
			log.Warnf("Error during loading config from vault: %s", err.Error())
			log.Info("Re-download config from cloud")
			loadedConfig, err = cluster.DownloadK8sConfig()
			if err != nil {
				return nil, err
			}

			log.Info("Store K8S config in vault")
			if err := StoreKubernetesConfig(cluster, loadedConfig); err != nil {
				return nil, err
			}

		} else {
			configStr, err := base64.StdEncoding.DecodeString(configSecret.GetValue(pkgSecret.K8SConfig))
			if err != nil {
				return nil, err
			}
			loadedConfig = []byte(configStr)
		}

		c.config = loadedConfig
	} else {
		log.Info("Config is loaded before")
	}
	return c.config, nil
}

// StoreKubernetesConfig stores the given K8S config in vault
func StoreKubernetesConfig(cluster CommonCluster, config []byte) error {

	encodedConfig := utils.EncodeStringToBase64(string(config))

	organizationID := cluster.GetOrganizationId()
	createSecretRequest := secret.CreateSecretRequest{
		Name: fmt.Sprintf("%s-config", cluster.GetName()),
		Type: pkgSecret.K8SConfig,
		Values: map[string]string{
			pkgSecret.K8SConfig: encodedConfig,
		},
		Tags: []string{pkgSecret.TagKubeConfig},
	}

	secretID, err := secret.Store.Store(organizationID, &createSecretRequest)
	if err != nil {
		log.Errorf("Error during storing config: %s", err.Error())
		return err
	}

	log.Info("Kubeconfig stored in vault")

	log.Info("Update cluster model in DB with config secret id")
	if err := cluster.SaveConfigSecretId(secretID); err != nil {
		log.Errorf("Error during saving config secret id: %s", err.Error())
		return err
	}

	return nil
}

func getSecret(organizationId uint, secretId string) (*secret.SecretsItemResponse, error) {
	return secret.Store.Get(organizationId, secretId)
}

//GetCommonClusterFromModel extracts CommonCluster from a ClusterModel
func GetCommonClusterFromModel(modelCluster *model.ClusterModel) (CommonCluster, error) {

	database := model.GetDB()

	cloudType := modelCluster.Cloud
	switch cloudType {
	case pkgCluster.Amazon:
		//Create Amazon struct
		awsCluster, err := CreateAWSClusterFromModel(modelCluster)
		if err != nil {
			return nil, err
		}

		log.Debug("Load Amazon props from database")
		database.Where(model.AmazonClusterModel{ClusterModelId: awsCluster.modelCluster.ID}).First(&awsCluster.modelCluster.Amazon)
		database.Model(&awsCluster.modelCluster.Amazon).Related(&awsCluster.modelCluster.Amazon.NodePools, "NodePools")

		return awsCluster, nil

	case pkgCluster.Azure:
		// Create Azure struct
		aksCluster, err := CreateAKSClusterFromModel(modelCluster)
		if err != nil {
			return nil, err
		}

		log.Info("Load Azure props from database")
		database.Where(model.AzureClusterModel{ClusterModelId: aksCluster.modelCluster.ID}).First(&aksCluster.modelCluster.Azure)
		database.Model(&aksCluster.modelCluster.Azure).Related(&aksCluster.modelCluster.Azure.NodePools, "NodePools")

		return aksCluster, nil

	case pkgCluster.Google:
		// Create Google struct
		gkeCluster, err := CreateGKEClusterFromModel(modelCluster)
		if err != nil {
			return nil, err
		}

		log.Info("Load Google props from database")
		database.Where(model.GoogleClusterModel{ClusterModelId: gkeCluster.modelCluster.ID}).First(&gkeCluster.modelCluster.Google)
		database.Model(&gkeCluster.modelCluster.Google).Related(&gkeCluster.modelCluster.Google.NodePools, "NodePools")

		return gkeCluster, nil

	case pkgCluster.Dummy:
		dummyCluster, err := CreateDummyClusterFromModel(modelCluster)
		if err != nil {
			return nil, err
		}
		log.Info("Load Dummy props from database")
		database.Where(model.DummyClusterModel{ClusterModelId: dummyCluster.modelCluster.ID}).First(&dummyCluster.modelCluster.Dummy)

		return dummyCluster, nil

	case pkgCluster.Kubernetes:
		// Create Kubernetes struct
		kubernetesCluster, err := CreateKubernetesClusterFromModel(modelCluster)
		if err != nil {
			return nil, err
		}

		log.Info("Load Kubernetes props from database")
		database.Where(model.KubernetesClusterModel{ClusterModelId: kubernetesCluster.modelCluster.ID}).First(&kubernetesCluster.modelCluster.Kubernetes)

		return kubernetesCluster, nil
	}

	return nil, pkgErrors.ErrorNotSupportedCloudType
}

//CreateCommonClusterFromRequest creates a CommonCluster from a request
func CreateCommonClusterFromRequest(createClusterRequest *pkgCluster.CreateClusterRequest, orgId uint) (CommonCluster, error) {

	if err := createClusterRequest.AddDefaults(); err != nil {
		return nil, err
	}

	// validate request
	if err := createClusterRequest.Validate(); err != nil {
		return nil, err
	}

	cloudType := createClusterRequest.Cloud
	switch cloudType {
	case pkgCluster.Amazon:
		//Create Amazon struct
		awsCluster, err := CreateAWSClusterFromRequest(createClusterRequest, orgId)
		if err != nil {
			return nil, err
		}
		return awsCluster, nil

	case pkgCluster.Azure:
		// Create Azure struct
		aksCluster, err := CreateAKSClusterFromRequest(createClusterRequest, orgId)
		if err != nil {
			return nil, err
		}
		return aksCluster, nil

	case pkgCluster.Google:
		// Create Google struct
		gkeCluster, err := CreateGKEClusterFromRequest(createClusterRequest, orgId)
		if err != nil {
			return nil, err
		}
		return gkeCluster, nil

	case pkgCluster.Dummy:
		// Create Dummy struct
		dummy, err := CreateDummyClusterFromRequest(createClusterRequest, orgId)
		if err != nil {
			return nil, err
		}

		return dummy, nil

	case pkgCluster.Kubernetes:
		// Create Kubernetes struct
		kubeCluster, err := CreateKubernetesClusterFromRequest(createClusterRequest, orgId)
		if err != nil {
			return nil, err
		}
		return kubeCluster, nil
	}

	return nil, pkgErrors.ErrorNotSupportedCloudType
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
	return pkgErrors.ErrStateStorePathEmpty
}

// CleanHelmFolder deletes helm path
func CleanHelmFolder(organizationName string) error {
	helmPath := config.GetHelmPath(organizationName)
	return os.RemoveAll(helmPath)
}
