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
	//ModifyCluster(*model.ClusterModel)
	//GetKubernetesConf()
	//GetKubernetesEndpoint()
}

func GetCommonClusterFromModel(modelCluster *model.ClusterModel) (CommonCluster, error) {
	cloudType := modelCluster.Cloud
	switch cloudType {
	case constants.Amazon:
		//Create Amazon struct
		awsCluster, err := CreateAWSClusterFromModel(modelCluster)
		if err != nil {
			return nil, err
		}
		return awsCluster, nil

	case constants.Azure:
		return nil, nil

	case constants.Google:
		return nil, nil
	}
	return nil, errors.New("Cluster type not found")
}

func CreateCommonClusterFromRequest(createClusterRequest *bTypes.CreateClusterRequest) (CommonCluster, error) {
	cloudType := createClusterRequest.Cloud
	switch cloudType {
	case constants.Amazon:
		_, errString := createClusterRequest.Properties.CreateClusterAmazon.Validate()
		if errString != "" {
			return nil, errors.New(errString)
		}
		//Create Amazon struct
		awsCluster, err := CreateAWSClusterFromRequest(createClusterRequest)
		if err != nil {
			return nil, err
		}
		return awsCluster, nil
	case constants.Azure:
		return nil, nil
	case constants.Google:
		return nil, nil
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
