// Copyright © 2019 Banzai Cloud
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package workflow

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"text/template"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-10-01/compute"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/banzaicloud/pipeline/internal/providers/pke/pkeworkflow/pkeworkflowadapter"
	"github.com/goph/emperror"
	"go.uber.org/cadence/activity"
)

// CreateVMSSActivityName is the default registration name of the activity
const CreateVMSSActivityName = "pke-azure-create-vmss"

// CreateVMSSActivity represents an activity for creating an Azure virtual machine scale set
type CreateVMSSActivity struct {
	azureClientFactory *AzureClientFactory
	tokenGenerator     *pkeworkflowadapter.TokenGenerator
}

// MakeCreateVMSSActivity returns a new CreateVMSSActivity
func MakeCreateVMSSActivity(azureClientFactory *AzureClientFactory, tokenGenerator *pkeworkflowadapter.TokenGenerator) CreateVMSSActivity {
	return CreateVMSSActivity{
		azureClientFactory: azureClientFactory,
		tokenGenerator:     tokenGenerator,
	}
}

// CreateVMSSActivityInput represents the input needed for executing a CreateVMSSActivity
type CreateVMSSActivityInput struct {
	OrganizationID    uint
	ClusterID         uint
	SecretID          string
	ClusterName       string
	ResourceGroupName string
	ScaleSet          VirtualMachineScaleSet
}

// VirtualMachineScaleSet represents an Azure virtual machine scale set
type VirtualMachineScaleSet struct {
	AdminUsername           string
	Image                   Image
	InstanceCount           int64
	InstanceType            string
	LBBackendAddressPoolIDs []string
	LBInboundNATPoolID      string
	Location                string
	Name                    string
	NetworkSecurityGroupID  string
	SSHPublicKey            string
	SubnetID                string
	UserDataScriptParams    map[string]string
	UserDataScriptTemplate  string
	Zones                   []string
}

type Image struct {
	Offer     string
	Publisher string
	SKU       string
	Version   string
}

type CreateVMSSActivityOutput struct {
	PrincipalID string
}

// Execute performs the activity
func (a CreateVMSSActivity) Execute(ctx context.Context, input CreateVMSSActivityInput) (output CreateVMSSActivityOutput, err error) {
	logger := activity.GetLogger(ctx).Sugar().With(
		"organization", input.OrganizationID,
		"cluster", input.ClusterName,
		"secret", input.SecretID,
		"resourceGroup", input.ResourceGroupName,
		"vmssName", input.ScaleSet.Name,
	)

	keyvals := []interface{}{
		"resourceGroup", input.ResourceGroupName,
		"vmssName", input.ScaleSet.Name,
	}

	logger.Info("create virtual machine scale set")

	userDataScriptTemplate, err := template.New(input.ScaleSet.Name + "UserDataScript").Parse(input.ScaleSet.UserDataScriptTemplate)
	if err != nil {
		return
	}

	_, token, err := a.tokenGenerator.GenerateClusterToken(input.OrganizationID, input.ClusterID)
	if err != nil {
		return
	}

	input.ScaleSet.UserDataScriptParams["PipelineToken"] = token

	var userDataScript strings.Builder
	err = userDataScriptTemplate.Execute(&userDataScript, input.ScaleSet.UserDataScriptParams)
	if err = emperror.Wrap(err, "failed to execute user data script template"); err != nil {
		return
	}

	cc, err := a.azureClientFactory.New(input.OrganizationID, input.SecretID)
	if err = emperror.Wrap(err, "failed to create cloud connection"); err != nil {
		return
	}

	client := cc.GetVirtualMachineScaleSetsClient()

	params := input.getCreateOrUpdateVirtualMachineScaleSetParams(userDataScript.String())

	logger.Debug("sending request to create or update virtual machine scale set")

	future, err := client.CreateOrUpdate(ctx, input.ResourceGroupName, input.ScaleSet.Name, params)
	if err = emperror.WrapWith(err, "sending request to create or update virtual machine scale set failed", keyvals...); err != nil {
		return
	}

	logger.Debug("waiting for the completion of create or update virtual machine scale set operation")

	err = future.WaitForCompletionRef(ctx, client.Client)
	if err = emperror.WrapWith(err, "waiting for the completion of create or update virtual machine scale set operation failed", keyvals...); err != nil {
		return
	}

	vmss, err := future.Result(client.VirtualMachineScaleSetsClient)
	if err = emperror.WrapWith(err, "getting virtual machine scale set create or update result failed", keyvals...); err != nil {
		return
	}

	if vmss.Identity != nil {
		output.PrincipalID = to.String(vmss.Identity.PrincipalID)
	}

	return
}

func (input CreateVMSSActivityInput) getCreateOrUpdateVirtualMachineScaleSetParams(UserDataScript string) compute.VirtualMachineScaleSet {
	var bapRefs []compute.SubResource
	if input.ScaleSet.LBBackendAddressPoolIDs != nil {
		for _, id := range input.ScaleSet.LBBackendAddressPoolIDs {
			if id != "" {
				bapRef := compute.SubResource{
					ID: to.StringPtr(id),
				}
				bapRefs = append(bapRefs, bapRef)
			}
		}
	}
	var inpRefs *[]compute.SubResource
	if input.ScaleSet.LBInboundNATPoolID != "" {
		inpRefs = &[]compute.SubResource{
			{
				ID: to.StringPtr(input.ScaleSet.LBInboundNATPoolID),
			},
		}
	}
	var nsgRef *compute.SubResource
	if input.ScaleSet.NetworkSecurityGroupID != "" {
		nsgRef = &compute.SubResource{
			ID: to.StringPtr(input.ScaleSet.NetworkSecurityGroupID),
		}
	}
	storageAccountType := compute.StorageAccountTypesStandardLRS

	return compute.VirtualMachineScaleSet{
		Identity: &compute.VirtualMachineScaleSetIdentity{
			Type: compute.ResourceIdentityTypeSystemAssigned,
		},
		Location: to.StringPtr(input.ScaleSet.Location),
		Sku: &compute.Sku{
			Capacity: to.Int64Ptr(input.ScaleSet.InstanceCount),
			Name:     to.StringPtr(input.ScaleSet.InstanceType),
		},
		Tags: *to.StringMapPtr(getOwnedTag(input.ClusterName).Map()),
		VirtualMachineScaleSetProperties: &compute.VirtualMachineScaleSetProperties{
			Overprovision: to.BoolPtr(false),
			UpgradePolicy: &compute.UpgradePolicy{
				Mode: compute.Manual,
			},
			VirtualMachineProfile: &compute.VirtualMachineScaleSetVMProfile{
				NetworkProfile: &compute.VirtualMachineScaleSetNetworkProfile{
					NetworkInterfaceConfigurations: &[]compute.VirtualMachineScaleSetNetworkConfiguration{
						{
							Name: to.StringPtr(fmt.Sprintf("%s-nic-1", input.ScaleSet.Name)),
							VirtualMachineScaleSetNetworkConfigurationProperties: &compute.VirtualMachineScaleSetNetworkConfigurationProperties{
								Primary:                     to.BoolPtr(true),
								EnableIPForwarding:          to.BoolPtr(true),
								EnableAcceleratedNetworking: to.BoolPtr(supportsAcceleratedNetworking(input.ScaleSet.InstanceType)),
								IPConfigurations: &[]compute.VirtualMachineScaleSetIPConfiguration{
									{
										Name: to.StringPtr(fmt.Sprintf("%s-pip-1", input.ScaleSet.Name)),
										VirtualMachineScaleSetIPConfigurationProperties: &compute.VirtualMachineScaleSetIPConfigurationProperties{
											Primary:                         to.BoolPtr(true),
											LoadBalancerBackendAddressPools: &bapRefs,
											LoadBalancerInboundNatPools:     inpRefs,
											Subnet: &compute.APIEntityReference{
												ID: to.StringPtr(input.ScaleSet.SubnetID),
											},
										},
									},
								},
								NetworkSecurityGroup: nsgRef,
							},
						},
					},
				},
				OsProfile: &compute.VirtualMachineScaleSetOSProfile{
					ComputerNamePrefix: to.StringPtr(input.ScaleSet.Name),
					AdminUsername:      to.StringPtr(input.ScaleSet.AdminUsername),
					CustomData:         to.StringPtr(base64.StdEncoding.EncodeToString([]byte(UserDataScript))),
					LinuxConfiguration: &compute.LinuxConfiguration{
						DisablePasswordAuthentication: to.BoolPtr(true),
						SSH: &compute.SSHConfiguration{
							PublicKeys: &[]compute.SSHPublicKey{
								{
									KeyData: to.StringPtr(input.ScaleSet.SSHPublicKey),
									Path:    to.StringPtr(fmt.Sprintf("/home/%s/.ssh/authorized_keys", input.ScaleSet.AdminUsername)),
								},
							},
						},
					},
				},
				StorageProfile: &compute.VirtualMachineScaleSetStorageProfile{
					ImageReference: &compute.ImageReference{
						Offer:     to.StringPtr(input.ScaleSet.Image.Offer),
						Publisher: to.StringPtr(input.ScaleSet.Image.Publisher),
						Sku:       to.StringPtr(input.ScaleSet.Image.SKU),
						Version:   to.StringPtr(input.ScaleSet.Image.Version),
					},
					OsDisk: &compute.VirtualMachineScaleSetOSDisk{
						CreateOption: compute.DiskCreateOptionTypesFromImage,
						ManagedDisk: &compute.VirtualMachineScaleSetManagedDiskParameters{
							StorageAccountType: storageAccountType,
						},
						DiskSizeGB: to.Int32Ptr(128),
						Caching:    compute.CachingTypesReadWrite,
						OsType:     compute.Linux,
					},
				},
			},
		},
		Zones: to.StringSlicePtr(input.ScaleSet.Zones),
	}
}

// supportsAcceleratedNetworking check if the instanceType supports the Accelerated Networking
// https://github.com/Azure/acs-engine/blob/master/pkg/helpers/helpers.go#L118
func supportsAcceleratedNetworking(instanceType string) bool {
	// TODO: ideally this information should come from CloudInfo
	switch instanceType {
	case "Standard_D3_v2", "Standard_D12_v2", "Standard_D3_v2_Promo", "Standard_D12_v2_Promo",
		"Standard_DS3_v2", "Standard_DS12_v2", "Standard_DS13-4_v2", "Standard_DS14-4_v2",
		"Standard_DS3_v2_Promo", "Standard_DS12_v2_Promo", "Standard_DS13-4_v2_Promo",
		"Standard_DS14-4_v2_Promo", "Standard_F4", "Standard_F4s", "Standard_D8_v3", "Standard_D8s_v3",
		"Standard_D32-8s_v3", "Standard_E8_v3", "Standard_E8s_v3", "Standard_D3_v2_ABC",
		"Standard_D12_v2_ABC", "Standard_F4_ABC", "Standard_F8s_v2", "Standard_D4_v2",
		"Standard_D13_v2", "Standard_D4_v2_Promo", "Standard_D13_v2_Promo", "Standard_DS4_v2",
		"Standard_DS13_v2", "Standard_DS14-8_v2", "Standard_DS4_v2_Promo", "Standard_DS13_v2_Promo",
		"Standard_DS14-8_v2_Promo", "Standard_F8", "Standard_F8s", "Standard_M64-16ms", "Standard_D16_v3",
		"Standard_D16s_v3", "Standard_D32-16s_v3", "Standard_D64-16s_v3", "Standard_E16_v3",
		"Standard_E16s_v3", "Standard_E32-16s_v3", "Standard_D4_v2_ABC", "Standard_D13_v2_ABC",
		"Standard_F8_ABC", "Standard_F16s_v2", "Standard_D5_v2", "Standard_D14_v2", "Standard_D5_v2_Promo",
		"Standard_D14_v2_Promo", "Standard_DS5_v2", "Standard_DS14_v2", "Standard_DS5_v2_Promo",
		"Standard_DS14_v2_Promo", "Standard_F16", "Standard_F16s", "Standard_M64-32ms",
		"Standard_M128-32ms", "Standard_D32_v3", "Standard_D32s_v3", "Standard_D64-32s_v3",
		"Standard_E32_v3", "Standard_E32s_v3", "Standard_E32-8s_v3", "Standard_E32-16_v3",
		"Standard_D5_v2_ABC", "Standard_D14_v2_ABC", "Standard_F16_ABC", "Standard_F32s_v2",
		"Standard_D15_v2", "Standard_D15_v2_Promo", "Standard_D15_v2_Nested", "Standard_DS15_v2",
		"Standard_DS15_v2_Promo", "Standard_DS15_v2_Nested", "Standard_D40_v3", "Standard_D40s_v3",
		"Standard_D15_v2_ABC", "Standard_M64ms", "Standard_M64s", "Standard_M128-64ms",
		"Standard_D64_v3", "Standard_D64s_v3", "Standard_E64_v3", "Standard_E64s_v3", "Standard_E64-16s_v3",
		"Standard_E64-32s_v3", "Standard_F64s_v2", "Standard_F72s_v2", "Standard_M128s", "Standard_M128ms",
		"Standard_L8s_v2", "Standard_L16s_v2", "Standard_L32s_v2", "Standard_L64s_v2", "Standard_L96s_v2",
		"SQLGL", "SQLGLCore", "Standard_D4_v3", "Standard_D4s_v3", "Standard_D2_v2", "Standard_DS2_v2",
		"Standard_E4_v3", "Standard_E4s_v3", "Standard_F2", "Standard_F2s", "Standard_F4s_v2",
		"Standard_D11_v2", "Standard_DS11_v2", "AZAP_Performance_ComputeV17C", "Standard_PB6s",
		"Standard_PB12s", "Standard_PB24s":
		return true
	default:
		return false
	}
}
