package cluster

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"syscall"

	bTypes "github.com/banzaicloud/banzai-types/components"
	"github.com/banzaicloud/banzai-types/constants"
	"github.com/banzaicloud/pipeline/model"
	"github.com/banzaicloud/pipeline/secret"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/terminal"
)

// TODO se who will win
var logger *logrus.Logger
var log *logrus.Entry

//CommonCluster interface for clusters
type CommonCluster interface {
	CreateCluster() error
	Persist() error
	GetK8sConfig() (*[]byte, error)
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
}

func GetSecret(cluster CommonCluster) (*secret.SecretsItemResponse, error) {
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

		return awsCluster, nil

	case constants.Azure:
		// Create Azure struct
		aksCluster, err := CreateAKSClusterFromModel(modelCluster)
		if err != nil {
			return nil, err
		}

		log.Info("Load Azure props from database")
		database.Where(model.AzureClusterModel{ClusterModelId: aksCluster.modelCluster.ID}).First(&aksCluster.modelCluster.Azure)

		return aksCluster, nil

	case constants.Google:
		// Create Google struct
		gkeCluster, err := CreateGKEClusterFromModel(modelCluster)
		if err != nil {
			return nil, err
		}

		log.Info("Load Google props from database")
		database.Where(model.GoogleClusterModel{ClusterModelId: gkeCluster.modelCluster.ID}).First(&gkeCluster.modelCluster.Google)

		return gkeCluster, nil
	}
	return nil, constants.ErrorNotSupportedCloudType
}

//CreateCommonClusterFromRequest creates a CommonCluster from a request
func CreateCommonClusterFromRequest(createClusterRequest *bTypes.CreateClusterRequest, orgId uint) (CommonCluster, error) {
	cloudType := createClusterRequest.Cloud
	switch cloudType {
	case constants.Amazon:
		err := createClusterRequest.Properties.CreateClusterAmazon.Validate()
		if err != nil {
			return nil, err
		}
		//Create Amazon struct
		awsCluster, err := CreateAWSClusterFromRequest(createClusterRequest, orgId)
		if err != nil {
			return nil, err
		}
		return awsCluster, nil
	case constants.Azure:

		err := createClusterRequest.Properties.CreateClusterAzure.Validate()
		if err != nil {
			return nil, err
		}

		// Create Azure struct
		aksCluster, err := CreateAKSClusterFromRequest(createClusterRequest, orgId)
		if err != nil {
			return nil, err
		}
		return aksCluster, nil

	case constants.Google:
		if err := createClusterRequest.Properties.CreateClusterGoogle.Validate(); err != nil {
			return nil, err
		}

		// Create Google struct
		gkeCluster, err := CreateGKEClusterFromRequest(createClusterRequest, orgId)
		if err != nil {
			return nil, err
		}

		return gkeCluster, nil
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

func CleanStateStore(cluster CommonCluster) error {
	stateStorePath := fmt.Sprintf("%s/%s", viper.GetString("statestore.path"), cluster.GetName())
	return os.RemoveAll(stateStorePath)
}
