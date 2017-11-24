package cluster

import (
	"github.com/banzaicloud/azure-aks-client/utils"
	log "github.com/sirupsen/logrus"
	"os"
)

func init() {
	// Log as JSON
	log.SetFormatter(&log.JSONFormatter{})
	log.SetOutput(os.Stdout)
	log.SetLevel(log.InfoLevel)

}

type ManagedCluster struct {
	Location   string     `json:"location"`
	Properties Properties `json:"properties"`
}

func GetTestManagedCluster(clientID, secret string) *ManagedCluster {
	return &ManagedCluster{
		Location: "eastus",
		Properties: Properties{
			DNSPrefix: "dnsprefix",
			AgentPoolProfiles: []AgentPoolProfiles{
				{
					Count:  1,
					Name:   "agentpool1",
					VMSize: "Standard_D2_v2",
				},
			},
			KubernetesVersion: "1.7.7",
			ServicePrincipalProfile: ServicePrincipalProfile{
				ClientID: utils.S(clientID),
				Secret:   utils.S(secret),
			},
			LinuxProfile: LinuxProfile{
				AdminUsername: "erospista",
				SSH: SSH{
					PublicKeys: &[]SSHPublicKey{
						{
							KeyData: utils.S(utils.ReadPubRSA("id_rsa.pub")),
						},
					},
				},
			},
		},
	}
}
