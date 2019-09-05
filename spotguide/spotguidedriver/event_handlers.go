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

package spotguidedriver

import (
	"context"
	"fmt"

	"emperror.dev/errors"

	"github.com/banzaicloud/pipeline/spotguide"
)

// OrganizationCreatedHandler handles OrganizationCreated events.
type OrganizationCreatedHandler interface {
	// OrganizationCreated handles an OrganizationCreated event.
	OrganizationCreated(ctx context.Context, event spotguide.OrganizationCreated) error
}

// OrganizationCreatedEventHandler handles an OrganizationCreated events.
type OrganizationCreatedEventHandler struct {
	handler OrganizationCreatedHandler
}

// NewOrganizationCreatedEventHandler returns a new OrganizationCreatedEventHandler instance.
func NewOrganizationCreatedEventHandler(handler OrganizationCreatedHandler) *OrganizationCreatedEventHandler {
	return &OrganizationCreatedEventHandler{
		handler: handler,
	}
}

// HandlerName implements the cqrs.EventHandler interface.
func (OrganizationCreatedEventHandler) HandlerName() string {
	return "spotguide_organization_handler"
}

// NewEvent implements the cqrs.EventHandler interface.
func (*OrganizationCreatedEventHandler) NewEvent() interface{} {
	return &spotguide.OrganizationCreated{}
}

// Handle implements the cqrs.EventHandler interface.
func (h *OrganizationCreatedEventHandler) Handle(ctx context.Context, event interface{}) error {
	e, ok := event.(*spotguide.OrganizationCreated)
	if !ok {
		return errors.NewWithDetails(
			"unexpected event type",
			"type", fmt.Sprintf("%T", event),
		)
	}

	return h.handler.OrganizationCreated(ctx, *e)
}
