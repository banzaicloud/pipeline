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

	"github.com/technosophos/moniker"
	corev1 "k8s.io/api/core/v1"
)

// Stable repository constants
const (
	StableRepository = "stable"
	BanzaiRepository = "banzaicloud-stable"
	LokiRepository   = "loki"
	HelmPostFix      = "helm"
)

const releaseNameMaxLen = 53

// the label Helm places on Kubernetes objects for differentiating between
// different instances: https://helm.sh/docs/chart_best_practices/#standard-labels
const HelmReleaseNameLabelLegacy = "release"
const HelmReleaseNameLabel = "app.kubernetes.io/instance"

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

// Install describes an Helm install request
type Install struct {
	// Name of the kubeconfig context to use
	KubeContext string `json:"kube_context"`

	// Namespace of Tiller
	Namespace string `json:"namespace" binding:"required"` // "kube-system"

	// Upgrade if Tiller is already installed
	Upgrade bool `json:"upgrade"`

	// Force allows to force upgrading tiller if deployed version is greater than current client version
	ForceUpgrade bool `json:"force_upgrade"`

	// Name of service account
	ServiceAccount string `json:"service_account" binding:"required"`

	// Use the canary Tiller image
	Canary bool `json:"canary_image"`

	// Override Tiller image
	ImageSpec string `json:"tiller_image"`

	// Limit the maximum number of revisions saved per release. Use 0 for no limit.
	MaxHistory int `json:"history_max"`

	// Tolerations to be applied onto the pod
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`

	// NodeAffinity
	NodeAffinity *corev1.NodeAffinity `json:"nodeAffinity,omitempty"`
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

// StatusResponse describes a Helm status response
type StatusResponse struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
	Name    string `json:"name"`
}

// DeleteResponse describes a deployment delete response
type DeleteResponse struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
	Name    string `json:"name"`
}

// InstallResponse describes a Helm install response
type InstallResponse struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
}

// CreateUpdateDeploymentResponse describes a create/update deployment response
type CreateUpdateDeploymentResponse struct {
	ReleaseName string               `json:"releaseName"`
	Notes       string               `json:"notes"`
	Resources   []DeploymentResource `json:"resources"`
}

// CreateUpdateDeploymentRequest describes a Helm deployment
type CreateUpdateDeploymentRequest struct {
	Name        string                 `json:"name" yaml:"name" binding:"required"`
	Version     string                 `json:"version,omitempty" yaml:"version,omitempty"`
	Package     []byte                 `json:"package,omitempty" yaml:"package,omitempty"`
	ReleaseName string                 `json:"releaseName" yaml:"releaseName"`
	ReUseValues bool                   `json:"reuseValues" yaml:"reuseValues"`
	Namespace   string                 `json:"namespace,omitempty" yaml:"namespace,omitempty"`
	DryRun      bool                   `json:"dryrun,omitempty" yaml:"dryrun,omitempty"`
	Wait        bool                   `json:"wait,omitempty" yaml:"wait,omitempty"`
	Timeout     int64                  `json:"timeout,omitempty" yaml:"timeout,omitempty"`
	Values      map[string]interface{} `json:"values,omitempty" yaml:"values,omitempty"`
	OdPcts      map[string]int         `json:"odpcts,omitempty" yaml:"odpcts,omitempty"`
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

// GetDeploymentResourcesResponse lists the resources of a helm deployment
type GetDeploymentResourcesResponse struct {
	DeploymentResources []DeploymentResource `json:"resources"`
}

// Describes a K8s resource
type DeploymentResource struct {
	Name string `json:"name"`
	Kind string `json:"kind"`
}

// GenerateReleaseName Generate Helm like release name
func GenerateReleaseName() string {
	namer := moniker.New()
	name := namer.NameSep("-")
	if len(name) > releaseNameMaxLen {
		name = name[:releaseNameMaxLen]
	}
	return name
}
