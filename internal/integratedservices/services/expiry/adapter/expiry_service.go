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
	"go.uber.org/cadence/.gen/go/shared"
	"go.uber.org/cadence/client"

	"github.com/banzaicloud/pipeline/internal/common"
	"github.com/banzaicloud/pipeline/internal/integratedservices/services/expiry"
	"github.com/banzaicloud/pipeline/internal/integratedservices/services/expiry/adapter/workflow"
)

// adjust this value if appropriate
const startToCloseDurationOffset = 24 * time.Hour

// asyncExpiryService Expirer implementation that uses cadence setup for executing the expiration
type asyncExpiryService struct {
	cadenceClient client.Client
	logger        common.Logger
}

func NewAsyncExpiryService(cadenceClient client.Client, logger common.Logger) expiry.ExpiryService {
	return asyncExpiryService{
		cadenceClient: cadenceClient,
		logger:        logger,
	}
}

func (a asyncExpiryService) Expire(ctx context.Context, clusterID uint, expiryDate string) error {
	startToCloseTimeout, err := expiry.CalculateDuration(time.Now(), expiryDate)
	if err != nil {
		return err
	}

	options := client.StartWorkflowOptions{
		ID:                           getWorkflowID(clusterID),
		TaskList:                     "pipeline",
		ExecutionStartToCloseTimeout: startToCloseTimeout + startToCloseDurationOffset,
		WorkflowIDReusePolicy:        client.WorkflowIDReusePolicyAllowDuplicate,
	}

	workflowInput := workflow.ExpiryJobWorkflowInput{
		ClusterID:  clusterID,
		ExpiryDate: expiryDate,
	}

	// cancel the workflow if already set up (support the update flow)
	if err := a.CancelExpiry(ctx, clusterID); err != nil {
		return errors.WrapIfWithDetails(err, "failed to setup expiry workflow", "clusterID", clusterID)
	}

	if _, err := a.cadenceClient.StartWorkflow(ctx, options, workflow.ExpiryJobWorkflowName, workflowInput); err != nil {
		return errors.WrapIfWithDetails(err, "failed to start the expiry workflow", "workflowId", options.ID)
	}

	a.logger.Info("expiry workflow successfully started", map[string]interface{}{"workflowID": getWorkflowID(clusterID)})
	return nil
}

func (a asyncExpiryService) CancelExpiry(ctx context.Context, clusterID uint) error {
	if err := a.cadenceClient.TerminateWorkflow(ctx, getWorkflowID(clusterID), "", "expiration service cancelled", nil); err != nil {
		if !IsEntityNotExistsError(err) {
			return errors.WrapIfWithDetails(err, "failed to cancel the expiry workflow", "clusterID", clusterID)
		}
	}

	a.logger.Info("expiry workflow successfully cancelled", map[string]interface{}{"workflowID": getWorkflowID(clusterID)})
	return nil
}

// computes the unique workflow id for the cluster (clusterID is unique in the system)
func getWorkflowID(clusterID uint) string {
	return fmt.Sprintf("%s-%d", workflow.ExpiryJobWorkflowName, clusterID)
}

func IsEntityNotExistsError(err error) bool {
	var ene *shared.EntityNotExistsError

	return errors.As(err, &ene)
}
