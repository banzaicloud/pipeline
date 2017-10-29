package cloud

import (
	"fmt"
	"github.com/kris-nova/kubicorn/apis/cluster"
	"github.com/kris-nova/kubicorn/cutil/kubeadm"
)

func getDOCluster(clusterType ClusterType) *cluster.Cluster {
	return &cluster.Cluster{
		Name:     clusterType.Name,
		Cloud:    cluster.CloudDigitalOcean,
		Location: "sfo2",
		SSH: &cluster.SSH{
			PublicKeyPath: "~/.ssh/id_rsa.pub",
			User:          "root",
		},
		KubernetesAPI: &cluster.KubernetesAPI{
			Port: "443",
		},
		Values: &cluster.Values{
			ItemMap: map[string]string{
				"INJECTEDTOKEN": kubeadm.GetRandomToken(),
			},
		},
		ServerPools: []*cluster.ServerPool{
			{
				Type:     cluster.ServerPoolTypeMaster,
				Name:     fmt.Sprintf("%s-master", clusterType.Name),
				MaxCount: 1,
				Image:    "ubuntu-16-04-x64",
				Size:     "1gb",
				BootstrapScripts: []string{
					"digitalocean_k8s_ubuntu_16.04_master.sh",
				},
			},
			{
				Type:     cluster.ServerPoolTypeNode,
				Name:     fmt.Sprintf("%s-node", clusterType.Name),
				MaxCount: 1,
				Image:    "ubuntu-16-04-x64",
				Size:     "1gb",
				BootstrapScripts: []string{
					"digitalocean_k8s_ubuntu_16.04_node.sh",
				},
			},
		},
	}
}
