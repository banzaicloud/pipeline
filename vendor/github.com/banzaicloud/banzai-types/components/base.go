package components

import (
	"github.com/banzaicloud/banzai-types/utils"
	"github.com/banzaicloud/banzai-types/components/amazon"
	"github.com/banzaicloud/banzai-types/components/azure"
	"github.com/banzaicloud/banzai-types/components/google"
	"bytes"
	"fmt"
	"github.com/banzaicloud/banzai-types/constants"
	"github.com/banzaicloud/banzai-types/components/database"
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
	Properties struct {
		CreateClusterAmazon *amazon.CreateClusterAmazon `json:"amazon,omitempty"`
		CreateClusterAzure  *azure.CreateClusterAzure   `json:"azure,omitempty"`
		CreateClusterGoogle *google.CreateClusterGoogle `json:"google,omitempty"`
	} `json:"properties" binding:"required"`
}

type UpdateClusterRequest struct {
	Cloud string     `json:"cloud" binding:"required"`
	UpdateProperties `json:"properties"`
}

type UpdateProperties struct {
	*amazon.UpdateClusterAmazon `json:"amazon,omitempty"`
	*azure.UpdateClusterAzure   `json:"azure,omitempty"`
	*google.UpdateClusterGoogle `json:"google,omitempty"`
}

func (e *BanzaiResponse) String() string {
	return utils.Convert2Json(e)
}

// String method prints formatted update request fields
func (r *UpdateClusterRequest) String() string {
	var buffer bytes.Buffer
	buffer.WriteString(fmt.Sprintf("Cloud: %s, ", r.Cloud))
	if r.Cloud == constants.Azure && r.UpdateClusterAzure != nil {
		// Write AKS
		buffer.WriteString(fmt.Sprintf("Agent count: %d",
			r.UpdateClusterAzure.AgentCount))
	} else if r.Cloud == constants.Amazon && r.UpdateClusterAzure != nil {
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
func (r *UpdateClusterRequest) Validate(defaultValue database.ClusterSimple) (bool, string) {

	r.preValidate()

	switch r.Cloud {
	case constants.Amazon:
		// amazon validate
		return r.UpdateClusterAmazon.Validate(defaultValue)
	case constants.Azure:
		// azure validate
		return r.UpdateClusterAzure.Validate(defaultValue)
	case constants.Google:
		// google validate
		return r.UpdateClusterGoogle.Validate(defaultValue)
	default:
		// not supported cloud type
		return false, "Not supported cloud type."
	}

}

// preValidate resets the azure fields
func (r *UpdateClusterRequest) preValidate() {
	switch r.Cloud {
	case constants.Amazon:
		// reset azure fields
		utils.LogInfo(constants.TagValidateUpdateCluster, "Reset azure fields")
		r.UpdateClusterAzure = nil
		break
	case constants.Azure:
		// reset field amazon fields
		utils.LogInfo(constants.TagValidateCreateCluster, "Reset amazon fields")
		r.UpdateClusterAmazon = nil
		break
	}
}
