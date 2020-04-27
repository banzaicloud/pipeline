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

package istiofeature

import (
	"context"
	"strings"

	"emperror.dev/errors"
	ghodss "github.com/ghodss/yaml"
	release2 "helm.sh/helm/v3/pkg/release"
	k8sHelm "k8s.io/helm/pkg/helm"
	pkgHelmRelease "k8s.io/helm/pkg/proto/hapi/release"
	"sigs.k8s.io/yaml"

	internalHelm "github.com/banzaicloud/pipeline/internal/helm"
	"github.com/banzaicloud/pipeline/src/helm"
)

type HelmService interface {
	InstallOrUpgrade(
		c ClusterProvider,
		release internalHelm.Release,
		opts internalHelm.Options,
	) error

	Delete(c ClusterProvider, releaseName, namespace string) error
}

type ClusterProvider interface {
	GetK8sConfig() ([]byte, error)
	GetID() uint
}

type ClusterProviderData struct {
	K8sConfig []byte
	ID        uint
}

func (c *ClusterProviderData) GetID() uint {
	return c.ID
}

func (c *ClusterProviderData) GetK8sConfig() ([]byte, error) {
	return c.K8sConfig, nil
}

type LegacyV2HelmService struct {
}

func (l *LegacyV2HelmService) InstallOrUpgrade(
	c ClusterProvider,
	release internalHelm.Release,
	opts internalHelm.Options,
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

func (l *LegacyV2HelmService) Delete(c ClusterProvider, releaseName, namespace string) error {
	return deleteDeployment(c, releaseName)
}

type HelmV3Service struct {
	service internalHelm.Service
}

func NewHelmV3Service(service internalHelm.Service) HelmService {
	return &HelmV3Service{service: service}
}

func (h *HelmV3Service) InstallOrUpgrade(
	c ClusterProvider,
	release internalHelm.Release,
	opts internalHelm.Options,
) error {
	ctx := context.Background()
	retrievedRelease, err := h.service.GetRelease(
		ctx,
		0,
		c.GetID(),
		release.ReleaseName,
		opts,
	)
	if err != nil {
		if internalHelm.ErrReleaseNotFound(err) {
			return h.service.InstallRelease(ctx, 0, c.GetID(), release, opts)
		}
		return errors.WrapIf(err, "failed to retrieve release")
	}
	if retrievedRelease.ReleaseInfo.Status == release2.StatusDeployed.String() {
		return h.service.UpgradeRelease(ctx, 0, c.GetID(), release, opts)
	}
	if retrievedRelease.ReleaseInfo.Status == release2.StatusFailed.String() {
		if err := h.service.DeleteRelease(ctx, 0, c.GetID(), release.ReleaseName, opts); err != nil {
			if !internalHelm.ErrReleaseNotFound(err) {
				return errors.WrapIf(err, "unable to delete release")
			}
		}
		return h.service.InstallRelease(ctx, 0, c.GetID(), release, opts)
	}
	return errors.Errorf("Release is in invalid state unable to upgrade: %s", retrievedRelease.ReleaseInfo.Status)
}

func (h *HelmV3Service) Delete(c ClusterProvider, releaseName, namespace string) error {
	if err := h.service.DeleteRelease(context.Background(), 0, c.GetID(), releaseName, internalHelm.Options{
		Namespace: namespace,
	}); err != nil {
		if internalHelm.ErrReleaseNotFound(err) {
			return nil
		}
		return errors.WrapIf(err, "unable to delete release")
	}
	return nil
}

func deleteDeployment(c ClusterProvider, releaseName string) error {
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

func installOrUpgradeDeployment(
	c ClusterProvider,
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
			if !upgrade {
				return nil
			}
			_, err = helm.UpgradeDeployment(releaseName, deploymentName, chartVersion, nil, values, false, kubeConfig, helm.GeneratePlatformHelmRepoEnv(), k8sHelm.UpgradeForce(true))
			if err != nil {
				return errors.WrapIfWithDetails(err, "could not upgrade deployment", "deploymentName", deploymentName)
			}
			return nil
		case pkgHelmRelease.Status_FAILED:
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

func ConvertStructure(in interface{}) (map[string]interface{}, error) {
	valuesOverride, err := ghodss.Marshal(in)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to marshal values")
	}

	// convert back to map[string]interface{}
	var mapStringValues map[string]interface{}
	err = yaml.UnmarshalStrict(valuesOverride, &mapStringValues)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to unmarshal values")
	}
	return mapStringValues, nil
}
