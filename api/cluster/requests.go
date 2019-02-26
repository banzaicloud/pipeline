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
	"context"
	"net/http"

	"github.com/banzaicloud/pipeline/pkg/providers"
	"github.com/banzaicloud/pipeline/secret"
	"github.com/mitchellh/mapstructure"

	"github.com/banzaicloud/pipeline/cluster"
	internalCluster "github.com/banzaicloud/pipeline/internal/cluster"
	internalPKE "github.com/banzaicloud/pipeline/internal/providers/pke"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	"github.com/banzaicloud/pipeline/pkg/cluster/pke"
	pkgClusterPKE "github.com/banzaicloud/pipeline/pkg/cluster/pke"
	pkgCommon "github.com/banzaicloud/pipeline/pkg/common"
)

// CreateClusterRequest defines the common interface of cluster creation requests
type CreateClusterRequest interface {
	CreateCluster(ctx context.Context, organizationID uint, userID uint) (cluster.Cluster, *pkgCommon.ErrorResponse)
}

// CreateClusterRequestBase describes the common base of cluster creation requests
type CreateClusterRequestBase struct {
	Name       string               `json:"name" yaml:"name" binding:"required"`
	PostHooks  pkgCluster.PostHooks `json:"postHooks" yaml:"postHooks"`
	SecretID   string               `json:"secretId" yaml:"secretId"`
	SecretIDs  []string             `json:"secretIds,omitempty" yaml:"secretIds,omitempty"`
	SecretName string               `json:"secretName" yaml:"secretName"`
	Type       string               `json:"type" yaml:"type" binding:"required"`
}

func getSecretByID(organizationID uint, secretID string) (*secret.SecretItemResponse, *pkgCommon.ErrorResponse) {
	if secretID == "" {
		return nil, nil
	}
	sir, err := secret.Store.Get(organizationID, secretID)
	if err == nil {
		return sir, nil
	}
	if err == secret.ErrSecretNotExists {
		return nil, &pkgCommon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "no secret exists with the specified ID",
			Error:   err.Error(),
		}
	}
	return nil, &pkgCommon.ErrorResponse{
		Code:    http.StatusInternalServerError,
		Message: "failed to retreive secret by ID",
		Error:   err.Error(),
	}
}

func getSecretByName(organizationID uint, secretName string) (*secret.SecretItemResponse, *pkgCommon.ErrorResponse) {
	if secretName == "" {
		return nil, nil
	}
	sir, err := secret.Store.GetByName(organizationID, secretName)
	if err == nil {
		return sir, nil
	}
	if err == secret.ErrSecretNotExists {
		return nil, &pkgCommon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "no secret exists with the specified name",
			Error:   err.Error(),
		}
	}
	return nil, &pkgCommon.ErrorResponse{
		Code:    http.StatusInternalServerError,
		Message: "failed to retreive secret by name",
		Error:   err.Error(),
	}
}

func getSecretWithType(organizationID uint, secretIDs []string, secretType string) (*secret.SecretItemResponse, *pkgCommon.ErrorResponse) {
	for _, id := range secretIDs {
		sir, err := secret.Store.Get(organizationID, id)
		if err == nil && sir.Type == secretType {
			return sir, nil
		}
		if err != secret.ErrSecretNotExists {
			return nil, &pkgCommon.ErrorResponse{
				Code:    http.StatusInternalServerError,
				Message: "failed to retreive secret by ID",
				Error:   err.Error(),
			}
		}
	}
	return nil, nil
}

func getSecretFromRequest(orgID uint, req CreateClusterRequestBase, providerID string) (*secret.SecretItemResponse, *pkgCommon.ErrorResponse) {
	sir, errRes := getSecretByID(orgID, req.SecretID)
	if errRes != nil {
		return nil, errRes
	}
	if sir == nil {
		sir, errRes = getSecretByName(orgID, req.SecretName)
	}
	if errRes != nil {
		return nil, errRes
	}
	if sir == nil {
		sir, errRes = getSecretWithType(orgID, req.SecretIDs, providerID)
	}
	if errRes != nil {
		return nil, errRes
	}
	if sir == nil {
		msg := "no suitable secret provided in request"
		return nil, &pkgCommon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: msg,
			Error:   msg,
		}
	}
	return sir, nil
}

// CreatePKEAWSClusterRequest represents a PKE-on-AWS cluster creation request
type CreatePKEAWSClusterRequest struct {
	CreateClusterRequestBase
	Region     string
	Network    pkgClusterPKE.Network    `json:"network,omitempty" yaml:"network,omitempty" binding:"required"`
	NodePools  pkgClusterPKE.NodePools  `json:"nodepools,omitempty" yaml:"nodepools,omitempty" binding:"required"`
	Kubernetes pkgClusterPKE.Kubernetes `json:"kubernetes,omitempty" yaml:"kubernetes,omitempty" binding:"required"`
	KubeADM    pkgClusterPKE.KubeADM    `json:"kubeadm,omitempty" yaml:"kubeadm,omitempty"`
	CRI        pkgClusterPKE.CRI        `json:"cri,omitempty" yaml:"cri,omitempty" binding:"required"`
}

func (req CreatePKEAWSClusterRequest) GetProviderID() string {
	return providers.Amazon
}

func createEC2PKENetworkFromRequest(network pkgClusterPKE.Network, userID uint) internalPKE.Network {
	n := internalPKE.Network{
		ServiceCIDR:      network.ServiceCIDR,
		PodCIDR:          network.PodCIDR,
		Provider:         internalPKE.NetworkProvider(network.Provider),
		APIServerAddress: network.APIServerAddress,
	}
	n.CreatedBy = userID
	return n
}

func convertRoles(roles pkgClusterPKE.Roles) internalPKE.Roles {
	result := make(internalPKE.Roles, len(roles))
	for i, role := range roles {
		result[i] = internalPKE.Role(role)
	}
	return result
}

func convertLabels(labels pkgClusterPKE.Labels) internalPKE.Labels {
	result := make(internalPKE.Labels, len(labels))
	for i, label := range labels {
		result[i] = label
	}
	return result
}

func convertTaints(taints pkgClusterPKE.Taints) internalPKE.Taints {
	result := make(internalPKE.Taints, len(taints))
	for i, taint := range taints {
		result[i] = internalPKE.Taint(taint)
	}
	return result
}

func convertHosts(hosts pkgClusterPKE.Hosts) internalPKE.Hosts {
	result := make(internalPKE.Hosts, len(hosts))
	for i, host := range hosts {
		result[i] = internalPKE.Host{
			Name:             host.Name,
			PrivateIP:        host.PrivateIP,
			NetworkInterface: host.NetworkInterface,
			Roles:            convertRoles(host.Roles),
			Labels:           convertLabels(host.Labels),
			Taints:           convertTaints(host.Taints),
		}
	}
	return result
}

func convertNodePoolProvider(provider pke.NodePoolProvider) internalPKE.NodePoolProvider {
	return internalPKE.NodePoolProvider(provider)
}

func createEC2ClusterPKENodePoolsFromRequest(pools pkgClusterPKE.NodePools, userID uint) internalPKE.NodePools {
	result := make(internalPKE.NodePools, len(pools))
	for i, pool := range pools {
		np := internalPKE.NodePool{
			Name:           pool.Name,
			Roles:          convertRoles(pool.Roles),
			Hosts:          convertHosts(pool.Hosts),
			Provider:       convertNodePoolProvider(pool.Provider),
			ProviderConfig: pool.ProviderConfig,
		}
		np.CreatedBy = userID
		result[i] = np
	}
	return result
}

func createEC2ClusterPKEFromRequest(kubernetes pkgClusterPKE.Kubernetes, userID uint) internalPKE.Kubernetes {
	k := internalPKE.Kubernetes{
		Version: kubernetes.Version,
		RBAC:    internalPKE.RBAC{Enabled: kubernetes.RBAC.Enabled},
	}
	k.CreatedBy = userID
	return k
}

func convertExtraArgs(extraArgs pkgClusterPKE.ExtraArgs) internalPKE.ExtraArgs {
	result := make(internalPKE.ExtraArgs, len(extraArgs))
	for i, arg := range extraArgs {
		result[i] = internalPKE.ExtraArg(arg)
	}
	return result
}

func createEC2ClusterPKEKubeADMFromRequest(kubernetes pkgClusterPKE.KubeADM, userID uint) internalPKE.KubeADM {
	a := internalPKE.KubeADM{
		ExtraArgs: convertExtraArgs(kubernetes.ExtraArgs),
	}
	a.CreatedBy = userID
	return a
}

func createEC2ClusterPKECRIFromRequest(cri pkgClusterPKE.CRI, userID uint) internalPKE.CRI {
	c := internalPKE.CRI{
		Runtime:       internalPKE.Runtime(cri.Runtime),
		RuntimeConfig: cri.RuntimeConfig,
	}
	c.CreatedBy = userID
	return c
}

func getMasterInstanceTypeAndImageFromNodePools(nodepools internalPKE.NodePools) (masterInstanceType string, masterImage string, err error) {
	for _, nodepool := range nodepools {
		for _, role := range nodepool.Roles {
			if role == internalPKE.RoleMaster {
				switch nodepool.Provider {
				case internalPKE.NPPAmazon:
					var providerConfig internalPKE.NodePoolProviderConfigAmazon
					if err = mapstructure.Decode(nodepool.ProviderConfig, &providerConfig); err != nil {
						return
					}
					masterInstanceType = providerConfig.AutoScalingGroup.InstanceType
					masterImage = providerConfig.AutoScalingGroup.Image
					return
				}
			}
		}
	}
	return
}

// CreateCluster creates a new PKE-on-AWS cluster based on the request
func (req CreatePKEAWSClusterRequest) CreateCluster(ctx context.Context, organizationID uint, userID uint) (cluster.Cluster, *pkgCommon.ErrorResponse) {
	sir, errRes := getSecretFromRequest(organizationID, req.CreateClusterRequestBase, req.GetProviderID())
	if errRes != nil {
		return nil, errRes
	}

	var (
		network    = createEC2PKENetworkFromRequest(req.Network, userID)
		nodepools  = createEC2ClusterPKENodePoolsFromRequest(req.NodePools, userID)
		kubernetes = createEC2ClusterPKEFromRequest(req.Kubernetes, userID)
		kubeADM    = createEC2ClusterPKEKubeADMFromRequest(req.KubeADM, userID)
		cri        = createEC2ClusterPKECRIFromRequest(req.CRI, userID)
	)

	instanceType, image, err := getMasterInstanceTypeAndImageFromNodePools(nodepools)
	if err != nil {
		return nil, &pkgCommon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "failed to parse node pool provider config",
			Error:   err.Error(),
		}
	}

	c, err := cluster.CreateEC2ClusterPKEFromClusterModel(&internalPKE.EC2PKEClusterModel{
		Cluster: internalCluster.ClusterModel{
			Name:           req.Name,
			Location:       req.Region,
			Cloud:          string(req.GetProviderID()),
			Distribution:   pkgCluster.PKE,
			OrganizationID: organizationID,
			RbacEnabled:    kubernetes.RBAC.Enabled,
			CreatedBy:      userID,
			SecretID:       sir.ID,
		},
		MasterInstanceType: instanceType,
		MasterImage:        image,
		Network:            network,
		NodePools:          nodepools,
		Kubernetes:         kubernetes,
		KubeADM:            kubeADM,
		CRI:                cri,
	})
	if err != nil {
		return nil, &pkgCommon.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: "failed to create PKE-on-AWS cluster from cluster model",
			Error:   err.Error(),
		}
	}
	return c, nil
}

// CreateCustomClusterRequest represents a custom cluster creation request
type CreateCustomClusterRequest struct {
	CreateClusterRequestBase
}

// CreateCluster creates a new custom cluster based on the request
func (req CreateCustomClusterRequest) CreateCluster(ctx context.Context, organizationID uint, userID uint) (cluster.Cluster, *pkgCommon.ErrorResponse) {
	return nil, nil // TODO
}
