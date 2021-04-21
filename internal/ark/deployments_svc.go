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

package ark

import (
	"context"

	"emperror.dev/errors"
	"github.com/banzaicloud/integrated-service-sdk/api/v1alpha1"
	"github.com/jinzhu/gorm"
	"github.com/sirupsen/logrus"

	"github.com/banzaicloud/pipeline/internal/ark/api"
	"github.com/banzaicloud/pipeline/internal/ark/client"
	"github.com/banzaicloud/pipeline/internal/ark/providers/amazon"
	"github.com/banzaicloud/pipeline/internal/ark/providers/azure"
	"github.com/banzaicloud/pipeline/internal/ark/providers/google"
	"github.com/banzaicloud/pipeline/internal/global"
	"github.com/banzaicloud/pipeline/internal/integratedservices/services/backup"
	pkgErrors "github.com/banzaicloud/pipeline/pkg/errors"
	"github.com/banzaicloud/pipeline/pkg/providers"
	"github.com/banzaicloud/pipeline/src/auth"
	"github.com/banzaicloud/pipeline/src/cluster"
	"github.com/banzaicloud/pipeline/src/secret"
)

// DeploymentsService is for managing ARK deployments
type DeploymentsService struct {
	org        *auth.Organization
	cluster    api.Cluster
	repository *DeploymentsRepository
	logger     logrus.FieldLogger

	client *client.Client
}

type secretContents map[string]cluster.InstallSecretRequestSpecItem

// DeploymentsServiceFactory creates and returns an initialized DeploymentsService instance
func DeploymentsServiceFactory(
	org *auth.Organization,
	cluster api.Cluster,
	db *gorm.DB,
	logger logrus.FieldLogger,
) *DeploymentsService {
	return NewDeploymentsService(org, cluster, NewDeploymentsRepository(org, cluster, db, logger), logger)
}

// NewDeploymentsService creates and returns an initialized DeploymentsService instance
func NewDeploymentsService(
	org *auth.Organization,
	cluster api.Cluster,
	repository *DeploymentsRepository,
	logger logrus.FieldLogger,
) *DeploymentsService {
	return &DeploymentsService{
		org:        org,
		cluster:    cluster,
		repository: repository,
		logger:     logger,
	}
}

// GetCluster gets an initialized api.Cluster implementation
func (s *DeploymentsService) GetCluster() api.Cluster {
	return s.cluster
}

// GetClient gets an initialized ARK client
func (s *DeploymentsService) GetClient() (*client.Client, error) {
	deployment, err := s.GetActiveDeployment()
	if err != nil {
		return nil, err
	}

	if s.client != nil {
		return s.client, nil
	}

	config, err := s.cluster.GetK8sConfig()
	if err != nil {
		return nil, errors.Wrap(err, "error getting k8s config")
	}

	client, err := client.New(config, deployment.Namespace, s.logger)
	if err != nil {
		return nil, errors.Wrap(err, "error getting ark client")
	}

	s.client = client

	return s.client, nil
}

// GetActiveDeployment gets the active ClusterBackupDeploymentsModel
func (s *DeploymentsService) GetActiveDeployment() (*ClusterBackupDeploymentsModel, error) {
	return s.repository.FindFirst()
}

func (s *DeploymentsService) Deploy(helmService HelmService, bucket *ClusterBackupBucketsModel,
	restoreMode bool, useClusterSecret bool, serviceAccountRoleARN string, useProviderSecret bool) error {
	var deployment *ClusterBackupDeploymentsModel
	req, err := s.deploy(bucket, restoreMode, useClusterSecret, serviceAccountRoleARN, useProviderSecret)
	if err != nil {
		return errors.Wrap(err, "error getting config request")
	}

	config, err := req.getChartConfig()
	if err != nil {
		return errors.Wrap(err, "error service getting config")
	}
	deployment, err = s.repository.Persist(&api.PersistDeploymentRequest{
		BucketID:    bucket.ID,
		Name:        config.Name,
		Namespace:   config.Namespace,
		RestoreMode: restoreMode,
	})
	if err != nil {
		return errors.Wrap(err, "error persisting deployment")
	}

	err = helmService.InstallDeployment(
		context.Background(),
		s.cluster.GetID(),
		config.Namespace,
		config.Chart,
		config.Name,
		config.ValueOverrides,
		config.Version,
		true,
	)
	if err != nil {
		err = errors.Wrap(err, "error deploying backup service")
		_ = s.repository.UpdateStatus(deployment, "ERROR", err.Error())
		_ = s.repository.Delete(deployment)
		return err
	}

	s.repository.UpdateStatus(deployment, "DEPLOYED", "") // nolint: errcheck

	return nil
}

func (s *DeploymentsService) Activate(service api.Service, bucket *ClusterBackupBucketsModel,
	restoreMode bool, useClusterSecret bool, serviceAccountRoleARN string, useProviderSecret bool) error {
	var deployment *ClusterBackupDeploymentsModel
	req, err := s.deploy(bucket, restoreMode, useClusterSecret, serviceAccountRoleARN, useProviderSecret)
	if err != nil {
		return errors.Wrap(err, "error getting config request")
	}

	config := GetChartConfig()
	deployment, err = s.repository.Persist(&api.PersistDeploymentRequest{
		BucketID:    bucket.ID,
		Name:        config.Name,
		Namespace:   config.Namespace,
		RestoreMode: restoreMode,
	})
	if err != nil {
		return errors.Wrap(err, "error persisting deployment")
	}

	valueOverrides, err := req.Get()
	if err != nil {
		return errors.Wrap(err, "error getting config values")
	}
	err = service.Activate(context.TODO(), s.cluster.GetID(),
		backup.IntegratedServiceName,
		map[string]interface{}{
			"chartValues": valueOverrides,
		},
	)

	if err != nil {
		err = errors.Wrap(err, "error activating backup service")
		_ = s.repository.UpdateStatus(deployment, "ERROR", err.Error())
		_ = s.repository.Delete(deployment)
		return err
	}

	client, err := s.GetClient()
	if err != nil {
		err = errors.Wrap(err, "error activating backup service")
		_ = s.repository.UpdateStatus(deployment, "ERROR", err.Error())
		_ = s.repository.Delete(deployment)
		return err
	}

	err = client.WaitForActivationPhase(backup.IntegratedServiceName, v1alpha1.Installed)
	if err != nil {
		err = errors.Wrap(err, "error activating backup service")
		_ = s.repository.UpdateStatus(deployment, "ERROR", err.Error())
		_ = s.repository.Delete(deployment)
		return err
	}

	s.repository.UpdateStatus(deployment, "DEPLOYED", "") // nolint: errcheck

	return nil
}

func (s *DeploymentsService) deploy(bucket *ClusterBackupBucketsModel,
	restoreMode bool, useClusterSecret bool, serviceAccountRoleARN string, useProviderSecret bool) (*ConfigRequest, error) {
	if !restoreMode {
		_, err := s.GetActiveDeployment()
		if err == nil {
			return nil, errors.New("already deployed")
		}
	}

	clusterSecret, err := s.cluster.GetSecretWithValidation()
	if err != nil {
		return nil, errors.Wrap(err, "error getting cluster secret")
	}

	bucketSecret, err := GetSecretWithValidation(bucket.SecretID, s.org.ID, bucket.Cloud)
	if err != nil {
		return nil, errors.Wrap(err, "error getting bucket secret")
	}

	var resourceGroup string
	if s.cluster.GetCloud() == providers.Azure {
		if m, ok := s.cluster.(api.AzureCluster); ok {
			resourceGroup = m.GetResourceGroupName()
		}
	}

	secretName := "velero"
	req := ConfigRequest{
		Cluster: clusterConfig{
			Name:         s.cluster.GetName(),
			Provider:     s.cluster.GetCloud(),
			Distribution: s.cluster.GetDistribution(),
			Location:     s.cluster.GetLocation(),
			RBACEnabled:  s.cluster.RbacEnabled(),
			azureClusterConfig: azureClusterConfig{
				ResourceGroup: resourceGroup,
			},
		},
		ClusterSecret: clusterSecret,
		Bucket: bucketConfig{
			Provider: bucket.Cloud,
			Name:     bucket.BucketName,
			Prefix:   bucket.Prefix,
			Location: bucket.Location,
			azureBucketConfig: azureBucketConfig{
				StorageAccount: bucket.StorageAccount,
				ResourceGroup:  bucket.ResourceGroup,
			},
		},
		SecretName:            secretName,
		BucketSecret:          bucketSecret,
		UseClusterSecret:      useClusterSecret,
		ServiceAccountRoleARN: serviceAccountRoleARN,
		UseProviderSecret:     useProviderSecret,
		RestoreMode:           restoreMode,
	}

	secretContents, err := req.getCredentialsSecret()
	if err != nil {
		return nil, err
	}
	// install secret
	_, err = installSecret(s.cluster, global.Config.Cluster.DisasterRecovery.Namespace, "velero", secretContents)
	if err != nil {
		return nil, errors.Wrap(err, "error installing Velero secret")
	}

	return &req, nil
}

// installSecret installs a secret to the specified cluster
func installSecret(cl interface {
	GetK8sConfig() ([]byte, error)
	GetOrganizationId() uint
}, namespace string, secretName string, secretContent secretContents) (string, error) {
	req := cluster.InstallSecretRequest{
		// Note: leave the Source field empty as the secret needs to be transformed
		Namespace: namespace,
		Update:    true,
		Spec:      secretContent,
	}

	k8sSecName, err := cluster.InstallSecret(cl, secretName, req)
	if err != nil {
		return "", errors.WrapIf(err, "failed to install secret to cluster")
	}

	return k8sSecName, nil
}

func (req ConfigRequest) getCredentialsSecret() (secretContents, error) {
	var config secretContents
	var BucketSecretContents string
	var ClusterSecretContents string
	var err error

	switch req.Cluster.Provider {
	case providers.Amazon:
		// In case of Amazon we set up one credential file with different profiles for cluster & bucket secret.
		// If UseClusterSecret is false there's no need for cluster secret, user will make sure node instance role has the right permissions
		ClusterSecretContents = ""
		if req.Bucket.Provider != providers.Amazon && req.UseClusterSecret {
			ClusterSecretContents, err = amazon.GetSecret(req.ClusterSecret, nil)
		}
		if err != nil {
			return config, nil
		}
	case providers.Google:
		ClusterSecretContents, err = google.GetSecret(req.ClusterSecret)
		if err != nil {
			return config, err
		}
	case providers.Azure:
		crgName := azure.GetAzureClusterResourceGroupName(req.Cluster.Distribution, req.Cluster.ResourceGroup, req.Cluster.Name, req.Cluster.Location)
		ClusterSecretContents, err = azure.GetSecretForCluster(req.ClusterSecret, crgName)
		if err != nil {
			return config, err
		}
	default:
		return config, pkgErrors.ErrorNotSupportedCloudType
	}

	switch req.Bucket.Provider {
	case providers.Amazon:
		var clusterSecret *secret.SecretItemResponse
		// put cluster secret if useClusterSecret == true otherwise will fallback to instance profile
		// which needs to be set up to contain snapshot permissions
		if req.Cluster.Provider == providers.Amazon && req.UseClusterSecret {
			clusterSecret = req.ClusterSecret
		}
		BucketSecretContents, err = amazon.GetSecret(clusterSecret, req.BucketSecret)
		if err != nil {
			return config, err
		}
	case providers.Google:
		BucketSecretContents, err = google.GetSecret(req.BucketSecret)
		if err != nil {
			return config, err
		}
	case providers.Azure:
		crgName := azure.GetAzureClusterResourceGroupName(req.Cluster.Distribution, req.Cluster.ResourceGroup, req.Cluster.Name, req.Cluster.Location)
		BucketSecretContents, err = azure.GetSecretForBucket(req.BucketSecret, req.Bucket.StorageAccount, req.Bucket.ResourceGroup, crgName)
		if err != nil {
			return config, err
		}
	default:
		return config, pkgErrors.ErrorNotSupportedCloudType
	}

	return secretContents{
		"cluster": cluster.InstallSecretRequestSpecItem{
			Value: ClusterSecretContents,
		},
		"cloud": cluster.InstallSecretRequestSpecItem{
			Value: BucketSecretContents,
		},
	}, err
}

// Remove deletes an ARK deployment
func (s *DeploymentsService) Deactivate(service api.Service) error {
	deployment, err := s.GetActiveDeployment()
	if err == gorm.ErrRecordNotFound {
		return errors.New("not deployed")
	}

	err = service.Deactivate(context.TODO(), s.cluster.GetID(), "backup")
	if err != nil {
		_ = s.repository.UpdateStatus(deployment, "ERROR", err.Error())
		return errors.Wrap(err, "error deleting deployment")
	}

	client, err := s.GetClient()
	if err != nil {
		err = errors.Wrap(err, "error activating backup service")
		_ = s.repository.UpdateStatus(deployment, "ERROR", err.Error())
		return err
	}
	err = client.WaitForActivationPhase(backup.IntegratedServiceName, v1alpha1.Installed)
	if err != nil {
		err = errors.Wrap(err, "error activating backup service")
		_ = s.repository.UpdateStatus(deployment, "ERROR", err.Error())
		return err
	}

	return s.repository.Delete(deployment)
}

// Remove deletes an ARK deployment
func (s *DeploymentsService) Remove(helmService HelmService) error {
	deployment, err := s.GetActiveDeployment()
	if err == gorm.ErrRecordNotFound {
		return errors.New("not deployed")
	}

	err = helmService.DeleteDeployment(context.Background(), deployment.ClusterID, deployment.Name, deployment.Namespace)
	if err != nil {
		_ = s.repository.UpdateStatus(deployment, "ERROR", err.Error())
		return errors.Wrap(err, "error deleting deployment")
	}

	return s.repository.Delete(deployment)
}
