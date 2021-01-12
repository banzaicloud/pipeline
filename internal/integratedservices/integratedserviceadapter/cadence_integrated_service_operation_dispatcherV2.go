// Copyright © 2021 Banzai Cloud
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
	"time"

	"emperror.dev/errors"
	"go.uber.org/cadence/.gen/go/shared"
	"go.uber.org/cadence/client"

	"github.com/banzaicloud/pipeline/internal/common"
	"github.com/banzaicloud/pipeline/internal/integratedservices"
	"github.com/banzaicloud/pipeline/internal/integratedservices/integratedserviceadapter/workflow"
)

// OperationDispatcher component dispatching operations related to the new version of the integrated services
type OperationDispatcher struct {
	cadenceClient client.Client
	logger        common.Logger
}

// NewCadenceOperationDispatcher returns the new version Uber Cadence based implementation of IntegratedServiceOperationDispatcher
func NewCadenceOperationDispatcher(
	cadenceClient client.Client,
	logger common.Logger,
) OperationDispatcher {
	return OperationDispatcher{
		cadenceClient: cadenceClient,
		logger:        logger,
	}
}

// DispatchApply dispatches an Apply request to an integrated service manager asynchronously
func (d OperationDispatcher) DispatchApply(ctx context.Context, clusterID uint, integratedServiceName string, spec integratedservices.IntegratedServiceSpec) error {
	return d.dispatchOperation(ctx, workflow.OperationApply, clusterID, integratedServiceName, spec)
}

// DispatchDeactivate dispatches a Deactivate request to an integrated service manager asynchronously
func (d OperationDispatcher) DispatchDeactivate(ctx context.Context, clusterID uint, integratedServiceName string, spec integratedservices.IntegratedServiceSpec) error {
	return d.dispatchOperation(ctx, workflow.OperationDeactivate, clusterID, integratedServiceName, spec)
}

func (d OperationDispatcher) dispatchOperation(ctx context.Context, op string, clusterID uint, integratedServiceName string, spec integratedservices.IntegratedServiceSpec) error {
	const workflowName = workflow.IntegratedServiceJobWorkflowV2Name
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

func (d OperationDispatcher) IsBeingDispatched(ctx context.Context, clusterID uint, integratedServiceName string) (bool, error) {
	const workflowName = workflow.IntegratedServiceJobWorkflowV2Name
	workflowID := getWorkflowID(workflowName, clusterID, integratedServiceName)

	response, err := d.cadenceClient.DescribeWorkflowExecution(ctx, workflowID, "")
	if err != nil {
		notExists := &shared.EntityNotExistsError{}
		if errors.As(err, &notExists) {
			return false, nil
		}
		return false, errors.WrapIfWithDetails(err, "unable to describe workflow for service",
			"workflowId", workflowID, "service", integratedServiceName)
	}
	return response.WorkflowExecutionInfo.CloseTime == nil, nil
}
