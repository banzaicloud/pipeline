package kubernetes

// CreateKubernetes describes Pipeline's Kubernetes fields of a CreateCluster request
type CreateKubernetes struct {
	Metadata map[string]string `json:"metadata,omitempty"`
}

// Validate validates Kubernetes cluster create request
func (kube *CreateKubernetes) Validate() error {
	return nil
}
