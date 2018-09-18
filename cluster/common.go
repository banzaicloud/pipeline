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
	"encoding/base64"
	"fmt"
	"os"
	"syscall"
	"time"

	"github.com/banzaicloud/pipeline/auth"
	"github.com/banzaicloud/pipeline/config"
	"github.com/banzaicloud/pipeline/internal/platform/database"
	"github.com/banzaicloud/pipeline/model"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	pkgCommon "github.com/banzaicloud/pipeline/pkg/common"
	pkgErrors "github.com/banzaicloud/pipeline/pkg/errors"
	modelOracle "github.com/banzaicloud/pipeline/pkg/providers/oracle/model"
	pkgSecret "github.com/banzaicloud/pipeline/pkg/secret"
	"github.com/banzaicloud/pipeline/secret"
	"github.com/banzaicloud/pipeline/utils"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/terminal"
)

// CommonCluster interface for clusters.
type CommonCluster interface {
	// Entity properties
	GetID() uint
	GetUID() string
	GetOrganizationId() uint
	GetCreatorID() uint
	GetName() string
	GetCloud() string
	GetDistribution() string
	GetLocation() string

	// Secrets
	GetSecretId() string
	GetSshSecretId() string
	SaveSshSecretId(string) error
	SaveConfigSecretId(string) error
	GetConfigSecretId() string
	GetSecretWithValidation() (*secret.SecretItemResponse, error)

	// Persistence
	Persist(string, string) error
	UpdateStatus(string, string) error
	DeleteFromDatabase() error

	// Cluster management
	CreateCluster() error
	ValidateCreationFields(r *pkgCluster.CreateClusterRequest) error
	UpdateCluster(*pkgCluster.UpdateClusterRequest, uint) error
	CheckEqualityToUpdate(*pkgCluster.UpdateClusterRequest) error
	AddDefaultsToUpdate(*pkgCluster.UpdateClusterRequest)
	DeleteCluster() error

	// Kubernetes
	DownloadK8sConfig() ([]byte, error)
	GetAPIEndpoint() (string, error)
	GetK8sConfig() ([]byte, error)
	RequiresSshPublicKey() bool
	RbacEnabled() bool
	NeedAdminRights() bool
	GetKubernetesUserName() (string, error)

	// Cluster info
	GetStatus() (*pkgCluster.GetClusterStatusResponse, error)
	GetClusterDetails() (*pkgCluster.DetailsResponse, error)
	ListNodeNames() (pkgCommon.NodeNames, error)
}

// CommonClusterBase holds the fields that is common to all cluster types
// also provides default implementation for common interface methods.
type CommonClusterBase struct {
	secret    *secret.SecretItemResponse
	sshSecret *secret.SecretItemResponse

	config []byte
}

// RequiresSshPublicKey returns true if an ssh public key is needed for the cluster for bootstrapping it.
// The default is false.
func (c *CommonClusterBase) RequiresSshPublicKey() bool {
	return false
}

func (c *CommonClusterBase) getSecret(cluster CommonCluster) (*secret.SecretItemResponse, error) {
	if c.secret == nil {
		log.Debug("Secret is nil.. load from vault")
		s, err := getSecret(cluster.GetOrganizationId(), cluster.GetSecretId())
		if err != nil {
			return nil, err
		}
		c.secret = s
	}

	err := c.secret.ValidateSecretType(cluster.GetCloud())
	if err != nil {
		return nil, err
	}

	return c.secret, err
}

func (c *CommonClusterBase) getSshSecret(cluster CommonCluster) (*secret.SecretItemResponse, error) {
	if c.sshSecret == nil {
		log.Debug("SSH secret is nil.. load from vault")
		s, err := getSecret(cluster.GetOrganizationId(), cluster.GetSshSecretId())
		if err != nil {
			return nil, err
		}
		c.sshSecret = s

		err = c.sshSecret.ValidateSecretType(pkgSecret.SSHSecretType)
		if err != nil {
			return nil, err
		}
	}

	return c.sshSecret, nil
}

func (c *CommonClusterBase) getConfig(cluster CommonCluster) ([]byte, error) {
	if c.config == nil {
		log.Debug("k8s config is nil.. load from vault")
		var loadedConfig []byte
		configSecret, err := getSecret(cluster.GetOrganizationId(), cluster.GetConfigSecretId())
		if err != nil {
			return nil, err
		}
		configStr, err := base64.StdEncoding.DecodeString(configSecret.GetValue(pkgSecret.K8SConfig))
		if err != nil {
			return nil, err
		}
		loadedConfig = []byte(configStr)

		c.config = loadedConfig
	}
	return c.config, nil
}

// StoreKubernetesConfig stores the given K8S config in vault
func StoreKubernetesConfig(cluster CommonCluster, config []byte) error {

	encodedConfig := utils.EncodeStringToBase64(string(config))

	organizationID := cluster.GetOrganizationId()
	clusterUidTag := fmt.Sprintf("clusterUID:%s", cluster.GetUID())

	createSecretRequest := secret.CreateSecretRequest{
		Name: fmt.Sprintf("cluster-%d-config", cluster.GetID()),
		Type: pkgSecret.K8SConfig,
		Values: map[string]string{
			pkgSecret.K8SConfig: encodedConfig,
		},
		Tags: []string{
			pkgSecret.TagKubeConfig,
			pkgSecret.TagBanzaiReadonly,
			clusterUidTag,
		},
	}

	secretID := secret.GenerateSecretID(&createSecretRequest)

	// Try to get the secret version first
	if configSecret, err := getSecret(organizationID, secretID); err != nil && err != secret.ErrSecretNotExists {
		return err
	} else if configSecret != nil {
		createSecretRequest.Version = &(configSecret.Version)
	}

	err := secret.Store.Update(organizationID, secretID, &createSecretRequest)
	if err != nil {
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

func getSecret(organizationId uint, secretId string) (*secret.SecretItemResponse, error) {
	return secret.Store.Get(organizationId, secretId)
}

// GetCommonClusterFromModel extracts CommonCluster from a ClusterModel
func GetCommonClusterFromModel(modelCluster *model.ClusterModel) (CommonCluster, error) {

	db := config.DB()

	cloudType := modelCluster.Cloud
	switch cloudType {
	case pkgCluster.Alibaba:
		//Create Alibaba struct
		alibabaCluster, err := CreateACSKClusterFromModel(modelCluster)
		if err != nil {
			return nil, err
		}

		log.Debug("Load Alibaba props from database")
		err = db.Where(model.ACSKClusterModel{ClusterModelId: alibabaCluster.modelCluster.ID}).First(&alibabaCluster.modelCluster.ACSK).Error
		if err != nil {
			return nil, err
		}

		err = db.Model(&alibabaCluster.modelCluster.ACSK).Related(&alibabaCluster.modelCluster.ACSK.NodePools, "NodePools").Error
		if err != nil {
			return nil, err
		}

		return alibabaCluster, nil

	case pkgCluster.Amazon:

		var c int
		err := db.Model(&model.EC2ClusterModel{}).Where(&model.EC2ClusterModel{ClusterModelId: modelCluster.ID}).Count(&c).Error
		if err != nil {
			return nil, err
		}

		if c > 0 {
			//Create Amazon struct
			awsCluster, err := CreateEC2ClusterFromModel(modelCluster)
			if err != nil {
				return nil, err
			}

			log.Debug("Load Amazon props from database")
			err = db.Where(model.EC2ClusterModel{ClusterModelId: awsCluster.modelCluster.ID}).First(&awsCluster.modelCluster.EC2).Error
			if err != nil {
				return nil, err
			}
			err = db.Model(&awsCluster.modelCluster.EC2).Related(&awsCluster.modelCluster.EC2.NodePools, "NodePools").Error

			return awsCluster, err
		}

		//Create Amazon EKS struct
		eksCluster, err := CreateEKSClusterFromModel(modelCluster)
		if err != nil {
			return nil, err
		}

		log.Debug("Load EKS props from database")
		err = db.Where(model.EKSClusterModel{ClusterModelId: eksCluster.modelCluster.ID}).First(&eksCluster.modelCluster.EKS).Error
		if err != nil {
			return nil, err
		}
		err = db.Model(&eksCluster.modelCluster.EKS).Related(&eksCluster.modelCluster.EKS.NodePools, "NodePools").Error

		return eksCluster, err

	case pkgCluster.Azure:
		// Create Azure struct
		aksCluster, err := CreateAKSClusterFromModel(modelCluster)
		if err != nil {
			return nil, err
		}

		log.Info("Load Azure props from database")
		err = db.Where(model.AKSClusterModel{ClusterModelId: aksCluster.modelCluster.ID}).First(&aksCluster.modelCluster.AKS).Error
		if err != nil {
			return nil, err
		}
		err = db.Model(&aksCluster.modelCluster.AKS).Related(&aksCluster.modelCluster.AKS.NodePools, "NodePools").Error

		return aksCluster, err

	case pkgCluster.Google:
		// Create Google struct
		gkeCluster, err := CreateGKEClusterFromModel(modelCluster)
		if err != nil {
			return nil, err
		}

		return gkeCluster, err

	case pkgCluster.Dummy:
		dummyCluster, err := CreateDummyClusterFromModel(modelCluster)
		if err != nil {
			return nil, err
		}

		log.Info("Load Dummy props from database")
		err = db.Where(model.DummyClusterModel{ClusterModelId: dummyCluster.modelCluster.ID}).First(&dummyCluster.modelCluster.Dummy).Error

		return dummyCluster, err

	case pkgCluster.Kubernetes:
		// Create Kubernetes struct
		kubernetesCluster, err := CreateKubernetesClusterFromModel(modelCluster)
		if err != nil {
			return nil, err
		}

		log.Info("Load Kubernetes props from database")
		err = db.Where(model.KubernetesClusterModel{ClusterModelId: kubernetesCluster.modelCluster.ID}).First(&kubernetesCluster.modelCluster.Kubernetes).Error
		if database.IsRecordNotFoundError(err) {
			// metadata not set so there's no properties in DB
			log.Warnf(err.Error())
			err = nil
		}

		return kubernetesCluster, err

	case pkgCluster.Oracle:
		// Create Oracle struct
		okeCluster, err := CreateOKEClusterFromModel(modelCluster)
		if err != nil {
			return nil, err
		}

		log.Info("Load Oracle props from database")
		err = db.Where(modelOracle.Cluster{ClusterModelID: okeCluster.modelCluster.ID}).Preload("NodePools.Subnets").Preload("NodePools.Labels").First(&okeCluster.modelCluster.OKE).Error

		return okeCluster, err
	}

	return nil, pkgErrors.ErrorNotSupportedCloudType
}

//CreateCommonClusterFromRequest creates a CommonCluster from a request
func CreateCommonClusterFromRequest(createClusterRequest *pkgCluster.CreateClusterRequest, orgId, userId uint) (CommonCluster, error) {

	if err := createClusterRequest.AddDefaults(); err != nil {
		return nil, err
	}

	// validate request
	if err := createClusterRequest.Validate(); err != nil {
		return nil, err
	}

	cloudType := createClusterRequest.Cloud
	switch cloudType {
	case pkgCluster.Alibaba:
		//Create Alibaba struct
		alibabaCluster, err := CreateACSKClusterFromRequest(createClusterRequest, orgId, userId)
		if err != nil {
			return nil, err
		}
		return alibabaCluster, nil

	case pkgCluster.Amazon:
		if createClusterRequest.Properties.CreateClusterEC2 != nil {
			//Create EC2 struct
			ec2Cluster, err := CreateEC2ClusterFromRequest(createClusterRequest, orgId, userId)
			if err != nil {
				return nil, err
			}
			return ec2Cluster, nil
		}

		//Create EKS struct
		eksCluster, err := CreateEKSClusterFromRequest(createClusterRequest, orgId, userId)
		if err != nil {
			return nil, err
		}
		return eksCluster, nil

	case pkgCluster.Azure:
		// Create AKS struct
		aksCluster, err := CreateAKSClusterFromRequest(createClusterRequest, orgId, userId)
		if err != nil {
			return nil, err
		}
		return aksCluster, nil

	case pkgCluster.Google:
		// Create GKE struct
		gkeCluster, err := CreateGKEClusterFromRequest(createClusterRequest, orgId, userId)
		if err != nil {
			return nil, err
		}
		return gkeCluster, nil

	case pkgCluster.Dummy:
		// Create Dummy struct
		dummy, err := CreateDummyClusterFromRequest(createClusterRequest, orgId, userId)
		if err != nil {
			return nil, err
		}

		return dummy, nil

	case pkgCluster.Kubernetes:
		// Create Kubernetes struct
		kubeCluster, err := CreateKubernetesClusterFromRequest(createClusterRequest, orgId, userId)
		if err != nil {
			return nil, err
		}
		return kubeCluster, nil

	case pkgCluster.Oracle:
		// Create OKE struct
		okeCluster, err := CreateOKEClusterFromRequest(createClusterRequest, orgId, userId)
		if err != nil {
			return nil, err
		}
		return okeCluster, nil

	}

	return nil, pkgErrors.ErrorNotSupportedCloudType
}

func getSigner(pemBytes []byte) (ssh.Signer, error) {
	signerwithoutpassphrase, err := ssh.ParsePrivateKey(pemBytes)
	if err != nil {
		log.Debug(err.Error())
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

// GetUserIdAndName returns userId and userName from DB
func GetUserIdAndName(modelCluster *model.ClusterModel) (userId uint, userName string) {
	userId = modelCluster.CreatedBy
	userName = auth.GetUserNickNameById(userId)
	return
}

// NewCreatorBaseFields creates a new CreatorBaseFields instance from createdAt and createdBy
func NewCreatorBaseFields(createdAt time.Time, createdBy uint) *pkgCommon.CreatorBaseFields {

	var userName string
	if createdBy != 0 {
		userName = auth.GetUserNickNameById(createdBy)
	}

	return &pkgCommon.CreatorBaseFields{
		CreatedAt:   utils.ConvertSecondsToTime(createdAt),
		CreatorName: userName,
		CreatorId:   createdBy,
	}
}
