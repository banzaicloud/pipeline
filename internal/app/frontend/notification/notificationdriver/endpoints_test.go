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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/banzaicloud/pipeline/internal/app/frontend/notification"
)

func TestMakeEndpoints_GetNotifications(t *testing.T) {
	notificationService := &notification.MockService{}

	notifications := notification.Notifications{
		Messages: []notification.Notification{
			{
				ID:       1,
				Message:  "message",
				Priority: 100,
			},
		},
	}

	notificationService.On("GetNotifications", mock.Anything).Return(notifications, nil)

	e := MakeEndpoints(notificationService).GetNotifications

	result, err := e(context.Background(), nil)

	require.NoError(t, err)
	assert.Equal(t, notifications, result)

	notificationService.AssertExpectations(t)
}
