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

	"github.com/stretchr/testify/assert"

	"github.com/banzaicloud/pipeline/client"
	intCluster "github.com/banzaicloud/pipeline/internal/cluster"
	"github.com/banzaicloud/pipeline/internal/pke"
	"github.com/banzaicloud/pipeline/internal/providers/azure/pke/driver"
	"github.com/banzaicloud/pipeline/pkg/cluster"
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
		Azuresubnet = client.PkeOnAzureNodePoolSubnet{
			Name: "test-subnet",
			Cidr: "1.1.1.1/16",
		}
		Nodepool = client.PkeOnAzureNodePool{
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
		Azurenetwork = client.PkeOnAzureClusterNetwork{
			Name: "test-net",
			Cidr: "1.1.1.1/10",
		}
		cri = client.CreatePkeClusterKubernetesCri{
			Runtime:       "containerd",
			RuntimeConfig: nil,
		}
		network = client.CreatePkeClusterKubernetesNetwork{
			PodCIDR:        "192.168.1.1/16",
			Provider:       "weave",
			ProviderConfig: nil,
			ServiceCIDR:    "11.11.1.1/16",
		}
		scaleOptions = client.ScaleOptions{
			Enabled:             false,
			DesiredCpu:          2,
			DesiredMem:          2048,
			DesiredGpu:          0,
			OnDemandPct:         55,
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
				Features:       []intCluster.Feature{},
				NodePools:      []driver.NodePool{},
			},
		},
		{
			Name: "FullRequest",
			in: CreatePKEOnAzureClusterRequest{
				Name:          Name,
				Features:      nil,
				SecretId:      SecretID,
				SshSecretId:   SSHSecretID,
				ScaleOptions:  scaleOptions,
				Type:          PKEOnAzure,
				Location:      Location,
				ResourceGroup: ResourceGroup,
				Nodepools:     []client.PkeOnAzureNodePool{Nodepool},
				Kubernetes: client.CreatePkeClusterKubernetes{
					Cri:     cri,
					Network: network,
					Rbac:    RBAC,
					Oidc:    client.CreatePkeClusterKubernetesOidc{Enabled: true},
					Version: Version,
				},
				Network: Azurenetwork,
			},
			out: driver.AzurePKEClusterCreationParams{
				CreatedBy: userID,
				Features:  []intCluster.Feature{},
				Kubernetes: pke.Kubernetes{
					Version: Version,
					RBAC:    RBAC,
					OIDC:    pke.OIDC{Enabled: true},
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
					CIDR:     Azurenetwork.Cidr,
					Location: Location,
				},
				NodePools: []driver.NodePool{
					{
						CreatedBy:    userID,
						Name:         Nodepool.Name,
						InstanceType: Nodepool.InstanceType,
						Subnet: driver.Subnet{
							Name: Azuresubnet.Name,
							CIDR: Azuresubnet.Cidr,
						},
						Zones:       Nodepool.Zones,
						Roles:       Nodepool.Roles,
						Labels:      Nodepool.Labels,
						Autoscaling: Nodepool.Autoscaling,
						Count:       int(Nodepool.Count),
						Min:         int(Nodepool.MinCount),
						Max:         int(Nodepool.MaxCount),
					},
				},
				OrganizationID: orgID,
				ResourceGroup:  ResourceGroup,
				ScaleOptions: cluster.ScaleOptions{
					Enabled:             scaleOptions.Enabled,
					DesiredCpu:          scaleOptions.DesiredCpu,
					DesiredMem:          scaleOptions.DesiredMem,
					DesiredGpu:          int(scaleOptions.DesiredGpu),
					OnDemandPct:         int(scaleOptions.OnDemandPct),
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
			assert.Equal(t, tt.out, out)
		})
	}
}
