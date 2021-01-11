// Copyright © 2020 Banzai Cloud
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

package pkeservice

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/banzaicloud/pipeline/internal/cluster"
)

func TestRegisterNodeStatus(t *testing.T) {
	ctx := context.Background()
	rand := new(MockidGenerator)
	rand.On("New").Return("totally-random")
	ts := time.Date(2020, 2, 22, 10, 10, 0, 0, time.UTC)
	clusterID := cluster.Identifier{
		OrganizationID: 1,
		ClusterID:      2,
		ClusterName:    "dummy",
	}

	t.Run("first step", func(t *testing.T) {
		store := new(cluster.MockStore)
		process := new(MockprocessService)
		logger := new(Mocklogger)
		svc := NewService(store, process, logger, rand)

		status := NodeStatus{Name: "node-host", NodePool: "pool", Ip: "127.0.0.1", Message: "Register node as ready at Pipeline", Phase: "pipeline-ready", Final: false, Status: Running, Timestamp: ts, ProcessID: ""}

		// incoming request logged
		logger.On("Info",
			"node status update",
			map[string]interface{}{"clusterID": uint(2), "message": "Register node as ready at Pipeline", "nodeIP": "127.0.0.1", "nodeName": "node-host", "nodePool": "pool", "phase": "pipeline-ready", "remoteTime": ts}).Return()

		// no existing (cluster creation, etc) process
		process.On("ListProcesses", ctx,
			Process{Status: Running, ResourceId: "2", OrgId: 1, ResourceType: "cluster"}).Return(nil, nil)
		// creates new process (because no process id in request)
		process.On("LogProcess", ctx,
			Process{Id: "totally-random", ParentId: "", OrgId: 1, Type: "pke-bootstrap", Log: "pipeline-ready: Register node as ready at Pipeline", ResourceId: "2/node-host", ResourceType: "node", Status: Running, StartedAt: ts, FinishedAt: nil, Events: nil}).Return(Process{Id: "totally-random"}, nil)
		// creates first event in process
		process.On("LogProcessEvent", ctx,
			ProcessEvent{Id: 0, ProcessId: "totally-random", Type: "pke-pipeline-ready", Log: "Register node as ready at Pipeline", Status: Running, Timestamp: ts}).Return(ProcessEvent{}, nil)

		r, err := svc.RegisterNodeStatus(ctx, clusterID, status)
		require.NoError(t, err)
		require.Equal(t, "totally-random", r.ProcessID)
	})

	t.Run("first step with running process", func(t *testing.T) {
		store := new(cluster.MockStore)
		process := new(MockprocessService)
		logger := new(Mocklogger)
		svc := NewService(store, process, logger, rand)

		status := NodeStatus{Name: "node-host", NodePool: "pool", Ip: "127.0.0.1", Message: "Register node as ready at Pipeline", Phase: "pipeline-ready", Final: false, Status: Running, Timestamp: ts, ProcessID: ""}

		// incoming request logged
		logger.On("Info",
			"node status update",
			map[string]interface{}{"clusterID": uint(2), "message": "Register node as ready at Pipeline", "nodeIP": "127.0.0.1", "nodeName": "node-host", "nodePool": "pool", "phase": "pipeline-ready", "remoteTime": ts}).Return()

		// one running (cluster creation, etc) process
		process.On("ListProcesses", ctx,
			Process{Status: Running, ResourceId: "2", OrgId: 1, ResourceType: "cluster"}).Return([]Process{{Id: "parent"}}, nil)
		// creates new process (because no process id in request)
		process.On("LogProcess", ctx,
			Process{Id: "totally-random", ParentId: "parent", OrgId: 1, Type: "pke-bootstrap", Log: "pipeline-ready: Register node as ready at Pipeline", ResourceId: "2/node-host", ResourceType: "node", Status: Running, StartedAt: ts, FinishedAt: nil, Events: nil}).Return(Process{Id: "totally-random"}, nil)
		// creates first event in process
		process.On("LogProcessEvent", ctx,
			ProcessEvent{Id: 0, ProcessId: "totally-random", Type: "pke-pipeline-ready", Log: "Register node as ready at Pipeline", Status: Running, Timestamp: ts}).Return(ProcessEvent{}, nil)

		r, err := svc.RegisterNodeStatus(ctx, clusterID, status)
		require.NoError(t, err)
		require.Equal(t, "totally-random", r.ProcessID)
	})

	t.Run("subsequent step", func(t *testing.T) {
		store := new(cluster.MockStore)
		process := new(MockprocessService)
		logger := new(Mocklogger)
		svc := NewService(store, process, logger, rand)

		status := NodeStatus{Name: "node-host", NodePool: "pool", Ip: "127.0.0.1", Message: "msg", Phase: "subseq", Final: false, Status: Running, Timestamp: ts, ProcessID: "previous"}

		// incoming request logged
		logger.On("Info",
			"node status update",
			map[string]interface{}{"clusterID": uint(2), "message": "msg", "nodeIP": "127.0.0.1", "nodeName": "node-host", "nodePool": "pool", "phase": "subseq", "remoteTime": ts}).Return()

		proc := Process{Id: "previous", ParentId: "", OrgId: 1, Type: "pke-bootstrap", Log: "pipeline-ready: Register node as ready at Pipeline", ResourceId: "2/node-host", ResourceType: "node", Status: Running, StartedAt: ts, FinishedAt: nil, Events: nil}
		// no running (cluster creation, etc) process
		process.On("ListProcesses", ctx,
			Process{Status: Running, ResourceId: "2", OrgId: 1, ResourceType: "cluster"}).Return(nil, nil)
		// process id supplied in request
		process.On("GetProcess", ctx, "previous").Return(proc, nil)

		process.On("LogProcess", ctx, proc).Return(proc, nil)
		process.On("LogProcessEvent", ctx,
			ProcessEvent{Id: 0, ProcessId: "previous", Type: "pke-subseq", Log: "msg", Status: Running, Timestamp: ts}).Return(ProcessEvent{}, nil)

		_, err := svc.RegisterNodeStatus(ctx, clusterID, status)
		require.NoError(t, err)
	})

	t.Run("subsequent error step", func(t *testing.T) {
		store := new(cluster.MockStore)
		process := new(MockprocessService)
		logger := new(Mocklogger)
		svc := NewService(store, process, logger, rand)

		status := NodeStatus{Name: "node-host", NodePool: "pool", Ip: "127.0.0.1", Message: "we have a problem", Phase: "subseq", Final: false, Status: Failed, Timestamp: ts, ProcessID: "previous"}

		// incoming request logged
		logger.On("Info",
			"node status update",
			map[string]interface{}{"clusterID": uint(2), "message": "we have a problem", "nodeIP": "127.0.0.1", "nodeName": "node-host", "nodePool": "pool", "phase": "subseq", "remoteTime": ts}).Return()

		proc := Process{Id: "previous", ParentId: "", OrgId: 1, Type: "pke-bootstrap", Log: "pipeline-ready: Register node as ready at Pipeline", ResourceId: "2/node-host", ResourceType: "node", Status: Running, StartedAt: ts, FinishedAt: nil, Events: nil}
		// no running (cluster creation, etc) process
		process.On("ListProcesses", ctx,
			Process{Status: Running, ResourceId: "2", OrgId: 1, ResourceType: "cluster"}).Return(nil, nil)
		// process id supplied in request
		process.On("GetProcess", ctx, "previous").Return(proc, nil)
		// still running, non-fatal
		process.On("LogProcess", ctx, proc).Return(proc, nil)
		// event itself is failed
		process.On("LogProcessEvent", ctx,
			ProcessEvent{Id: 0, ProcessId: "previous", Type: "pke-subseq", Log: "we have a problem", Status: Failed, Timestamp: ts}).Return(ProcessEvent{}, nil)

		_, err := svc.RegisterNodeStatus(ctx, clusterID, status)
		require.NoError(t, err)
	})

	t.Run("last success step with workflow (no signal)", func(t *testing.T) {
		store := new(cluster.MockStore)
		process := new(MockprocessService)
		logger := new(Mocklogger)
		svc := NewService(store, process, logger, rand)

		newts := ts.Add(time.Second)
		status := NodeStatus{Name: "node-host", NodePool: "pool", Ip: "127.0.0.1", Message: "done", Phase: "subseq", Final: true, Status: Finished, Timestamp: newts, ProcessID: "previous"}

		// incoming request logged
		logger.On("Info",
			"node status update",
			map[string]interface{}{"clusterID": uint(2), "message": "done", "nodeIP": "127.0.0.1", "nodeName": "node-host", "nodePool": "pool", "phase": "subseq", "remoteTime": newts}).Return()

		proc := Process{Id: "previous", ParentId: "running-flow", OrgId: 1, Type: "pke-bootstrap", Log: "pipeline-ready: Register node as ready at Pipeline", ResourceId: "2/node-host", ResourceType: "node", Status: Running, StartedAt: ts, FinishedAt: nil, Events: nil}
		// one running (cluster creation, etc) process
		process.On("ListProcesses", ctx,
			Process{Status: Running, ResourceId: "2", OrgId: 1, ResourceType: "cluster"}).Return([]Process{{Id: "running-flow"}}, nil)
		// process id supplied in request
		process.On("GetProcess", ctx, "previous").Return(proc, nil)
		// finishing process
		proc.Status = Finished
		proc.FinishedAt = &newts
		proc.Log = "done"
		process.On("LogProcess", ctx, proc).Return(proc, nil)
		// event itself is failed
		process.On("LogProcessEvent", ctx,
			ProcessEvent{Id: 0, ProcessId: "previous", Type: "pke-subseq", Log: "done", Status: Finished, Timestamp: newts}).Return(ProcessEvent{}, nil)

		_, err := svc.RegisterNodeStatus(ctx, clusterID, status)
		require.NoError(t, err)
	})

	t.Run("last failed step with workflow signal", func(t *testing.T) {
		store := new(cluster.MockStore)
		process := new(MockprocessService)
		logger := new(Mocklogger)
		svc := NewService(store, process, logger, rand)

		newts := ts.Add(time.Second)
		status := NodeStatus{Name: "node-host", NodePool: "pool", Ip: "127.0.0.1", Message: "big trouble", Phase: "hard", Final: true, Status: Failed, Timestamp: newts, ProcessID: "previous"}

		// incoming request logged
		logger.On("Info",
			"node status update",
			map[string]interface{}{"clusterID": uint(2), "message": "big trouble", "nodeIP": "127.0.0.1", "nodeName": "node-host", "nodePool": "pool", "phase": "hard", "remoteTime": newts}).Return()

		proc := Process{Id: "previous", ParentId: "running-flow", OrgId: 1, Type: "pke-bootstrap", Log: "pipeline-ready: Register node as ready at Pipeline", ResourceId: "2/node-host", ResourceType: "node", Status: Running, StartedAt: ts, FinishedAt: nil, Events: nil}
		// one running (cluster creation, etc) process
		process.On("ListProcesses", ctx,
			Process{Status: Running, ResourceId: "2", OrgId: 1, ResourceType: "cluster"}).Return([]Process{{Id: "running-flow"}}, nil)
		// process id supplied in request
		process.On("GetProcess", ctx, "previous").Return(proc, nil)
		// finishing process
		proc.Status = Failed
		proc.FinishedAt = &newts
		proc.Log = "big trouble"
		process.On("LogProcess", ctx, proc).Return(proc, nil)
		// event itself is failed
		process.On("LogProcessEvent", ctx,
			ProcessEvent{Id: 0, ProcessId: "previous", Type: "pke-hard", Log: "big trouble", Status: Failed, Timestamp: newts}).Return(ProcessEvent{}, nil)
		// process signalled
		// newServiceError("node-host failed: big trouble")
		process.On("SignalProcess", ctx, "running-flow", "node-bootstrap-failed", mock.AnythingOfType("serviceError")).Return(nil)

		_, err := svc.RegisterNodeStatus(ctx, clusterID, status)
		require.NoError(t, err)
	})
}
