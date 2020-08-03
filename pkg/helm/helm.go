// Copyright © 2018 Banzai Cloud
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
	"time"
)

// the label Helm places on Kubernetes objects for differentiating between
// different instances: https://helm.sh/docs/chart_best_practices/#standard-labels
const (
	HelmReleaseNameLabelLegacy = "release"
	HelmReleaseNameLabel       = "app.kubernetes.io/instance"
)

// GetHelmReleaseName returns the helm release name placed by helm deployment Kubernetes objects
// it checks for label with key `HelmReleaseNameLabel`, if no such label is present than falls back
// the legacy label key
func GetHelmReleaseName(labels map[string]string) string {
	if labels == nil {
		return ""
	}

	if labels[HelmReleaseNameLabel] != "" {
		return labels[HelmReleaseNameLabel]
	}

	return labels[HelmReleaseNameLabelLegacy]
}

// EndpointResponse describes a service public endpoints
type EndpointResponse struct {
	Endpoints []*EndpointItem `json:"endpoints"`
}

// EndpointItem describes a service public endpoint
type EndpointItem struct {
	Name         string           `json:"name"`
	Host         string           `json:"host"`
	Ports        map[string]int32 `json:"ports"`
	EndPointURLs []*EndPointURLs  `json:"urls"`
}

// EndPointURLs describes an endpoint url
type EndPointURLs struct {
	Path        string `json:"path"`
	URL         string `json:"url"`
	ReleaseName string `json:"releaseName"`
}

// ListDeploymentResponse describes a deployment list response
type ListDeploymentResponse struct {
	Name         string    `json:"releaseName"`
	Chart        string    `json:"chart"`
	ChartName    string    `json:"chartName"`
	ChartVersion string    `json:"chartVersion"`
	Version      int32     `json:"version"`
	UpdatedAt    time.Time `json:"updatedAt"`
	Status       string    `json:"status"`
	Namespace    string    `json:"namespace"`
	CreatedAt    time.Time `json:"createdAt,omitempty"`
	Supported    bool      `json:"supported"`
	WhiteListed  bool      `json:"whiteListed"`
	Rejected     bool      `json:"rejected"`
}

// DeploymentStatusResponse describes a deployment status response
type DeploymentStatusResponse struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
}

// GetDeploymentResponse describes the details of a helm deployment
type GetDeploymentResponse struct {
	ReleaseName  string                 `json:"releaseName"`
	Chart        string                 `json:"chart"`
	ChartName    string                 `json:"chartName"`
	ChartVersion string                 `json:"chartVersion"`
	Namespace    string                 `json:"namespace"`
	Version      int32                  `json:"version"`
	Status       string                 `json:"status"`
	Description  string                 `json:"description"`
	CreatedAt    time.Time              `json:"createdAt,omitempty"`
	Updated      time.Time              `json:"updatedAt,omitempty"`
	Notes        string                 `json:"notes"`
	Values       map[string]interface{} `json:"values"`
}
