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

package clusterfeatureadapter

import (
	"context"

	"emperror.dev/errors"
	"github.com/banzaicloud/pipeline/helm"
	"github.com/banzaicloud/pipeline/internal/clusterfeature"
	"github.com/goph/logur"
	k8sHelm "k8s.io/helm/pkg/helm"
	"k8s.io/helm/pkg/proto/hapi/release"
)

// component in chrge for installing features from helm charts
type featureHelmService struct {
	logger logur.Logger
}

func (hs *featureHelmService) InstallDeployment(
	ctx context.Context,
	orgName string,
	kubeConfig []byte,
	namespace string,
	deploymentName string,
	releaseName string,
	values []byte,
	chartVersion string,
	wait bool,
) error {

	deployments, err := helm.ListDeployments(&releaseName, "", kubeConfig)
	if err != nil {
		hs.logger.Error("failed to fetch deployments", map[string]interface{}{"deployment": deploymentName})
		return err
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

	if foundRelease != nil {
		switch foundRelease.GetInfo().GetStatus().GetCode() {
		case release.Status_DEPLOYED:
			hs.logger.Info("deployment is already installed", map[string]interface{}{"deployment": deploymentName})
			return nil
		case release.Status_FAILED:
			err = helm.DeleteDeployment(releaseName, kubeConfig)
			if err != nil {
				hs.logger.Error("failed to delete failed deployment", map[string]interface{}{"deployment": deploymentName})
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
		helm.GenerateHelmRepoEnv(orgName),
		options...,
	)

	if err != nil {
		hs.logger.Error("failed to create deployment", map[string]interface{}{"deployment": deploymentName})
		return err
	}

	hs.logger.Info("installed deployment", map[string]interface{}{"deployment": deploymentName})
	return nil
}

func (hs *featureHelmService) DeleteDeployment(ctx context.Context, kubeConfig []byte, releaseName string) error {

	deployments, err := helm.ListDeployments(&releaseName, "", kubeConfig)
	if err != nil {
		hs.logger.Error("failed to fetch deployments", map[string]interface{}{"release": releaseName})
		return err
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

	if foundRelease != nil {
		err = helm.DeleteDeployment(releaseName, kubeConfig)
		if err != nil {
			hs.logger.Error("failed to delete deployment", map[string]interface{}{"deployment": releaseName})
			return err
		}
	}

	return nil

}

func (hs *featureHelmService) UpdateDeployment(ctx context.Context, orgName string, kubeConfig []byte, namespace string,
	deploymentName string, releaseName string, values []byte, chartVersion string) error {

	deployments, err := helm.ListDeployments(&releaseName, "", kubeConfig)
	if err != nil {
		return errors.WrapIf(err, "unable to fetch deployments	")
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

	if foundRelease != nil {
		switch foundRelease.GetInfo().GetStatus().GetCode() {
		case release.Status_DEPLOYED:
			_, err = helm.UpgradeDeployment(
				releaseName,
				deploymentName,
				chartVersion,
				nil,
				values,
				false,
				kubeConfig,
				helm.GenerateHelmRepoEnv(orgName))
			if err != nil {
				return errors.WrapIfWithDetails(err, "could not upgrade deployment", "deploymentName", deploymentName)
			}
			return nil
		}
	}

	return nil
}

func NewHelmService(logger logur.Logger) clusterfeature.HelmService {
	return &featureHelmService{
		logger: logur.WithFields(logger, map[string]interface{}{"helm-service": "comp"}),
	}
}
