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

package helm

import (
	"context"

	"emperror.dev/errors"
	k8sHelm "k8s.io/helm/pkg/helm"
	"k8s.io/helm/pkg/proto/hapi/release"

	"github.com/banzaicloud/pipeline/helm"
	"github.com/banzaicloud/pipeline/internal/common"
)

// ClusterService provides a thin access layer to clusters.
type ClusterService interface {
	// GetCluster retrieves the cluster representation based on the cluster identifier.
	GetCluster(ctx context.Context, clusterID uint) (*Cluster, error)
}

// Cluster represents a Kubernetes cluster.
type Cluster struct {
	OrganizationName string
	KubeConfig       []byte
}

// HelmService provides an interface for using Helm on a specific cluster.
type HelmService struct {
	clusters ClusterService

	logger common.Logger
}

// NewHelmService returns a new HelmService.
func NewHelmService(clusters ClusterService, logger common.Logger) *HelmService {
	return &HelmService{
		clusters: clusters,

		logger: logger.WithFields(map[string]interface{}{"component": "helm"}),
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

	cluster, err := s.clusters.GetCluster(ctx, clusterID)
	if err != nil {
		return err
	}

	foundRelease, err := s.findRelease(releaseName, cluster)
	if err != nil {
		return errors.WithDetails(err, "chart", chartName)
	}

	if foundRelease != nil {
		switch foundRelease.GetInfo().GetStatus().GetCode() {
		case release.Status_DEPLOYED:
			logger.Info("deployment is already installed")

			return nil
		case release.Status_FAILED:
			err := helm.DeleteDeployment(releaseName, cluster.KubeConfig)
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
		cluster.KubeConfig,
		helm.GenerateHelmRepoEnv(cluster.OrganizationName), // TODO: refactor!!!!!!
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

	cluster, err := s.clusters.GetCluster(ctx, clusterID)
	if err != nil {
		return err
	}

	foundRelease, err := s.findRelease(releaseName, cluster)
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
				cluster.KubeConfig,
				helm.GenerateHelmRepoEnv(cluster.OrganizationName), // TODO: refactor!!!!!!
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

	cluster, err := s.clusters.GetCluster(ctx, clusterID)
	if err != nil {
		return err
	}

	foundRelease, err := s.findRelease(releaseName, cluster)
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
				cluster.KubeConfig,
				helm.GenerateHelmRepoEnv(cluster.OrganizationName), // TODO: refactor!!!!!!
			)
			if err != nil {
				return errors.WrapIfWithDetails(
					err, "failed to upgrade deployment",
					"chart", chartName,
					"release", releaseName,
				)
			}

		case release.Status_FAILED:
			if err := helm.DeleteDeployment(releaseName, cluster.KubeConfig); err != nil {
				return errors.WrapIfWithDetails(
					err, "failed to delete deployment",
					"chart", chartName,
					"release", releaseName,
				)
			}

			options := []k8sHelm.InstallOption{
				//k8sHelm.InstallWait(wait),
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
				cluster.KubeConfig,
				helm.GenerateHelmRepoEnv(cluster.OrganizationName), // TODO: refactor!!!!!!
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
			//k8sHelm.InstallWait(wait),
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
			cluster.KubeConfig,
			helm.GenerateHelmRepoEnv(cluster.OrganizationName), // TODO: refactor!!!!!!
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
func (s *HelmService) DeleteDeployment(ctx context.Context, clusterID uint, releaseName string) error {
	logger := s.logger.WithContext(ctx).WithFields(map[string]interface{}{"release": releaseName})
	logger.Info("deleting deployment")

	cluster, err := s.clusters.GetCluster(ctx, clusterID)
	if err != nil {
		return err
	}

	foundRelease, err := s.findRelease(releaseName, cluster)
	if err != nil {
		return err
	}

	if foundRelease != nil {
		err = helm.DeleteDeployment(releaseName, cluster.KubeConfig)
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

func (s *HelmService) findRelease(releaseName string, cluster *Cluster) (*release.Release, error) {
	deployments, err := helm.ListDeployments(&releaseName, "", cluster.KubeConfig)
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
