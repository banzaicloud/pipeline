package api

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/banzaicloud/banzai-types/components"
	"github.com/banzaicloud/pipeline/auth"
	"github.com/banzaicloud/pipeline/model"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func GetUsers(c *gin.Context) {

	log := logger.WithFields(logrus.Fields{"tag": "GetUsers"})
	log.Info("Fetching users")

	organization := auth.GetCurrentOrganization(c.Request)

	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if idParam != "" && err != nil {
		message := fmt.Sprintf("error parsing user id: %s", err)
		log.Info(message)
		c.JSON(http.StatusBadRequest, components.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: message,
			Error:   message,
		})
		return
	}

	var users []auth.User
	db := model.GetDB()
	err = db.Model(organization).Related(&users, "Users").Error
	if err != nil {
		message := "failed to fetch users"
		log.Info(message + ": " + err.Error())
		c.AbortWithStatusJSON(http.StatusInternalServerError, components.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: message,
			Error:   message,
		})
	} else if id == 0 {
		c.JSON(http.StatusOK, users)
	} else if len(users) == 1 {
		c.JSON(http.StatusOK, users[0])
	} else if len(users) > 1 {
		message := fmt.Sprintf("multiple users found with id: %d", id)
		log.Info(message)
		c.AbortWithStatusJSON(http.StatusConflict, components.ErrorResponse{
			Code:    http.StatusConflict,
			Message: message,
			Error:   message,
		})
	} else {
		message := fmt.Sprintf("user not found with id: %d", id)
		log.Info(message)
		c.AbortWithStatusJSON(http.StatusNotFound, components.ErrorResponse{
			Code:    http.StatusNotFound,
			Message: message,
			Error:   message,
		})
	}
}
