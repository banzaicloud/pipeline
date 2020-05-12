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

package deployment

import (
	"context"
	"strings"

	"emperror.dev/errors"

	"github.com/banzaicloud/pipeline/internal/helm"
)

type HelmService interface {
	GetChartDescription(name, version string) (string, error)
}

func NewHelmService(facade helm.Service) HelmService {
	return &Helm3Service{
		facade: facade,
	}
}

type Helm3Service struct {
	facade helm.Service
}

func (h *Helm3Service) GetChartDescription(name, version string) (string, error) {
	repoAndChart := strings.Split(name, "/")
	if len(repoAndChart) != 2 {
		return "", errors.Errorf("missing repo ref from chart name %s", name)
	}
	chart, err := h.facade.GetChart(context.TODO(), 0, helm.ChartFilter{
		Repo:    []string{repoAndChart[0]},
		Name:    []string{repoAndChart[1]},
		Version: []string{version},
	}, helm.Options{})
	if err != nil {
		return "", err
	}
	return chart.GetDescription(version)
}
