package cluster

import (
	"os"
	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/banzaicloud/azure-aks-client/utils"
	clientAuthorization "github.com/banzaicloud/azure-aks-client/service/authorization"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/Azure/go-autorest/autorest"
	"github.com/banzaicloud/azure-aks-client/service/resources"
	"github.com/banzaicloud/azure-aks-client/service/network"
	"github.com/banzaicloud/azure-aks-client/service/containerservice"
	"github.com/banzaicloud/azure-aks-client/service/compute"
	"github.com/banzaicloud/azure-aks-client/service/subscriptions"
)

const AzureClientId = "AZURE_CLIENT_ID"
const AzureClientSecret = "AZURE_CLIENT_SECRET"
const AzureSubscriptionId = "AZURE_SUBSCRIPTION_ID"
const AzureTenantId = "AZURE_TENANT_ID"

type AKSCredential struct {
	ClientId       string
	ClientSecret   string
	SubscriptionId string
	TenantId       string
}

type Sdk struct {
	ServicePrincipal        *ServicePrincipal
	authorizer              autorest.Authorizer
	managedClusterClient    *containerservice.ManagedClustersClient
	vmSizeClient            *compute.VirtualMachineSizesClient
	subscriptionsClient     *subscriptions.Client
	containerServicesClient *containerservice.ContainerServicesClient
	virtualMachineClient    *compute.VirtualMachinesClient
	interfaceClient         *network.InterfacesClient
	virtualNetworksClient   *network.VirtualNetworksClient
	subnetClient            *network.SubnetClient
	ipClient                *network.IPClient
	securityGroupClient     *network.SecurityGroupsClient
	roleAssignmentsClient   *clientAuthorization.RoleAssignmentsClient
	roleDefinitionsClient   *clientAuthorization.RoleDefinitionsClient
	groupClient             *resources.ResourceGroupClient
}

type ServicePrincipal struct {
	ClientID           string
	ClientSecret       string
	SubscriptionID     string
	TenantId           string
	HashMap            map[string]string
	AuthenticatedToken *adal.ServicePrincipalToken
}

func (a *AKSCredential) Validate() error {
	msg := "missing credential: "
	if len(a.ClientId) == 0 {
		return utils.NewErr(msg + "clientId")
	}
	if len(a.ClientSecret) == 0 {
		return utils.NewErr(msg + "ClientSecret")
	}
	if len(a.SubscriptionId) == 0 {
		return utils.NewErr(msg + "SubscriptionId")
	}
	if len(a.TenantId) == 0 {
		return utils.NewErr(msg + "TenantId")
	}
	return nil
}

// CreateSdk create azure sdk which contains all required client
func CreateSdk(credentials *AKSCredential) (*Sdk, error) {
	var AKSCred *AKSCredential
	if credentials != nil {
		AKSCred = credentials
	} else {
		AKSCred = &AKSCredential{}
		AKSCred.ClientId = os.Getenv(AzureClientId)
		AKSCred.ClientSecret = os.Getenv(AzureClientSecret)
		AKSCred.SubscriptionId = os.Getenv(AzureSubscriptionId)
		AKSCred.TenantId = os.Getenv(AzureTenantId)
	}

	err := AKSCred.Validate()
	if err != nil {
		return nil, err
	}

	sdk := Sdk{
		ServicePrincipal: &ServicePrincipal{
			ClientID:       AKSCred.ClientId,
			ClientSecret:   AKSCred.ClientSecret,
			SubscriptionID: AKSCred.SubscriptionId,
			TenantId:       AKSCred.TenantId,
			HashMap: map[string]string{
				AzureClientId:       AKSCred.ClientId,
				AzureClientSecret:   AKSCred.ClientSecret,
				AzureSubscriptionId: AKSCred.SubscriptionId,
				AzureTenantId:       AKSCred.TenantId,
			},
		},
	}

	return &sdk, nil
}

// GetSubscriptionsClient returns SubscriptionsClient
func (sdk *Sdk) GetSubscriptionsClient() (*subscriptions.Client, error) {
	if sdk.subscriptionsClient == nil {
		authorizer, err := sdk.GetAuthorizer()
		if err != nil {
			return nil, err
		}

		sdk.subscriptionsClient = subscriptions.NewClient(authorizer, sdk.GetSubscriptionID())
	}
	return sdk.subscriptionsClient, nil
}

// GetManagedClusterClient returns ManagedClustersClient
func (sdk *Sdk) GetManagedClusterClient() (*containerservice.ManagedClustersClient, error) {
	if sdk.managedClusterClient == nil {
		authorizer, err := sdk.GetAuthorizer()
		if err != nil {
			return nil, err
		}

		sdk.managedClusterClient = containerservice.NewManagedClustersClient(authorizer, sdk.GetSubscriptionID())
	}
	return sdk.managedClusterClient, nil
}

// GetContainerServicesClient returns ContainerServicesClient
func (sdk *Sdk) GetContainerServicesClient() (*containerservice.ContainerServicesClient, error) {
	if sdk.containerServicesClient == nil {
		authorizer, err := sdk.GetAuthorizer()
		if err != nil {
			return nil, err
		}

		sdk.containerServicesClient = containerservice.NewContainerServicesClient(authorizer, sdk.GetSubscriptionID())
	}
	return sdk.containerServicesClient, nil
}

// GetVirtualMachineSizesClient returns VirtualMachineSizesClient
func (sdk *Sdk) GetVirtualMachineSizesClient() (*compute.VirtualMachineSizesClient, error) {
	if sdk.vmSizeClient == nil {
		authorizer, err := sdk.GetAuthorizer()
		if err != nil {
			return nil, err
		}

		sdk.vmSizeClient = compute.NewVirtualMachineSizesClient(authorizer, sdk.GetSubscriptionID())
	}
	return sdk.vmSizeClient, nil
}

// GetVirtualMachineClient returns VirtualMachinesClient
func (sdk *Sdk) GetVirtualMachineClient() (*compute.VirtualMachinesClient, error) {
	if sdk.virtualMachineClient == nil {
		authorizer, err := sdk.GetAuthorizer()
		if err != nil {
			return nil, err
		}

		sdk.virtualMachineClient = compute.NewVirtualMachinesClient(authorizer, sdk.GetSubscriptionID())
	}
	return sdk.virtualMachineClient, nil
}

// GetSecurityGroupsClient returns SecurityGroupsClient
func (sdk *Sdk) GetSecurityGroupsClient() (*network.SecurityGroupsClient, error) {
	if sdk.securityGroupClient == nil {
		authorizer, err := sdk.GetAuthorizer()
		if err != nil {
			return nil, err
		}

		sdk.securityGroupClient = network.NewSecurityGroupsClient(authorizer, sdk.GetSubscriptionID())
	}
	return sdk.securityGroupClient, nil
}

// GetIPClient returns IPClient
func (sdk *Sdk) GetIPClient() (*network.IPClient, error) {
	if sdk.ipClient == nil {
		authorizer, err := sdk.GetAuthorizer()
		if err != nil {
			return nil, err
		}

		sdk.ipClient = network.NewIPClient(authorizer, sdk.GetSubscriptionID())
	}
	return sdk.ipClient, nil
}

// GetSubnetClient returns SubnetClient
func (sdk *Sdk) GetSubnetClient() (*network.SubnetClient, error) {
	if sdk.subnetClient == nil {
		authorizer, err := sdk.GetAuthorizer()
		if err != nil {
			return nil, err
		}

		sdk.subnetClient = network.NewSubnetClient(authorizer, sdk.GetSubscriptionID())
	}
	return sdk.subnetClient, nil
}

// GetVirtualNetworksClient returns VirtualNetworksClient
func (sdk *Sdk) GetVirtualNetworksClient() (*network.VirtualNetworksClient, error) {
	if sdk.virtualNetworksClient == nil {
		authorizer, err := sdk.GetAuthorizer()
		if err != nil {
			return nil, err
		}

		sdk.virtualNetworksClient = network.NewVirtualNetworksClient(authorizer, sdk.GetSubscriptionID())
	}
	return sdk.virtualNetworksClient, nil
}

// GetInterfacesClient returns InterfacesClient
func (sdk *Sdk) GetInterfacesClient() (*network.InterfacesClient, error) {
	if sdk.interfaceClient == nil {
		authorizer, err := sdk.GetAuthorizer()
		if err != nil {
			return nil, err
		}

		sdk.interfaceClient = network.NewInterfacesClient(authorizer, sdk.GetSubscriptionID())
	}
	return sdk.interfaceClient, nil
}

// GetResourceGroupClient returns ResourceGroupClient
func (sdk *Sdk) GetResourceGroupClient() (*resources.ResourceGroupClient, error) {
	if sdk.groupClient == nil {
		authorizer, err := sdk.GetAuthorizer()
		if err != nil {
			return nil, err
		}

		sdk.groupClient = resources.NewResourceGroupClient(authorizer, sdk.GetSubscriptionID())
	}
	return sdk.groupClient, nil
}

// GetRoleDefinitionsClient returns RoleDefinitionsClient
func (sdk *Sdk) GetRoleDefinitionsClient() (*clientAuthorization.RoleDefinitionsClient, error) {
	if sdk.roleDefinitionsClient == nil {
		authorizer, err := sdk.GetAuthorizer()
		if err != nil {
			return nil, err
		}

		sdk.roleDefinitionsClient = clientAuthorization.NewRoleDefinitionClient(authorizer, sdk.GetSubscriptionID())
	}
	return sdk.roleDefinitionsClient, nil
}

// GetRoleAssignmentsClient returns RoleAssignmentsClient
func (sdk *Sdk) GetRoleAssignmentsClient() (*clientAuthorization.RoleAssignmentsClient, error) {
	if sdk.roleAssignmentsClient == nil {
		authorizer, err := sdk.GetAuthorizer()
		if err != nil {
			return nil, err
		}

		sdk.roleAssignmentsClient = clientAuthorization.NewRoleAssignmentsClient(authorizer, sdk.GetSubscriptionID())
	}
	return sdk.roleAssignmentsClient, nil
}

// GetSubscriptionID returns subscriptionID
func (sdk *Sdk) GetSubscriptionID() string {
	return sdk.ServicePrincipal.SubscriptionID
}

// GetAuthorizer returns the authorizer from client credentials.
func (sdk *Sdk) GetAuthorizer() (autorest.Authorizer, error) {
	if sdk.authorizer == nil {
		authorizer, err := auth.NewClientCredentialsConfig(sdk.ServicePrincipal.ClientID, sdk.ServicePrincipal.ClientSecret, sdk.ServicePrincipal.TenantId).Authorizer()
		if err != nil {
			return nil, err
		}
		sdk.authorizer = authorizer
	}

	return sdk.authorizer, nil
}
