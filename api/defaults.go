package api

import (
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/banzaicloud/banzai-types/constants"
	"github.com/banzaicloud/banzai-types/components"
	"fmt"
	"github.com/go-errors/errors"
	"net/http"
	"github.com/banzaicloud/pipeline/model/defaults"
)

func GetDefaults(c *gin.Context) {

	log := logger.WithFields(logrus.Fields{"tag": constants.TagGetDefaults})

	cloudType := c.Param("type")
	log.Infof("Cloud type: %s", cloudType)

	resp, err := getDefaultClusterProfile(cloudType)

	if err != nil {
		log.Errorf("Error during getting defaults to %s: %s", cloudType, err.Error())
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
	return errors.New(fmt.Sprintf("Not supported cloud type: %s", ct)) // todo move to constants
}

func getDefaultClusterProfile(cloudType string) (*components.ClusterProfileRespone, error) {

	ds := defaults.GetDefaults()
	for _, d := range ds {
		if d.GetType() == cloudType {
			return d.GetDefaultProfile(), nil
		}
	}

	return nil, notSupportedCloudType(cloudType)

}
