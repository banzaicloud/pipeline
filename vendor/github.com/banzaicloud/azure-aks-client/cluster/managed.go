package cluster

import (
	"github.com/Azure/azure-sdk-for-go/services/containerservice/mgmt/2017-09-30/containerservice"
	cError "github.com/banzaicloud/azure-aks-client/errors"
	"github.com/banzaicloud/azure-aks-client/utils"
	"regexp"
)

// GetManagedCluster creates a ManagedCluster type from CreateClusterRequest
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
							KeyData: utils.S(request.SSHPubKey),
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

// CreateClusterRequest describes a cluster creation request
type CreateClusterRequest struct {
	Name              string
	Location          string
	ResourceGroup     string
	KubernetesVersion string
	SSHPubKey         string
	Profiles          []containerservice.AgentPoolProfile
}

// Validate validates create request
func (c CreateClusterRequest) Validate() error {

	if len(c.Name) == 0 {
		return cError.ErrClusterNameEmpty
	} else if len(c.Name) >= 32 {
		return cError.ErrClusterNameTooLong
	}
	if isMatch, _ := regexp.MatchString(RegexpForName, c.Name); !isMatch {
		return cError.ErrClusterNameRegexp
	}

	return nil
}

// RegexpForName describes cluster name regexp
const RegexpForName = "^[a-z0-9_]{0,31}[a-z0-9]$"
