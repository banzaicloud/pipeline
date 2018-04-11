package components

import (
	"bytes"
	"fmt"
	"github.com/banzaicloud/banzai-types/components/amazon"
	"github.com/banzaicloud/banzai-types/components/azure"
	"github.com/banzaicloud/banzai-types/components/byoc"
	"github.com/banzaicloud/banzai-types/components/dummy"
	"github.com/banzaicloud/banzai-types/components/google"
	"github.com/banzaicloud/banzai-types/constants"
)

type BanzaiResponse struct {
	StatusCode int    `json:"status_code,omitempty"`
	Message    string `json:"message,omitempty"`
}

type CreateClusterRequest struct {
	Name             string `json:"name" binding:"required"`
	Location         string `json:"location"`
	Cloud            string `json:"cloud" binding:"required"`
	NodeInstanceType string `json:"nodeInstanceType"`
	SecretId         string `json:"secret_id" binding:"required"`
	Properties       struct {
		CreateClusterAmazon *amazon.CreateClusterAmazon `json:"amazon,omitempty"`
		CreateClusterAzure  *azure.CreateClusterAzure   `json:"azure,omitempty"`
		CreateClusterGoogle *google.CreateClusterGoogle `json:"google,omitempty"`
		CreateClusterDummy  *dummy.CreateClusterDummy   `json:"dummy,omitempty"`
		CreateBYOC          *byoc.CreateBYOC            `json:"byoc,omitempty"`
	} `json:"properties" binding:"required"`
}

type ErrorResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Error   string `json:"error"`
}

type GetClusterStatusResponse struct {
	Status           string `json:"status"`
	Name             string `json:"name"`
	Location         string `json:"location"`
	Cloud            string `json:"cloud"`
	NodeInstanceType string `json:"nodeInstanceType"`
	ResourceID       uint   `json:"id"`
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
	*amazon.UpdateClusterAmazon `json:"amazon,omitempty"`
	*azure.UpdateClusterAzure   `json:"azure,omitempty"`
	*google.UpdateClusterGoogle `json:"google,omitempty"`
	*dummy.UpdateClusterDummy   `json:"dummy,omitempty"`
}

// String method prints formatted update request fields
func (r *UpdateClusterRequest) String() string {
	var buffer bytes.Buffer
	buffer.WriteString(fmt.Sprintf("Cloud: %s, ", r.Cloud))
	if r.Cloud == constants.Azure && r.UpdateClusterAzure != nil {
		// Write AKS
		buffer.WriteString(fmt.Sprintf("Agent count: %d",
			r.UpdateClusterAzure.AgentCount))
	} else if r.Cloud == constants.Amazon && r.UpdateClusterAmazon != nil {
		// Write AWS Node
		if r.UpdateClusterAmazon.UpdateAmazonNode != nil {
			buffer.WriteString(fmt.Sprintf("Min count: %d, Max count: %d",
				r.UpdateClusterAmazon.MinCount,
				r.UpdateClusterAmazon.MaxCount))
		}
	} else if r.Cloud == constants.Google && r.UpdateClusterGoogle != nil {
		// Write GKE Master
		if r.UpdateClusterGoogle.Master != nil {
			buffer.WriteString(fmt.Sprintf("Master version: %s",
				r.UpdateClusterGoogle.Master.Version))
		}

		// Write GKE Node version
		buffer.WriteString(fmt.Sprintf("Node version: %s", r.UpdateClusterGoogle.NodeVersion))
		if r.UpdateClusterGoogle.NodePools != nil {
			buffer.WriteString(fmt.Sprintf("Node pools: %v", r.UpdateClusterGoogle.NodePools))
		}
	} else if r.Cloud == constants.Dummy && r.UpdateClusterDummy != nil {
		// Write Dummy node
		if r.UpdateClusterDummy.Node != nil {
			buffer.WriteString(fmt.Sprintf("Node count: %d, k8s version: %s",
				r.UpdateClusterDummy.Node.Count,
				r.UpdateClusterDummy.Node.KubernetesVersion))
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
	case constants.BYOC:
		// byoc validate
		return r.Properties.CreateBYOC.Validate()
	default:
		// not supported cloud type
		return constants.ErrorNotSupportedCloudType
	}
}

func (r *CreateClusterRequest) validateMainFields() error {
	if r.Cloud != constants.BYOC {
		if len(r.Location) == 0 {
			return constants.ErrorLocationEmpty
		}

		if len(r.NodeInstanceType) == 0 {
			return constants.ErrorNodeInstanceTypeEmpty
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
		return r.UpdateClusterAmazon.Validate()
	case constants.Azure:
		// azure validate
		return r.UpdateClusterAzure.Validate()
	case constants.Google:
		// google validate
		return r.UpdateClusterGoogle.Validate()
	case constants.Dummy:
		return r.UpdateClusterDummy.Validate()
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
		r.UpdateClusterAzure = nil
		r.UpdateClusterGoogle = nil
		break
	case constants.Azure:
		// reset other fields
		r.UpdateClusterAmazon = nil
		r.UpdateClusterGoogle = nil
		break
	case constants.Google:
		// reset other fields
		r.UpdateClusterAmazon = nil
		r.UpdateClusterAzure = nil
	}
}

type ClusterProfileResponse struct {
	Name             string `json:"name" binding:"required"`
	Location         string `json:"location" binding:"required"`
	Cloud            string `json:"cloud" binding:"required"`
	NodeInstanceType string `json:"nodeInstanceType" binding:"required"`
	Properties       struct {
		Amazon *amazon.ClusterProfileAmazon `json:"amazon,omitempty"`
		Azure  *azure.ClusterProfileAzure   `json:"azure,omitempty"`
		Google *google.ClusterProfileGoogle `json:"google,omitempty"`
	} `json:"properties" binding:"required"`
}

type ClusterProfileRequest struct {
	Name             string `json:"name" binding:"required"`
	Location         string `json:"location" binding:"required"`
	Cloud            string `json:"cloud" binding:"required"`
	NodeInstanceType string `json:"nodeInstanceType" binding:"required"`
	Properties       struct {
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
	} `json:"filter,omitempty"`
}

type InstanceFilter struct {
	Zone string    `json:"zone,omitempty"`
	Tags []*string `json:"tags,omitempty"`
}

type KubernetesFilter struct {
	Zone string `json:"zone,omitempty"`
}

type GetCloudInfoResponse struct {
	Type               string                 `json:"type" binding:"required"`
	NameRegexp         string                 `json:"nameRegexp,omitempty"`
	Locations          []string               `json:"locations,omitempty"`
	NodeInstanceType   map[string]MachineType `json:"nodeInstanceType,omitempty"`
	KubernetesVersions interface{}            `json:"kubernetes_versions,omitempty"`
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
