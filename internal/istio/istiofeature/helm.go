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
	"strings"

	"emperror.dev/errors"
	k8sHelm "k8s.io/helm/pkg/helm"
	pkgHelmRelease "k8s.io/helm/pkg/proto/hapi/release"

	"github.com/banzaicloud/pipeline/src/helm"
)

type clusterProvider interface {
	GetK8sConfig() ([]byte, error)
}

type clusterProviderData struct {
	k8sConfig []byte
}

func (c *clusterProviderData) GetK8sConfig() ([]byte, error) {
	return c.k8sConfig, nil
}

func deleteDeployment(c clusterProvider, releaseName string) error {
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
	c clusterProvider,
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
