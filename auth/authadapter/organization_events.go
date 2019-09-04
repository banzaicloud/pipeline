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

package authadapter

import (
	"context"

	"emperror.dev/errors"

	"github.com/banzaicloud/pipeline/auth"
)

// OrganizationEventDispatcher dispatches organization events.
type OrganizationEventDispatcher struct {
	eventBus EventBus
}

// NewOrganizationEventDispatcher returns a new OrganizationEventDispatcher instance.
func NewOrganizationEventDispatcher(eventBus EventBus) OrganizationEventDispatcher {
	return OrganizationEventDispatcher{
		eventBus: eventBus,
	}
}

// OrganizationCreated dispatches an OrganizationCreated event.
func (e OrganizationEventDispatcher) OrganizationCreated(ctx context.Context, event auth.OrganizationCreated) error {
	err := e.eventBus.Publish(ctx, event)
	if err != nil {
		return errors.WithDetails(
			errors.WithMessage(err, "failed to dispatch event"),
			"event", "OrganizationCreated",
			"userId", event.UserID,
			"organizationId", event.ID,
		)
	}

	return nil
}
