package cluster

import (
	"github.com/banzaicloud/azure-aks-client/utils"
	"regexp"
	"github.com/banzaicloud/banzai-types/constants"
)

type ManagedCluster struct {
	Location   string     `json:"location"`
	Properties Properties `json:"properties"`
}

func GetManagedCluster(request CreateClusterRequest, clientId string, secret string) *ManagedCluster {
	return &ManagedCluster{
		Location: request.Location,
		Properties: Properties{
			DNSPrefix: "dnsprefix",
			AgentPoolProfiles: []AgentPoolProfiles{
				{
					Count:  request.AgentCount,
					Name:   request.AgentName,
					VMSize: request.VMSize,
				},
			},
			KubernetesVersion: request.KubernetesVersion,
			ServicePrincipalProfile: ServicePrincipalProfile{
				ClientID: utils.S(clientId),
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

type CreateClusterRequest struct {
	Name              string
	Location          string
	VMSize            string
	ResourceGroup     string
	AgentCount        int
	AgentName         string
	KubernetesVersion string
}

func (c CreateClusterRequest) Validate() error {

	if len(c.Name) == 0 {
		return constants.ErrorAzureClusterNameEmpty
	} else if len(c.Name) >= 32 {
		return constants.ErrorAzureClusterNameTooLong
	}
	if isMatch, _ := regexp.MatchString("^[a-z0-9_]{0,31}[a-z0-9]$", c.Name); !isMatch {
		return constants.ErrorAzureClusterNameRegexp
	}

	return nil
}
