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
	"encoding/base64"
	"strings"
	"time"

	"emperror.dev/errors"
	"github.com/banzaicloud/pipeline/internal/global"
	k8sHelm "k8s.io/helm/pkg/helm"
	"k8s.io/helm/pkg/proto/hapi/release"
	"sigs.k8s.io/yaml"

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

// LegacyHelmService provides an interface for using Helm on a specific cluster.
type LegacyHelmService struct {
	clusters      internalhelm.ClusterService
	serviceFacade internalhelm.Service
	logger        common.Logger
}

// NewLegacyHelmService returns a new LegacyHelmService.
func NewLegacyHelmService(clusters internalhelm.ClusterService, service internalhelm.Service, logger common.Logger) internalhelm.UnifiedReleaser {
	return &LegacyHelmService{
		clusters:      clusters,
		serviceFacade: service,
		logger:        logger.WithFields(map[string]interface{}{"component": "helm"}),
	}
}

// InstallDeployment installs a deployment on a specific cluster.
// If it's already installed, InstallDeployment does nothing.
// If it's in a FAILED state, InstallDeployment attempts to delete it first.
func (s *LegacyHelmService) InstallDeployment(
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
func (s *LegacyHelmService) UpdateDeployment(
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

func (s *LegacyHelmService) ApplyDeployment(
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
func (s *LegacyHelmService) DeleteDeployment(ctx context.Context, clusterID uint, releaseName, namespace string) error {
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

func (s *LegacyHelmService) GetDeployment(ctx context.Context, clusterID uint, releaseName, namespace string) (*pkgHelm.GetDeploymentResponse, error) {
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

func (s *LegacyHelmService) InstallOrUpgrade(
	c internalhelm.ClusterDataProvider,
	release internalhelm.Release,
	opts internalhelm.Options,
) error {
	values, err := yaml.Marshal(release.Values)
	if err != nil {
		return errors.WrapIf(err, "failed to marshal release values")
	}
	return installOrUpgradeDeployment(
		c,
		release.Namespace,
		release.ChartName,
		release.ReleaseName,
		values,
		release.Version,
		opts.Wait,
		opts.Install,
	)
}

func installOrUpgradeDeployment(
	c internalhelm.ClusterDataProvider,
	namespace string,
	deploymentName string,
	releaseName string,
	values []byte,
	chartVersion string,
	wait bool,
	upgrade bool,
) error {
	kubeConfig, err := c.GetK8sConfig()
	if err != nil {
		return errors.WrapIf(err, "could not get k8s config")
	}

	deployments, err := helm.ListDeployments(&releaseName, "", kubeConfig)
	if err != nil {
		return errors.WrapIf(err, "unable to fetch deployments from helm")
	}

	var foundRelease *release.Release
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
		case release.Status_DEPLOYED:
			if !upgrade {
				return nil
			}
			_, err = helm.UpgradeDeployment(releaseName, deploymentName, chartVersion, nil, values, false, kubeConfig, helm.GeneratePlatformHelmRepoEnv(), k8sHelm.UpgradeForce(true))
			if err != nil {
				return errors.WrapIfWithDetails(err, "could not upgrade deployment", "deploymentName", deploymentName)
			}
			return nil
		case release.Status_FAILED:
			err = helm.DeleteDeployment(releaseName, kubeConfig)
			if err != nil {
				return errors.WrapIfWithDetails(err, "failed to delete failed deployment", "deploymentName", deploymentName)
			}
		}
	}

	options := []k8sHelm.InstallOption{
		k8sHelm.InstallWait(wait),
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
		helm.GeneratePlatformHelmRepoEnv(),
		options...,
	)
	if err != nil {
		return errors.WrapIfWithDetails(err, "could not deploy", "deploymentName", deploymentName)
	}

	return nil
}

func (s *LegacyHelmService) Delete(c internalhelm.ClusterDataProvider, releaseName, namespace string) error {
	kubeConfig, err := c.GetK8sConfig()
	if err != nil {
		return errors.WrapIf(err, "could not get k8s config")
	}

	err = helm.DeleteDeployment(releaseName, kubeConfig)
	if err != nil {
		e := errors.Cause(err)
		if e != nil && strings.Contains(e.Error(), "not found") {
			return nil
		}
		return errors.WrapIf(err, "could not remove deployment")
	}

	return nil
}

func (s *LegacyHelmService) AddRepositoryIfNotExists(repository internalhelm.Repository) error {
	repos, err := s.serviceFacade.ListRepositories(context.Background(), 0)
	if err != nil {
		return err
	}
	for _, r := range repos {
		if r.URL == repository.URL {
			return nil
		}
	}
	return s.serviceFacade.AddRepository(context.Background(), 0, repository)
}

func (s *LegacyHelmService) GetRelease(c internalhelm.ClusterDataProvider, releaseName, namespace string) (internalhelm.Release, error) {
	kubeConfig, err := c.GetK8sConfig()
	if err != nil {
		return internalhelm.Release{}, err
	}

	helmClient, err := pkgHelm.NewClient(kubeConfig, global.LogrusLogger())
	if err != nil {
		return internalhelm.Release{}, err
	}
	defer helmClient.Close()

	releaseContent, err := helmClient.ReleaseContent(releaseName)
	if err != nil {
		return internalhelm.Release{}, err
	}

	createdAt := time.Unix(releaseContent.GetRelease().GetInfo().GetFirstDeployed().GetSeconds(), 0)
	updatedAt := time.Unix(releaseContent.GetRelease().GetInfo().GetLastDeployed().GetSeconds(), 0)

	notes := base64.StdEncoding.EncodeToString([]byte(releaseContent.GetRelease().GetInfo().GetStatus().GetNotes()))

	chartValues, err := internalhelm.ConvertBytes([]byte(releaseContent.GetRelease().GetChart().GetValues().GetRaw()))
	if err != nil {
		return internalhelm.Release{}, errors.WrapIf(err, "failed to decode chart values")
	}

	overrideValues, err := internalhelm.ConvertBytes([]byte(releaseContent.GetRelease().GetConfig().GetRaw()))
	if err != nil {
		return internalhelm.Release{}, errors.WrapIf(err, "failed to decode override values")
	}

	return internalhelm.Release{
		ReleaseName:    releaseContent.GetRelease().GetName(),
		ChartName:      releaseContent.GetRelease().GetChart().GetMetadata().GetName(),
		Namespace:      releaseContent.GetRelease().GetNamespace(),
		Values:         chartValues,
		Version:        releaseContent.GetRelease().GetChart().GetMetadata().GetVersion(),
		ReleaseVersion: releaseContent.GetRelease().GetVersion(),
		ReleaseInfo: internalhelm.ReleaseInfo{
			FirstDeployed: createdAt,
			LastDeployed:  updatedAt,
			Description:   releaseContent.GetRelease().GetInfo().GetDescription(),
			Status:        releaseContent.GetRelease().GetInfo().GetStatus().GetCode().String(),
			Notes:         notes,
			Values:        overrideValues,
		},
	}, nil
}

func (s *LegacyHelmService) IsV3() bool {
	return false
}
