package cloud

import (
	"github.com/banzaicloud/azure-aks-client/cluster"
	"github.com/banzaicloud/azure-aks-client/initapi"
	"github.com/banzaicloud/azure-aks-client/utils"
)

var sdk *cluster.Sdk

func init() {
	sdk = initapi.Init()
}

func GetAKSCluster(clusterType ClusterType) *cluster.ManagedCluster {
	return &cluster.ManagedCluster{
		Location: clusterType.Location,
		Properties: cluster.Properties{
			DNSPrefix: "pipeline",
			AgentPoolProfiles: []cluster.AgentPoolProfiles{
				{
					Count:  1,
					Name:   "pipeline",
					VMSize: clusterType.NodeInstanceType,
				},
			},
			KubernetesVersion: "1.7.7",
			ServicePrincipalProfile: cluster.ServicePrincipalProfile{
				ClientID: s(sdk.ServicePrincipal.ClientID),
				Secret:   s(sdk.ServicePrincipal.ClientSecret),
			},
			LinuxProfile: cluster.LinuxProfile{
				AdminUsername: "ubuntu",
				SSH: cluster.SSH{
					PublicKeys: &[]cluster.SSHPublicKey{
						{
							KeyData: s(utils.ReadPubRSA("id_rsa.pub")),
						},
					},
				},
			},
		},
	}
}

func s(input string) *string {
	s := input
	return &s
}
