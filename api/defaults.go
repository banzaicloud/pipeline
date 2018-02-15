package api

import (
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/banzaicloud/banzai-types/constants"
	"github.com/banzaicloud/banzai-types/components"
	"net/http"
	"github.com/banzaicloud/pipeline/model/defaults"
	"github.com/pkg/errors"
)

const (
	cloudTypeKey = "type"
	nameKey      = "name"
)

func GetDefaults(c *gin.Context) {

	log := logger.WithFields(logrus.Fields{"tag": constants.TagGetClusterProfile})

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
	log := logger.WithFields(logrus.Fields{"tag": constants.TagSetClusterProfile})

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

func getDefaultClusterProfile(cloudType string) ([]components.ClusterProfileRespone, error) {

	var response []components.ClusterProfileRespone
	if profiles, err := defaults.GetAllProfiles(cloudType); err != nil {
		// error during getting profiles
		return nil, err
	} else {
		for _, p := range profiles {
			r := p.GetProfile()
			response = append(response, *r)
		}
		return response, nil
	}

}

// convertRequestToProfile converts a ClusterProfileRequest into ClusterProfile
func convertRequestToProfile(request *components.ClusterProfileRequest) (defaults.ClusterProfile, error) {

	switch request.Cloud {
	case constants.Amazon:
		var awsProfile defaults.AWSProfile
		awsProfile.UpdateProfile(request, false)
		return &awsProfile, nil
	case constants.Azure:
		var aksProfile defaults.AKSProfile
		aksProfile.UpdateProfile(request, false)
		return &aksProfile, nil
	case constants.Google:
		var gkeProfile defaults.GKEProfile
		gkeProfile.UpdateProfile(request, false)
		return &gkeProfile, nil
	default:
		return nil, constants.NotSupportedCloudType
	}

}

func UpdateClusterProfile(c *gin.Context) {
	log := logger.WithFields(logrus.Fields{"tag": constants.TagUpdateClusterProfile})

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
	log.Debug("Parsing request succeeded")

	if "default" == profileRequest.ProfileName { // todo move to constants
		log.Error("The default profile cannot be updated") // todo move to constants
		c.JSON(http.StatusBadRequest, components.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "The default profile cannot be updated",
			Error:   "The default profile cannot be updated",
		})
		return
	}

	if profile, err := defaults.GetProfile(cloudType, profileRequest.ProfileName); err != nil {
		log.Error(errors.Wrap(err, "Error during getting profile"))
		c.JSON(http.StatusInternalServerError, components.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: "Error during getting profile",
			Error:   err.Error(),
		})
	} else if err := profile.UpdateProfile(&profileRequest, true); err != nil {
		log.Error(errors.Wrap(err, "Error during update profile"))
		c.JSON(http.StatusInternalServerError, components.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: "Error during update profile",
			Error:   err.Error(),
		})
	} else {
		log.Infof("Update succeeded")
		c.Status(http.StatusCreated)
	}

}

func DeleteClusterProfile(c *gin.Context) {
	log := logger.WithFields(logrus.Fields{"tag": constants.TagDeleteClusterProfile})

	cloudType := c.Param(cloudTypeKey)
	name := c.Param(nameKey)
	log.Infof("Delete profile: %s[%s]", name, cloudType)

	if "default" == name { // todo move to constants
		log.Error("The default profile cannot be deleted")
		c.JSON(http.StatusBadRequest, components.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "The default profile cannot be deleted",
			Error:   "The default profile cannot be deleted",
		})
		return
	}

	if profile, err := defaults.GetProfile(cloudType, name); err != nil {
		log.Error(errors.Wrap(err, "Error during getting profile"))
		c.JSON(http.StatusInternalServerError, components.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: "Error during getting profile",
			Error:   err.Error(),
		})
	} else {
		log.Info("Getting profile succeeded")
		log.Info("Delete from database")
		if err := profile.DeleteProfile(); err != nil {
			log.Error(errors.Wrap(err, "Error during profile delete"))
			c.JSON(http.StatusInternalServerError, components.ErrorResponse{
				Code:    http.StatusInternalServerError,
				Message: "Error during profile delete",
				Error:   err.Error(),
			})
		} else {
			log.Info("Delete from database succeeded")
			c.Status(http.StatusOK)
		}
	}

}
