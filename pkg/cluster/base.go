// Copyright © 2018 Banzai Cloud
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

package cluster

import (
	"bytes"
	"fmt"
	"time"

	"github.com/banzaicloud/pipeline/pkg/cluster/acsk"
	"github.com/banzaicloud/pipeline/pkg/cluster/aks"
	"github.com/banzaicloud/pipeline/pkg/cluster/banzaicloud"
	"github.com/banzaicloud/pipeline/pkg/cluster/dummy"
	"github.com/banzaicloud/pipeline/pkg/cluster/eks"
	"github.com/banzaicloud/pipeline/pkg/cluster/gke"
	"github.com/banzaicloud/pipeline/pkg/cluster/kubernetes"
	pkgCommon "github.com/banzaicloud/pipeline/pkg/common"
	pkgErrors "github.com/banzaicloud/pipeline/pkg/errors"
	oke "github.com/banzaicloud/pipeline/pkg/providers/oracle/cluster"
	"k8s.io/api/core/v1"
)

// ### [ Cluster statuses ] ### //
const (
	Creating = "CREATING"
	Running  = "RUNNING"
	Updating = "UPDATING"
	Deleting = "DELETING"
	Warning  = "WARNING"
	Error    = "ERROR"

	CreatingMessage = "Cluster is creating"
	RunningMessage  = "Cluster is running"
	UpdatingMessage = "Cluster is updating"
	DeletingMessage = "Cluster is deleting"
)

// Cloud constants
const (
	Alibaba    = "alibaba"
	Amazon     = "amazon"
	Azure      = "azure"
	Google     = "google"
	Dummy      = "dummy"
	Kubernetes = "kubernetes"
	Oracle     = "oracle"
)

// Distribution constants
const (
	ACSK        = "acsk"
	EKS         = "eks"
	AKS         = "aks"
	GKE         = "gke"
	OKE         = "oke"
	BanzaiCloud = "banzaicloud"
	Unknown     = "unknown"
)

// constants for posthooks
const (
	StoreKubeConfig                        = "StoreKubeConfig"
	SetupPrivileges                        = "SetupPrivileges"
	CreatePipelineNamespacePostHook        = "CreatePipelineNamespacePostHook"
	InstallHelmPostHook                    = "InstallHelmPostHook"
	InstallIngressControllerPostHook       = "InstallIngressControllerPostHook"
	InstallKubernetesDashboardPostHook     = "InstallKubernetesDashboardPostHook"
	InstallClusterAutoscalerPostHook       = "InstallClusterAutoscalerPostHook"
	InstallHorizontalPodAutoscalerPostHook = "InstallHorizontalPodAutoscalerPostHook"
	InstallMonitoring                      = "InstallMonitoring"
	InstallLogging                         = "InstallLogging"
	RegisterDomainPostHook                 = "RegisterDomainPostHook"
	LabelNodes                             = "LabelNodes"
	TaintHeadNodes                         = "TaintHeadNodes"
	InstallPVCOperator                     = "InstallPVCOperator"
	InstallAnchoreImageValidator           = "InstallAnchoreImageValidator"
	RestoreFromBackup                      = "RestoreFromBackup"
	InitSpotConfig                         = "InitSpotConfig"
)

// Provider name regexp
const (
	RegexpAWSName = `^[A-z0-9-_]{1,255}$`
	RegexpAKSName = `^[a-z0-9_]{0,31}[a-z0-9]$`
	RegexpGKEName = `^[a-z]$|^[a-z][a-z0-9-]{0,38}[a-z0-9]$`
)

// ### [ Keywords ] ###
const (
	KeyWordLocation          = "location"
	KeyWordInstanceType      = "instanceType"
	KeyWordKubernetesVersion = "k8sVersion"
	KeyWordImage             = "image"
)

// CreateClusterRequest describes a create cluster request
type CreateClusterRequest struct {
	Name         string                   `json:"name" yaml:"name" binding:"required"`
	Location     string                   `json:"location" yaml:"location"`
	Cloud        string                   `json:"cloud" yaml:"cloud" binding:"required"`
	Distribution string                   `json:"distribution,omitempty" yaml:"distribution,omitempty"`
	SecretId     string                   `json:"secretId" yaml:"secretId"`
	SecretIds    []string                 `json:"secretIds,omitempty" yaml:"secretIds,omitempty"`
	SecretName   string                   `json:"secretName" yaml:"secretName"`
	ProfileName  string                   `json:"profileName" yaml:"profileName"`
	PostHooks    PostHooks                `json:"postHooks" yaml:"postHooks"`
	Properties   *CreateClusterProperties `json:"properties" yaml:"properties" binding:"required"`
}

// CreateClusterProperties contains the cluster flavor specific properties.
type CreateClusterProperties struct {
	CreateClusterACSK        *acsk.CreateClusterACSK               `json:"acsk,omitempty" yaml:"acsk,omitempty"`
	CreateClusterEKS         *eks.CreateClusterEKS                 `json:"eks,omitempty" yaml:"eks,omitempty"`
	CreateClusterAKS         *aks.CreateClusterAKS                 `json:"aks,omitempty" yaml:"aks,omitempty"`
	CreateClusterGKE         *gke.CreateClusterGKE                 `json:"gke,omitempty" yaml:"gke,omitempty"`
	CreateClusterDummy       *dummy.CreateClusterDummy             `json:"dummy,omitempty" yaml:"dummy,omitempty"`
	CreateClusterKubernetes  *kubernetes.CreateClusterKubernetes   `json:"kubernetes,omitempty" yaml:"kubernetes,omitempty"`
	CreateClusterOKE         *oke.Cluster                          `json:"oke,omitempty" yaml:"oke,omitempty"`
	CreateClusterBanzaiCloud *banzaicloud.CreateClusterBanzaiCloud `json:"clusterTopology,omitempty" yaml:"clusterTopology,omitempty"`
}

// PostHookParam describes posthook params in create request
type PostHookParam interface{}

// GenTLSForLogging describes the TLS related params for Logging
type GenTLSForLogging struct {
	TLSEnabled       bool   `json:"tlsEnabled" binding:"required"`
	GenTLSSecretName string `json:"genTlsSecretName"`
	Namespace        string `json:"namespace"`
	TLSHost          string `json:"tlsHost"`
}

// LoggingParam describes the logging posthook params
type LoggingParam struct {
	BucketName       string           `json:"bucketName" binding:"required"`
	Region           string           `json:"region"`
	ResourceGroup    string           `json:"resourceGroup"`
	StorageAccount   string           `json:"storageAccount"`
	SecretId         string           `json:"secretId"`
	SecretName       string           `json:"secretName"`
	GenTLSForLogging GenTLSForLogging `json:"tls" binding:"required"`
}

// AnchoreParam describes the anchore posthook params
type AnchoreParam struct {
	AllowAll string `json:"allowAll"`
}

func (p LoggingParam) String() string {
	return fmt.Sprintf("bucketName: %s, region: %s, secretId: %s", p.BucketName, p.Region, p.SecretId)
}

// PostHooks describes a {cluster_id}/posthooks API request
type PostHooks map[string]PostHookParam

// GetClusterStatusResponse describes Pipeline's GetClusterStatus API response
type GetClusterStatusResponse struct {
	Status        string                     `json:"status"`
	StatusMessage string                     `json:"statusMessage,omitempty"`
	Name          string                     `json:"name"`
	Location      string                     `json:"location"`
	Cloud         string                     `json:"cloud"`
	Distribution  string                     `json:"distribution"`
	Spot          bool                       `json:"spot,omitempty"`
	Logging       bool                       `json:"logging"`
	Monitoring    bool                       `json:"monitoring"`
	SecurityScan  bool                       `json:"securityscan"`
	Version       string                     `json:"version,omitempty"`
	ResourceID    uint                       `json:"id"`
	NodePools     map[string]*NodePoolStatus `json:"nodePools,omitempty"`
	pkgCommon.CreatorBaseFields

	// If region not available fall back to Location
	Region string `json:"region,omitempty"`
}

// NodePoolStatus describes cluster's node status
type NodePoolStatus struct {
	Autoscaling  bool   `json:"autoscaling"`
	Count        int    `json:"count,omitempty"`
	InstanceType string `json:"instanceType,omitempty"`
	SpotPrice    string `json:"spotPrice,omitempty"`
	Preemptible  bool   `json:"preemptible,omitempty"`
	MinCount     int    `json:"minCount,omitempty"`
	MaxCount     int    `json:"maxCount,omitempty"`
	Image        string `json:"image,omitempty"`
	Version      string `json:"version,omitempty"`

	pkgCommon.CreatorBaseFields
}

// GetClusterConfigResponse describes Pipeline's GetConfig API response
type GetClusterConfigResponse struct {
	Status int    `json:"status"`
	Data   string `json:"data"`
}

// UpdateClusterRequest describes an update cluster request
type UpdateClusterRequest struct {
	Cloud            string `json:"cloud" binding:"required"`
	UpdateProperties `json:"properties"`
}

// UpdateProperties describes Pipeline's UpdateCluster request properties
type UpdateProperties struct {
	ACSK  *acsk.UpdateClusterACSK     `json:"acsk,omitempty"`
	EKS   *eks.UpdateClusterAmazonEKS `json:"eks,omitempty"`
	AKS   *aks.UpdateClusterAzure     `json:"aks,omitempty"`
	GKE   *gke.UpdateClusterGoogle    `json:"gke,omitempty"`
	Dummy *dummy.UpdateClusterDummy   `json:"dummy,omitempty"`
	OKE   *oke.Cluster                `json:"oke,omitempty"`
}

// String method prints formatted update request fields
func (r *UpdateClusterRequest) String() string { // todo expand
	var buffer bytes.Buffer
	buffer.WriteString(fmt.Sprintf("Cloud: %s, ", r.Cloud))
	if r.Cloud == Azure && r.AKS != nil && r.AKS.NodePools != nil {
		// Write AKS
		buffer.WriteString(fmt.Sprintf("Node pools: %v", &r.AKS.NodePools))
	} else if r.Cloud == Amazon {
		if r.EKS != nil {
			// Write EKS Node
			for name, nodePool := range r.UpdateProperties.EKS.NodePools {
				buffer.WriteString(fmt.Sprintf("NodePool %s Min count: %d, Max count: %d, Count: %d, Image: %s, Autoscaling: %v, InstanceType: %s, Spot price: %s",
					name,
					nodePool.MinCount,
					nodePool.MaxCount,
					nodePool.Count,
					nodePool.Image,
					nodePool.Autoscaling,
					nodePool.InstanceType,
					nodePool.SpotPrice,
				))
			}
		}
	} else if r.Cloud == Google && r.GKE != nil {
		// Write GKE Master
		if r.GKE.Master != nil {
			buffer.WriteString(fmt.Sprintf("Master version: %s",
				r.GKE.Master.Version))
		}

		// Write GKE Node version
		buffer.WriteString(fmt.Sprintf("Node version: %s", r.GKE.NodeVersion))
		if r.GKE.NodePools != nil {
			buffer.WriteString(fmt.Sprintf("Node pools: %v", r.GKE.NodePools))
		}
	} else if r.Cloud == Dummy && r.Dummy != nil {
		// Write Dummy node
		if r.Dummy.Node != nil {
			buffer.WriteString(fmt.Sprintf("Node count: %d, k8s version: %s",
				r.Dummy.Node.Count,
				r.Dummy.Node.KubernetesVersion))
		}
	} else if r.Cloud == Oracle && r.OKE != nil {
		buffer.WriteString(fmt.Sprintf("Master version: %s", r.OKE.Version))
		for name, nodePool := range r.UpdateProperties.OKE.NodePools {
			buffer.WriteString(fmt.Sprintf("NodePool %s Count: %d Version: %s Image: %s Shape: %s Labels: %v",
				name,
				nodePool.Count,
				nodePool.Version,
				nodePool.Image,
				nodePool.Shape,
				nodePool.Labels))
		}
	}

	return buffer.String()
}

// AddDefaults puts default values to optional field(s)
func (r *CreateClusterRequest) AddDefaults() error {
	if r.Distribution != "" {
		return nil
	}
	switch r.Cloud {
	case Amazon:
		return r.Properties.CreateClusterEKS.AddDefaults(r.Location)
	case Oracle:
		return r.Properties.CreateClusterOKE.AddDefaults()
	default:
		return nil
	}
}

// Validate checks the request fields
func (r *CreateClusterRequest) Validate() error {

	if err := r.validateMainFields(); err != nil {
		return err
	}

	if r.Distribution != "" {
		switch r.Cloud {
		case Amazon:
			// TODO(Ecsy): validation
			return nil
		default:
			// not supported cloud type
			return pkgErrors.ErrorNotSupportedCloudType
		}
	}

	switch r.Cloud {
	case Alibaba:
		// alibaba validate
		return r.Properties.CreateClusterACSK.Validate()
	case Amazon:
		// eks validate
		return r.Properties.CreateClusterEKS.Validate()
	case Azure:
		// aks validate
		return r.Properties.CreateClusterAKS.Validate()
	case Google:
		// gke validate
		return r.Properties.CreateClusterGKE.Validate()
	case Dummy:
		// dummy validate
		return r.Properties.CreateClusterDummy.Validate()
	case Kubernetes:
		// kubernetes validate
		return r.Properties.CreateClusterKubernetes.Validate()
	case Oracle:
		// oracle validate
		return r.Properties.CreateClusterOKE.Validate(false)
	default:
		// not supported cloud type
		return pkgErrors.ErrorNotSupportedCloudType
	}
}

// validateMainFields checks the request's main fields
func (r *CreateClusterRequest) validateMainFields() error {
	if r.Cloud != Kubernetes {
		if len(r.Location) == 0 {
			return pkgErrors.ErrorLocationEmpty
		}
	}
	return nil
}

// Validate checks the request fields
func (r *UpdateClusterRequest) Validate() error {

	r.preValidate()

	switch r.Cloud {
	case Alibaba:
		// alibaba validate
		return r.ACSK.Validate()
	case Amazon:
		// eks validate
		return r.EKS.Validate()
	case Azure:
		// aks validate
		return r.AKS.Validate()
	case Google:
		// gke validate
		return r.GKE.Validate()
	case Dummy:
		return r.Dummy.Validate()
	case Oracle:
		// oracle validate
		return r.OKE.Validate(true)
	default:
		// not supported cloud type
		return pkgErrors.ErrorNotSupportedCloudType
	}

}

// preValidate resets other cloud type fields
func (r *UpdateClusterRequest) preValidate() {

	switch r.Cloud {
	case Alibaba:
		// reset other fields
		r.AKS = nil
		r.GKE = nil
		r.OKE = nil
		break
	case Amazon:
		// reset other fields
		r.ACSK = nil
		r.AKS = nil
		r.GKE = nil
		r.OKE = nil
		break
	case Azure:
		// reset other fields
		r.ACSK = nil
		r.GKE = nil
		r.OKE = nil
		break
	case Google:
		// reset other fields
		r.ACSK = nil
		r.AKS = nil
		r.OKE = nil
	case Oracle:
		// reset other fields
		r.ACSK = nil
		r.AKS = nil
		r.GKE = nil
	}
}

// ClusterProfileResponse describes Pipeline's ClusterProfile API responses
type ClusterProfileResponse struct {
	Name       string                    `json:"name" binding:"required"`
	Location   string                    `json:"location" binding:"required"`
	Cloud      string                    `json:"cloud" binding:"required"`
	Properties *ClusterProfileProperties `json:"properties" binding:"required"`
}

// ClusterProfileRequest describes CreateClusterProfile request
type ClusterProfileRequest struct {
	Name       string                    `json:"name" binding:"required"`
	Location   string                    `json:"location" binding:"required"`
	Cloud      string                    `json:"cloud" binding:"required"`
	Properties *ClusterProfileProperties `json:"properties" binding:"required"`
}

type ClusterProfileProperties struct {
	ACSK *acsk.ClusterProfileACSK `json:"acsk,omitempty"`
	EKS  *eks.ClusterProfileEKS   `json:"eks,omitempty"`
	AKS  *aks.ClusterProfileAKS   `json:"aks,omitempty"`
	GKE  *gke.ClusterProfileGKE   `json:"gke,omitempty"`
	OKE  *oke.Cluster             `json:"oke,omitempty"`
}

// CloudInfoRequest describes Cloud info requests
type CloudInfoRequest struct {
	OrganizationId uint             `json:"-"`
	SecretId       string           `json:"secretId,omitempty"`
	Filter         *CloudInfoFilter `json:"filter,omitempty"`
}

// CloudInfoFilter describes a filter in cloud info
type CloudInfoFilter struct {
	Fields           []string          `json:"fields,omitempty"`
	InstanceType     *InstanceFilter   `json:"instanceType,omitempty"`
	KubernetesFilter *KubernetesFilter `json:"k8sVersion,omitempty"`
	ImageFilter      *ImageFilter      `json:"image,omitempty"`
}

// InstanceFilter describes instance filter of cloud info
type InstanceFilter struct {
	Location string `json:"location,omitempty"`
}

// ImageFilter describes image filter of cloud info
type ImageFilter struct {
	Location string    `json:"location,omitempty"`
	Tags     []*string `json:"tags,omitempty"`
}

// KubernetesFilter describes K8S version filter of cloud info
type KubernetesFilter struct {
	Location string `json:"location,omitempty"`
}

// GetCloudInfoResponse describes Pipeline's Cloud info API response
type GetCloudInfoResponse struct {
	Type               string                 `json:"type" binding:"required"`
	NameRegexp         string                 `json:"nameRegexp,omitempty"`
	Locations          []string               `json:"locations,omitempty"`
	NodeInstanceType   map[string]MachineType `json:"instanceType,omitempty"`
	KubernetesVersions interface{}            `json:"kubernetesVersions,omitempty"`
	Image              map[string][]string    `json:"image,omitempty"`
}

// MachineType describes an string slice which contains machine types
type MachineType []string

// SupportedClustersResponse describes the supported cloud providers
type SupportedClustersResponse struct {
	Items []SupportedClusterItem `json:"items"`
}

// SupportedClusterItem describes a supported cloud provider
type SupportedClusterItem struct {
	Name    string `json:"name" binding:"required"`
	Key     string `json:"key" binding:"required"`
	Enabled bool   `json:"enabled"`
	Icon    string `json:"icon"`
}

// CreateClusterResponse describes Pipeline's CreateCluster API response
type CreateClusterResponse struct {
	Name       string `json:"name"`
	ResourceID uint   `json:"id"`
}

// DetailsResponse describes Pipeline's GetClusterDetails API response
type DetailsResponse struct {
	GetClusterStatusResponse
	Id            uint                        `json:"id"`
	SecretId      string                      `json:"secretId"`
	SecretName    string                      `json:"secretName"`
	MasterVersion string                      `json:"masterVersion,omitempty"`
	Endpoint      string                      `json:"endpoint,omitempty"`
	NodePools     map[string]*NodePoolDetails `json:"nodePools,omitempty"`
	Master        map[string]ResourceSummary  `json:"master,omitempty"`
	TotalSummary  *ResourceSummary            `json:"totalSummary,omitempty"`
}

// PodDetailsResponse describes a pod
type PodDetailsResponse struct {
	Name          string            `json:"name"`
	Namespace     string            `json:"namespace"`
	CreatedAt     time.Time         `json:"createdAt"`
	Labels        map[string]string `json:"labels,omitempty"`
	RestartPolicy string            `json:"restartPolicy,omitempty"`
	Conditions    []v1.PodCondition `json:"conditions,omitempty"`
	Summary       *ResourceSummary  `json:"resourceSummary,omitempty"`
}

// NodePoolDetails describes a cluster's node details
type NodePoolDetails struct {
	pkgCommon.CreatorBaseFields
	NodePoolStatus
	ResourceSummary map[string]ResourceSummary `json:"resourceSummary,omitempty"`
}

// ResourceSummary describes a node's resource summary with CPU and Memory capacity/request/limit/allocatable
type ResourceSummary struct {
	Cpu    *CPU    `json:"cpu,omitempty"`
	Memory *Memory `json:"memory,omitempty"`
	Status string  `json:"status,omitempty"`
}

// CPU describes CPU resource summary
type CPU struct {
	ResourceSummaryItem
}

// Memory describes Memory resource summary
type Memory struct {
	ResourceSummaryItem
}

// ResourceSummaryItem describes a resource summary with capacity/request/limit/allocatable
type ResourceSummaryItem struct {
	Capacity    string `json:"capacity,omitempty"`
	Allocatable string `json:"allocatable,omitempty"`
	Limit       string `json:"limit,omitempty"`
	Request     string `json:"request,omitempty"`
}

// CreateClusterRequest creates a CreateClusterRequest model from profile
func (p *ClusterProfileResponse) CreateClusterRequest(createRequest *CreateClusterRequest) (*CreateClusterRequest, error) {
	response := &CreateClusterRequest{
		Name:        createRequest.Name,
		Location:    p.Location,
		Cloud:       p.Cloud,
		SecretId:    createRequest.SecretId,
		ProfileName: p.Name,
		Properties:  &CreateClusterProperties{},
	}

	switch p.Cloud { // TODO(Ecsy): distribution???
	case Alibaba:
		response.Properties.CreateClusterACSK = &acsk.CreateClusterACSK{
			RegionID:  p.Properties.ACSK.RegionID,
			ZoneID:    p.Properties.ACSK.ZoneID,
			NodePools: p.Properties.ACSK.NodePools,
		}
	case Amazon:
		response.Properties.CreateClusterEKS = &eks.CreateClusterEKS{
			NodePools: p.Properties.EKS.NodePools,
			Version:   p.Properties.EKS.Version,
		}
	case Azure:
		a := createRequest.Properties.CreateClusterAKS
		if a == nil || len(a.ResourceGroup) == 0 {
			return nil, pkgErrors.ErrorResourceGroupRequired
		}
		response.Properties.CreateClusterAKS = &aks.CreateClusterAKS{
			ResourceGroup:     a.ResourceGroup,
			KubernetesVersion: p.Properties.AKS.KubernetesVersion,
			NodePools:         p.Properties.AKS.NodePools,
		}
	case Google:
		response.Properties.CreateClusterGKE = &gke.CreateClusterGKE{
			NodeVersion: p.Properties.GKE.NodeVersion,
			NodePools:   p.Properties.GKE.NodePools,
			Master:      p.Properties.GKE.Master,
		}
	case Oracle:
		response.Properties.CreateClusterOKE = &oke.Cluster{
			Version:   p.Properties.OKE.Version,
			NodePools: p.Properties.OKE.NodePools,
		}
	}

	return response, nil
}
