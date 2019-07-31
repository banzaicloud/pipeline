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
	"emperror.dev/emperror"
	"go.uber.org/cadence/workflow"
)

const ScrapeSharedSpotguidesWorkflowName = "scrape-shared-spotguides"

type ScrapeSharedSpotguidesWorkflow struct {
	manager *SpotguideManager
}

type ClusterDNSRecordsDeleter interface {
	Delete(organizationID uint, clusterUID string) error
}

func NewScrapeSharedSpotguidesWorkflow(manager *SpotguideManager) ScrapeSharedSpotguidesWorkflow {
	return ScrapeSharedSpotguidesWorkflow{
		manager: manager,
	}
}

func (a ScrapeSharedSpotguidesWorkflow) Execute(ctx workflow.Context) error {
	return emperror.Wrap(a.manager.scrapeSharedSpotguides(), "failed to scrape shared spotguides")
}
