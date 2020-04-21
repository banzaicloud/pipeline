// Copyright © 2019 Banzai Cloud
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

package helm2

import (
	"context"

	"emperror.dev/errors"
	k8sHelm "k8s.io/helm/pkg/helm"
	"k8s.io/helm/pkg/proto/hapi/release"

	internalhelm "github.com/banzaicloud/pipeline/internal/helm"

	"github.com/banzaicloud/pipeline/internal/common"
	pkgHelm "github.com/banzaicloud/pipeline/pkg/helm"
	"github.com/banzaicloud/pipeline/src/helm"
)

// Cluster represents a Kubernetes cluster.
type Cluster struct {
	OrganizationName string
	KubeConfig       []byte
}

// HelmService provides an interface for using Helm on a specific cluster.
type HelmService struct {
	clusters internalhelm.ClusterService

	logger common.Logger
}

// NewHelmService returns a new HelmService.
func NewHelmService(clusters internalhelm.ClusterService, logger common.Logger) *HelmService {
	return &HelmService{
		clusters: clusters,
		logger:   logger.WithFields(map[string]interface{}{"component": "helm"}),
	}
}

// InstallDeployment installs a deployment on a specific cluster.
// If it's already installed, InstallDeployment does nothing.
// If it's in a FAILED state, InstallDeployment attempts to delete it first.
func (s *HelmService) InstallDeployment(
	ctx context.Context,
	clusterID uint,
	namespace string,
	chartName string,
	releaseName string,
	values []byte,
	chartVersion string,
	wait bool,
) error {
	logger := s.logger.WithContext(ctx).WithFields(map[string]interface{}{"chart": chartName, "release": releaseName})
	logger.Info("installing deployment")

	kubeConfig, err := s.clusters.GetKubeConfig(ctx, clusterID)
	if err != nil {
		return err
	}

	foundRelease, err := findRelease(releaseName, kubeConfig)
	if err != nil {
		return errors.WithDetails(err, "chart", chartName)
	}

	if foundRelease != nil {
		switch foundRelease.GetInfo().GetStatus().GetCode() {
		case release.Status_DEPLOYED:
			logger.Info("deployment is already installed")

			return nil
		case release.Status_FAILED:
			err := helm.DeleteDeployment(releaseName, kubeConfig)
			if err != nil {
				return errors.WrapIfWithDetails(
					err, "failed to delete deployment",
					"chart", chartName,
					"release", releaseName,
				)
			}
		}
	}

	options := []k8sHelm.InstallOption{
		k8sHelm.InstallWait(wait),
		k8sHelm.ValueOverrides(values),
	}
	_, err = helm.CreateDeployment(
		chartName,
		chartVersion,
		nil,
		namespace,
		releaseName,
		false,
		nil,
		kubeConfig,
		helm.GeneratePlatformHelmRepoEnv(), // TODO: refactor!!!!!!
		options...,
	)
	if err != nil {
		return errors.WrapIfWithDetails(
			err, "failed to install deployment",
			"chart", chartName,
			"release", releaseName,
		)
	}

	logger.Info("deployment installed successfully")

	return nil
}

// UpdateDeployment updates an existing deployment on a specific cluster.
// If the deployment is not installed yet, UpdateDeployment does nothing.
func (s *HelmService) UpdateDeployment(
	ctx context.Context,
	clusterID uint,
	namespace string,
	chartName string,
	releaseName string,
	values []byte,
	chartVersion string,
) error {
	logger := s.logger.WithContext(ctx).WithFields(map[string]interface{}{"chart": chartName, "release": releaseName})
	logger.Info("updating deployment")

	kubeConfig, err := s.clusters.GetKubeConfig(ctx, clusterID)
	if err != nil {
		return err
	}

	foundRelease, err := findRelease(releaseName, kubeConfig)
	if err != nil {
		return errors.WithDetails(err, "chart", chartName)
	}

	if foundRelease != nil {
		switch foundRelease.GetInfo().GetStatus().GetCode() {
		case release.Status_DEPLOYED:
			_, err = helm.UpgradeDeployment(
				releaseName,
				chartName,
				chartVersion,
				nil,
				values,
				false,
				kubeConfig,
				helm.GeneratePlatformHelmRepoEnv(), // TODO: refactor!!!!!!
			)
			if err != nil {
				return errors.WrapIfWithDetails(
					err, "failed to update deployment",
					"chart", chartName,
					"release", releaseName,
				)
			}
		}
	}

	logger.Info("deployment updated successfully")

	return nil
}

func (s *HelmService) ApplyDeployment(
	ctx context.Context,
	clusterID uint,
	namespace string,
	chartName string,
	releaseName string,
	values []byte,
	chartVersion string,
) error {
	logger := s.logger.WithContext(ctx).WithFields(map[string]interface{}{"chart": chartName, "release": releaseName})
	logger.Info("applying deployment")

	kubeConfig, err := s.clusters.GetKubeConfig(ctx, clusterID)
	if err != nil {
		return err
	}

	foundRelease, err := findRelease(releaseName, kubeConfig)
	if err != nil {
		return errors.WithDetails(err, "chart", chartName)
	}

	if foundRelease != nil {
		switch foundRelease.GetInfo().GetStatus().GetCode() {
		case release.Status_DEPLOYED:
			_, err := helm.UpgradeDeployment(
				releaseName,
				chartName,
				chartVersion,
				nil,
				values,
				false,
				kubeConfig,
				helm.GeneratePlatformHelmRepoEnv(), // TODO: refactor!!!!!!
			)
			if err != nil {
				return errors.WrapIfWithDetails(
					err, "failed to upgrade deployment",
					"chart", chartName,
					"release", releaseName,
				)
			}

		case release.Status_FAILED:
			if err := helm.DeleteDeployment(releaseName, kubeConfig); err != nil {
				return errors.WrapIfWithDetails(
					err, "failed to delete deployment",
					"chart", chartName,
					"release", releaseName,
				)
			}

			options := []k8sHelm.InstallOption{
				// k8sHelm.InstallWait(wait),
				k8sHelm.ValueOverrides(values),
			}
			_, err = helm.CreateDeployment(
				chartName,
				chartVersion,
				nil,
				namespace,
				releaseName,
				false,
				nil,
				kubeConfig,
				helm.GeneratePlatformHelmRepoEnv(), // TODO: refactor!!!!!!
				options...,
			)
			if err != nil {
				return errors.WrapIfWithDetails(
					err, "failed to install deployment",
					"chart", chartName,
					"release", releaseName,
				)
			}
		}
	} else {
		options := []k8sHelm.InstallOption{
			// k8sHelm.InstallWait(wait),
			k8sHelm.ValueOverrides(values),
		}
		_, err = helm.CreateDeployment(
			chartName,
			chartVersion,
			nil,
			namespace,
			releaseName,
			false,
			nil,
			kubeConfig,
			helm.GeneratePlatformHelmRepoEnv(), // TODO: refactor!!!!!!
			options...,
		)
		if err != nil {
			return errors.WrapIfWithDetails(
				err, "failed to install deployment",
				"chart", chartName,
				"release", releaseName,
			)
		}
	}

	logger.Info("deployment applied successfully")

	return nil
}

// DeleteDeployment deletes a deployment from a specific cluster.
func (s *HelmService) DeleteDeployment(ctx context.Context, clusterID uint, releaseName, namespace string) error {
	logger := s.logger.WithContext(ctx).WithFields(map[string]interface{}{"release": releaseName})
	logger.Info("deleting deployment")

	kubeConfig, err := s.clusters.GetKubeConfig(ctx, clusterID)
	if err != nil {
		return err
	}

	foundRelease, err := findRelease(releaseName, kubeConfig)
	if err != nil {
		return err
	}

	if foundRelease != nil {
		err = helm.DeleteDeployment(releaseName, kubeConfig)
		if err != nil {
			return errors.WrapIfWithDetails(
				err, "failed to delete deployment",
				"release", releaseName,
			)
		}
	}

	logger.Info("deployment deleted successfully")

	return nil
}

func (s *HelmService) GetDeployment(ctx context.Context, clusterID uint, releaseName, namespace string) (*pkgHelm.GetDeploymentResponse, error) {
	logger := s.logger.WithContext(ctx).WithFields(map[string]interface{}{"release": releaseName})
	logger.Info("getting deployment")

	kubeConfig, err := s.clusters.GetKubeConfig(ctx, clusterID)
	if err != nil {
		return nil, err
	}

	return helm.GetDeployment(releaseName, kubeConfig)
}

func findRelease(releaseName string, k8sConfig []byte) (*release.Release, error) {
	deployments, err := helm.ListDeployments(&releaseName, "", k8sConfig)
	if err != nil {
		return nil, errors.WrapIfWithDetails(err, "failed to fetch deployments", "release", releaseName)
	}

	var foundRelease *release.Release

	if deployments != nil {
		for _, rel := range deployments.Releases {
			if rel.Name == releaseName {
				foundRelease = rel
				break
			}
		}
	}

	return foundRelease, nil
}
