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

package api

import (
	"testing"

	"github.com/banzaicloud/pipeline/internal/pke"
	"github.com/banzaicloud/pipeline/internal/providers/azure/pke/driver"
	"github.com/banzaicloud/pipeline/pkg/cluster"
	"gotest.tools/assert"
)

const (
	orgID  = 1
	userID = 1

	Name          = "test"
	SecretID      = "test-secret"
	SSHSecretID   = "ssh-secret"
	Location      = "test-location"
	ResourceGroup = "test-resource-group"
	Instancetype  = "azure-instance"
	Version       = "12.2.2"
	RBAC          = false
)

func TestToAzurePKEClusterCreationParams(t *testing.T) {
	var (
		Azuresubnet = AzureSubnet{
			Name: "test-subnet",
			CIDR: "1.1.1.1/16",
		}
		Nodepool = AzureNodePool{
			Labels:       nil,
			Name:         "nodepool1",
			Roles:        []string{"role"},
			Subnet:       Azuresubnet,
			Zones:        []string{"zone"},
			InstanceType: Instancetype,
			Autoscaling:  false,
			Count:        2,
			MinCount:     1,
			MaxCount:     3,
		}
		Azurenetwork = AzureNetwork{
			Name: "test-net",
			CIDR: "1.1.1.1/10",
		}
		cri = CRI{
			Runtime:       "containerd",
			RuntimeConfig: nil,
		}
		network = Network{
			PodCIDR:        "192.168.1.1/16",
			Provider:       "weave",
			ProviderConfig: nil,
			ServiceCIDR:    "11.11.1.1/16",
		}
		scaleOptions = ScaleOptions{
			Enabled:             false,
			DesiredCPU:          2,
			DesiredMEM:          2048,
			DesiredGPU:          0,
			OnDemandPCT:         55,
			Excludes:            nil,
			KeepDesiredCapacity: false,
		}
	)

	var conversionTest = []struct {
		Name string
		in   CreatePKEOnAzureClusterRequest
		out  driver.AzurePKEClusterCreationParams
	}{
		{
			Name: "EmptyRequest",
			in:   CreatePKEOnAzureClusterRequest{},
			out: driver.AzurePKEClusterCreationParams{
				OrganizationID: orgID,
				CreatedBy:      userID,
				NodePools:      make([]driver.NodePool, 0),
			},
		},
		{
			Name: "FullRequest",
			in: CreatePKEOnAzureClusterRequest{
				CreateClusterRequestBase: CreateClusterRequestBase{
					Name:         Name,
					Features:     nil,
					SecretID:     SecretID,
					SSHSecretID:  SSHSecretID,
					ScaleOptions: scaleOptions,
					Type:         PKEOnAzure,
				},
				Location:      Location,
				ResourceGroup: ResourceGroup,
				NodePools:     []AzureNodePool{Nodepool},
				Kubernetes: Kubernetes{
					CRI:     cri,
					Network: network,
					RBAC:    RBAC,
					Version: Version,
				},
				Network: Azurenetwork,
			},
			out: driver.AzurePKEClusterCreationParams{
				CreatedBy: userID,
				Kubernetes: pke.Kubernetes{
					Version: Version,
					RBAC:    RBAC,
					Network: pke.Network{
						ServiceCIDR:    network.ServiceCIDR,
						PodCIDR:        network.PodCIDR,
						Provider:       network.Provider,
						ProviderConfig: network.ProviderConfig,
					},
					CRI: pke.CRI{
						Runtime:       cri.Runtime,
						RuntimeConfig: cri.RuntimeConfig,
					},
				},
				Name: Name,
				Network: driver.VirtualNetwork{
					Name:     Azurenetwork.Name,
					CIDR:     Azurenetwork.CIDR,
					Location: Location,
				},
				NodePools: []driver.NodePool{
					{
						CreatedBy:    userID,
						Name:         Nodepool.Name,
						InstanceType: Nodepool.InstanceType,
						Subnet: driver.Subnet{
							Name: Azuresubnet.Name,
							CIDR: Azuresubnet.CIDR,
						},
						Zones:       Nodepool.Zones,
						Roles:       Nodepool.Roles,
						Labels:      Nodepool.Labels,
						Autoscaling: Nodepool.Autoscaling,
						Count:       Nodepool.Count,
						Min:         Nodepool.MinCount,
						Max:         Nodepool.MaxCount,
					},
				},
				OrganizationID: orgID,
				ResourceGroup:  ResourceGroup,
				ScaleOptions: cluster.ScaleOptions{
					Enabled:             scaleOptions.Enabled,
					DesiredCpu:          scaleOptions.DesiredCPU,
					DesiredMem:          scaleOptions.DesiredMEM,
					DesiredGpu:          scaleOptions.DesiredGPU,
					OnDemandPct:         scaleOptions.OnDemandPCT,
					Excludes:            scaleOptions.Excludes,
					KeepDesiredCapacity: scaleOptions.KeepDesiredCapacity,
				},
				SecretID:    SecretID,
				SSHSecretID: SSHSecretID,
			},
		},
	}
	for _, tt := range conversionTest {
		t.Run(tt.Name, func(t *testing.T) {
			out := tt.in.ToAzurePKEClusterCreationParams(orgID, userID)
			assert.DeepEqual(t, tt.out, out)
		})
	}
}
