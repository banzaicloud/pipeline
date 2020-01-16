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
	"time"

	"emperror.dev/errors"
	"go.uber.org/cadence/workflow"
)

const ScrapeSharedSpotguidesWorkflowName = "scrape-shared-spotguides"
const ScrapeSharedSpotguidesActivityName = "scrape-shared-spotguides-activty"

type ScrapeSharedSpotguidesActivity struct {
	manager *SpotguideManager
}

func NewScrapeSharedSpotguidesActivity(manager *SpotguideManager) ScrapeSharedSpotguidesActivity {
	return ScrapeSharedSpotguidesActivity{
		manager: manager,
	}
}

func (a ScrapeSharedSpotguidesActivity) Execute(_ context.Context) error {
	return errors.WrapIf(a.manager.scrapeSharedSpotguides(), "failed to scrape shared spotguides")
}

func ScrapeSharedSpotguidesWorkflow(ctx workflow.Context) error {

	ao := workflow.ActivityOptions{
		ScheduleToStartTimeout: 5 * time.Minute,
		StartToCloseTimeout:    10 * time.Minute,
		WaitForCancellation:    true,
	}

	ctx = workflow.WithActivityOptions(ctx, ao)

	return workflow.ExecuteActivity(ctx, ScrapeSharedSpotguidesActivityName).Get(ctx, nil)
}
