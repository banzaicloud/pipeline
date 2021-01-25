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

	v1 "k8s.io/api/core/v1"

	"github.com/banzaicloud/pipeline/internal/cluster/distribution/eks/ekscluster"
	"github.com/banzaicloud/pipeline/pkg/cluster/aks"
	"github.com/banzaicloud/pipeline/pkg/cluster/gke"
	"github.com/banzaicloud/pipeline/pkg/cluster/kubernetes"
	"github.com/banzaicloud/pipeline/pkg/cluster/pke"
	pkgCommon "github.com/banzaicloud/pipeline/pkg/common"
	pkgErrors "github.com/banzaicloud/pipeline/pkg/errors"
)

// ### [ Cluster statuses ] ### //
const (
	Creating = "CREATING"
	Running  = "RUNNING"
	Updating = "UPDATING"
	Deleting = "DELETING"
	Warning  = "WARNING"
	Error    = "ERROR"

	CreatingMessage = "Cluster creation is in progress"
	RunningMessage  = "Cluster is running"
	UpdatingMessage = "Update is in progress"
	DeletingMessage = "Termination is in progress"
)

// Cloud constants
const (
	Amazon     = "amazon"
	Azure      = "azure"
	Google     = "google"
	Kubernetes = "kubernetes"
	Vsphere    = "vsphere"
)

// Distribution constants
const (
	EKS     = "eks"
	AKS     = "aks"
	GKE     = "gke"
	PKE     = "pke"
	Unknown = "unknown"
)

// constants for posthooks
const (
	InstallIngressControllerPostHook       = "InstallIngressControllerPostHook"
	InstallKubernetesDashboardPostHook     = "InstallKubernetesDashboardPostHook"
	InstallClusterAutoscalerPostHook       = "InstallClusterAutoscalerPostHook"
	InstallHorizontalPodAutoscalerPostHook = "InstallHorizontalPodAutoscalerPostHook"
	RestoreFromBackup                      = "RestoreFromBackup"
	InitSpotConfig                         = "InitSpotConfig"
	DeployInstanceTerminationHandler       = "DeployInstanceTerminationHandler"
)

// CreateClusterRequest describes a create cluster request
type CreateClusterRequest struct {
	Name         string                   `json:"name" yaml:"name" binding:"required"`
	Location     string                   `json:"location" yaml:"location"`
	Cloud        string                   `json:"cloud" yaml:"cloud" binding:"required"`
	SecretId     string                   `json:"secretId" yaml:"secretId"`
	SecretIds    []string                 `json:"secretIds,omitempty" yaml:"secretIds,omitempty"`
	SecretName   string                   `json:"secretName" yaml:"secretName"`
	PostHooks    PostHooks                `json:"postHooks" yaml:"postHooks"`
	Properties   *CreateClusterProperties `json:"properties" yaml:"properties" binding:"required"`
	ScaleOptions *ScaleOptions            `json:"scaleOptions,omitempty" yaml:"scaleOptions,omitempty"`
}

// CreateClusterProperties contains the cluster flavor specific properties.
type CreateClusterProperties struct {

	CreateClusterEKS        *ekscluster.CreateClusterEKS        `json:"eks,omitempty" yaml:"eks,omitempty"`
	CreateClusterAKS        *aks.CreateClusterAKS               `json:"aks,omitempty" yaml:"aks,omitempty"`
	CreateClusterGKE        *gke.CreateClusterGKE               `json:"gke,omitempty" yaml:"gke,omitempty"`
	CreateClusterKubernetes *kubernetes.CreateClusterKubernetes `json:"kubernetes,omitempty" yaml:"kubernetes,omitempty"`
	CreateClusterPKE        *pke.CreateClusterPKE               `json:"pke,omitempty" yaml:"pke,omitempty"`
}

// ScaleOptions describes scale options
type ScaleOptions struct {
	Enabled             bool     `json:"enabled"`
	DesiredCpu          float64  `json:"desiredCpu" binding:"min=1"`
	DesiredMem          float64  `json:"desiredMem" binding:"min=1"`
	DesiredGpu          int      `json:"desiredGpu" binding:"min=0"`
	OnDemandPct         int      `json:"onDemandPct,omitempty" binding:"min=0,max=100"`
	Excludes            []string `json:"excludes,omitempty"`
	KeepDesiredCapacity bool     `json:"keepDesiredCapacity"`
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
	OIDCEnabled   bool                       `json:"oidcEnabled,omitempty"`
	Version       string                     `json:"version,omitempty"`
	ResourceID    uint                       `json:"id"`
	NodePools     map[string]*NodePoolStatus `json:"nodePools"`
	pkgCommon.CreatorBaseFields

	// If region not available fall back to Location
	Region    string     `json:"region,omitempty"`
	StartedAt *time.Time `json:"startedAt,omitempty"`
}

// NodePoolStatus describes cluster's node status
type NodePoolStatus struct {
	Autoscaling  bool              `json:"autoscaling,omitempty"`
	Count        int               `json:"count"`
	InstanceType string            `json:"instanceType,omitempty"`
	SpotPrice    string            `json:"spotPrice,omitempty"`
	Preemptible  bool              `json:"preemptible,omitempty"`
	MinCount     int               `json:"minCount"`
	MaxCount     int               `json:"maxCount"`
	Image        string            `json:"image,omitempty"`
	Version      string            `json:"version,omitempty"`
	Labels       map[string]string `json:"labels,omitempty"`
	Vcpu         int               `json:"vcpu,omitempty"`
	Ram          int               `json:"ram,omitempty"`
	Template     string            `json:"template,omitempty"`

	pkgCommon.CreatorBaseFields
}

// GetClusterConfigResponse describes Pipeline's GetConfig API response
type GetClusterConfigResponse struct {
	Status int    `json:"status"`
	Data   string `json:"data"`
}

// GetNodePoolsResponse describes node pools of a cluster
type GetNodePoolsResponse struct {
	ScaleEnabled            bool                             `json:"scaleEnabled"`
	NodePools               map[string]*ActualNodePoolStatus `json:"nodePools,omitempty"`
	ClusterTotalResources   map[string]float64               `json:"clusterTotalResources,omitempty"`
	ClusterDesiredResources map[string]float64               `json:"clusterDesiredResources,omitempty"`
	ClusterStatus           string                           `json:"status,omitempty"`
	Cloud                   string                           `json:"cloud"`
	Distribution            string                           `json:"distribution"`
	Location                string                           `json:"location"`
}

type ActualNodePoolStatus struct {
	NodePoolStatus
	ActualCount int `json:"actualCount"`
}

// UpdateNodePoolsRequest describes an update node pools request
type UpdateNodePoolsRequest struct {
	NodePools map[string]*NodePoolData `json:"nodePools,omitempty"`
}

// NodePoolData describes node pool size
type NodePoolData struct {
	Count int `json:"count"`
}

// UpdateClusterRequest describes an update cluster request
type UpdateClusterRequest struct {
	Cloud            string `json:"cloud" binding:"required"`
	UpdateProperties `json:"properties"`
	ScaleOptions     *ScaleOptions `json:"scaleOptions,omitempty" yaml:"scaleOptions,omitempty"`
}

// UpdateProperties describes Pipeline's UpdateCluster request properties
type UpdateProperties struct {
	EKS *ekscluster.UpdateClusterAmazonEKS `json:"eks,omitempty"`
	AKS *aks.UpdateClusterAzure            `json:"aks,omitempty"`
	GKE *gke.UpdateClusterGoogle           `json:"gke,omitempty"`
	PKE *pke.UpdateClusterPKE              `json:"pke,omitempty"`
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
	}

	return buffer.String()
}

// AddDefaults puts default values to optional field(s)
func (r *CreateClusterRequest) AddDefaults() error {
	switch r.Cloud {
	case Amazon:
		if r.Properties.CreateClusterPKE != nil {
			return r.Properties.CreateClusterPKE.AddDefaults()
		}
		return r.Properties.CreateClusterEKS.AddDefaults(r.Location)
	default:
		return nil
	}
}

// Validate checks the request fields
func (r *CreateClusterRequest) Validate() error {
	if err := r.validateMainFields(); err != nil {
		return err
	}

	switch r.Cloud {
	case Amazon:
		// eks validate
		if r.Properties.CreateClusterPKE != nil {
			// r.Properties.CreateClusterPKE.Validate()
			return nil
		}
		return r.Properties.CreateClusterEKS.Validate()
	case Azure:
		// aks validate
		return r.Properties.CreateClusterAKS.Validate()
	case Google:
		// gke validate
		return r.Properties.CreateClusterGKE.Validate()
	case Kubernetes:
		// kubernetes validate
		return r.Properties.CreateClusterKubernetes.Validate()
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
	if r.ScaleOptions != nil && r.ScaleOptions.Enabled {
		if len(r.Location) == 0 {
			return pkgErrors.ErrorLocationEmpty
		}
	}
	return nil
}

// Validate checks the request fields
func (r *UpdateClusterRequest) Validate() error {
	r.preValidate()
	if r.PKE != nil {
		return r.PKE.Validate()
	}

	switch r.Cloud {
	case Amazon:
		return r.EKS.Validate()
	case Azure:
		return r.AKS.Validate()
	case Google:
		return r.GKE.Validate()
	default:
		return pkgErrors.ErrorNotSupportedCloudType
	}
}

// preValidate resets other cloud type fields
func (r *UpdateClusterRequest) preValidate() {
	switch r.Cloud {
	case Amazon:
		// reset other fields
		r.AKS = nil
		r.GKE = nil
		break
	case Azure:
		// reset other fields
		r.GKE = nil
		r.EKS = nil
		break
	case Google:
		// reset other fields
		r.AKS = nil
		r.EKS = nil
	}
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
	Type               string                  `json:"type" binding:"required"`
	NameRegexp         string                  `json:"nameRegexp,omitempty"`
	Locations          []string                `json:"locations,omitempty"`
	NodeInstanceType   map[string]MachineTypes `json:"instanceType,omitempty"`
	KubernetesVersions interface{}             `json:"kubernetesVersions,omitempty"`
	Image              map[string][]string     `json:"image,omitempty"`
}

// MachineTypes describes a string slice which contains machine types
type MachineTypes []string

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

// NodePoolLabel describes labels on a node pool
type NodePoolLabel struct {
	Name     string `json:"name"`
	Value    string `json:"value"`
	Reserved bool   `json:"reserved"`
}
