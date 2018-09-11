package kubernetes

// CreateKubernetes describes Pipeline's Kubernetes fields of a CreateCluster request
type CreateClusterKubernetes struct {
	Metadata map[string]string `json:"metadata,omitempty"`
}

// Validate validates Kubernetes cluster create request
func (kube *CreateClusterKubernetes) Validate() error {
	return nil
}
