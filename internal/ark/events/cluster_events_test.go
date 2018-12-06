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
	"testing"

	evbus "github.com/asaskevich/EventBus"
	"github.com/banzaicloud/pipeline/cluster"
)

func TestClusterCreatedEvent(t *testing.T) {
	oid := uint(1)

	clusterEventBus := evbus.New()
	publisher := cluster.NewClusterEvents(clusterEventBus)

	ok := false
	listener := NewClusterEvents(clusterEventBus)
	listener.NotifyClusterCreated(func(clusterID uint) {
		if clusterID == oid {
			ok = true
		}
	})

	publisher.ClusterCreated(oid)

	clusterEventBus.WaitAsync()

	if !ok {
		t.Fail()
	}
}

func TestClusterDeletedEvent(t *testing.T) {
	oid := uint(1)
	cname := "clustername"

	clusterEventBus := evbus.New()
	publisher := cluster.NewClusterEvents(clusterEventBus)

	ok := false
	listener := NewClusterEvents(clusterEventBus)
	listener.NotifyClusterDeleted(func(orgID uint, clusterName string) {
		if orgID == oid && clusterName == cname {
			ok = true
		}
	})

	publisher.ClusterDeleted(oid, cname)

	clusterEventBus.WaitAsync()

	if !ok {
		t.Fail()
	}
}
