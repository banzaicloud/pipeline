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

// ActiveNotifications is the list of notifications that are currently active.
type ActiveNotifications struct {
	Messages []Notification `json:"messages"`
}

// Notification represents a single notification.
type Notification struct {
	ID       uint   `json:"id"`
	Message  string `json:"message"`
	Priority int8   `json:"priority"`
}

// Service provides an interface to notifications.
type Service struct {
	store Store
}

// Store is a data persistence layer for notifications.
type Store interface {
	// GetActiveNotifications returns the list of active notifications.
	GetActiveNotifications(ctx context.Context) ([]Notification, error)
}

// NewService returns a new Service.
func NewService(store Store) *Service {
	return &Service{
		store: store,
	}
}

// GetActiveNotifications returns the list of active notifications.
func (s *Service) GetActiveNotifications(ctx context.Context) (ActiveNotifications, error) {
	notifications, err := s.store.GetActiveNotifications(ctx)
	if err != nil {
		return ActiveNotifications{}, err
	}

	return ActiveNotifications{Messages: notifications}, nil
}
