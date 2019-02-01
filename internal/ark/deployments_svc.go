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
	"encoding/json"

	"github.com/goph/emperror"

	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	k8sHelm "k8s.io/helm/pkg/helm"
	pkgHelmRelease "k8s.io/helm/pkg/proto/hapi/release"

	"github.com/banzaicloud/pipeline/auth"
	"github.com/banzaicloud/pipeline/helm"
	"github.com/banzaicloud/pipeline/internal/ark/api"
	"github.com/banzaicloud/pipeline/internal/ark/client"
	"github.com/banzaicloud/pipeline/pkg/providers"
)

const (
	// helm deployment timeout
	deployTimeout = 90
)

// DeploymentsService is for managing ARK deployments
type DeploymentsService struct {
	org        *auth.Organization
	cluster    api.Cluster
	repository *DeploymentsRepository
	logger     logrus.FieldLogger

	client *client.Client
}

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

// Deploy deploys ARK with helm configured to use the given bucket and mode
func (s *DeploymentsService) Deploy(bucket *ClusterBackupBucketsModel, restoreMode bool) error {

	var deployment *ClusterBackupDeploymentsModel
	if !restoreMode {
		_, err := s.GetActiveDeployment()
		if err == nil {
			return errors.New("already deployed")
		}
	}

	clusterSecret, err := s.cluster.GetSecretWithValidation()
	if err != nil {
		return errors.Wrap(err, "error getting cluster secret")
	}

	bucketSecret, err := GetSecretWithValidation(bucket.SecretID, s.org.ID, bucket.Cloud)
	if err != nil {
		return errors.Wrap(err, "error getting bucket secret")
	}

	var resourceGroup string
	if s.cluster.GetCloud() == providers.Azure {
		m := s.cluster.(api.AKSCluster)
		resourceGroup = m.GetResourceGroupName()
	}

	config, err := s.getChartConfig(ConfigRequest{
		Cluster: clusterConfig{
			Name:        s.cluster.GetName(),
			Provider:    s.cluster.GetCloud(),
			Location:    s.cluster.GetLocation(),
			RBACEnabled: s.cluster.RbacEnabled(),
			azureClusterConfig: azureClusterConfig{
				ResourceGroup: resourceGroup,
			},
		},
		ClusterSecret: clusterSecret,

		Bucket: bucketConfig{
			Provider: bucket.Cloud,
			Name:     bucket.BucketName,
			Location: bucket.Location,
			azureBucketConfig: azureBucketConfig{
				StorageAccount: bucket.StorageAccount,
				ResourceGroup:  bucket.ResourceGroup,
			},
		},
		BucketSecret: bucketSecret,

		RestoreMode: restoreMode,
	})

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

	err = s.installDeployment(
		config.Namespace,
		config.Chart,
		config.Name,
		config.ValueOverrides,
		"InstallArk",
		config.Version,
		deployTimeout,
	)
	if err != nil {
		err = errors.Wrap(err, "error deploying ark")
		s.repository.UpdateStatus(deployment, "ERROR", err.Error())
		s.repository.Delete(deployment)
		return err
	}

	s.repository.UpdateStatus(deployment, "DEPLOYED", "")

	return nil
}

// Remove deletes an ARK deployment
func (s *DeploymentsService) Remove() error {

	deployment, err := s.GetActiveDeployment()
	if err == gorm.ErrRecordNotFound {
		return errors.New("not deployed")
	}

	config, err := s.cluster.GetK8sConfig()
	if err != nil {
		err = errors.Wrap(err, "error getting k8s config")
		s.repository.UpdateStatus(deployment, "ERROR", err.Error())
		return err
	}

	err = helm.DeleteDeployment(deployment.Name, config)
	if err != nil {
		s.repository.UpdateStatus(deployment, "ERROR", err.Error())
		return errors.Wrap(err, "error deleting deployment")
	}

	return s.repository.Delete(deployment)
}

func (s *DeploymentsService) getChartConfig(req ConfigRequest) (config ChartConfig, err error) {

	config = GetChartConfig()

	arkConfig, err := req.Get()
	if err != nil {
		err = errors.Wrap(err, "error getting config")
		return
	}

	arkJSON, err := json.Marshal(arkConfig)
	if err != nil {
		err = errors.Wrap(err, "json convert failed")
		return
	}

	config.ValueOverrides = arkJSON

	return
}

func (s *DeploymentsService) installDeployment(
	namespace string,
	deploymentName string,
	releaseName string,
	values []byte,
	actionName string,
	chartVersion string,
	timeout int64,
) error {

	kubeConfig, err := s.cluster.GetK8sConfig()
	if err != nil {
		return emperror.Wrap(err, "unable to fetch k8s config")
	}

	deployments, err := helm.ListDeployments(&releaseName, "", kubeConfig)
	if err != nil {
		return emperror.Wrap(err, "unable to fetch deployments from helm")
	}

	var foundRelease *pkgHelmRelease.Release
	if deployments != nil {
		for _, release := range deployments.Releases {
			if release.Name == releaseName {
				foundRelease = release
				break
			}
		}
	}

	if foundRelease != nil {
		switch foundRelease.GetInfo().GetStatus().GetCode() {
		case pkgHelmRelease.Status_DEPLOYED:
			s.logger.Infof("'%s' is already installed", deploymentName)
			return nil
		case pkgHelmRelease.Status_FAILED:
			err = helm.DeleteDeployment(releaseName, kubeConfig)
			if err != nil {
				s.logger.Errorf("Failed to deleted failed deployment '%s' due to: %s", deploymentName, err.Error())
				return err
			}
		}
	}

	options := []k8sHelm.InstallOption{
		k8sHelm.InstallWait(true),
		k8sHelm.ValueOverrides(values),
	}

	_, err = helm.CreateDeployment(
		deploymentName,
		chartVersion,
		nil,
		namespace,
		releaseName,
		false,
		nil,
		kubeConfig,
		helm.GenerateHelmRepoEnv(s.org.Name),
		options...,
	)
	if err != nil {
		s.logger.Errorf("Deploying '%s' failed due to: %s", deploymentName, err.Error())
		return err
	}

	s.logger.Infof("'%s' installed", deploymentName)

	return nil
}
