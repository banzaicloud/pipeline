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
	"github.com/stretchr/testify/require"

	"github.com/banzaicloud/pipeline/internal/app/frontend/notification"
)

func TestMakeMakeGetActiveNotificationsEndpoint(t *testing.T) {
	notificationService := &notification.MockService{}

	ctx := context.Background()

	activeNotifications := notification.ActiveNotifications{
		Messages: []notification.Notification{
			{
				ID:       1,
				Message:  "message",
				Priority: 100,
			},
		},
	}

	notificationService.On("GetActiveNotifications", ctx).Return(activeNotifications, nil)

	e := MakeGetActiveNotificationsEndpoint(notificationService)

	result, err := e(ctx, nil)

	require.NoError(t, err)
	assert.Equal(t, activeNotifications, result)

	notificationService.AssertExpectations(t)
}
