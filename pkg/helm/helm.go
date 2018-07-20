package helm

import "github.com/technosophos/moniker"

// ### [ Constants to helm]
const (
	HELM_RETRY_ATTEMPT_CONFIG = "helm.retryAttempt"
	HELM_RETRY_SLEEP_SECONDS  = "helm.retrySleepSeconds"
)

// Stable repository constants
const (
	StableRepository = "stable"
	BanzaiRepository = "banzaicloud-stable"
	HelmPostFix      = "helm"
)

const releaseNameMaxLen = 53

// Install describes an Helm install request
type Install struct {
	// Name of the kubeconfig context to use
	KubeContext string `json:"kube_context"`

	// Namespace of Tiller
	Namespace string `json:"namespace" binding:"required"` // "kube-system"

	// Upgrade if Tiller is already installed
	Upgrade bool `json:"upgrade"`

	// Name of service account
	ServiceAccount string `json:"service_account" binding:"required"`

	// Use the canary Tiller image
	Canary bool `json:"canary_image"`

	// Override Tiller image
	ImageSpec string `json:"tiller_image"`

	// Limit the maximum number of revisions saved per release. Use 0 for no limit.
	MaxHistory int `json:"history_max"`
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
	ReleaseName string `json:"releaseName"`
	Notes       string `json:"notes"`
}

// CreateUpdateDeploymentRequest describes a Helm deployment
type CreateUpdateDeploymentRequest struct {
	Name        string                 `json:"name" binding:"required"`
	ReleaseName string                 `json:"releaseName"`
	Version     int32                  `json:"version"`
	ReUseValues bool                   `json:"reuseValues"`
	Namespace   string                 `json:"namespace"`
	Values      map[string]interface{} `json:"values,omitempty"`
}

// ListDeploymentResponse describes a deployment list response
type ListDeploymentResponse struct {
	Name         string `json:"releaseName"`
	Chart        string `json:"chart"`
	ChartName    string `json:"chartName"`
	ChartVersion string `json:"chartVersion"`
	Version      int32  `json:"version"`
	Updated      string `json:"updatedAt"`
	Status       string `json:"status"`
	Namespace    string `json:"namespace"`
	CreatedAt    string `json:"createdAt,omitempty"`
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
	CreatedAt    string                 `json:"createdAt,omitempty"`
	Updated      string                 `json:"updatedAt,omitempty"`
	Notes        string                 `json:"notes"`
	Values       map[string]interface{} `json:"values"`
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
