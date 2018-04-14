package kubernetes

type CreateKubernetes struct {
	Metadata map[string]string `json:"metadata,omitempty"`
}

func (kube *CreateKubernetes) Validate() error {
	return nil
}
