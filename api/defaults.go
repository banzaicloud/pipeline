package api

import (
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/banzaicloud/banzai-types/constants"
	"github.com/banzaicloud/banzai-types/components"
	"fmt"
	"net/http"
	"github.com/banzaicloud/pipeline/model/defaults"
	"github.com/pkg/errors"
)

const cloudTypeKey = "type"

func GetDefaults(c *gin.Context) {

	log := logger.WithFields(logrus.Fields{"tag": constants.TagGetDefaults})

	cloudType := c.Param(cloudTypeKey)
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

func AddClusterProfile(c *gin.Context) {
	log := logger.WithFields(logrus.Fields{"tag": constants.TagSetDefaults})

	cloudType := c.Param(cloudTypeKey)
	log.Infof("Cloud type: %s", cloudType)

	log.Debug("Bind json into ClusterProfileRequest struct")
	// bind request body to struct
	var profileRequest components.ClusterProfileRequest
	if err := c.BindJSON(&profileRequest); err != nil {
		log.Error(errors.Wrap(err, "Error parsing request"))
		c.JSON(http.StatusBadRequest, components.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error parsing request",
			Error:   err.Error(),
		})
		return
	}
	log.Debugf("Parsing request succeeded %#v", profileRequest)
	log.Infof("Convert ClusterProfileRequest into ClusterProfile model with name: %s", profileRequest.ProfileName)

	if prof, err := convertRequestToProfile(&profileRequest); err != nil {
		log.Error("Error during convert profile: &s", err.Error())
		c.JSON(http.StatusBadRequest, components.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error during convert profile",
			Error:   err.Error(),
		})
	} else if !prof.IsDefinedBefore() {
		log.Info("Convert succeeded")
		log.Info("Save cluster profile into database")
		if err := prof.SaveInstance(); err != nil {
			log.Errorf("Error during persist cluster profile: %s", err.Error())
			c.JSON(http.StatusInternalServerError, components.ErrorResponse{
				Code:    http.StatusInternalServerError,
				Message: "Error during persist cluster profile",
				Error:   err.Error(),
			})
		} else {
			log.Info("Save cluster profile succeeded")
			c.Status(http.StatusCreated)
		}
	} else {
		log.Error("Cluster profile with the given name is already exists")
		c.JSON(http.StatusBadRequest, components.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Cluster profile with the given name is already exists, please update not create profile",
			Error:   "Cluster profile with the given name is already exists, please update not create profile",
		})
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

// convertRequestToProfile converts a ClusterProfileRequest into ClusterProfile
func convertRequestToProfile(request *components.ClusterProfileRequest) (defaults.ClusterProfile, error) {

	switch request.Cloud {
	case constants.Amazon:
		return &defaults.AWSProfile{
			DefaultModel:       defaults.DefaultModel{Name: request.ProfileName},
			Location:           request.Location,
			NodeInstanceType:   request.NodeInstanceType,
			NodeImage:          request.Properties.Amazon.Node.Image,
			MasterInstanceType: request.Properties.Amazon.Master.InstanceType,
			MasterImage:        request.Properties.Amazon.Master.Image,
			NodeSpotPrice:      request.Properties.Amazon.Node.SpotPrice,
			NodeMinCount:       request.Properties.Amazon.Node.MinCount,
			NodeMaxCount:       request.Properties.Amazon.Node.MaxCount,
		}, nil
	case constants.Azure:
		return &defaults.AKSProfile{
			DefaultModel:      defaults.DefaultModel{Name: request.ProfileName},
			Location:          request.Location,
			NodeInstanceType:  request.NodeInstanceType,
			AgentCount:        request.Properties.Azure.Node.AgentCount,
			AgentName:         request.Properties.Azure.Node.AgentName,
			KubernetesVersion: request.Properties.Azure.Node.KubernetesVersion,
		}, nil
	case constants.Google:
		return &defaults.GKEProfile{
			DefaultModel:     defaults.DefaultModel{Name: request.ProfileName},
			Location:         request.Location,
			NodeInstanceType: request.NodeInstanceType,
			NodeCount:        request.Properties.Google.Node.Count,
			NodeVersion:      request.Properties.Google.Node.Version,
			MasterVersion:    request.Properties.Google.Master.Version,
		}, nil
	default:
		return nil, notSupportedCloudType(request.Cloud)
	}

}
