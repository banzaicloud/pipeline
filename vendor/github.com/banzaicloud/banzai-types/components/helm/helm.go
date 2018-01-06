package helm

type Install struct {
	KubeContext string `json:"kube_context"`
	Namespace   string `json:"namespace"` // "kube-system"
	Upgrade     bool   `json:"upgrade"`
}
