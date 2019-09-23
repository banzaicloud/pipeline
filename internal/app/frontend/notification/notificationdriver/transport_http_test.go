// Copyright © 2019 Banzai Cloud
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

package notificationdriver

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"emperror.dev/emperror"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/banzaicloud/pipeline/internal/app/frontend/notification"
)

func TestMakeHTTPHandler_GetActiveNotifications(t *testing.T) {
	service := new(notification.MockService)

	notifications := notification.Notifications{
		Messages: []notification.Notification{
			{
				ID:       1,
				Message:  "message",
				Priority: 100,
			},
		},
	}

	service.On("GetNotifications", mock.Anything).Return(notifications, nil)

	handler := mux.NewRouter()
	RegisterHTTPHandlers(MakeEndpoints(service), handler.PathPrefix("/notifications").Subrouter(), emperror.NewNoopHandler())

	ts := httptest.NewServer(handler)
	defer ts.Close()

	tsClient := ts.Client()

	resp, err := tsClient.Get(ts.URL + "/notifications")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	decoder := json.NewDecoder(resp.Body)

	var notificationResp notification.Notifications

	err = decoder.Decode(&notificationResp)
	require.NoError(t, err)

	assert.Equal(t, notifications, notificationResp)

	service.AssertExpectations(t)
}
