package cluster

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"syscall"

	bTypes "github.com/banzaicloud/banzai-types/components"
	"github.com/banzaicloud/banzai-types/constants"
	"github.com/banzaicloud/pipeline/config"
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
	Persist(string) error
	GetK8sConfig() ([]byte, error)
	GetName() string
	GetType() string
	GetStatus() (*bTypes.GetClusterStatusResponse, error)
	DeleteCluster() error
	UpdateCluster(*bTypes.UpdateClusterRequest) error
	GetID() uint
	GetSecretID() string
	GetModel() *model.ClusterModel
	CheckEqualityToUpdate(*bTypes.UpdateClusterRequest) error
	AddDefaultsToUpdate(*bTypes.UpdateClusterRequest)
	GetAPIEndpoint() (string, error)
	DeleteFromDatabase() error
	GetOrg() uint
	UpdateStatus(string) error
	GetClusterDetails() (*bTypes.ClusterDetailsResponse, error)
	ValidateCreationFields(r *bTypes.CreateClusterRequest) error
	GetSecretWithValidation() (*secret.SecretsItemResponse, error)
}

type commonSecret struct {
	secret *secret.SecretsItemResponse
}

func (cs *commonSecret) get(cluster CommonCluster) (*secret.SecretsItemResponse, error) {
	if cs.secret == nil {
		log.Info("secret is nil.. load from vault")
		s, err := getSecret(cluster)
		if err != nil {
			return nil, err
		}
		cs.secret = s
	} else {
		log.Info("Secret is loaded before")
	}

	err := cs.secret.ValidateSecretType(cluster.GetType())
	if err != nil {
		return nil, err
	}

	return cs.secret, err
}

func getSecret(cluster CommonCluster) (*secret.SecretsItemResponse, error) {
	org := strconv.FormatUint(uint64(cluster.GetOrg()), 10)
	return secret.Store.Get(org, cluster.GetSecretID())
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
		KubernetesCluster, err := CreateKubernetesClusterFromModel(modelCluster)
		if err != nil {
			return nil, err
		}

		log.Info("Load Kubernetes props from database")
		database.Where(model.KubernetesClusterModel{ClusterModelId: KubernetesCluster.modelCluster.ID}).First(&KubernetesCluster.modelCluster.Kubernetes)

		return KubernetesCluster, nil
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

func CleanStateStore(clusterName string) error {
	stateStorePath := config.GetStateStorePath(clusterName)
	return os.RemoveAll(stateStorePath)
}
