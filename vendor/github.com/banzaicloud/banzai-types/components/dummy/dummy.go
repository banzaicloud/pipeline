package dummy

// CreateClusterDummy describes Pipeline's Dummy fields of a CreateCluster request
type CreateClusterDummy struct {
	Node *Node `json:"node,omitempty"`
}

// Node describes Dummy's node fields of a CreateCluster/Update request
type Node struct {
	KubernetesVersion string `json:"kubernetes_version"`
	Count             int    `json:"count"`
}

// UpdateClusterDummy describes Dummy's node fields of an UpdateCluster request
type UpdateClusterDummy struct {
	Node *Node `json:"node,omitempty"`
}

// Validate validates cluster create request
func (d *CreateClusterDummy) Validate() error {

	if d.Node == nil {
		d.Node = &Node{
			KubernetesVersion: "DummyKubernetesVersion",
			Count:             1,
		}
	}

	return nil
}

// Validate validates the update request
func (r *UpdateClusterDummy) Validate() error {
	if r.Node == nil {
		r.Node = &Node{
			KubernetesVersion: "DummyKubernetesVersion",
			Count:             1,
		}
	}
	return nil
}
