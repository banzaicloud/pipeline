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

package clusterfeatureadapter

import (
	"context"
	"fmt"
	"time"

	"emperror.dev/errors"
	"go.uber.org/cadence/client"

	"github.com/banzaicloud/pipeline/internal/clusterfeature"
	"github.com/banzaicloud/pipeline/internal/clusterfeature/clusterfeatureadapter/workflow"
	"github.com/banzaicloud/pipeline/internal/common"
)

// MakeCadenceFeatureOperationDispatcher returns an Uber Cadence based implementation of FeatureOperationDispatcher
func MakeCadenceFeatureOperationDispatcher(
	cadenceClient client.Client,
	logger common.Logger,
) CadenceFeatureOperationDispatcher {
	return CadenceFeatureOperationDispatcher{
		cadenceClient: cadenceClient,
		logger:        logger,
	}
}

// CadenceFeatureOperationDispatcher implements a feature operation dispatcher using Uber Cadence
type CadenceFeatureOperationDispatcher struct {
	cadenceClient client.Client
	logger        common.Logger
}

// DispatchApply dispatches an Apply request to a feature manager asynchronously
func (d CadenceFeatureOperationDispatcher) DispatchApply(ctx context.Context, clusterID uint, featureName string, spec clusterfeature.FeatureSpec) error {
	return d.dispatchOperation(ctx, workflow.OperationApply, clusterID, featureName, spec)
}

// DispatchDeactivate dispatches a Deactivate request to a feature manager asynchronously
func (d CadenceFeatureOperationDispatcher) DispatchDeactivate(ctx context.Context, clusterID uint, featureName string) error {
	return d.dispatchOperation(ctx, workflow.OperationDeactivate, clusterID, featureName, nil)
}

func (d CadenceFeatureOperationDispatcher) dispatchOperation(ctx context.Context, op string, clusterID uint, featureName string, spec clusterfeature.FeatureSpec) error {
	const workflowName = workflow.ClusterFeatureJobWorkflowName
	workflowID := getWorkflowID(workflowName, clusterID, featureName)
	const signalName = workflow.ClusterFeatureJobSignalName
	signalArg := workflow.ClusterFeatureJobSignalInput{
		Operation:     op,
		FeatureSpec:   spec,
		RetryInterval: 1 * time.Minute,
	}
	options := client.StartWorkflowOptions{
		TaskList:                     "pipeline",
		ExecutionStartToCloseTimeout: 40 * time.Minute,
		WorkflowIDReusePolicy:        client.WorkflowIDReusePolicyAllowDuplicate,
	}
	workflowInput := workflow.ClusterFeatureJobWorkflowInput{
		ClusterID:   clusterID,
		FeatureName: featureName,
	}
	_, err := d.cadenceClient.SignalWithStartWorkflow(ctx, workflowID, signalName, signalArg, options, workflowName, workflowInput)
	if err != nil {
		return errors.WrapIfWithDetails(err, "signal with start workflow failed", "workflowId", workflowID)
	}
	return nil
}

func getWorkflowID(workflowName string, clusterID uint, featureName string) string {
	return fmt.Sprintf("%s-%d-%s", workflowName, clusterID, featureName)
}
