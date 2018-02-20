package cluster

import (
	"github.com/banzaicloud/azure-aks-client/utils"
	"regexp"
	"github.com/pkg/errors"
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

	msg := "Only numbers, lowercase letters and underscores are allowed under name property. In addition, the value cannot end with an underscore, and must also be less than 32 characters long."
	emptyMsg := "The name should not be empty."
	if len(c.Name) == 0 {
		return errors.New(emptyMsg)
	} else if len(c.Name) >= 32 {
		return errors.New("Cluster name is greater than or equal 32")
	}
	if isMatch, _ := regexp.MatchString("^[a-z0-9_]{0,31}[a-z0-9]$", c.Name); !isMatch {
		return errors.New(msg)
	}

	return nil
}
