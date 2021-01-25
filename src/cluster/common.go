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
	"net"
	"net/url"
	"strings"
	"time"

	"emperror.dev/errors"
	"k8s.io/client-go/tools/clientcmd"
	logrusadapter "logur.dev/adapter/logrus"

	"github.com/banzaicloud/pipeline/internal/cluster/clusteradapter/clustermodel"
	"github.com/banzaicloud/pipeline/internal/common/commonadapter"
	"github.com/banzaicloud/pipeline/internal/global"
	"github.com/banzaicloud/pipeline/internal/platform/database"
	"github.com/banzaicloud/pipeline/internal/providers/azure/azureadapter"
	"github.com/banzaicloud/pipeline/internal/providers/azure/pke"
	"github.com/banzaicloud/pipeline/internal/providers/azure/pke/adapter"
	pkeAzureAdapter "github.com/banzaicloud/pipeline/internal/providers/azure/pke/driver/commoncluster"
	"github.com/banzaicloud/pipeline/internal/providers/kubernetes/kubernetesadapter"
	vsphereadapter "github.com/banzaicloud/pipeline/internal/providers/vsphere/pke/adapter"
	pkeVsphereAdapter "github.com/banzaicloud/pipeline/internal/providers/vsphere/pke/driver/commoncluster"
	"github.com/banzaicloud/pipeline/internal/secret/secrettype"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	pkgCommon "github.com/banzaicloud/pipeline/pkg/common"
	pkgErrors "github.com/banzaicloud/pipeline/pkg/errors"
	"github.com/banzaicloud/pipeline/src/auth"
	"github.com/banzaicloud/pipeline/src/model"
	"github.com/banzaicloud/pipeline/src/secret"
	"github.com/banzaicloud/pipeline/src/utils"
)

// CommonCluster interface for clusters.
type CommonCluster interface {
	// Entity properties
	GetID() uint
	GetUID() string
	GetOrganizationId() uint
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
	Persist() error
	DeleteFromDatabase() error

	// Cluster management
	CreateCluster() error
	ValidateCreationFields(r *pkgCluster.CreateClusterRequest) error
	UpdateCluster(*pkgCluster.UpdateClusterRequest, uint) error
	UpdateNodePools(*pkgCluster.UpdateNodePoolsRequest, uint) error
	CheckEqualityToUpdate(*pkgCluster.UpdateClusterRequest) error
	AddDefaultsToUpdate(*pkgCluster.UpdateClusterRequest)
	DeleteCluster() error
	GetScaleOptions() *pkgCluster.ScaleOptions
	SetScaleOptions(*pkgCluster.ScaleOptions)

	// Kubernetes
	GetAPIEndpoint() (string, error)
	GetK8sConfig() ([]byte, error)
	GetK8sUserConfig() ([]byte, error)
	RequiresSshPublicKey() bool
	RbacEnabled() bool

	// Cluster info
	GetStatus() (*pkgCluster.GetClusterStatusResponse, error)
	IsReady() (bool, error)
	NodePoolExists(nodePoolName string) bool

	SetStatus(status, statusMessage string) error
}

// CommonClusterBase holds the fields that is common to all cluster types
// also provides default implementation for common interface methods.
type CommonClusterBase struct {
	secret    *secret.SecretItemResponse
	sshSecret *secret.SecretItemResponse

	config []byte
}

// ErrConfigNotExists means that a cluster has no kubeconfig stored in vault (probably didn't successfully start yet)
var ErrConfigNotExists = fmt.Errorf("Kubernetes config is not available for the cluster")

// RequiresSshPublicKey returns true if an ssh public key is needed for the cluster for bootstrapping it.
// The default is false.
func (c *CommonClusterBase) RequiresSshPublicKey() bool {
	return false
}

func (c *CommonClusterBase) getSecret(cluster CommonCluster) (*secret.SecretItemResponse, error) {
	if c.secret == nil {
		s, err := getSecret(cluster.GetOrganizationId(), cluster.GetSecretId())
		if err != nil {
			return nil, err
		}
		c.secret = s
	}

	if err := secret.ValidateSecretType(c.secret, cluster.GetCloud()); err != nil {
		return nil, err
	}

	return c.secret, nil
}

func (c *CommonClusterBase) getSshSecret(cluster CommonCluster) (*secret.SecretItemResponse, error) {
	if c.sshSecret == nil {
		s, err := getSecret(cluster.GetOrganizationId(), cluster.GetSshSecretId())
		if err != nil {
			return nil, errors.WithDetails(err, "cluster", cluster.GetName())
		}
		c.sshSecret = s

		err = secret.ValidateSecretType(c.sshSecret, secrettype.SSHSecretType)
		if err != nil {
			return nil, errors.WithDetails(err, "cluster", cluster.GetName())
		}
	}

	return c.sshSecret, nil
}

func (c *CommonClusterBase) getConfig(cluster CommonCluster) ([]byte, error) {
	if c.config == nil {
		var loadedConfig []byte
		secretId := cluster.GetConfigSecretId()
		if secretId == "" {
			return nil, ErrConfigNotExists
		}
		configSecret, err := getSecret(cluster.GetOrganizationId(), secretId)
		if err != nil {
			return nil, errors.Wrap(err, "can't get config from Vault")
		}
		configStr, err := base64.StdEncoding.DecodeString(configSecret.Values[secrettype.K8SConfig])
		if err != nil {
			return nil, errors.Wrap(err, "can't decode Kubernetes config")
		}
		loadedConfig = configStr

		c.config = loadedConfig
	}
	return c.config, nil
}

// StoreKubernetesConfig stores the given K8S config in vault
func StoreKubernetesConfig(cluster CommonCluster, config []byte) error {
	var configYaml string

	if azurePKEClusterGetter, ok := cluster.(interface {
		GetPKEOnAzureCluster() pke.Cluster
	}); ok {
		azurePKECluster := azurePKEClusterGetter.GetPKEOnAzureCluster()

		var apiServerAccessPointAddress string
		if azurePKECluster.APIServerAccessPoints.Exists("public") {
			apiServerAccessPointAddress = azurePKECluster.AccessPoints.Get("public").Address
		} else if azurePKECluster.APIServerAccessPoints.Exists("private") {
			apiServerAccessPointAddress = azurePKECluster.AccessPoints.Get("private").Address
		} else {
			return errors.New("missing api server access point")
		}

		apiConfig, err := clientcmd.Load(config)
		if err != nil {
			return errors.WrapIf(err, "failed to load kubernetes API config")
		}

		ctx := apiConfig.Contexts[apiConfig.CurrentContext]
		cluster := apiConfig.Clusters[ctx.Cluster]

		apiServerUrl, err := url.Parse(cluster.Server)
		if err != nil {
			return errors.WrapIf(err, "couldn't parse API server url from config")
		}

		// replace host in api server url with the selected api server access point
		_, p, err := net.SplitHostPort(apiServerUrl.Host)
		if err != nil {
			return errors.WrapIf(err, "couldn't parse API server host and port from config")
		}

		apiServerUrl.Host = net.JoinHostPort(apiServerAccessPointAddress, p)
		cluster.Server = apiServerUrl.String()

		raw, err := clientcmd.Write(*apiConfig)
		if err != nil {
			return errors.WrapIf(err, "couldn't serialize API config yaml")
		}
		configYaml = string(raw)
	} else {
		configYaml = string(config)
	}

	encodedConfig := utils.EncodeStringToBase64(configYaml)

	organizationID := cluster.GetOrganizationId()
	clusterUidTag := fmt.Sprintf("clusterUID:%s", cluster.GetUID())

	createSecretRequest := secret.CreateSecretRequest{
		Name: fmt.Sprintf("cluster-%d-config", cluster.GetID()),
		Type: secrettype.Kubernetes,
		Values: map[string]string{
			secrettype.K8SConfig: encodedConfig,
		},
		Tags: []string{
			secret.TagKubeConfig,
			secret.TagBanzaiReadonly,
			clusterUidTag,
		},
	}

	secretID := secret.GenerateSecretID(&createSecretRequest)

	// Try to get the secret version first
	if _, err := getSecret(organizationID, secretID); err != nil && err != secret.ErrSecretNotExists {
		return err
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

func updateScaleOptions(scaleOptions *clustermodel.ScaleOptions, requestScaleOptions *pkgCluster.ScaleOptions) {
	if scaleOptions == nil || requestScaleOptions == nil {
		return
	}
	excludes := strings.Join(requestScaleOptions.Excludes, clustermodel.InstanceTypeSeparator)
	scaleOptions.Enabled = requestScaleOptions.Enabled
	scaleOptions.DesiredCpu = requestScaleOptions.DesiredCpu
	scaleOptions.DesiredMem = requestScaleOptions.DesiredMem
	scaleOptions.DesiredGpu = requestScaleOptions.DesiredGpu
	scaleOptions.OnDemandPct = requestScaleOptions.OnDemandPct
	scaleOptions.Excludes = excludes
	scaleOptions.KeepDesiredCapacity = requestScaleOptions.KeepDesiredCapacity
}

func getScaleOptionsFromModel(scaleOptions clustermodel.ScaleOptions) *pkgCluster.ScaleOptions {
	if scaleOptions.ID != 0 {
		scaleOpt := &pkgCluster.ScaleOptions{
			Enabled:             scaleOptions.Enabled,
			DesiredCpu:          scaleOptions.DesiredCpu,
			DesiredMem:          scaleOptions.DesiredMem,
			DesiredGpu:          scaleOptions.DesiredGpu,
			OnDemandPct:         scaleOptions.OnDemandPct,
			KeepDesiredCapacity: scaleOptions.KeepDesiredCapacity,
		}
		if len(scaleOptions.Excludes) > 0 {
			scaleOpt.Excludes = strings.Split(scaleOptions.Excludes, clustermodel.InstanceTypeSeparator)
		}
		return scaleOpt
	}
	return nil
}

// GetCommonClusterFromModel extracts CommonCluster from a ClusterModel
func GetCommonClusterFromModel(modelCluster *model.ClusterModel) (CommonCluster, error) {
	db := global.DB()

	if modelCluster.Distribution == pkgCluster.PKE {
		logger := commonadapter.NewLogger(logrusadapter.New(log))
		switch modelCluster.Cloud {
		case pkgCluster.Azure:
			return pkeAzureAdapter.MakeCommonClusterGetter(secret.Store, adapter.NewClusterStore(db, logger)).GetByID(modelCluster.ID)
		case pkgCluster.Vsphere:
			return pkeVsphereAdapter.MakeCommonClusterGetter(secret.Store, vsphereadapter.NewClusterStore(db)).GetByID(modelCluster.ID)
		default:
			return createCommonClusterWithDistributionFromModel(modelCluster)
		}
	}

	switch modelCluster.Cloud {
	case pkgCluster.Amazon:
		// Create Amazon EKS struct
		eksCluster, err := CreateEKSClusterFromModel(modelCluster)

		return eksCluster, err

	case pkgCluster.Azure:
		// Create Azure struct
		aksCluster := CreateAKSClusterFromModel(modelCluster)

		err := db.Preload("NodePools").
			Where(azureadapter.AKSClusterModel{ID: aksCluster.modelCluster.ID}).First(&aksCluster.modelCluster.AKS).Error

		return aksCluster, err

	case pkgCluster.Google:
		// Create Google struct
		gkeCluster, err := CreateGKEClusterFromModel(modelCluster)
		if err != nil {
			return nil, err
		}

		return gkeCluster, err

	case pkgCluster.Kubernetes:
		// Create Kubernetes struct
		kubernetesCluster, err := CreateKubernetesClusterFromModel(modelCluster)
		if err != nil {
			return nil, err
		}

		err = db.Where(kubernetesadapter.KubernetesClusterModel{ID: kubernetesCluster.modelCluster.ID}).First(&kubernetesCluster.modelCluster.Kubernetes).Error
		if database.IsRecordNotFoundError(err) {
			// metadata not set so there's no properties in DB
			log.Warnf(err.Error())
			err = nil
		}

		return kubernetesCluster, err
	}

	return nil, pkgErrors.ErrorNotSupportedCloudType
}

// CreateCommonClusterFromRequest creates a CommonCluster from a request
func CreateCommonClusterFromRequest(createClusterRequest *pkgCluster.CreateClusterRequest, orgId uint, userId uint) (CommonCluster, error) {
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
		// Check for PKE
		if createClusterRequest.Properties.CreateClusterPKE != nil {
			return createCommonClusterWithDistributionFromRequest(createClusterRequest, orgId, userId)
		}
		// Create EKS struct
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

	case pkgCluster.Kubernetes:
		// Create Kubernetes struct
		kubeCluster, err := CreateKubernetesClusterFromRequest(createClusterRequest, orgId, userId)
		if err != nil {
			return nil, err
		}
		return kubeCluster, nil
	}

	return nil, pkgErrors.ErrorNotSupportedCloudType
}

// createCommonClusterWithDistributionFromRequest creates a CommonCluster from a request
func createCommonClusterWithDistributionFromRequest(createClusterRequest *pkgCluster.CreateClusterRequest, orgId uint, userId uint) (*EC2ClusterPKE, error) {
	switch createClusterRequest.Cloud {
	case pkgCluster.Amazon:
		return CreateEC2ClusterPKEFromRequest(createClusterRequest, orgId, userId)

	default:
		return nil, pkgErrors.ErrorNotSupportedCloudType
	}
}

func createCommonClusterWithDistributionFromModel(modelCluster *model.ClusterModel) (*EC2ClusterPKE, error) {
	if modelCluster.Distribution != pkgCluster.PKE {
		return nil, pkgErrors.ErrorNotSupportedDistributionType
	}

	switch modelCluster.Cloud {
	case pkgCluster.Amazon:
		return CreateEC2ClusterPKEFromModel(modelCluster)

	default:
		return nil, pkgErrors.ErrorNotSupportedCloudType
	}
}

// NewCreatorBaseFields creates a new CreatorBaseFields instance from createdAt and createdBy
func NewCreatorBaseFields(createdAt time.Time, createdBy uint) *pkgCommon.CreatorBaseFields {
	var userName string
	if createdBy != 0 {
		userName = auth.GetUserNickNameById(createdBy)
	}

	return &pkgCommon.CreatorBaseFields{
		CreatedAt:   createdAt,
		CreatorName: userName,
		CreatorId:   createdBy,
	}
}
