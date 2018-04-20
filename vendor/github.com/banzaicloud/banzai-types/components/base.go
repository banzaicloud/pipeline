package components

import (
	"bytes"
	"fmt"
	"github.com/banzaicloud/banzai-types/components/amazon"
	"github.com/banzaicloud/banzai-types/components/azure"
	"github.com/banzaicloud/banzai-types/components/dummy"
	"github.com/banzaicloud/banzai-types/components/google"
	"github.com/banzaicloud/banzai-types/components/kubernetes"
	"github.com/banzaicloud/banzai-types/constants"
)

type BanzaiResponse struct {
	StatusCode int    `json:"status_code,omitempty"`
	Message    string `json:"message,omitempty"`
}

type CreateClusterRequest struct {
	Name        string `json:"name" binding:"required"`
	Location    string `json:"location"`
	Cloud       string `json:"cloud" binding:"required"`
	SecretId    string `json:"secret_id" binding:"required"`
	ProfileName string `json:"profile_name"`
	Properties  struct {
		CreateClusterAmazon *amazon.CreateClusterAmazon  `json:"amazon,omitempty"`
		CreateClusterAzure  *azure.CreateClusterAzure    `json:"azure,omitempty"`
		CreateClusterGoogle *google.CreateClusterGoogle  `json:"google,omitempty"`
		CreateClusterDummy  *dummy.CreateClusterDummy    `json:"dummy,omitempty"`
		CreateKubernetes    *kubernetes.CreateKubernetes `json:"kubernetes,omitempty"`
	} `json:"properties" binding:"required"`
}

type ErrorResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Error   string `json:"error"`
}

type GetClusterStatusResponse struct {
	Status     string                     `json:"status"`
	Name       string                     `json:"name"`
	Location   string                     `json:"location"`
	Cloud      string                     `json:"cloud"`
	ResourceID uint                       `json:"id"`
	NodePools  map[string]*NodePoolStatus `json:"nodePools,omitempty"`
}

type NodePoolStatus struct {
	Count          int    `json:"count,omitempty"`
	InstanceType   string `json:"instance_type,omitempty"`
	ServiceAccount string `json:"service_account,omitempty"`
	SpotPrice      string `json:"spot_price,omitempty"`
	MinCount       int    `json:"min_count,omitempty"`
	MaxCount       int    `json:"max_count,omitempty"`
	Image          string `json:"image,omitempty"`
}

type GetClusterConfigResponse struct {
	Status int    `json:"status"`
	Data   string `json:"data"`
}

type UpdateClusterResponse struct {
	Status int `json:"status"`
}

type UpdateClusterRequest struct {
	Cloud            string `json:"cloud" binding:"required"`
	UpdateProperties `json:"properties"`
}

type DeleteClusterResponse struct {
	Status     int    `json:"status"`
	Name       string `json:"name"`
	Message    string `json:"message"`
	ResourceID uint   `json:"id"`
}

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
	if r.Cloud == constants.Azure && r.Azure != nil && r.Azure.NodePools != nil {
		// Write AKS
		buffer.WriteString(fmt.Sprintf("Node pools: %v",
			&r.Azure.NodePools))
	} else if r.Cloud == constants.Amazon && r.Amazon != nil {
		// Write AWS Node
		for name, nodePool := range r.UpdateProperties.Amazon.NodePools {
			buffer.WriteString(fmt.Sprintf("NodePool %s Min count: %d, Max count: %d",
				name,
				nodePool.MinCount,
				nodePool.MaxCount))
		}
	} else if r.Cloud == constants.Google && r.Google != nil {
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
	} else if r.Cloud == constants.Dummy && r.Dummy != nil {
		// Write Dummy node
		if r.Dummy.Node != nil {
			buffer.WriteString(fmt.Sprintf("Node count: %d, k8s version: %s",
				r.Dummy.Node.Count,
				r.Dummy.Node.KubernetesVersion))
		}
	}

	return buffer.String()
}

// The Validate method checks the request fields
func (r *CreateClusterRequest) Validate() error {

	if err := r.validateMainFields(); err != nil {
		return err
	}

	switch r.Cloud {
	case constants.Amazon:
		// amazon validate
		return r.Properties.CreateClusterAmazon.Validate()
	case constants.Azure:
		// azure validate
		return r.Properties.CreateClusterAzure.Validate()
	case constants.Google:
		// google validate
		return r.Properties.CreateClusterGoogle.Validate()
	case constants.Dummy:
		// dummy validate
		return r.Properties.CreateClusterDummy.Validate()
	case constants.Kubernetes:
		// kubernetes validate
		return r.Properties.CreateKubernetes.Validate()
	default:
		// not supported cloud type
		return constants.ErrorNotSupportedCloudType
	}
}

func (r *CreateClusterRequest) validateMainFields() error {
	if r.Cloud != constants.Kubernetes {
		if len(r.Location) == 0 {
			return constants.ErrorLocationEmpty
		}
	}
	return nil
}

// The Validate method checks the request fields
func (r *UpdateClusterRequest) Validate() error {

	r.preValidate()

	switch r.Cloud {
	case constants.Amazon:
		// amazon validate
		return r.Amazon.Validate()
	case constants.Azure:
		// azure validate
		return r.Azure.Validate()
	case constants.Google:
		// google validate
		return r.Google.Validate()
	case constants.Dummy:
		return r.Dummy.Validate()
	default:
		// not supported cloud type
		return constants.ErrorNotSupportedCloudType
	}

}

// preValidate resets other cloud type fields
func (r *UpdateClusterRequest) preValidate() {
	switch r.Cloud {
	case constants.Amazon:
		// reset other fields
		r.Azure = nil
		r.Google = nil
		break
	case constants.Azure:
		// reset other fields
		r.Amazon = nil
		r.Google = nil
		break
	case constants.Google:
		// reset other fields
		r.Amazon = nil
		r.Azure = nil
	}
}

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

type CloudInfoRequest struct {
	OrganizationId uint   `json:"-"`
	SecretId       string `json:"secret_id,omitempty"`
	Filter         *struct {
		Fields           []string          `json:"fields,omitempty"`
		InstanceType     *InstanceFilter   `json:"instanceType,omitempty"`
		KubernetesFilter *KubernetesFilter `json:"k8sVersion,omitempty"`
		ImageFilter      *ImageFilter      `json:"image,omitempty"`
	} `json:"filter,omitempty"`
}

type InstanceFilter struct {
	Location string `json:"location,omitempty"`
}

type ImageFilter struct {
	Location string    `json:"location,omitempty"`
	Tags     []*string `json:"tags,omitempty"`
}

type KubernetesFilter struct {
	Location string `json:"location,omitempty"`
}

type GetCloudInfoResponse struct {
	Type               string                 `json:"type" binding:"required"`
	NameRegexp         string                 `json:"nameRegexp,omitempty"`
	Locations          []string               `json:"locations,omitempty"`
	NodeInstanceType   map[string]MachineType `json:"nodeInstanceType,omitempty"`
	KubernetesVersions interface{}            `json:"kubernetes_versions,omitempty"`
	Image              map[string][]string    `json:"image,omitempty"`
}

type MachineType []string

type SupportedClustersResponse struct {
	Items []SupportedClusterItem `json:"items"`
}

type SupportedClusterItem struct {
	Name string `json:"name" binding:"required"`
	Key  string `json:"key" binding:"required"`
}

type SupportedFilters struct {
	Keys []string `json:"keys"`
}

type CreateClusterResponse struct {
	Name       string `json:"name"`
	ResourceID uint   `json:"id"`
}

// todo expand with more fields
type ClusterDetailsResponse struct {
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
	case constants.Amazon:
		response.Properties.CreateClusterAmazon = &amazon.CreateClusterAmazon{
			NodePools: p.Properties.Amazon.NodePools,
			Master: &amazon.CreateAmazonMaster{
				InstanceType: p.Properties.Amazon.Master.InstanceType,
				Image:        p.Properties.Amazon.Master.Image,
			},
		}
	case constants.Azure:
		a := createRequest.Properties.CreateClusterAzure
		if a == nil || len(a.ResourceGroup) == 0 {
			return nil, constants.ErrorResourceGroupRequired
		}
		response.Properties.CreateClusterAzure = &azure.CreateClusterAzure{
			ResourceGroup:     a.ResourceGroup,
			KubernetesVersion: p.Properties.Azure.KubernetesVersion,
			NodePools:         p.Properties.Azure.NodePools,
		}
	case constants.Google:
		g := createRequest.Properties.CreateClusterGoogle
		if g == nil || len(g.Project) == 0 {
			return nil, constants.ErrorProjectRequired
		}
		response.Properties.CreateClusterGoogle = &google.CreateClusterGoogle{
			Project:     g.Project,
			NodeVersion: p.Properties.Google.NodeVersion,
			NodePools:   p.Properties.Google.NodePools,
			Master:      p.Properties.Google.Master,
		}
	}

	return response, nil
}
