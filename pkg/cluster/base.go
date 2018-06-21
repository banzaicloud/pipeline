package cluster

import (
	"bytes"
	"fmt"
	"github.com/banzaicloud/pipeline/pkg/cluster/amazon"
	"github.com/banzaicloud/pipeline/pkg/cluster/azure"
	"github.com/banzaicloud/pipeline/pkg/cluster/dummy"
	"github.com/banzaicloud/pipeline/pkg/cluster/google"
	"github.com/banzaicloud/pipeline/pkg/cluster/kubernetes"
	pkgErrors "github.com/banzaicloud/pipeline/pkg/errors"
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

const (
	Amazon     = "amazon"
	Azure      = "azure"
	Google     = "google"
	Dummy      = "dummy"
	Kubernetes = "kubernetes"
)

// constants for posthooks
const (
	StoreKubeConfig                  = "StoreKubeConfig"
	PersistKubernetesKeys            = "PersistKubernetesKeys"
	UpdatePrometheusPostHook         = "UpdatePrometheusPostHook"
	InstallHelmPostHook              = "InstallHelmPostHook"
	InstallIngressControllerPostHook = "InstallIngressControllerPostHook"
	InstallClusterAutoscalerPostHook = "InstallClusterAutoscalerPostHook"
	InstallMonitoring                = "InstallMonitoring"
	InstallLogging                   = "InstallLogging"
	RegisterDomainPostHook           = "RegisterDomainPostHook"
)

// ### [ Constants to Azure cluster default values ] ### //
const (
	AzureDefaultAgentName         = "agentpool1"
	AzureDefaultKubernetesVersion = "1.9.2"
)

// ### [ Constants to common cluster default values ] ### //
const (
	DefaultNodeMinCount = 1
	DefaultNodeMaxCount = 2
)

// ### [ Constants to Amazon cluster default values ] ### //
const (
	AmazonDefaultMasterInstanceType = "m4.xlarge"
	AmazonDefaultNodeSpotPrice      = "0.2"
	AmazonDefaultImage              = "ami-16bfeb6f"
)

// ### [ Constants to Google cluster default values ] ### //
const (
	GoogleDefaultNodePoolName = "default-pool"
)

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

// BanzaiResponse describes Pipeline's responses
type BanzaiResponse struct {
	StatusCode int    `json:"status_code,omitempty"`
	Message    string `json:"message,omitempty"`
}

// CreateClusterRequest describes a create cluster request
type CreateClusterRequest struct {
	Name              string   `json:"name" binding:"required"`
	Location          string   `json:"location"`
	Cloud             string   `json:"cloud" binding:"required"`
	SecretId          string   `json:"secret_id" binding:"required"`
	ProfileName       string   `json:"profile_name"`
	PostHookFunctions []string `json:"postHooks"`
	Properties        struct {
		CreateClusterAmazon *amazon.CreateClusterAmazon  `json:"amazon,omitempty"`
		CreateClusterAzure  *azure.CreateClusterAzure    `json:"azure,omitempty"`
		CreateClusterGoogle *google.CreateClusterGoogle  `json:"google,omitempty"`
		CreateClusterDummy  *dummy.CreateClusterDummy    `json:"dummy,omitempty"`
		CreateKubernetes    *kubernetes.CreateKubernetes `json:"kubernetes,omitempty"`
	} `json:"properties" binding:"required"`
}

// ErrorResponse describes Pipeline's responses when an error occurred
type ErrorResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Error   string `json:"error"`
}

// GetClusterStatusResponse describes Pipeline's GetClusterStatus API response
type GetClusterStatusResponse struct {
	Status        string                     `json:"status"`
	StatusMessage string                     `json:"status_message,omitempty"`
	Name          string                     `json:"name"`
	Location      string                     `json:"location"`
	Cloud         string                     `json:"cloud"`
	ResourceID    uint                       `json:"id"`
	NodePools     map[string]*NodePoolStatus `json:"nodePools,omitempty"`
}

// NodePoolStatus describes cluster's node status
type NodePoolStatus struct {
	Count          int    `json:"count,omitempty"`
	InstanceType   string `json:"instanceType,omitempty"`
	ServiceAccount string `json:"service_account,omitempty"`
	SpotPrice      string `json:"spot_price,omitempty"`
	MinCount       int    `json:"min_count,omitempty"`
	MaxCount       int    `json:"max_count,omitempty"`
	Image          string `json:"image,omitempty"`
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
		return r.Properties.CreateClusterAmazon.AddDefaults()
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
		break
	case Azure:
		// reset other fields
		r.Amazon = nil
		r.Google = nil
		break
	case Google:
		// reset other fields
		r.Amazon = nil
		r.Azure = nil
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
	} `json:"properties" binding:"required"`
}

// CloudInfoRequest describes Cloud info requests
type CloudInfoRequest struct {
	OrganizationId uint             `json:"-"`
	SecretId       string           `json:"secret_id,omitempty"`
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
	KubernetesVersions interface{}            `json:"kubernetes_versions,omitempty"`
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

// ClusterDetailsResponse describes Pipeline's GetClusterDetails API response
type ClusterDetailsResponse struct {
	// todo expand with more fields
	Name string `json:"name"`
	Id   uint   `json:"id"`
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
	}

	return response, nil
}
