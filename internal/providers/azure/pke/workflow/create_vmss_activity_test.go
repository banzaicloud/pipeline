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
	"strings"
	"testing"
	"text/template"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-10-01/compute"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/stretchr/testify/assert"
)

func TestGetCreateOrUpdateVirtualMachineScaleSetParams(t *testing.T) {
	t.Run("typical input", func(t *testing.T) {
		input := CreateVMSSActivityInput{
			OrganizationID:    1,
			ClusterID:         123,
			SecretID:          "0123456789abcdefghijklmnopqrstuvwxyz",
			ClusterName:       "test-cluster",
			ResourceGroupName: "test-rg",
			ScaleSet: VirtualMachineScaleSet{
				AdminUsername: "test-admin",
				Image: Image{
					Offer:     "CentOS-CI",
					Publisher: "OpenLogic",
					SKU:       "7-CI",
					Version:   "7.6.20190306",
				},
				InstanceCount:           int64(42),
				InstanceType:            "Standard_B2s",
				LBBackendAddressPoolIDs: []string{"/subscriptions/test-subscription/resourceGroups/test-rg/providers/Microsoft.Network/loadBalancers/test-lb/backendAddressPools/test-bap"},
				LBInboundNATPoolIDs:     []string{"/subscriptions/test-subscription/resourceGroups/test-rg/providers/Microsoft.Network/loadBalancers/test-lb/inboundNatPools/test-inp"},
				Location:                "test-location",
				Name:                    "test-vmss",
				NetworkSecurityGroupID:  "/subscriptions/test-subscription/resourceGroups/test-rg/providers/Microsoft.Network/networkSecurityGroups/test-nsg",
				SSHPublicKey:            "ssh-rsa 2048bitBASE64key test-key",
				SubnetID:                "/subscriptions/test-subscription/resourceGroups/test-rg/providers/Microsoft.Network/virtualNetworks/test-vnet/subnets/test-subnet",
				UserDataScriptTemplate:  "#!/bin/bash\necho \"{{ .Message }}\" > {{ .FilePath }}",
				UserDataScriptParams: map[string]string{
					"Message":  "I was here",
					"FilePath": "/tmp/where-i-was",
				},
				Zones: []string{"1", "2", "3"},
			},
		}
		expected := compute.VirtualMachineScaleSet{
			Identity: &compute.VirtualMachineScaleSetIdentity{
				Type: compute.ResourceIdentityTypeSystemAssigned,
			},
			Location: to.StringPtr("test-location"),
			Sku: &compute.Sku{
				Capacity: to.Int64Ptr(42),
				Name:     to.StringPtr("Standard_B2s"),
			},
			Tags: map[string]*string{
				"kubernetesCluster-test-cluster": to.StringPtr("owned"),
				"banzaicloud-pipeline-managed":   to.StringPtr("true"),
			},
			VirtualMachineScaleSetProperties: &compute.VirtualMachineScaleSetProperties{
				Overprovision: to.BoolPtr(false),
				UpgradePolicy: &compute.UpgradePolicy{Mode: compute.Manual},
				VirtualMachineProfile: &compute.VirtualMachineScaleSetVMProfile{
					NetworkProfile: &compute.VirtualMachineScaleSetNetworkProfile{
						NetworkInterfaceConfigurations: &[]compute.VirtualMachineScaleSetNetworkConfiguration{
							{
								Name: to.StringPtr("test-vmss-nic-1"),
								VirtualMachineScaleSetNetworkConfigurationProperties: &compute.VirtualMachineScaleSetNetworkConfigurationProperties{
									Primary:                     to.BoolPtr(true),
									EnableIPForwarding:          to.BoolPtr(true),
									EnableAcceleratedNetworking: to.BoolPtr(false),
									IPConfigurations: &[]compute.VirtualMachineScaleSetIPConfiguration{
										{
											Name: to.StringPtr("test-vmss-pip-1"),
											VirtualMachineScaleSetIPConfigurationProperties: &compute.VirtualMachineScaleSetIPConfigurationProperties{
												Primary: to.BoolPtr(true),
												LoadBalancerBackendAddressPools: &[]compute.SubResource{
													{
														ID: to.StringPtr("/subscriptions/test-subscription/resourceGroups/test-rg/providers/Microsoft.Network/loadBalancers/test-lb/backendAddressPools/test-bap"),
													},
												},
												LoadBalancerInboundNatPools: &[]compute.SubResource{
													{
														ID: to.StringPtr("/subscriptions/test-subscription/resourceGroups/test-rg/providers/Microsoft.Network/loadBalancers/test-lb/inboundNatPools/test-inp"),
													},
												},
												Subnet: &compute.APIEntityReference{
													ID: to.StringPtr("/subscriptions/test-subscription/resourceGroups/test-rg/providers/Microsoft.Network/virtualNetworks/test-vnet/subnets/test-subnet"),
												},
											},
										},
									},
									NetworkSecurityGroup: &compute.SubResource{
										ID: to.StringPtr("/subscriptions/test-subscription/resourceGroups/test-rg/providers/Microsoft.Network/networkSecurityGroups/test-nsg"),
									},
								},
							},
						},
					},
					OsProfile: &compute.VirtualMachineScaleSetOSProfile{
						ComputerNamePrefix: to.StringPtr("test-vmss"),
						AdminUsername:      to.StringPtr("test-admin"),
						CustomData:         to.StringPtr("IyEvYmluL2Jhc2gKZWNobyAiSSB3YXMgaGVyZSIgPiAvdG1wL3doZXJlLWktd2Fz"),
						LinuxConfiguration: &compute.LinuxConfiguration{
							DisablePasswordAuthentication: to.BoolPtr(true),
							SSH: &compute.SSHConfiguration{
								PublicKeys: &[]compute.SSHPublicKey{
									{
										Path:    to.StringPtr("/home/test-admin/.ssh/authorized_keys"),
										KeyData: to.StringPtr("ssh-rsa 2048bitBASE64key test-key"),
									},
								},
							},
						},
					},
					StorageProfile: &compute.VirtualMachineScaleSetStorageProfile{
						ImageReference: &compute.ImageReference{
							Offer:     to.StringPtr("CentOS-CI"),
							Publisher: to.StringPtr("OpenLogic"),
							Sku:       to.StringPtr("7-CI"),
							Version:   to.StringPtr("7.6.20190306"),
						},
						OsDisk: &compute.VirtualMachineScaleSetOSDisk{
							Caching:      compute.CachingTypesReadWrite,
							OsType:       compute.Linux,
							CreateOption: compute.DiskCreateOptionTypesFromImage,
							ManagedDisk: &compute.VirtualMachineScaleSetManagedDiskParameters{
								StorageAccountType: compute.StorageAccountTypesStandardLRS,
							},
							DiskSizeGB: to.Int32Ptr(128),
						},
					},
				},
			},
			Zones: &[]string{"1", "2", "3"},
		}
		var userDataScript strings.Builder
		assert.NoError(t, template.Must(template.New("TestUserTemplate").Parse(input.ScaleSet.UserDataScriptTemplate)).Execute(&userDataScript, input.ScaleSet.UserDataScriptParams))
		result := input.getCreateOrUpdateVirtualMachineScaleSetParams(userDataScript.String())
		assert.Equal(t, expected, result)
	})
}
