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

package adapter

import (
	"context"
	"fmt"
	"time"

	"emperror.dev/errors"
	"go.uber.org/cadence/client"

	"github.com/banzaicloud/pipeline/internal/common"
	"github.com/banzaicloud/pipeline/internal/integratedservices/services/expiry"
	"github.com/banzaicloud/pipeline/internal/integratedservices/services/expiry/adapter/workflow"
)

// asyncExpirer Expirer implementation that uses cadence setup for executing the expiration
type asyncExpirer struct {
	cadenceClient client.Client
	logger        common.Logger
}

func NewAsyncExpirer(cadenceClient client.Client, logger common.Logger) asyncExpirer {
	return asyncExpirer{
		cadenceClient: cadenceClient,
		logger:        logger,
	}
}

func (a asyncExpirer) Expire(ctx context.Context, clusterID uint, expiryDate string) error {
	workflowID := fmt.Sprintf("%s-%d-%s", workflow.ExpiryJobWorkflowName, clusterID, expiry.ExpiryInternalServiceName)

	signalArg := workflow.ExpiryJobSignalInput{
		ClusterID:  clusterID,
		ExpiryDate: expiryDate,
	}

	options := client.StartWorkflowOptions{
		TaskList:                     "pipeline",
		ExecutionStartToCloseTimeout: 3 * time.Hour,
		WorkflowIDReusePolicy:        client.WorkflowIDReusePolicyAllowDuplicate,
	}

	workflowInput := workflow.ExpiryJobWorkflowInput{
		ClusterID:  clusterID,
		ExpiryDate: expiryDate,
	}

	if _, err := a.cadenceClient.SignalWithStartWorkflow(ctx, workflowID, workflow.ExpiryJobSignalName, signalArg, options,
		workflow.ExpiryJobWorkflowName, workflowInput); err != nil {
		return errors.WrapIfWithDetails(err, "signal with start workflow failed", "workflowId", workflowID)
	}

	return nil
}
