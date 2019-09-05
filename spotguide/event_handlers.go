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

package spotguide

import (
	"context"

	"emperror.dev/errors"

	"github.com/banzaicloud/pipeline/internal/common"
)

// OrganizationCreated event is triggered when an organization is created in the system.
type OrganizationCreated struct {
	// ID is the created organization ID.
	ID uint

	// UserID is the ID of the user whose login triggered the organization being created.
	UserID uint
}

// InitialSpotguideScrapeHandler handles OrganizationCreated events and triggers spotguide scraping for them.
type InitialSpotguideScrapeHandler struct {
	spotguideManager *SpotguideManager
	logger           common.Logger
}

// NewInitialSpotguideScrapeHandler returns a new InitialSpotguideScrapeHandler instance.
func NewInitialSpotguideScrapeHandler(spotguideManager *SpotguideManager, logger common.Logger) InitialSpotguideScrapeHandler {
	return InitialSpotguideScrapeHandler{
		spotguideManager: spotguideManager,
		logger:           logger,
	}
}

// OrganizationCreated scrapes spotguides for new organizations.
func (h InitialSpotguideScrapeHandler) OrganizationCreated(ctx context.Context, event OrganizationCreated) error {
	h.logger.Info(
		"starting initial spotguide scraping",
		map[string]interface{}{
			"organizationId": event.ID,
		},
	)

	err := h.spotguideManager.ScrapeSpotguides(event.ID, event.UserID)
	if err != nil {
		h.logger.Warn(
			errors.WithMessage(err, "failed to scrape Spotguide repositories").Error(),
			map[string]interface{}{
				"organizationId": event.ID,
			},
		)
	}

	return nil
}
