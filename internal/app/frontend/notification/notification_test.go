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

package notification

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//go:generate sh -c "which mockery > /dev/null && mockery -name Store -inpkg -testonly || true"

func TestService_GetNotifications(t *testing.T) {
	store := &MockStore{}

	ctx := context.Background()

	notifications := []Notification{
		{
			ID:       1,
			Message:  "message",
			Priority: 100,
		},
	}

	store.On("GetActiveNotifications", ctx).Return(notifications, nil)

	service := NewService(store)

	activeNotifications, err := service.GetNotifications(ctx)

	require.NoError(t, err)
	assert.Equal(
		t,
		Notifications{
			Messages: notifications,
		},
		activeNotifications,
	)

	store.AssertExpectations(t)
}
