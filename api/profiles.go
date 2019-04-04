// Copyright © 2018 Banzai Cloud
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package api

import (
	"net/http"

	"github.com/banzaicloud/pipeline/internal/platform/database"
	"github.com/banzaicloud/pipeline/model/defaults"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	pkgCommon "github.com/banzaicloud/pipeline/pkg/common"
	pkgErrors "github.com/banzaicloud/pipeline/pkg/errors"
	oracle "github.com/banzaicloud/pipeline/pkg/providers/oracle/model"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
)

const (
	distributionTypeKey = "distribution"
	nameKey             = "name"
)

// GetClusterProfiles handles /profiles/cluster/:type GET api endpoint.
// Sends back the saved cluster profiles
func GetClusterProfiles(c *gin.Context) {

	distributionType := c.Param(distributionTypeKey)
	log.Infof("Start getting saved cluster profiles [%s]", distributionType)

	resp, err := getProfiles(distributionType)
	if err != nil {
		log.Errorf("Error during getting defaults to %s: %s", distributionType, err.Error())
		c.JSON(http.StatusBadRequest, pkgCommon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: err.Error(),
			Error:   err.Error(),
		})
	} else {
		c.JSON(http.StatusOK, resp)
	}

}

// AddClusterProfile handles /profiles/cluster/:type POST api endpoint.
// Saves ClusterProfileRequest data into the database.
// Saving failed if profile with the given name is already exists
func AddClusterProfile(c *gin.Context) {

	log.Info("Start getting save cluster profile")

	log.Debug("Bind json into ClusterProfileRequest struct")
	// bind request body to struct
	var profileRequest pkgCluster.ClusterProfileRequest
	if err := c.BindJSON(&profileRequest); err != nil {
		log.Error(errors.Wrap(err, "Error parsing request"))
		c.JSON(http.StatusBadRequest, pkgCommon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error parsing request",
			Error:   err.Error(),
		})
		return
	}
	log.Info("Parsing request succeeded")
	log.Infof("Convert ClusterProfileRequest into ClusterProfile model with name: %s", profileRequest.Name)

	// convert request into ClusterProfile model
	if prof, err := convertRequestToProfile(&profileRequest); err != nil {
		log.Error("Error during convert profile: &s", err.Error())
		c.JSON(http.StatusBadRequest, pkgCommon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error during convert profile",
			Error:   err.Error(),
		})
	} else if !prof.IsDefinedBefore() {
		// name is free
		log.Info("Convert succeeded")
		log.Info("Save cluster profile into database")
		if err := prof.SaveInstance(); err != nil {
			// save failed
			log.Errorf("Error during persist cluster profile: %s", err.Error())
			c.JSON(http.StatusInternalServerError, pkgCommon.ErrorResponse{
				Code:    http.StatusInternalServerError,
				Message: "Error during persist cluster profile",
				Error:   err.Error(),
			})
		} else {
			// save succeeded
			log.Info("Save cluster profile succeeded")
			c.Status(http.StatusCreated)
		}
	} else {
		// profile with given name is already exists
		log.Error("Cluster profile with the given name is already exists")
		c.JSON(http.StatusBadRequest, pkgCommon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Cluster profile with the given name is already exists, please update not create profile",
			Error:   "Cluster profile with the given name is already exists, please update not create profile",
		})
	}

}

// getProfiles loads cluster profiles from database by distribution
func getProfiles(distribution string) ([]pkgCluster.ClusterProfileResponse, error) {

	var response []pkgCluster.ClusterProfileResponse
	profiles, err := defaults.GetAllProfiles(distribution)
	if err != nil {
		// error during getting profiles
		return nil, err
	}
	for _, p := range profiles {
		r := p.GetProfile()
		response = append(response, *r)
	}
	return response, nil

}

// convertRequestToProfile converts a ClusterProfileRequest into ClusterProfile
func convertRequestToProfile(request *pkgCluster.ClusterProfileRequest) (defaults.ClusterProfile, error) {

	switch request.Cloud {
	case pkgCluster.Amazon:
		var eksProfile defaults.EKSProfile
		eksProfile.UpdateProfile(request, false)
		return &eksProfile, nil
	case pkgCluster.Azure:
		var aksProfile defaults.AKSProfile
		aksProfile.UpdateProfile(request, false)
		return &aksProfile, nil
	case pkgCluster.Google:
		var gkeProfile defaults.GKEProfile
		gkeProfile.UpdateProfile(request, false)
		return &gkeProfile, nil
	case pkgCluster.Oracle:
		var okeProfile oracle.Profile
		okeProfile.UpdateProfile(request, false)
		return &okeProfile, nil
	default:
		return nil, pkgErrors.ErrorNotSupportedCloudType
	}

}

// UpdateClusterProfile handles /cluster/profiles/:type PUT api endpoint.
// Updates existing cluster profiles.
// Updating failed if the name is the default name.
func UpdateClusterProfile(c *gin.Context) {

	log.Debug("Bind json into ClusterProfileRequest struct")
	// bind request body to struct
	var profileRequest pkgCluster.ClusterProfileRequest
	if err := c.BindJSON(&profileRequest); err != nil {
		log.Error(errors.Wrap(err, "Error parsing request"))
		c.JSON(http.StatusBadRequest, pkgCommon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error parsing request",
			Error:   err.Error(),
		})
		return
	}
	log.Debug("Parsing request succeeded")

	if defaults.GetDefaultProfileName() == profileRequest.Name {
		// default profiles cannot updated
		log.Error("The default profile cannot be updated") // todo move to constants
		c.JSON(http.StatusBadRequest, pkgCommon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "The default profile cannot be updated",
			Error:   "The default profile cannot be updated",
		})
		return
	}

	log.Infof("Load cluster from database: %s[%s]", profileRequest.Name, profileRequest.Cloud)

	if profileRequest.Properties.ACSK != nil {
		profileRequest.Properties.ACK = profileRequest.Properties.ACSK
	}

	distribution := pkgCluster.Unknown
	switch profileRequest.Cloud {
	case pkgCluster.Amazon:
		distribution = pkgCluster.EKS
	case pkgCluster.Azure:
		distribution = pkgCluster.AKS
	case pkgCluster.Google:
		distribution = pkgCluster.GKE
	case pkgCluster.Oracle:
		distribution = pkgCluster.OKE
	default:
		log.Infoln("Not supported cloud type", profileRequest.Cloud)
		c.JSON(http.StatusBadRequest, pkgCommon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Not supported cloud type",
			Error:   "Not supported cloud type",
		})
		return
	}

	// load cluster profile from database
	if profile, err := defaults.GetProfile(distribution, profileRequest.Name); err != nil {
		// load from db failed
		log.Error(errors.Wrap(err, "Error during getting profile"))
		sendBackGetProfileErrorResponse(c, err)
	} else if err := profile.UpdateProfile(&profileRequest, true); err != nil {
		// updating failed
		log.Error(errors.Wrap(err, "Error during update profile"))
		c.JSON(http.StatusInternalServerError, pkgCommon.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: "Error during update profile",
			Error:   err.Error(),
		})
	} else {
		// update success
		log.Infof("Update succeeded")
		c.Status(http.StatusCreated)
	}

}

// DeleteClusterProfile handles /cluster/profiles/:type/:name DELETE api endpoint.
// Deletes saved cluster profile.
// Deleting failed if the name is the default name.
func DeleteClusterProfile(c *gin.Context) {

	distribution := c.Param(distributionTypeKey)
	name := c.Param(nameKey)
	log.Infof("Start deleting cluster profile: %s[%s]", name, distribution)

	if defaults.GetDefaultProfileName() == name {
		// default profile cannot deleted
		log.Error("The default profile cannot be deleted")
		c.JSON(http.StatusBadRequest, pkgCommon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "The default profile cannot be deleted",
			Error:   "The default profile cannot be deleted",
		})
		return
	}

	log.Infof("Load cluster profile from database: %s[%s]", name, distribution)

	// load cluster profile from database
	if profile, err := defaults.GetProfile(distribution, name); err != nil {
		// load from database failed
		log.Error(errors.Wrap(err, "Error during getting profile"))
		sendBackGetProfileErrorResponse(c, err)
	} else {
		log.Info("Getting profile succeeded")
		log.Info("Delete from database")
		if err := profile.DeleteProfile(); err != nil {
			// delete from db failed
			log.Error(errors.Wrap(err, "Error during profile delete"))
			c.JSON(http.StatusInternalServerError, pkgCommon.ErrorResponse{
				Code:    http.StatusInternalServerError,
				Message: "Error during profile delete",
				Error:   err.Error(),
			})
		} else {
			// delete succeeded
			log.Info("Delete from database succeeded")
			c.Status(http.StatusOK)
		}
	}

}

func sendBackGetProfileErrorResponse(c *gin.Context, err error) {
	statusCode := http.StatusBadRequest
	msg := "Error during getting profile"
	if database.IsRecordNotFoundError(err) {
		statusCode = http.StatusNotFound
		msg = "Profile not found"
	}

	c.JSON(statusCode, pkgCommon.ErrorResponse{
		Code:    statusCode,
		Message: msg,
		Error:   err.Error(),
	})
}
