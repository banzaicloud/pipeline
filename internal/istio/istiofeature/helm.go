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

	"github.com/goph/emperror"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	k8sHelm "k8s.io/helm/pkg/helm"
	pkgHelmRelease "k8s.io/helm/pkg/proto/hapi/release"

	"github.com/banzaicloud/pipeline/auth"
	"github.com/banzaicloud/pipeline/cluster"
	"github.com/banzaicloud/pipeline/helm"
)

func deleteDeployment(c cluster.CommonCluster, releaseName string) error {
	kubeConfig, err := c.GetK8sConfig()
	if err != nil {
		return emperror.Wrap(err, "could not get k8s config")
	}

	err = helm.DeleteDeployment(releaseName, kubeConfig)
	if err != nil {
		e := errors.Cause(err)
		if e != nil && strings.Contains(e.Error(), "not found") {
			return nil
		}
		return emperror.Wrap(err, "could not remove deployment")
	}

	return nil
}

func installDeployment(
	c cluster.CommonCluster,
	namespace string,
	deploymentName string,
	releaseName string,
	values []byte,
	chartVersion string,
	wait bool,
	logger logrus.FieldLogger,
) error {
	kubeConfig, err := c.GetK8sConfig()
	if err != nil {
		return emperror.Wrap(err, "could not get k8s config")
	}

	org, err := auth.GetOrganizationById(c.GetOrganizationId())
	if err != nil {
		return emperror.Wrap(err, "could not get organization")
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
			logger.Infof("'%s' is already installed", deploymentName)
			return nil
		case pkgHelmRelease.Status_FAILED:
			err = helm.DeleteDeployment(releaseName, kubeConfig)
			if err != nil {
				logger.Errorf("Failed to deleted failed deployment '%s' due to: %s", deploymentName, err.Error())
				return err
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
		helm.GenerateHelmRepoEnv(org.Name),
		options...,
	)
	if err != nil {
		logger.Errorf("Deploying '%s' failed due to: %s", deploymentName, err.Error())
		return err
	}

	logger.Infof("'%s' installed", deploymentName)

	return nil
}
