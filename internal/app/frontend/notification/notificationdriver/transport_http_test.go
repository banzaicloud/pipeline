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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/banzaicloud/pipeline/internal/app/frontend/notification"
)

func TestMakeHTTPHandler_GetActiveNotifications(t *testing.T) {
	service := &notification.MockService{}

	activeNotifications := notification.ActiveNotifications{
		Messages: []notification.Notification{
			{
				ID:       1,
				Message:  "message",
				Priority: 100,
			},
		},
	}

	service.On("GetActiveNotifications", mock.Anything).Return(activeNotifications, nil)

	handler := MakeHTTPHandler(MakeEndpoints(service), emperror.NewNoopHandler())

	ts := httptest.NewServer(handler)
	defer ts.Close()

	tsClient := ts.Client()

	resp, err := tsClient.Get(ts.URL)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, resp.StatusCode, http.StatusOK)

	decoder := json.NewDecoder(resp.Body)

	var notifications notification.ActiveNotifications

	err = decoder.Decode(&notifications)
	require.NoError(t, err)

	assert.Equal(t, activeNotifications, notifications)
}
