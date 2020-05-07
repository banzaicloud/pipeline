// Copyright © 2020 Banzai Cloud
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

package helmadapter

import (
	"context"

	"emperror.dev/errors"
	release2 "helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/storage/driver"
	"sigs.k8s.io/yaml"

	legacyHelm "github.com/banzaicloud/pipeline/src/helm"

	"github.com/banzaicloud/pipeline/internal/common"
	"github.com/banzaicloud/pipeline/internal/helm"
	helm2 "github.com/banzaicloud/pipeline/pkg/helm"
)

// helm3UnifiedReleaser component providing helm3 implementation for integrated services
type helm3UnifiedReleaser struct {
	helmService helm.Service
	logger      common.Logger
}

func NewUnifiedHelm3Releaser(service helm.Service, logger common.Logger) helm.UnifiedReleaser {
	return &helm3UnifiedReleaser{
		helmService: service,
		logger:      logger,
	}
}

func (h helm3UnifiedReleaser) ApplyDeployment(
	ctx context.Context,
	clusterID uint,
	namespace string,
	chartName string,
	releaseName string,
	values []byte,
	chartVersion string,
) error {
	var valuesMap map[string]interface{}
	if err := yaml.Unmarshal(values, &valuesMap); err != nil {
		return errors.WrapIf(err, "failed to unmarshal values")
	}

	options := helm.Options{
		Namespace:    namespace,
		DryRun:       false,
		GenerateName: false,
		Wait:         false,
		ReuseValues:  false,
		Install:      true,
	}
	release := helm.Release{
		ReleaseName: releaseName,
		ChartName:   chartName,
		Namespace:   namespace,
		Version:     chartVersion,
		Values:      valuesMap,
	}

	return h.helmService.UpgradeRelease(ctx, 0, clusterID, release, options)
}

// for clustersetup!
func (h *helm3UnifiedReleaser) InstallDeployment(
	ctx context.Context,
	clusterID uint,
	namespace string,
	chartName string,
	releaseName string,
	values []byte,
	chartVersion string,
	wait bool,
) error {
	var valuesMap map[string]interface{}
	if err := yaml.Unmarshal(values, &valuesMap); err != nil {
		return errors.WrapIf(err, "failed to unmarshal values")
	}

	options := helm.Options{
		Namespace:    namespace,
		DryRun:       false,
		GenerateName: false,
		Wait:         wait,
		ReuseValues:  false,
	}
	release := helm.Release{
		ReleaseName: releaseName,
		ChartName:   chartName,
		Namespace:   namespace,
		Version:     chartVersion,
		Values:      valuesMap,
	}

	retrievedRelease, err := h.helmService.GetRelease(ctx, 0, clusterID, releaseName, helm.Options{
		Namespace: namespace,
	})
	if err != nil {
		if errors.Is(err, driver.ErrReleaseNotFound) {
			return h.helmService.InstallRelease(ctx, 0, clusterID, release, options)
		}
		return errors.WrapIf(err, "failed to retrieve release")
	}
	if retrievedRelease.ReleaseInfo.Status == release2.StatusDeployed.String() {
		return nil
	}
	if retrievedRelease.ReleaseInfo.Status == release2.StatusFailed.String() {
		if err := h.DeleteDeployment(ctx, clusterID, releaseName, namespace); err != nil {
			return errors.WrapIf(err, "unable to delete release")
		}
		return h.helmService.InstallRelease(ctx, 0, clusterID, release, options)
	}
	return errors.Errorf("release is in an invalid state: %s", release.ReleaseInfo.Status)
}

func (h *helm3UnifiedReleaser) DeleteDeployment(ctx context.Context, clusterID uint, releaseName, namespace string) error {
	err := h.helmService.DeleteRelease(ctx, 0, clusterID, releaseName, helm.Options{
		Namespace: namespace,
	})
	if err != nil {
		if helm.ErrReleaseNotFound(err) {
			return nil
		}
		return errors.WrapIf(err, "unable to delete release")
	}
	return nil
}

func (h *helm3UnifiedReleaser) GetDeployment(ctx context.Context, clusterID uint, releaseName, namespace string) (*helm2.GetDeploymentResponse, error) {
	release, err := h.helmService.GetRelease(ctx, 0, clusterID, releaseName, helm.Options{
		Namespace: namespace,
	})
	if err != nil {
		// return the same error as the helm2 implementation on release not found
		if errors.Is(err, driver.ErrReleaseNotFound) {
			return nil, &legacyHelm.DeploymentNotFoundError{
				HelmError: err,
			}
		}
		return nil, errors.WrapIf(err, "failed to retrieve release")
	}

	// TODO identify the minimum set of required fields, map only those
	return &helm2.GetDeploymentResponse{
		ReleaseName:  release.ReleaseName,
		Chart:        release.ChartName,
		ChartName:    release.ChartName,
		ChartVersion: release.Version,
		Namespace:    release.Namespace,
		Version:      0,
		Status:       release.ReleaseInfo.Status,
	}, nil
}

func (h *helm3UnifiedReleaser) InstallOrUpgrade(
	c helm.ClusterDataProvider,
	release helm.Release,
	opts helm.Options,
) error {
	ctx := context.Background()
	retrievedRelease, err := h.helmService.GetRelease(
		ctx,
		0,
		c.GetID(),
		release.ReleaseName,
		opts,
	)
	if err != nil {
		if helm.ErrReleaseNotFound(err) {
			return h.helmService.InstallRelease(ctx, 0, c.GetID(), release, opts)
		}
		return errors.WrapIf(err, "failed to retrieve release")
	}
	if retrievedRelease.ReleaseInfo.Status == release2.StatusDeployed.String() {
		return h.helmService.UpgradeRelease(ctx, 0, c.GetID(), release, opts)
	}
	if retrievedRelease.ReleaseInfo.Status == release2.StatusFailed.String() {
		if err := h.helmService.DeleteRelease(ctx, 0, c.GetID(), release.ReleaseName, opts); err != nil {
			if !helm.ErrReleaseNotFound(err) {
				return errors.WrapIf(err, "unable to delete release")
			}
		}
		return h.helmService.InstallRelease(ctx, 0, c.GetID(), release, opts)
	}
	return errors.Errorf("Release is in invalid state unable to upgrade: %s", retrievedRelease.ReleaseInfo.Status)
}

func (h *helm3UnifiedReleaser) Delete(c helm.ClusterDataProvider, releaseName, namespace string) error {
	if err := h.helmService.DeleteRelease(context.Background(), 0, c.GetID(), releaseName, helm.Options{
		Namespace: namespace,
	}); err != nil {
		if helm.ErrReleaseNotFound(err) {
			return nil
		}
		return errors.WrapIf(err, "unable to delete release")
	}
	return nil
}

func (h *helm3UnifiedReleaser) AddRepositoryIfNotExists(repository helm.Repository) error {
	repos, err := h.helmService.ListRepositories(context.Background(), 0)
	if err != nil {
		return err
	}
	for _, r := range repos {
		if r.URL == repository.URL {
			return nil
		}
	}
	return h.helmService.AddRepository(context.Background(), 0, repository)
}

func (h *helm3UnifiedReleaser) GetRelease(c helm.ClusterDataProvider, releaseName, namespace string) (helm.Release, error) {
	return h.helmService.GetRelease(context.TODO(), 0, c.GetID(), releaseName, helm.Options{
		Namespace: namespace,
	})
}
