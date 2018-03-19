package components

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/banzaicloud/banzai-types/components/amazon"
	"github.com/banzaicloud/banzai-types/components/azure"
	"github.com/banzaicloud/banzai-types/components/google"
	"github.com/banzaicloud/banzai-types/constants"
)

type BanzaiResponse struct {
	StatusCode int    `json:"status_code,omitempty"`
	Message    string `json:"message,omitempty"`
}

type CreateClusterRequest struct {
	Name             string `json:"name" binding:"required"`
	Location         string `json:"location" binding:"required"`
	Cloud            string `json:"cloud" binding:"required"`
	NodeInstanceType string `json:"nodeInstanceType" binding:"required"`
	Properties       struct {
		CreateClusterAmazon *amazon.CreateClusterAmazon `json:"amazon,omitempty"`
		CreateClusterAzure  *azure.CreateClusterAzure   `json:"azure,omitempty"`
		CreateClusterGoogle *google.CreateClusterGoogle `json:"google,omitempty"`
	} `json:"properties" binding:"required"`
}

type ErrorResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Error   string `json:"error"`
}

type GetClusterStatusResponse struct {
	Status           int    `json:"status"`
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

//type GetClusterStatusResponse struct {
//	Status int `json:"status"`
//}

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
	}

	return buffer.String()
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
	default:
		// not supported cloud type
		return errors.New("Not supported cloud type.")
	}

}

// preValidate resets the azure fields
func (r *UpdateClusterRequest) preValidate() {
	switch r.Cloud {
	case constants.Amazon:
		// reset azure fields
		r.UpdateClusterAzure = nil
		break
	case constants.Azure:
		// reset field amazon fields
		r.UpdateClusterAmazon = nil
		break
	}
}

type ClusterProfileResponse struct {
	Name             string `json:"instanceName" binding:"required"`
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
	Name             string `json:"instanceName" binding:"required"`
	Location         string `json:"location" binding:"required"`
	Cloud            string `json:"cloud" binding:"required"`
	NodeInstanceType string `json:"nodeInstanceType" binding:"required"`
	Properties       struct {
		Amazon *amazon.ClusterProfileAmazon `json:"amazon,omitempty"`
		Azure  *azure.ClusterProfileAzure   `json:"azure,omitempty"`
		Google *google.ClusterProfileGoogle `json:"google,omitempty"`
	} `json:"properties" binding:"required"`
}
