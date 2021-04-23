// Copyright © 2021 Banzai Cloud
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

package clustersetup

import (
	"context"

	"emperror.dev/errors"
)

const HelmInstallActivityName = "cluster-setup-helm-install"

type HelmInstallActivity struct {
	helmService HelmService
}

// NewHelmInstallActivity returns a new HelmInstallActivity.
func NewHelmInstallActivity(helmService HelmService) HelmInstallActivity {
	return HelmInstallActivity{
		helmService: helmService,
	}
}

type HelmInstallActivityInput struct {
	ClusterID    uint
	Namespace    string
	ReleaseName  string
	ChartName    string
	ChartVersion string
	Values       []byte
}

func (a HelmInstallActivity) Execute(ctx context.Context, input HelmInstallActivityInput) error {
	if a.helmService == nil {
		return errors.New("missing helm service dependency")
	}

	err := a.helmService.ApplyDeployment(
		ctx,
		input.ClusterID,
		input.Namespace,
		input.ChartName,
		input.ReleaseName,
		input.Values,
		input.ChartVersion,
	)
	if err != nil {
		return errors.WrapIff(err, "failed to deploy %s@%s chart", input.ChartName, input.ChartVersion)
	}

	return nil
}
