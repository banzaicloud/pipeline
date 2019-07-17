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

package notification

import (
	"net/http"

	"emperror.dev/emperror"
	"github.com/gin-gonic/gin"

	"github.com/banzaicloud/pipeline/config"
	pkgCommon "github.com/banzaicloud/pipeline/pkg/common"
)

// GetNotificationsResponse Api object to be mapped to Get notifications request
// swagger:model GetNotificationsResponse
type GetNotificationsResponse struct {
	Messages []MessagesResponse `json:"messages"`
}

type MessagesResponse struct {
	Id       uint   `json:"id"`
	Message  string `json:"message"`
	Priority int8   `json:"priority"`
}

// swagger:route GET /notifications  GetNotifications
//
// Lists all notifications
//
//     Produces:
//     - application/json
//
//     Schemes: http
//
//     Security:
//
//     Responses:
//       200: GetNotificationsResponse
func GetNotifications(c *gin.Context) {
	log.Info("Fetching notifications")

	if response, err := getValidNotifications(); err != nil {
		log.Errorf("Error during listing valid notifications: %s", err.Error())
		c.JSON(http.StatusBadRequest, pkgCommon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Error during listing valid notifications",
			Error:   err.Error(),
		})
	} else {
		c.JSON(http.StatusOK, GetNotificationsResponse{Messages: response})
	}
}

func getValidNotifications() ([]MessagesResponse, error) {
	var notifications []NotificationModel

	db := config.DB()

	err := db.Find(&notifications, "NOW() BETWEEN initial_time AND end_time").Error
	if err != nil {
		return nil, emperror.Wrap(err, "failed to find notifications")
	}
	response := make([]MessagesResponse, 0)

	for _, notification := range notifications {
		response = append(response, MessagesResponse{
			Id:       notification.ID,
			Message:  notification.Message,
			Priority: notification.Priority,
		})
	}
	return response, nil
}
