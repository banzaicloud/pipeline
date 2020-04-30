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
	"github.com/banzaicloud/pipeline/.gen/pipeline/pipeline"
	intPKE "github.com/banzaicloud/pipeline/internal/pke"
	"github.com/banzaicloud/pipeline/internal/providers/azure/pke"
	"github.com/banzaicloud/pipeline/internal/providers/azure/pke/driver"
	"github.com/banzaicloud/pipeline/pkg/cluster"
)

const PKEOnAzure = pke.PKEOnAzure

type CreatePKEOnAzureClusterRequest pipeline.CreatePkeOnAzureClusterRequest

func (req CreatePKEOnAzureClusterRequest) ToAzurePKEClusterCreationParams(organizationID, userID uint) driver.ClusterCreationParams {
	var accessPoints pke.AccessPoints
	for _, apName := range req.AccessPoints {
		accessPoints = append(accessPoints, pke.AccessPoint{Name: apName})
	}

	var apiServerAccessPoints pke.APIServerAccessPoints
	for _, ap := range req.ApiServerAccessPoints {
		apiServerAccessPoints = append(apiServerAccessPoints, pke.APIServerAccessPoint(ap))
	}

	return driver.ClusterCreationParams{
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
		AccessPoints:          accessPoints,
		APIServerAccessPoints: apiServerAccessPoints,
		NodePools:             azureRequestToClusterNodepools(req.Nodepools, userID),
		HTTPProxy: intPKE.HTTPProxy{
			HTTP:       clientPKEClusterHTTPProxyOptionsToPKEHTTPProxyOptions(req.Proxy.Http),
			HTTPS:      clientPKEClusterHTTPProxyOptionsToPKEHTTPProxyOptions(req.Proxy.Https),
			Exceptions: req.Proxy.Exceptions,
		},
	}
}

func clientPKEClusterHTTPProxyOptionsToPKEHTTPProxyOptions(o pipeline.PkeClusterHttpProxyOptions) intPKE.HTTPProxyOptions {
	return intPKE.HTTPProxyOptions{
		Host:     o.Host,
		Port:     uint16(o.Port),
		SecretID: o.SecretId,
		Scheme:   o.Scheme,
	}
}

type UpdatePKEOnAzureClusterRequest pipeline.UpdatePkeOnAzureClusterRequest

func (req UpdatePKEOnAzureClusterRequest) ToAzurePKEClusterUpdateParams(clusterID, userID uint) driver.ClusterUpdateParams {
	return driver.ClusterUpdateParams{
		ClusterID: clusterID,
		NodePools: azureRequestToClusterNodepools(req.Nodepools, userID),
	}
}

func azureRequestToClusterNodepools(request []pipeline.PkeOnAzureNodePool, userID uint) []driver.NodePool {
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
