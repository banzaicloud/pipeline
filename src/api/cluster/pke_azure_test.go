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

	"github.com/banzaicloud/pipeline/.gen/pipeline/pipeline"
	"github.com/banzaicloud/pipeline/internal/pke"
	azurePke "github.com/banzaicloud/pipeline/internal/providers/azure/pke"
	"github.com/banzaicloud/pipeline/internal/providers/azure/pke/driver"
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
		Azuresubnet = pipeline.PkeonAzureNodePoolSubnet{
			Name: "test-subnet",
			Cidr: "1.1.1.1/16",
		}
		Nodepool = pipeline.PkeonAzureNodePool{
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
		Azurenetwork = pipeline.PkeonAzureClusterNetwork{
			Name: "test-net",
			Cidr: "1.1.1.1/10",
		}
		cri = pipeline.CreatePkeClusterKubernetesCri{
			Runtime:       "containerd",
			RuntimeConfig: nil,
		}
		network = pipeline.CreatePkeClusterKubernetesNetwork{
			PodCIDR:        "192.168.1.1/16",
			Provider:       "weave",
			ProviderConfig: nil,
			ServiceCIDR:    "11.11.1.1/16",
		}
	)

	conversionTest := []struct {
		Name string
		in   CreatePKEOnAzureClusterRequest
		out  driver.ClusterCreationParams
	}{
		{
			Name: "EmptyRequest",
			in:   CreatePKEOnAzureClusterRequest{},
			out: driver.ClusterCreationParams{
				OrganizationID: orgID,
				CreatedBy:      userID,
				NodePools:      []driver.NodePool{},
			},
		},
		{
			Name: "FullRequest",
			in: CreatePKEOnAzureClusterRequest{
				Name:          Name,
				SecretId:      SecretID,
				SshSecretId:   SSHSecretID,
				Type:          PKEOnAzure,
				Location:      Location,
				ResourceGroup: ResourceGroup,
				Nodepools:     []pipeline.PkeonAzureNodePool{Nodepool},
				Kubernetes: pipeline.CreatePkeClusterKubernetes{
					Cri:     cri,
					Network: network,
					Rbac:    RBAC,
					Oidc:    pipeline.CreatePkeClusterKubernetesOidc{Enabled: true},
					Version: Version,
				},
				Network:               Azurenetwork,
				AccessPoints:          []string{"private", "public"},
				ApiServerAccessPoints: []string{"private", "public"},
			},
			out: driver.ClusterCreationParams{
				CreatedBy: userID,
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
				OrganizationID:        orgID,
				ResourceGroup:         ResourceGroup,
				SecretID:              SecretID,
				SSHSecretID:           SSHSecretID,
				AccessPoints:          azurePke.AccessPoints{{Name: "private"}, {Name: "public"}},
				APIServerAccessPoints: azurePke.APIServerAccessPoints{"private", "public"},
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
