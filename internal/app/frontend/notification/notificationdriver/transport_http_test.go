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
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/banzaicloud/pipeline/internal/app/frontend/notification"
)

func TestMakeHTTPHandler_GetNotifications(t *testing.T) {
	notifications := notification.Notifications{
		Messages: []notification.Notification{
			{
				ID:       1,
				Message:  "message",
				Priority: 100,
			},
		},
	}

	handler := mux.NewRouter()
	RegisterHTTPHandlers(
		Endpoints{
			GetNotifications: func(ctx context.Context, request interface{}) (response interface{}, err error) {
				return notifications, nil
			},
		},
		handler.PathPrefix("/notifications").Subrouter(),
	)

	ts := httptest.NewServer(handler)
	defer ts.Close()

	resp, err := ts.Client().Get(ts.URL + "/notifications")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var notificationResp notification.Notifications

	err = json.NewDecoder(resp.Body).Decode(&notificationResp)
	require.NoError(t, err)

	assert.Equal(t, notifications, notificationResp)
}
