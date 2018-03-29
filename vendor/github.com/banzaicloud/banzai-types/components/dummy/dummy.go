package dummy

type CreateClusterDummy struct {
	Node *Node `json:"node,omitempty"`
}

type Node struct {
	KubernetesVersion string `json:"kubernetes_version"`
	Count             int    `json:"count"`
}

type UpdateClusterDummy struct {
	Node *Node `json:"node,omitempty"`
}

func (d *CreateClusterDummy) Validate() error {

	if d.Node == nil {
		d.Node = &Node{
			KubernetesVersion: "DummyKubernetesVersion",
			Count:             1,
		}
	}

	return nil
}

func (r *UpdateClusterDummy) Validate() error {
	if r.Node == nil {
		r.Node = &Node{
			KubernetesVersion: "DummyKubernetesVersion",
			Count:             1,
		}
	}
	return nil
}
