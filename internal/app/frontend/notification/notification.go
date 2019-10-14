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
)

// Notifications is the list of notifications active.
type Notifications struct {
	Messages []Notification `json:"messages"`
}

// Notification represents a single notification.
type Notification struct {
	ID       uint   `json:"id"`
	Message  string `json:"message"`
	Priority int8   `json:"priority"`
}

// Service provides an interface to notifications.
//go:generate mga gen kit endpoint --outdir notificationdriver --with-oc Service
//go:generate mockery -name Service -inpkg
type Service interface {
	// GetNotifications returns the list of notifications.
	GetNotifications(ctx context.Context) (Notifications, error)
}

type service struct {
	store Store
}

// NewService returns a new Service.
func NewService(store Store) Service {
	return &service{
		store: store,
	}
}

// Store is a data persistence layer for notifications.
type Store interface {
	// GetActiveNotifications returns the list of active notifications.
	GetActiveNotifications(ctx context.Context) ([]Notification, error)
}

// GetActiveNotifications returns the list of active notifications.
func (s *service) GetNotifications(ctx context.Context) (Notifications, error) {
	notifications, err := s.store.GetActiveNotifications(ctx)
	if err != nil {
		return Notifications{}, err
	}

	// The response is not nillable
	if notifications == nil {
		notifications = make([]Notification, 0)
	}

	return Notifications{Messages: notifications}, nil
}
