package cluster

type AccessProfiles struct {
	ClusterAdmin ClusterAdmin `json:"clusterAdmin"`
	ClusterUser  ClusterUser  `json:"clusterUser"`
}

type ClusterAdmin struct {
	KubeConfig string `json:"kubeConfig"`
}

type ClusterUser struct {
	KubeConfig string `json:"kubeConfig"`
}
