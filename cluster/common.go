package cluster

import (
	"fmt"
	bTypes "github.com/banzaicloud/banzai-types/components"
	"github.com/banzaicloud/banzai-types/constants"
	"github.com/banzaicloud/pipeline/model"
	"github.com/go-errors/errors"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/terminal"
	"os"
	"strings"
	"syscall"
	"k8s.io/client-go/tools/clientcmd"
	"io/ioutil"
)

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
	GetModel() *model.ClusterModel
	CheckEqualityToUpdate(*bTypes.UpdateClusterRequest) error
	AddDefaultsToUpdate(*bTypes.UpdateClusterRequest)
	//ModifyCluster(*model.ClusterModel)
	//GetKubernetesConf()
	//GetKubernetesEndpoint()
}

func GetCommonClusterFromModel(modelCluster *model.ClusterModel) (CommonCluster, error) {

	database := model.GetDB()

	cloudType := modelCluster.Cloud
	switch cloudType {
	case constants.Amazon:
		//Create Amazon struct
		awsCluster, err := CreateAWSClusterFromModel(modelCluster)
		if err != nil {
			return nil, err
		}

		log.Info("Load Amazon props from database")
		database.Where(model.AzureClusterModel{ClusterModelId: awsCluster.modelCluster.ID}).First(&awsCluster.modelCluster.Amazon)

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
		// Create Azure struct
		gkeCluster, err := CreateGKEClusterFromModel(modelCluster)
		if err != nil {
			return nil, err
		}

		log.Info("Load Google props from database")
		database.Where(model.AzureClusterModel{ClusterModelId: gkeCluster.modelCluster.ID}).First(&gkeCluster.modelCluster.Google)

		return gkeCluster, nil
	}
	return nil, errors.New("Cluster type not found")
}

func CreateCommonClusterFromRequest(createClusterRequest *bTypes.CreateClusterRequest) (CommonCluster, error) {
	cloudType := createClusterRequest.Cloud
	switch cloudType {
	case constants.Amazon:
		isOk, errString := createClusterRequest.Properties.CreateClusterAmazon.Validate()
		if !isOk {
			return nil, errors.New(errString)
		}
		//Create Amazon struct
		awsCluster, err := CreateAWSClusterFromRequest(createClusterRequest)
		if err != nil {
			return nil, err
		}
		return awsCluster, nil
	case constants.Azure:

		isOk, errString := createClusterRequest.Properties.CreateClusterAzure.Validate()
		if !isOk {
			return nil, errors.New(errString)
		}

		// Create Azure struct
		aksCluster, err := CreateAKSClusterFromRequest(createClusterRequest)
		if err != nil {
			return nil, err
		}
		return aksCluster, nil

		return nil, nil
	case constants.Google:
		if isOk, errString := createClusterRequest.Properties.CreateClusterGoogle.Validate(); !isOk {
			return nil, errors.New(errString)
		}

		// Create Google struct
		gkeCluster, err := CreateGKEClusterFromRequest(createClusterRequest)
		if err != nil {
			return nil, err
		}

		return gkeCluster, nil
	}
	return nil, errors.New("Cluster type not found")
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

func writeKubernetesKeys(kubeConfigPath string, localPath string) {
	log.Infof("Starting to write kubernetes related certs/keys for: %s", kubeConfigPath)
	config, err := clientcmd.BuildConfigFromFlags("", kubeConfigPath)
	if err != nil {
		log.Errorf("Getting kubernetes config failed from: %s", kubeConfigPath)
		return
	}
	log.Infof("Getting kubernetes config succeeded!")
	ioutil.WriteFile(localPath+"/client-key-data.pem", config.KeyData, 0644)
	ioutil.WriteFile(localPath+"/client-certificate-data.pem", config.CertData, 0644)
	ioutil.WriteFile(localPath+"/certificate-authority-data.pem", config.CAData, 0644)
	log.Infof("Writing kubernetes related certs/keys succeeded for: %s", kubeConfigPath)
}

func getKubeConfigPath(path string) (string, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := os.Mkdir(path, 0777); err != nil {
			return "", err
		}
	}
	return fmt.Sprintf("%s/config", path), nil
}
