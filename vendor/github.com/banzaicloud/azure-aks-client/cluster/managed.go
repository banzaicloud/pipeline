package cluster

import (
	"github.com/Azure/azure-sdk-for-go/services/containerservice/mgmt/2017-09-30/containerservice"
	"github.com/banzaicloud/azure-aks-client/utils"
	"github.com/banzaicloud/banzai-types/constants"
	"regexp"
)

func GetManagedCluster(request *CreateClusterRequest, clientId string, secret string) *containerservice.ManagedCluster {
	return &containerservice.ManagedCluster{
		ManagedClusterProperties: &containerservice.ManagedClusterProperties{
			ProvisioningState: nil,
			DNSPrefix:         utils.S("dnsprefix"),
			Fqdn:              nil,
			KubernetesVersion: &request.KubernetesVersion,
			AgentPoolProfiles: &request.Profiles,
			LinuxProfile: &containerservice.LinuxProfile{
				AdminUsername: utils.S("pipeline"),
				SSH: &containerservice.SSHConfiguration{
					PublicKeys: &[]containerservice.SSHPublicKey{
						{
							KeyData: utils.S(utils.ReadPubRSA("id_rsa.pub")),
						},
					},
				},
			},
			ServicePrincipalProfile: &containerservice.ServicePrincipalProfile{
				ClientID: &clientId,
				Secret:   &secret,
			},
		},
		Name:     &request.Name,
		Location: &request.Location,
	}
}

type CreateClusterRequest struct {
	Name              string
	Location          string
	ResourceGroup     string
	KubernetesVersion string
	Profiles          []containerservice.AgentPoolProfile
}

func (c CreateClusterRequest) Validate() error {

	if len(c.Name) == 0 {
		return constants.ErrorAzureClusterNameEmpty
	} else if len(c.Name) >= 32 {
		return constants.ErrorAzureClusterNameTooLong
	}
	if isMatch, _ := regexp.MatchString(RegexpForName, c.Name); !isMatch {
		return constants.ErrorAzureClusterNameRegexp
	}

	return nil
}

const RegexpForName = "^[a-z0-9_]{0,31}[a-z0-9]$"
