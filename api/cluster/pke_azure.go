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
	"github.com/banzaicloud/pipeline/client"
	intCluster "github.com/banzaicloud/pipeline/internal/cluster"
	intPKE "github.com/banzaicloud/pipeline/internal/pke"
	"github.com/banzaicloud/pipeline/internal/providers/azure/pke"
	"github.com/banzaicloud/pipeline/internal/providers/azure/pke/driver"
	"github.com/banzaicloud/pipeline/pkg/cluster"
)

const PKEOnAzure = pke.PKEOnAzure

type CreatePKEOnAzureClusterRequest client.CreatePkeOnAzureClusterRequest

func (req CreatePKEOnAzureClusterRequest) ToAzurePKEClusterCreationParams(organizationID, userID uint) driver.AzurePKEClusterCreationParams {
	features := make([]intCluster.Feature, len(req.Features))
	for i, f := range req.Features {
		features[i] = intCluster.Feature{
			Kind:   f.Kind,
			Params: f.Params,
		}
	}

	return driver.AzurePKEClusterCreationParams{
		Name:           req.Name,
		OrganizationID: organizationID,
		CreatedBy:      userID,
		ResourceGroup:  req.ResourceGroup,
		ScaleOptions: cluster.ScaleOptions{
			Enabled:             req.ScaleOptions.Enabled,
			DesiredCpu:          req.ScaleOptions.DesiredCpu,
			DesiredMem:          req.ScaleOptions.DesiredMem,
			DesiredGpu:          int(req.ScaleOptions.DesiredGpu),
			OnDemandPct:         int(req.ScaleOptions.OnDemandPct),
			Excludes:            req.ScaleOptions.Excludes,
			KeepDesiredCapacity: req.ScaleOptions.KeepDesiredCapacity,
		},
		SecretID:    req.SecretId,
		SSHSecretID: req.SshSecretId,
		Kubernetes: intPKE.Kubernetes{
			Version: req.Kubernetes.Version,
			RBAC:    req.Kubernetes.Rbac,
			Network: intPKE.Network{
				ServiceCIDR:    req.Kubernetes.Network.ServiceCIDR,
				PodCIDR:        req.Kubernetes.Network.PodCIDR,
				Provider:       req.Kubernetes.Network.Provider,
				ProviderConfig: req.Kubernetes.Network.ProviderConfig,
			},
			CRI: intPKE.CRI{
				Runtime:       req.Kubernetes.Cri.Runtime,
				RuntimeConfig: req.Kubernetes.Cri.RuntimeConfig,
			},
			OIDC: intPKE.OIDC{
				Enabled: req.Kubernetes.Oidc.Enabled,
			},
		},
		Network: driver.VirtualNetwork{
			Name:     req.Network.Name,
			CIDR:     req.Network.Cidr,
			Location: req.Location,
		},
		NodePools: requestToClusterNodepools(req.Nodepools, userID),
		Features:  features,
	}
}

type UpdatePKEOnAzureClusterRequest client.UpdatePkeOnAzureClusterRequest

func (req UpdatePKEOnAzureClusterRequest) ToAzurePKEClusterUpdateParams(clusterID, userID uint) driver.AzurePKEClusterUpdateParams {
	return driver.AzurePKEClusterUpdateParams{
		ClusterID: clusterID,
		NodePools: requestToClusterNodepools(req.Nodepools, userID),
	}
}

func requestToClusterNodepools(request []client.PkeOnAzureNodePool, userID uint) []driver.NodePool {
	nodepools := make([]driver.NodePool, len(request))
	for i, node := range request {
		nodepools[i] = driver.NodePool{
			CreatedBy:    userID,
			Name:         node.Name,
			InstanceType: node.InstanceType,
			Subnet: driver.Subnet{
				Name: node.Subnet.Name,
				CIDR: node.Subnet.Cidr,
			},
			Zones:       node.Zones,
			Roles:       node.Roles,
			Labels:      node.Labels,
			Autoscaling: node.Autoscaling,
			Count:       int(node.Count),
			Min:         int(node.MinCount),
			Max:         int(node.MaxCount),
		}
	}
	return nodepools
}
