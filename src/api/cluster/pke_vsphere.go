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
	"github.com/banzaicloud/pipeline/internal/providers/vsphere/pke"
	"github.com/banzaicloud/pipeline/internal/providers/vsphere/pke/driver"
	"github.com/banzaicloud/pipeline/src/secret"
)

const PKEOnVsphere = pke.PKEOnVsphere

type CreatePKEOnVsphereClusterRequest pipeline.CreatePkeOnVsphereClusterRequest

func (req CreatePKEOnVsphereClusterRequest) ToVspherePKEClusterCreationParams(organizationID, userID uint) driver.VspherePKEClusterCreationParams {
	storagetSecretID := req.StorageSecretId
	if storagetSecretID == "" && req.StorageSecretName != "" {
		storagetSecretID = secret.GenerateSecretIDFromName(req.StorageSecretName)
	}

	return driver.VspherePKEClusterCreationParams{
		Name:            req.Name,
		OrganizationID:  organizationID,
		CreatedBy:       userID,
		SecretID:        req.SecretId,
		StorageSecretID: storagetSecretID,
		SSHSecretID:     req.SshSecretId,
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
		NodePools: vsphereRequestToClusterNodepools(req.Nodepools, userID),
		HTTPProxy: intPKE.HTTPProxy{
			HTTP:       clientPKEClusterHTTPProxyOptionsToPKEHTTPProxyOptions(req.Proxy.Http),
			HTTPS:      clientPKEClusterHTTPProxyOptionsToPKEHTTPProxyOptions(req.Proxy.Https),
			Exceptions: req.Proxy.Exceptions,
		},
		ResourcePoolName:    req.ResourcePool,
		FolderName:          req.Folder,
		DatastoreName:       req.Datastore,
		LoadBalancerIPRange: req.LoadBalancerIPRange,
	}
}

type UpdatePKEOnVsphereClusterRequest pipeline.UpdatePkeOnVsphereClusterRequest

func (req UpdatePKEOnVsphereClusterRequest) ToVspherePKEClusterUpdateParams(clusterID, userID uint) driver.VspherePKEClusterUpdateParams {
	return driver.VspherePKEClusterUpdateParams{
		ClusterID: clusterID,
		NodePools: vsphereRequestToClusterNodepools(req.Nodepools, userID),
	}
}

func vsphereRequestToClusterNodepools(request []pipeline.PkeOnVsphereNodePool, userID uint) []driver.NodePool {
	nodepools := make([]driver.NodePool, len(request))
	for i, node := range request {
		nodepools[i] = driver.NodePool{
			CreatedBy:     userID,
			Name:          node.Name,
			Roles:         node.Roles,
			Labels:        node.Labels,
			Size:          int(node.Size),
			AdminUsername: node.AdminUsername,
			VCPU:          int(node.Vcpu),
			RAM:           int(node.Ram),
			TemplateName:  node.Template,
		}
	}
	return nodepools
}
