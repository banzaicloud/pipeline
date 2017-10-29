package cloud

import (
	"fmt"

	"github.com/kris-nova/kubicorn/apis/cluster"
	"github.com/kris-nova/kubicorn/cutil/kubeadm"
	"github.com/kris-nova/kubicorn/cutil/uuid"
)

func getAWSCluster(clusterType ClusterType) *cluster.Cluster {
	return &cluster.Cluster{
		Name:     clusterType.Name,
		Cloud:    cluster.CloudAmazon,
		Location: clusterType.Location,
		SSH: &cluster.SSH{
			PublicKeyPath: "~/.ssh/id_rsa.pub",
			User:          "ubuntu",
		},
		KubernetesAPI: &cluster.KubernetesAPI{
			Port: "443",
		},
		Network: &cluster.Network{
			Type:       cluster.NetworkTypePublic,
			CIDR:       "10.0.0.0/16",
			InternetGW: &cluster.InternetGW{},
		},
		Values: &cluster.Values{
			ItemMap: map[string]string{
				"INJECTEDTOKEN": kubeadm.GetRandomToken(),
			},
		},
		ServerPools: []*cluster.ServerPool{
			{
				Type:     cluster.ServerPoolTypeMaster,
				Name:     fmt.Sprintf("%s.master", clusterType.Name),
				MinCount: 1,
				MaxCount: 1,
				Image:    clusterType.MasterImage, //"ami-835b4efa"
				Size:     clusterType.NodeInstanceType,
				BootstrapScripts: []string{
					"amazon_k8s_ubuntu_16.04_master_pipeline.sh",
				},
				Subnets: []*cluster.Subnet{
					{
						Name:     fmt.Sprintf("%s.master", clusterType.Name),
						CIDR:     "10.0.0.0/24",
						Location: clusterType.Location,
					},
				},

				Firewalls: []*cluster.Firewall{
					{
						Name: fmt.Sprintf("%s.master-external-%s", clusterType.Name, uuid.TimeOrderedUUID()),
						IngressRules: []*cluster.IngressRule{
							{
								IngressFromPort: "22",
								IngressToPort:   "22",
								IngressSource:   "0.0.0.0/0",
								IngressProtocol: "tcp",
							},
							{
								IngressFromPort: "443",
								IngressToPort:   "443",
								IngressSource:   "0.0.0.0/0",
								IngressProtocol: "tcp",
							},
							{
								IngressFromPort: "30080",
								IngressToPort:   "30080",
								IngressSource:   "0.0.0.0/0",
								IngressProtocol: "tcp",
							},
							{
								IngressFromPort: "0",
								IngressToPort:   "65535",
								IngressSource:   "10.0.100.0/24",
								IngressProtocol: "-1",
							},
						},
					},
				},
			},
			{
				Type:     cluster.ServerPoolTypeNode,
				Name:     fmt.Sprintf("%s.node", clusterType.Name),
				MinCount: clusterType.NodeMin,
				MaxCount: clusterType.NodeMax,
				Image:    clusterType.NodeImage, //"ami-835b4efa"
				Size:     clusterType.NodeInstanceType,
				AwsConfiguration: &cluster.AwsConfiguration{
					SpotPrice: clusterType.NodeInstanceSpotPrice,
				},
				BootstrapScripts: []string{
					"amazon_k8s_ubuntu_16.04_node_pipeline.sh",
				},
				Subnets: []*cluster.Subnet{
					{
						Name:     fmt.Sprintf("%s.node", clusterType.Name),
						CIDR:     "10.0.100.0/24",
						Location: clusterType.Location,
					},
				},
				Firewalls: []*cluster.Firewall{
					{
						Name: fmt.Sprintf("%s.node-external-%s", clusterType.Name, uuid.TimeOrderedUUID()),
						IngressRules: []*cluster.IngressRule{
							{
								IngressFromPort: "22",
								IngressToPort:   "22",
								IngressSource:   "0.0.0.0/0",
								IngressProtocol: "tcp",
							},
							{
								IngressFromPort: "0",
								IngressToPort:   "65535",
								IngressSource:   "10.0.0.0/24",
								IngressProtocol: "-1",
							},
						},
					},
				},
			},
		},
	}
}
