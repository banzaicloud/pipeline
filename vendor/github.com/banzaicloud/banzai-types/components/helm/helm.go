package helm

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

type ErrorResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Error   string `json:"error"`
}

type EndpointResponse struct {
	Endpoints []*EndpointItem `json:"endpoints"`
}

type EndpointItem struct {
	Name         string          `json:"name"`
	Host         string          `json:"host"`
	EndPointURLs []*EndPointURLs `json:"urls"`
}

type EndPointURLs struct {
	ServiceName string `json:"servicename"`
	URL         string `json:"url"`
}

type StatusResponse struct {
	Status int `json:"status"`
	Message string `json:"message"`
	Name string `json:"name"`
}

type DeleteResponse struct {
	Status int `json:"status"`
	Message string `json:"message"`
	Name string `json:"name"`
}

type InstallResponse struct {
	Status int `json:"status"`
	Message string `json:"message"`
}

type CreateDeploymentResponse struct {
	ReleaseName string `json:"release_name"`
	Notes string `json:"notes"`
}

//DeploymentType definition to describe a Helm deployment
type CreateDeploymentRequest struct {
	Name        string      `json:"name" binding:"required"`
	ReleaseName string      `json:"releasename"`
	Version     string      `json:"version"`
	Values      interface{} `json:"values"`
}

type ListDeploymentResponse struct {
	Name string `json:"name"`
	Chart string `json:"chart"`
	Version int32 `json:"version"`
	Updated string `json:"updated"`
	Status string `json:"status"`
}

type DeploymentStatusResponse struct {
	Status int `json:"status"`
	Message string `json:"message"`
}
