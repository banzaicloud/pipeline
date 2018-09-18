// Copyright © 2018 Banzai Cloud
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

package cluster

import (
	"context"

	"github.com/banzaicloud/pipeline/pkg/cluster"
	"github.com/goph/emperror"
	"github.com/sirupsen/logrus"
)

// RetryPendingOperations retries pending operations that stuck in the queue (eg. because Pipeline was restarted).
func RetryPendingOperations(manager *Manager, logger logrus.FieldLogger) error {
	creatingClusters, err := manager.GetClustersByStatus(context.Background(), cluster.Creating)
	if err != nil {
		return emperror.Wrap(err, "failed to retry CREATING operations")
	}

	if len(creatingClusters) < 1 {
		logger.Info("no clusters stuck in CREATING status, good job")

		return nil
	}

	for _, c := range creatingClusters {
		if c.GetCloud() != "google" { // TODO: remove safeguard once all cluster create works
			continue
		}

		creationCtx := CreationContext{
			OrganizationID: c.GetOrganizationId(),
			UserID:         c.GetCreatorID(),
			Name:           c.GetName(),
			SecretID:       c.GetSecretId(),
			Provider:       c.GetCloud(),
			PostHooks:      BasePostHookFunctions,
		}

		creator := NewRecoveryClusterCreator(c)

		_, err = manager.CreateCluster(context.Background(), creationCtx, creator)
		if err != nil {
			errorHandler.Handle(err)
		}
	}

	return nil
}
