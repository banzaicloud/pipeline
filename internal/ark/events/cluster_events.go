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

type clusterEvents interface {
	NotifyClusterDeleted(fn interface{})
}

type eventBus interface {
	SubscribeAsync(topic string, fn interface{}, transactional bool) error
}

type clusterEventBus struct {
	eb eventBus
}

const (
	clusterCreatedTopic = "cluster_created"
	clusterDeletedTopic = "cluster_deleted"
)

// NewClusterEvents gives back a new clusterEventBus
func NewClusterEvents(eb eventBus) *clusterEventBus {
	return &clusterEventBus{
		eb: eb,
	}
}

// NotifyClusterCreated subscribes to clusterCreatedTopic
func (c *clusterEventBus) NotifyClusterCreated(fn interface{}) {
	c.eb.SubscribeAsync(clusterCreatedTopic, fn, false)
}

// NotifyClusterDeleted subscribes to clusterDeletedTopic
func (c *clusterEventBus) NotifyClusterDeleted(fn interface{}) {
	c.eb.SubscribeAsync(clusterDeletedTopic, fn, false)
}
