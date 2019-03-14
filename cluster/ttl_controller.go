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

package cluster

import (
	"context"
	"time"

	intCluster "github.com/banzaicloud/pipeline/internal/cluster"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	"github.com/goph/emperror"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/util/workqueue"
)

type clusterEventsSubscriber interface {
	SubscribeAsync(topic string, fn interface{}, transactional bool) error
}

// TtlController periodically checks running clusters and delete those with cluster age exceeding the configured TTL
type TtlController struct {
	manager *Manager

	// clusterEvents is the event bus through which cluster created and deleted notifications are received
	clusterEvents clusterEventsSubscriber

	// queue is where incoming work is placed to de-dup and to allow "easy"
	// rate limited re-queues on errors
	queue workqueue.RateLimitingInterface

	logger       logrus.FieldLogger
	errorHandler emperror.Handler
}

func (c *TtlController) Start() error {
	c.logger.Info("starting cluster TTL controller")
	clusters, err := c.manager.GetAllClusters(context.Background())

	if err != nil {
		return emperror.Wrap(err, "retrieving clusters failed")
	}

	for _, cluster := range clusters {

		c.enqueueCluster(cluster.GetID())
	}

	// we are interested in clusters created later
	c.clusterEvents.SubscribeAsync(clusterCreatedTopic, c.enqueueCluster, false)

	// we are interested in clusters being updated as their TTL setting may change
	c.clusterEvents.SubscribeAsync(clusterUpdatedTopic, c.enqueueCluster, false)

	go c.runWorker()

	return nil

}

func (c *TtlController) Stop() {
	c.logger.Info("shutting cluster TTL controller")
	c.queue.ShutDown()
}

// NewTtlController instantiates a new cluster TTL controller
func NewTtlController(manager *Manager, clusterEvents clusterEventsSubscriber, logger logrus.FieldLogger, errorHandler emperror.Handler) *TtlController {
	return &TtlController{
		manager:       manager,
		clusterEvents: clusterEvents,
		queue:         workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "ttl-controller"),
		logger:        logger,
		errorHandler:  errorHandler,
	}
}

func (c *TtlController) enqueueCluster(clusterID uint) {
	if !c.queue.ShuttingDown() {
		c.queue.Add(clusterID)
	}
}

// runWorker runs the loop that processes clusters taken from the workqueue
func (c *TtlController) runWorker() {
	// loop until we are told to quit
	for c.processNextCluster() {
	}
}

// processNextCluster takes one cluster id off the queue for processing.
// It returns false when it's time to quit
func (c *TtlController) processNextCluster() bool {
	// get next cluster off the queue
	clusterID, quit := c.queue.Get()
	if quit {
		return false
	}

	// tell to the queue that we finished processing the work item
	defer c.queue.Done(clusterID)

	err := c.handleCluster(clusterID.(uint))
	if err != nil {
		// processing the cluster failed; requeue to be retried later
		c.errorHandler.Handle(err)

		c.queue.AddRateLimited(clusterID)
	} else {
		// successfully processed cluster; tell the queue to stop tracking the work item for retries
		c.queue.Forget(clusterID)
	}

	return true
}

func (c *TtlController) handleCluster(clusterID uint) error {
	cluster, err := c.manager.GetClusterByIDOnly(context.Background(), clusterID)

	if err != nil && !intCluster.IsClusterNotFoundError(err) {
		return emperror.WrapWith(err, "failed to retrieve cluster", "clusterID", clusterID)
	}

	clusterDetail, err := cluster.GetStatus()
	if err != nil {
		return emperror.WrapWith(err, "failed to retrieve cluster details", "clusterID", clusterID)
	}

	statusHistory, err := c.manager.GetClusterStatusHistory(context.Background(), clusterID)
	if err != nil {
		return emperror.Wrap(err, "failed to retrieve cluster status  history")
	}

	clusterStartedAt := c.getClusterStartTime(statusHistory)

	log := c.logger.WithFields(logrus.Fields{
		"organization": cluster.GetOrganizationId(),
		"clusterID":    cluster.GetID(),
		"cluster":      cluster.GetName(),
		"status":       clusterDetail.Status,
		"created_at":   clusterDetail.CreatedAt,
		"ttlMinutes":   clusterDetail.TtlMinutes,
	})

	if clusterStartedAt.IsZero() {
		log = log.WithField("started_at", "")
	} else {
		log = log.WithField("started_at", clusterStartedAt)
	}

	// check only running clusters that have a TTL assigned
	if !c.hasTtl(clusterDetail) {
		log.Info("cluster has no TTL set, skip further processing")

		return nil
	}

	if !c.isClusterRunning(clusterDetail) {
		log.Infof("cluster is not in any of [%s, %s] states, skip further processing", pkgCluster.Running, pkgCluster.Warning)

		return nil
	}

	log.Debug("check if cluster has reached end of life")

	if c.isClusterEndOfLife(clusterStartedAt, clusterDetail.TtlMinutes) {
		log.Info("deleting cluster as it has reached end of life")

		err = c.manager.DeleteCluster(context.Background(), cluster, false)
		if err != nil {
			return emperror.WrapWith(err, "failed to initiate cluster deletion", "clusterID", clusterID)
		}

	} else {
		// schedule for later processing
		log.Debug("cluster has not reached end of life yet, schedule it for re-check")
		c.queue.AddAfter(clusterID, time.Duration(5*time.Minute))
	}

	return nil
}

// hasTtl returns true if the cluster has a TTL set
func (c *TtlController) hasTtl(clusterDetail *pkgCluster.GetClusterStatusResponse) bool {
	return clusterDetail.TtlMinutes > 0
}

// isClusterRunning returns true if the cluster is up an running regardless of the health of the cluster
func (c *TtlController) isClusterRunning(clusterDetail *pkgCluster.GetClusterStatusResponse) bool {
	// deleting a cluster that is in updating state may fail on some cloud provider thus clusters in updating state should
	// not be considered for deletion in case of end of life. Process these clusters once they finished updating.
	return clusterDetail.Status == pkgCluster.Running || clusterDetail.Status == pkgCluster.Warning
}

// isClusterEndOfLife returns true if cluster has reached end of life according
func (c *TtlController) isClusterEndOfLife(clusterStarTime time.Time, ttlMinutes uint) bool {
	if clusterStarTime.IsZero() {
		return false
	}

	clusterEndTime := clusterStarTime.Add(time.Duration(ttlMinutes) * time.Minute)

	return time.Now().After(clusterEndTime)
}

// getClusterStartTime returns the time when cluster status changed from creating -> running|warning
// if no such status transition found in cluster status history than returns zero time
func (c *TtlController) getClusterStartTime(statusHistory *pkgCluster.StatusHistory) time.Time {
	if statusHistory == nil {
		return time.Time{}
	}

	// find the time when cluster become running
	found := false
	var statusChange *pkgCluster.StatusChange

	for _, statusChange = range statusHistory.StatusChanges {
		if statusChange.FromStatus == pkgCluster.Creating && (statusChange.ToStatus == pkgCluster.Running || statusChange.ToStatus == pkgCluster.Warning) {
			found = true
			break
		}
	}

	if !found {
		return time.Time{}
	}

	return statusChange.CreatedAt
}
