package api

import (
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/banzaicloud/banzai-types/constants"
	"github.com/banzaicloud/banzai-types/components"
	"fmt"
	"github.com/go-errors/errors"
	"github.com/banzaicloud/banzai-types/components/amazon"
	"github.com/banzaicloud/banzai-types/components/azure"
	"github.com/banzaicloud/banzai-types/components/google"
	"net/http"
	"github.com/banzaicloud/pipeline/model/defaults"
)

// functions
const (
	create = "create"
	update = "update"
)

func GetDefaults(c *gin.Context) {

	log := logger.WithFields(logrus.Fields{"tag": constants.TagGetDefaults})

	function := c.Param("function")
	cloudType := c.Param("type")

	log.Infof("function: %s, type: %s", function, cloudType)

	var resp interface{}
	var err error

	switch function {
	case create:
		resp, err = getCreateClusterRequest(cloudType)
	case update:
		resp, err = getUpdateClusterRequest(cloudType)
	default:
		err = notSupportedFunctionType(function)
	}

	if err != nil {
		log.Errorf("Error during getting defaults to %s function: %s", function, err.Error())
		c.JSON(http.StatusBadRequest, components.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: err.Error(),
			Error:   err.Error(),
		})
	} else {
		c.JSON(http.StatusOK, resp)
	}

}

// notSupportedCloudType returns an error with 'not supported cloud type' message
func notSupportedCloudType(ct string) error {
	return errors.New(fmt.Sprintf("Not supported cloud type: %s", ct))
}

// notSupportedFunctionType returns an error with 'not supported function' message
func notSupportedFunctionType(f string) error {
	return errors.New(fmt.Sprintf("Not supported function: %s", f))
}

func getCreateClusterRequest(cloudType string) (*components.CreateClusterRequest, error) {

	ds := defaults.GetDefaults()
	for _, d := range ds {
		if d.GetType() == cloudType {
			return d.GetDefaultCreateClusterRequest(), nil
		}
	}

	return nil, notSupportedCloudType(cloudType)

}

func getUpdateClusterRequest(cloudType string) (*components.UpdateClusterRequest, error) {
	switch cloudType {
	case constants.Amazon:
		return getUpdateClusterRequestAmazon(), nil
	case constants.Azure:
		return getUpdateClusterRequestAzure(), nil
	case constants.Google:
		return getUpdateClusterRequestGoogle(), nil
	default:
		err := notSupportedCloudType(cloudType)
		log.Error(err.Error())
		return nil, err
	}
}

func getUpdateClusterRequestAmazon() *components.UpdateClusterRequest {
	return &components.UpdateClusterRequest{
		Cloud: constants.Amazon,
		UpdateProperties: components.UpdateProperties{
			UpdateClusterAmazon: &amazon.UpdateClusterAmazon{
				UpdateAmazonNode: &amazon.UpdateAmazonNode{
					MinCount: 2,
					MaxCount: 3,
				},
			},
		},
	}
}

func getUpdateClusterRequestAzure() *components.UpdateClusterRequest {
	return &components.UpdateClusterRequest{
		Cloud: constants.Azure,
		UpdateProperties: components.UpdateProperties{
			UpdateClusterAzure: &azure.UpdateClusterAzure{
				UpdateAzureNode: &azure.UpdateAzureNode{
					AgentCount: 2,
				},
			},
		},
	}
}

func getUpdateClusterRequestGoogle() *components.UpdateClusterRequest {
	return &components.UpdateClusterRequest{
		Cloud: constants.Google,
		UpdateProperties: components.UpdateProperties{
			UpdateClusterGoogle: &google.UpdateClusterGoogle{
				GoogleNode: &google.GoogleNode{
					Count: 2,
				},
			},
		},
	}
}
