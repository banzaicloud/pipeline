package helm

type Install struct {
	KubeContext string `json:"kube_context"`
	Namespace   string `json:"namespace" binding:"required"` // "kube-system"
	Upgrade     bool   `json:"upgrade"`
	ServiceAccount string `json:"service_account" binding:"required"`
}
