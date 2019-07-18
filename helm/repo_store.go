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
	"github.com/goph/logur"
	"k8s.io/helm/pkg/repo"
)

// ChartDetails describes a chart details
type ChartDetails struct {
	Name     string          `json:"name"`
	Repo     string          `json:"repo"`
	Versions []*ChartVersion `json:"versions"`
}

// ChartVersion describes a chart verion
type ChartVersion struct {
	Chart  *repo.ChartVersion `json:"chart"`
	Values string             `json:"values"`
	Readme string             `json:"readme"`
}

// ChartList describe a chart list
type ChartList struct {
	Name   string               `json:"name"`
	Charts []repo.ChartVersions `json:"charts"`
}

// RepositoryStore
type RepositoryStore interface {
	AddRepo(helmChartRepo *repo.Entry) (bool, error)
	DeleteRepo(repoName string) error
	ModifyRepo(repoName string, newRepo *repo.Entry) error
	UpdateRepo(repoName string) error
	GetRepos() ([]*repo.Entry, error)
	GetCharts(queryName, queryRepo, queryVersion, queryKeyword string) ([]ChartList, error)
	GetChart(chartRepo, chartName, chartVersion string) (details *ChartDetails, err error)
	DownloadChart(name, version string) (string, error)
}

func CreateDefaultRepoStore(orgName string, log logur.Logger) (RepositoryStore, error) {
	return newFileRepositoryStore(orgName, log)
}
