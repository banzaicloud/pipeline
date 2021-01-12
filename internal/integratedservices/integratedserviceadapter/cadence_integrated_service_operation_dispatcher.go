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

package integratedserviceadapter

import (
	"context"
	"fmt"
	"time"

	"emperror.dev/errors"
	"go.uber.org/cadence/client"

	"github.com/banzaicloud/pipeline/internal/common"
	"github.com/banzaicloud/pipeline/internal/integratedservices"
	"github.com/banzaicloud/pipeline/internal/integratedservices/integratedserviceadapter/workflow"
)

// MakeCadenceIntegratedServiceOperationDispatcher returns an Uber Cadence based implementation of IntegratedServiceOperationDispatcher
func MakeCadenceIntegratedServiceOperationDispatcher(
	cadenceClient client.Client,
	logger common.Logger,
) CadenceIntegratedServiceOperationDispatcher {
	return CadenceIntegratedServiceOperationDispatcher{
		cadenceClient: cadenceClient,
		logger:        logger,
	}
}

// CadenceIntegratedServiceOperationDispatcher implements an integrated service operation dispatcher using Uber Cadence
type CadenceIntegratedServiceOperationDispatcher struct {
	cadenceClient client.Client
	logger        common.Logger
}

// DispatchApply dispatches an Apply request to an integrated service manager asynchronously
func (d CadenceIntegratedServiceOperationDispatcher) DispatchApply(ctx context.Context, clusterID uint, integratedServiceName string, spec integratedservices.IntegratedServiceSpec) error {
	return d.dispatchOperation(ctx, workflow.OperationApply, clusterID, integratedServiceName, spec)
}

// DispatchDeactivate dispatches a Deactivate request to an integrated service manager asynchronously
func (d CadenceIntegratedServiceOperationDispatcher) DispatchDeactivate(ctx context.Context, clusterID uint, integratedServiceName string, spec integratedservices.IntegratedServiceSpec) error {
	return d.dispatchOperation(ctx, workflow.OperationDeactivate, clusterID, integratedServiceName, spec)
}

func (d CadenceIntegratedServiceOperationDispatcher) dispatchOperation(ctx context.Context, op string, clusterID uint, integratedServiceName string, spec integratedservices.IntegratedServiceSpec) error {
	const workflowName = workflow.IntegratedServiceJobWorkflowName
	workflowID := getWorkflowID(workflowName, clusterID, integratedServiceName)
	const signalName = workflow.IntegratedServiceJobSignalName
	signalArg := workflow.IntegratedServiceJobSignalInput{
		Operation:              op,
		IntegratedServiceSpecs: spec,
		RetryInterval:          1 * time.Minute,
	}
	options := client.StartWorkflowOptions{
		TaskList:                     "pipeline",
		ExecutionStartToCloseTimeout: 3 * time.Hour,
		WorkflowIDReusePolicy:        client.WorkflowIDReusePolicyAllowDuplicate,
	}
	workflowInput := workflow.IntegratedServiceJobWorkflowInput{
		ClusterID:             clusterID,
		IntegratedServiceName: integratedServiceName,
	}
	_, err := d.cadenceClient.SignalWithStartWorkflow(ctx, workflowID, signalName, signalArg, options, workflowName, workflowInput)
	if err != nil {
		return errors.WrapIfWithDetails(err, "signal with start workflow failed", "workflowId", workflowID)
	}
	return nil
}

func (d CadenceIntegratedServiceOperationDispatcher) IsBeingDispatched(ctx context.Context, clusterID uint, integratedServiceName string) (bool, error) {
	return false, errors.New("method not applicable for the v1 implementation")
}

func getWorkflowID(workflowName string, clusterID uint, integratedServiceName string) string {
	return fmt.Sprintf("%s-%d-%s", workflowName, clusterID, integratedServiceName)
}
