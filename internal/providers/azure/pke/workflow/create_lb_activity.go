package workflow

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/network/mgmt/network"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/goph/emperror"
	"go.uber.org/cadence/activity"
)

// CreateLoadBalancerActivityName is the default registration name of the activity
const CreateLoadBalancerActivityName = "pke-azure-create-load-balancer"

// CreateLoadBalancerActivity represents an activity for creating an Azure load balancer
type CreateLoadBalancerActivity struct {
	azureClientFactory *AzureClientFactory
}

// MakeCreateLoadBalancerActivity returns a new CreateLoadBalancerActivity
func MakeCreateLoadBalancerActivity(azureClientFactory *AzureClientFactory) CreateLoadBalancerActivity {
	return CreateLoadBalancerActivity{
		azureClientFactory: azureClientFactory,
	}
}

// CreateLoadBalancerActivityInput represents the input needed for executing a CreateLoadBalancerActivity
type CreateLoadBalancerActivityInput struct {
	Name                     string
	Location                 string
	SKU                      string
	BackendAddressPools      []BackendAddressPool
	FrontendIPConfigurations []FrontendIPConfiguration
	LoadBalancingRules       []LoadBalancingRule
	Probes                   []Probe

	ResourceGroupName string
	OrganizationID    uint
	ClusterName       string
	SecretID          string
}

type BackendAddressPool struct {
	Name string
}

type FrontendIPConfiguration struct {
	Name            string
	PublicIPAddress PublicIPAddress
	Zones           []string
}

type LoadBalancingRule struct {
	Name      string
	ProbeName string
}

type Probe struct {
	Name     string
	Port     int32
	Protocol string
}

type PublicIPAddress struct {
	Location string
	Name     string
	SKU      string
}

// Execute performs the activity
func (a CreateLoadBalancerActivity) Execute(ctx context.Context, input CreateLoadBalancerActivityInput) error {
	logger := activity.GetLogger(ctx).Sugar().With(
		"organization", input.OrganizationID,
		"cluster", input.ClusterName,
		"secret", input.SecretID,
		"resourceGroup", input.ResourceGroupName,
		"loadBalancer", input.Name,
	)

	keyvals := []interface{}{
		"resourceGroup", input.ResourceGroupName,
		"loadBalancer", input.Name,
	}

	logger.Info("create load balancer")

	cc, err := a.azureClientFactory.New(input.OrganizationID, input.SecretID)
	if err != nil {
		return emperror.Wrap(err, "failed to create cloud connection")
	}

	client := cc.GetLoadBalancersClient()

	var params network.LoadBalancer
	fillLoadBalancerParams(&params, input, client.SubscriptionID)

	future, err := client.CreateOrUpdate(ctx, input.ResourceGroupName, input.Name, params)
	if err != nil {
		return emperror.WrapWith(err, "sending request to create or update load balancer failed", keyvals...)
	}

	logger.Debug("waiting for the completion of create or update load balancer operation")

	err = future.WaitForCompletionRef(ctx, client.Client)
	if err != nil {
		return emperror.WrapWith(err, "waiting for the completion of create or update load balancer operation failed", keyvals...)
	}

	_, err = future.Result(client.LoadBalancersClient)
	if err != nil {
		return emperror.WrapWith(err, "getting load balancer create or update result failed", keyvals...)
	}

	return nil
}

func fillLoadBalancerParams(params *network.LoadBalancer, input CreateLoadBalancerActivityInput, subscriptionID string) {
	if params == nil {
		return
	}

	if params.LoadBalancerPropertiesFormat == nil {
		params.LoadBalancerPropertiesFormat = new(network.LoadBalancerPropertiesFormat)
	}

	if l := len(input.BackendAddressPools); l > 0 {
		backendAddressPools := make([]network.BackendAddressPool, l)
		for i, bap := range input.BackendAddressPools {
			backendAddressPools[i].Name = to.StringPtr(bap.Name)
		}
		params.LoadBalancerPropertiesFormat.BackendAddressPools = &backendAddressPools
	}

	if l := len(input.FrontendIPConfigurations); l > 0 {
		frontendIPConfigurations := make([]network.FrontendIPConfiguration, l)
		for i, fic := range input.FrontendIPConfigurations {
			frontendIPConfigurations[i] = network.FrontendIPConfiguration{
				Name: to.StringPtr(fic.Name),
				FrontendIPConfigurationPropertiesFormat: &network.FrontendIPConfigurationPropertiesFormat{
					PrivateIPAllocationMethod: network.Dynamic,
					PublicIPAddress: &network.PublicIPAddress{
						Name:     to.StringPtr(fic.PublicIPAddress.Name),
						Location: to.StringPtr(fic.PublicIPAddress.Location),
						PublicIPAddressPropertiesFormat: &network.PublicIPAddressPropertiesFormat{
							PublicIPAddressVersion:   network.IPv4,
							PublicIPAllocationMethod: network.Static,
						},
						Sku: &network.PublicIPAddressSku{
							Name: network.PublicIPAddressSkuName(fic.PublicIPAddress.SKU),
						},
					},
				},
			}
			if len(fic.Zones) > 0 {
				frontendIPConfigurations[i].Zones = to.StringSlicePtr(fic.Zones)
			}
		}
		params.LoadBalancerPropertiesFormat.FrontendIPConfigurations = &frontendIPConfigurations
	}

	if l := len(input.LoadBalancingRules); l > 0 {
		loadBalancingRules := make([]network.LoadBalancingRule, l)
		for i, lbr := range input.LoadBalancingRules {
			loadBalancingRules[i] = network.LoadBalancingRule{
				Name: to.StringPtr(lbr.Name),
			}
			if lbr.ProbeName != "" {
				if loadBalancingRules[i].LoadBalancingRulePropertiesFormat == nil {
					loadBalancingRules[i].LoadBalancingRulePropertiesFormat = new(network.LoadBalancingRulePropertiesFormat)
				}
				loadBalancingRules[i].LoadBalancingRulePropertiesFormat.Probe = &network.SubResource{
					ID: to.StringPtr(loadBalancerProbeID(subscriptionID, input.ResourceGroupName, input.Name, lbr.ProbeName)),
				}
			}
		}
		params.LoadBalancerPropertiesFormat.LoadBalancingRules = &loadBalancingRules
	}

	if l := len(input.Probes); l > 0 {
		probes := make([]network.Probe, l)
		for i, p := range input.Probes {
			probes[i] = network.Probe{
				Name: to.StringPtr(p.Name),
				ProbePropertiesFormat: &network.ProbePropertiesFormat{
					Port:     to.Int32Ptr(p.Port),
					Protocol: network.ProbeProtocol(p.Protocol),
				},
			}
		}
		params.LoadBalancerPropertiesFormat.Probes = &probes
	}

	params.Location = to.StringPtr(input.Location)

	if params.Sku == nil {
		params.Sku = new(network.LoadBalancerSku)
	}
	params.Sku.Name = network.LoadBalancerSkuName(input.SKU)

	params.Tags = *to.StringMapPtr(tagsFrom(getOwnedTag(input.ClusterName)))
}

func loadBalancerProbeID(subscriptionID, resourceGroupName, loadBalancerName, probeName string) string {
	return fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/loadBalancers/%s/probes/%s", subscriptionID, resourceGroupName, loadBalancerName, probeName)
}
