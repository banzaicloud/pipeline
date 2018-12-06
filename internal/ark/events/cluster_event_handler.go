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

package events

import (
	"github.com/goph/emperror"
	"github.com/jinzhu/gorm"
	"github.com/sirupsen/logrus"

	"github.com/banzaicloud/pipeline/internal/ark"
)

// ClusterEventHandler is for handling cluster events
type ClusterEventHandler struct {
	events clusterEvents
	db     *gorm.DB
	logger logrus.FieldLogger
}

// NewClusterEventHandler handles arriving cluster events such as 'cluster_deleted'
func NewClusterEventHandler(events clusterEvents, db *gorm.DB, logger logrus.FieldLogger) *ClusterEventHandler {
	eh := &ClusterEventHandler{
		events: events,
		db:     db,
		logger: logger,
	}

	eh.events.NotifyClusterDeleted(func(orgID uint, clusterName string) {
		eh.DeleteStaleARKDeployments(orgID)
	})

	return eh
}

// RemoveStaleDeployments deletes stale ARK deployment records from database
func (eh *ClusterEventHandler) DeleteStaleARKDeployments(orgID uint) error {
	var deployments []*ark.ClusterBackupDeploymentsModel
	log := eh.logger.WithField("org", orgID)
	log.Debug("removing stale ark deployment records")

	err := eh.db.Where(ark.ClusterBackupDeploymentsModel{OrganizationID: orgID}).Preload("Cluster").Find(&deployments).Error
	if err != nil {
		return err
	}

	for _, deployment := range deployments {
		if deployment.ID > 0 && deployment.Cluster.ID == 0 {
			err = eh.db.Delete(&deployment).Error
			if err != nil {
				log.Error(emperror.Wrap(err, "could not delete deployment record"))
			}
		}
	}

	return nil
}
