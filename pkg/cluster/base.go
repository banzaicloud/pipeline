package cluster

import (
	"bytes"
	"fmt"

	"github.com/banzaicloud/pipeline/pkg/cluster/amazon"
	"github.com/banzaicloud/pipeline/pkg/cluster/azure"
	"github.com/banzaicloud/pipeline/pkg/cluster/dummy"
	"github.com/banzaicloud/pipeline/pkg/cluster/google"
	"github.com/banzaicloud/pipeline/pkg/cluster/kubernetes"
	pkgCommon "github.com/banzaicloud/pipeline/pkg/common"
	pkgErrors "github.com/banzaicloud/pipeline/pkg/errors"
	oracle "github.com/banzaicloud/pipeline/pkg/providers/oracle/cluster"
	"k8s.io/api/core/v1"
)

// ### [ Cluster statuses ] ### //
const (
	Creating = "CREATING"
	Running  = "RUNNING"
	Updating = "UPDATING"
	Deleting = "DELETING"
	Error    = "ERROR"

	CreatingMessage = "Cluster is creating"
	RunningMessage  = "Cluster is running"
	UpdatingMessage = "Cluster is updating"
	DeletingMessage = "Cluster is deleting"
)

// Cluster provider constants
const (
	Amazon     = "amazon"
	Azure      = "azure"
	Google     = "google"
	Dummy      = "dummy"
	Kubernetes = "kubernetes"
	Oracle     = "oracle"
)

// constants for posthooks
const (
	StoreKubeConfig                    = "StoreKubeConfig"
	PersistKubernetesKeys              = "PersistKubernetesKeys"
	UpdatePrometheusPostHook           = "UpdatePrometheusPostHook"
	InstallHelmPostHook                = "InstallHelmPostHook"
	InstallIngressControllerPostHook   = "InstallIngressControllerPostHook"
	InstallKubernetesDashboardPostHook = "InstallKubernetesDashboardPostHook"
	InstallClusterAutoscalerPostHook   = "InstallClusterAutoscalerPostHook"
	InstallMonitoring                  = "InstallMonitoring"
	InstallLogging                     = "InstallLogging"
	RegisterDomainPostHook             = "RegisterDomainPostHook"
	LabelNodes                         = "LabelNodes"
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
	Name        string    `json:"name" binding:"required"`
	Location    string    `json:"location"`
	Cloud       string    `json:"cloud" binding:"required"`
	SecretId    string    `json:"secretId" binding:"required"`
	ProfileName string    `json:"profileName"`
	PostHooks   PostHooks `json:"postHooks"`
	Properties  struct {
		CreateClusterAmazon *amazon.CreateClusterAmazon  `json:"amazon,omitempty"`
		CreateClusterAzure  *azure.CreateClusterAzure    `json:"azure,omitempty"`
		CreateClusterGoogle *google.CreateClusterGoogle  `json:"google,omitempty"`
		CreateClusterDummy  *dummy.CreateClusterDummy    `json:"dummy,omitempty"`
		CreateKubernetes    *kubernetes.CreateKubernetes `json:"kubernetes,omitempty"`
		CreateClusterOracle *oracle.Cluster              `json:"oracle,omitempty"`
	} `json:"properties" binding:"required"`
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
	Region           string           `json:"region" binding:"required"`
	SecretId         string           `json:"secretId" binding:"required"`
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
	ResourceID    uint                       `json:"id"`
	NodePools     map[string]*NodePoolStatus `json:"nodePools,omitempty"`
	pkgCommon.CreatorBaseFields

	// ONLY in case of GKE
	Region string `json:"region,omitempty"`
}

// NodePoolStatus describes cluster's node status
type NodePoolStatus struct {
	Autoscaling  bool   `json:"autoscaling,omitempty"`
	Count        int    `json:"count,omitempty"`
	InstanceType string `json:"instanceType,omitempty"`
	SpotPrice    string `json:"spotPrice,omitempty"`
	MinCount     int    `json:"minCount,omitempty"`
	MaxCount     int    `json:"maxCount,omitempty"`
	Image        string `json:"image,omitempty"`
}

// GetClusterConfigResponse describes Pipeline's GetConfig API response
type GetClusterConfigResponse struct {
	Status int    `json:"status"`
	Data   string `json:"data"`
}

// UpdateClusterResponse describes Pipeline's UpdateCluster API response
type UpdateClusterResponse struct {
	Status int `json:"status"`
}

// UpdateClusterRequest describes an update cluster request
type UpdateClusterRequest struct {
	Cloud            string `json:"cloud" binding:"required"`
	UpdateProperties `json:"properties"`
}

// DeleteClusterResponse describes Pipeline's DeleteCluster API response
type DeleteClusterResponse struct {
	Status     int    `json:"status"`
	Name       string `json:"name"`
	Message    string `json:"message"`
	ResourceID uint   `json:"id"`
}

// UpdateProperties describes Pipeline's UpdateCluster request properties
type UpdateProperties struct {
	Amazon *amazon.UpdateClusterAmazon `json:"amazon,omitempty"`
	Azure  *azure.UpdateClusterAzure   `json:"azure,omitempty"`
	Google *google.UpdateClusterGoogle `json:"google,omitempty"`
	Dummy  *dummy.UpdateClusterDummy   `json:"dummy,omitempty"`
	Oracle *oracle.Cluster             `json:"oracle,omitempty"`
}

// String method prints formatted update request fields
func (r *UpdateClusterRequest) String() string {
	var buffer bytes.Buffer
	buffer.WriteString(fmt.Sprintf("Cloud: %s, ", r.Cloud))
	if r.Cloud == Azure && r.Azure != nil && r.Azure.NodePools != nil {
		// Write AKS
		buffer.WriteString(fmt.Sprintf("Node pools: %v",
			&r.Azure.NodePools))
	} else if r.Cloud == Amazon && r.Amazon != nil {
		// Write AWS Node
		for name, nodePool := range r.UpdateProperties.Amazon.NodePools {
			buffer.WriteString(fmt.Sprintf("NodePool %s Min count: %d, Max count: %d",
				name,
				nodePool.MinCount,
				nodePool.MaxCount))
		}
	} else if r.Cloud == Google && r.Google != nil {
		// Write GKE Master
		if r.Google.Master != nil {
			buffer.WriteString(fmt.Sprintf("Master version: %s",
				r.Google.Master.Version))
		}

		// Write GKE Node version
		buffer.WriteString(fmt.Sprintf("Node version: %s", r.Google.NodeVersion))
		if r.Google.NodePools != nil {
			buffer.WriteString(fmt.Sprintf("Node pools: %v", r.Google.NodePools))
		}
	} else if r.Cloud == Dummy && r.Dummy != nil {
		// Write Dummy node
		if r.Dummy.Node != nil {
			buffer.WriteString(fmt.Sprintf("Node count: %d, k8s version: %s",
				r.Dummy.Node.Count,
				r.Dummy.Node.KubernetesVersion))
		}
	}

	return buffer.String()
}

// AddDefaults puts default values to optional field(s)
func (r *CreateClusterRequest) AddDefaults() error {
	switch r.Cloud {
	case Amazon:
		return r.Properties.CreateClusterAmazon.AddDefaults(r.Location)
	case Oracle:
		return r.Properties.CreateClusterOracle.AddDefaults()
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
		// amazon validate
		return r.Properties.CreateClusterAmazon.Validate()
	case Azure:
		// azure validate
		return r.Properties.CreateClusterAzure.Validate()
	case Google:
		// google validate
		return r.Properties.CreateClusterGoogle.Validate()
	case Dummy:
		// dummy validate
		return r.Properties.CreateClusterDummy.Validate()
	case Kubernetes:
		// kubernetes validate
		return r.Properties.CreateKubernetes.Validate()
	case Oracle:
		// oracle validate
		return r.Properties.CreateClusterOracle.Validate(false)
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
	case Amazon:
		// amazon validate
		return r.Amazon.Validate()
	case Azure:
		// azure validate
		return r.Azure.Validate()
	case Google:
		// google validate
		return r.Google.Validate()
	case Dummy:
		return r.Dummy.Validate()
	case Oracle:
		// oracle validate
		return r.Oracle.Validate(true)
	default:
		// not supported cloud type
		return pkgErrors.ErrorNotSupportedCloudType
	}

}

// preValidate resets other cloud type fields
func (r *UpdateClusterRequest) preValidate() {
	switch r.Cloud {
	case Amazon:
		// reset other fields
		r.Azure = nil
		r.Google = nil
		r.Oracle = nil
		break
	case Azure:
		// reset other fields
		r.Amazon = nil
		r.Google = nil
		r.Oracle = nil
		break
	case Google:
		// reset other fields
		r.Amazon = nil
		r.Azure = nil
		r.Oracle = nil
	case Oracle:
		// reset other fields
		r.Amazon = nil
		r.Azure = nil
		r.Google = nil
	}
}

// ClusterProfileResponse describes Pipeline's ClusterProfile API responses
type ClusterProfileResponse struct {
	Name       string `json:"name" binding:"required"`
	Location   string `json:"location" binding:"required"`
	Cloud      string `json:"cloud" binding:"required"`
	Properties struct {
		Amazon *amazon.ClusterProfileAmazon `json:"amazon,omitempty"`
		Azure  *azure.ClusterProfileAzure   `json:"azure,omitempty"`
		Google *google.ClusterProfileGoogle `json:"google,omitempty"`
		Oracle *oracle.Cluster              `json:"oracle,omitempty"`
	} `json:"properties" binding:"required"`
}

// ClusterProfileRequest describes CreateClusterProfile request
type ClusterProfileRequest struct {
	Name       string `json:"name" binding:"required"`
	Location   string `json:"location" binding:"required"`
	Cloud      string `json:"cloud" binding:"required"`
	Properties struct {
		Amazon *amazon.ClusterProfileAmazon `json:"amazon,omitempty"`
		Azure  *azure.ClusterProfileAzure   `json:"azure,omitempty"`
		Google *google.ClusterProfileGoogle `json:"google,omitempty"`
		Oracle *oracle.Cluster              `json:"oracle,omitempty"`
	} `json:"properties" binding:"required"`
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
	pkgCommon.CreatorBaseFields
	Name          string                     `json:"name"`
	Id            uint                       `json:"id"`
	SecretId      string                     `json:"secretId"`
	SecretName    string                     `json:"secretName"`
	Location      string                     `json:"location"`
	MasterVersion string                     `json:"masterVersion,omitempty"`
	Endpoint      string                     `json:"endpoint,omitempty"`
	NodePools     map[string]*NodeDetails    `json:"nodePools,omitempty"`
	Master        map[string]ResourceSummary `json:"master,omitempty"`
	TotalSummary  *ResourceSummary           `json:"totalSummary,omitempty"`

	// ONLY in case of GKE
	Region string `json:"region,omitempty"`
}

// PodDetailsResponse describes a pod
type PodDetailsResponse struct {
	Name          string            `json:"name"`
	Namespace     string            `json:"namespace"`
	CreatedAt     string            `json:"createdAt"`
	Labels        map[string]string `json:"labels,omitempty"`
	RestartPolicy string            `json:"restartPolicy,omitempty"`
	Conditions    []v1.PodCondition `json:"conditions,omitempty"`
	Summary       *ResourceSummary  `json:"resourceSummary,omitempty"`
}

// NodeDetails describes a cluster's node details
type NodeDetails struct {
	pkgCommon.CreatorBaseFields
	Version         string                     `json:"version,omitempty"`
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
		Properties: struct {
			CreateClusterAmazon *amazon.CreateClusterAmazon  `json:"amazon,omitempty"`
			CreateClusterAzure  *azure.CreateClusterAzure    `json:"azure,omitempty"`
			CreateClusterGoogle *google.CreateClusterGoogle  `json:"google,omitempty"`
			CreateClusterDummy  *dummy.CreateClusterDummy    `json:"dummy,omitempty"`
			CreateKubernetes    *kubernetes.CreateKubernetes `json:"kubernetes,omitempty"`
			CreateClusterOracle *oracle.Cluster              `json:"oracle,omitempty"`
		}{},
	}

	switch p.Cloud {
	case Amazon:
		response.Properties.CreateClusterAmazon = &amazon.CreateClusterAmazon{
			NodePools: p.Properties.Amazon.NodePools,
			Master: &amazon.CreateAmazonMaster{
				InstanceType: p.Properties.Amazon.Master.InstanceType,
				Image:        p.Properties.Amazon.Master.Image,
			},
		}
	case Azure:
		a := createRequest.Properties.CreateClusterAzure
		if a == nil || len(a.ResourceGroup) == 0 {
			return nil, pkgErrors.ErrorResourceGroupRequired
		}
		response.Properties.CreateClusterAzure = &azure.CreateClusterAzure{
			ResourceGroup:     a.ResourceGroup,
			KubernetesVersion: p.Properties.Azure.KubernetesVersion,
			NodePools:         p.Properties.Azure.NodePools,
		}
	case Google:
		response.Properties.CreateClusterGoogle = &google.CreateClusterGoogle{
			NodeVersion: p.Properties.Google.NodeVersion,
			NodePools:   p.Properties.Google.NodePools,
			Master:      p.Properties.Google.Master,
		}
	case Oracle:
		response.Properties.CreateClusterOracle = &oracle.Cluster{
			Version:   p.Properties.Oracle.Version,
			NodePools: p.Properties.Oracle.NodePools,
		}
	}

	return response, nil
}
