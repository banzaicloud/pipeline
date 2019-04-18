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
	"encoding/base64"
	"testing"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-10-01/compute"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/stretchr/testify/assert"
)

func TestGetCreateOrUpdateVirtualMachineScaleSetParams(t *testing.T) {
	t.Run("typical input", func(t *testing.T) {
		input := CreateVMSSActivityInput{
			OrganizationID:    1,
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
				InstanceCount:          int64(42),
				InstanceType:           "Standard_B2s",
				LBBackendAddressPoolID: "/subscriptions/test-subscription/resourceGroups/test-rg/providers/Microsoft.Network/loadBalancers/test-lb/backendAddressPools/test-bap",
				LBInboundNATPoolID:     "/subscriptions/test-subscription/resourceGroups/test-rg/providers/Microsoft.Network/loadBalancers/test-lb/inboundNatPools/test-inp",
				Location:               "test-location",
				Name:                   "test-vmss",
				NetworkSecurityGroupID: "/subscriptions/test-subscription/resourceGroups/test-rg/providers/Microsoft.Network/networkSecurityGroups/test-nsg",
				SSHPublicKey:           "ssh-rsa 2048bitBASE64key test-key",
				SubnetID:               "/subscriptions/test-subscription/resourceGroups/test-rg/providers/Microsoft.Network/virtualNetworks/test-vnet/subnets/test-subnet",
				UserDataScript:         base64.StdEncoding.EncodeToString([]byte("#!/bin/bash\necho \"I was here\" > /tmp/where-was-i")),
				Zones:                  []string{"1", "2", "3"},
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
			},
			VirtualMachineScaleSetProperties: &compute.VirtualMachineScaleSetProperties{
				VirtualMachineProfile: &compute.VirtualMachineScaleSetVMProfile{
					NetworkProfile: &compute.VirtualMachineScaleSetNetworkProfile{
						NetworkInterfaceConfigurations: &[]compute.VirtualMachineScaleSetNetworkConfiguration{
							{
								VirtualMachineScaleSetNetworkConfigurationProperties: &compute.VirtualMachineScaleSetNetworkConfigurationProperties{
									IPConfigurations: &[]compute.VirtualMachineScaleSetIPConfiguration{
										{
											VirtualMachineScaleSetIPConfigurationProperties: &compute.VirtualMachineScaleSetIPConfigurationProperties{
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
						AdminUsername: to.StringPtr("test-admin"),
						CustomData:    to.StringPtr("IyEvYmluL2Jhc2gKZWNobyAiSSB3YXMgaGVyZSIgPiAvdG1wL3doZXJlLXdhcy1p"),
						LinuxConfiguration: &compute.LinuxConfiguration{
							DisablePasswordAuthentication: to.BoolPtr(true),
							SSH: &compute.SSHConfiguration{
								PublicKeys: &[]compute.SSHPublicKey{
									{
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
							CreateOption: compute.DiskCreateOptionTypesFromImage,
							ManagedDisk: &compute.VirtualMachineScaleSetManagedDiskParameters{
								StorageAccountType: compute.StorageAccountTypesStandardLRS,
							},
						},
					},
				},
			},
			Zones: &[]string{"1", "2", "3"},
		}
		result := input.getCreateOrUpdateVirtualMachineScaleSetParams()
		assert.Equal(t, expected, result)
	})
}
