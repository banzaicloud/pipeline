package workflow

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/network/mgmt/network"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/goph/emperror"
	"go.uber.org/cadence/activity"
)

// CreateNSGActivityName is the default registration name of the activity
const CreateLBActivityName = "pke-azure-create-lb"

// CreateNSGActivity represents an activity for creating an Azure network security group
type CreateLBActivity struct {
	azureClientFactory *AzureClientFactory
}

// MakeCreateNSGActivity returns a new CreateNSGActivity
func MakeCreateLBActivity(azureClientFactory *AzureClientFactory) CreateLBActivity {
	return CreateLBActivity{
		azureClientFactory: azureClientFactory,
	}
}

type CreateLBActivityInput struct {
	Name              string
	Location          string
	Rules             []SecurityRule
	ResourceGroupName string
	OrganizationID    uint
	ClusterName       string
	SecretID          string
}

func (a CreateLBActivity) Execute(ctx context.Context, input CreateLBActivityInput) (string, error) {

	inboundIP := input.ResourceGroupName + "-pip-in"
	//outboundIP := input.ResourceGroupName + "-pip-out"
	lbName := input.ResourceGroupName + "-lb"

	logger := activity.GetLogger(ctx).Sugar().With(
		"organization", input.OrganizationID,
		"cluster", input.ClusterName,
		"secret", input.SecretID,
		"resourceGroup", input.ResourceGroupName,
		"nsgName", input.Name,
	)

	keyvals := []interface{}{
		"resourceGroup", input.ResourceGroupName,
		"lbName", input.Name,
	}

	logger.Info("create network security group")

	cc, err := a.azureClientFactory.New(input.OrganizationID, input.SecretID)
	if err != nil {
		return "", emperror.Wrap(err, "failed to create cloud connection")
	}

	ipClient := cc.GetPublicIPClient()
	// Create Inbound IP
	future, err := ipClient.CreateOrUpdate(
		ctx,
		input.ResourceGroupName,
		inboundIP,
		network.PublicIPAddress{
			Name:     to.StringPtr(inboundIP),
			Location: to.StringPtr(input.Location),
			PublicIPAddressPropertiesFormat: &network.PublicIPAddressPropertiesFormat{
				PublicIPAddressVersion:   network.IPv4,
				PublicIPAllocationMethod: network.Static,
			},
		},
	)
	if err != nil {
		return "", emperror.WrapWith(err, "sending request to create public ip failed", keyvals...)
	}

	// Create Outbound IP

	//
	logger.Debug("waiting for the completion of create or update public ip operation")

	err = future.WaitForCompletionRef(ctx, ipClient.Client)
	if err != nil {
		return "", emperror.WrapWith(err, "waiting for the completion of create or update public ip operation failed", keyvals...)
	}

	// TODO check what output we need
	pip, err := future.Result(ipClient.PublicIPAddressesClient)
	if err != nil {
		return "", emperror.WrapWith(err, "getting public ip create or update result failed", keyvals...)
	}

	lbClient := cc.GetLoadBalancerClient()
	future, err := lbClient.CreateOrUpdate(ctx,
		input.ResourceGroupName,
		lbName,
		network.LoadBalancer{
			Location: to.StringPtr(input.Location),
			LoadBalancerPropertiesFormat: &network.LoadBalancerPropertiesFormat{
				FrontendIPConfigurations: &[]network.FrontendIPConfiguration{
					{
						Name: &frontEndIPConfigName,
						FrontendIPConfigurationPropertiesFormat: &network.FrontendIPConfigurationPropertiesFormat{
							PrivateIPAllocationMethod: network.Dynamic,
							PublicIPAddress:           &pip,
						},
					},
				},
				BackendAddressPools: &[]network.BackendAddressPool{
					{
						Name: &backEndAddressPoolName,
					},
				},
				Probes: &[]network.Probe{
					{
						Name: &probeName,
						ProbePropertiesFormat: &network.ProbePropertiesFormat{
							Protocol:          network.ProbeProtocolHTTP,
							Port:              to.Int32Ptr(80),
							IntervalInSeconds: to.Int32Ptr(15),
							NumberOfProbes:    to.Int32Ptr(4),
							RequestPath:       to.StringPtr("healthprobe.aspx"),
						},
					},
				},
				LoadBalancingRules: &[]network.LoadBalancingRule{
					{
						Name: to.StringPtr("lbRule"),
						LoadBalancingRulePropertiesFormat: &network.LoadBalancingRulePropertiesFormat{
							Protocol:             network.TransportProtocolTCP,
							FrontendPort:         to.Int32Ptr(80),
							BackendPort:          to.Int32Ptr(80),
							IdleTimeoutInMinutes: to.Int32Ptr(4),
							EnableFloatingIP:     to.BoolPtr(false),
							LoadDistribution:     network.Default,
							FrontendIPConfiguration: &network.SubResource{
								ID: to.StringPtr(fmt.Sprintf("/%s/%s/frontendIPConfigurations/%s", idPrefix, lbName, frontEndIPConfigName)),
							},
							BackendAddressPool: &network.SubResource{
								ID: to.StringPtr(fmt.Sprintf("/%s/%s/backendAddressPools/%s", idPrefix, lbName, backEndAddressPoolName)),
							},
							Probe: &network.SubResource{
								ID: to.StringPtr(fmt.Sprintf("/%s/%s/probes/%s", idPrefix, lbName, probeName)),
							},
						},
					},
				},
				InboundNatRules: &[]network.InboundNatRule{
					{
						Name: to.StringPtr("natRule1"),
						InboundNatRulePropertiesFormat: &network.InboundNatRulePropertiesFormat{
							Protocol:             network.TransportProtocolTCP,
							FrontendPort:         to.Int32Ptr(21),
							BackendPort:          to.Int32Ptr(22),
							EnableFloatingIP:     to.BoolPtr(false),
							IdleTimeoutInMinutes: to.Int32Ptr(4),
							FrontendIPConfiguration: &network.SubResource{
								ID: to.StringPtr(fmt.Sprintf("/%s/%s/frontendIPConfigurations/%s", idPrefix, lbName, frontEndIPConfigName)),
							},
						},
					},
					{
						Name: to.StringPtr("natRule2"),
						InboundNatRulePropertiesFormat: &network.InboundNatRulePropertiesFormat{
							Protocol:             network.TransportProtocolTCP,
							FrontendPort:         to.Int32Ptr(23),
							BackendPort:          to.Int32Ptr(22),
							EnableFloatingIP:     to.BoolPtr(false),
							IdleTimeoutInMinutes: to.Int32Ptr(4),
							FrontendIPConfiguration: &network.SubResource{
								ID: to.StringPtr(fmt.Sprintf("/%s/%s/frontendIPConfigurations/%s", idPrefix, lbName, frontEndIPConfigName)),
							},
						},
					},
				},
			},
		})

	return "", nil
}
