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

func NewAsyncExpirer(cadenceClient client.Client, logger common.Logger) asyncExpiryService {
	return asyncExpiryService{
		cadenceClient: cadenceClient,
		logger:        logger,
	}
}

func (a asyncExpiryService) Expire(ctx context.Context, clusterID uint, expiryDate string) error {

	startToCloseTimeout, err := startToCloseDuration(expiryDate)
	if err != nil {
		return err
	}

	options := client.StartWorkflowOptions{
		ID:                           getWorkflowID(clusterID),
		TaskList:                     "pipeline",
		ExecutionStartToCloseTimeout: startToCloseTimeout,
		WorkflowIDReusePolicy:        client.WorkflowIDReusePolicyAllowDuplicate,
	}

	workflowInput := workflow.ExpiryJobWorkflowInput{
		ClusterID:  clusterID,
		ExpiryDate: expiryDate,
	}

	// cancel the workflow if already exists to support the update flow
	// the error is ignored on purpose here
	if err := a.cadenceClient.CancelWorkflow(ctx, getWorkflowID(clusterID), ""); err != nil {
		a.logger.Debug("failed to cancel the workflow ( on apply )", map[string]interface{}{"workflowID": getWorkflowID(clusterID)})
	}

	if _, err := a.cadenceClient.StartWorkflow(ctx, options, workflow.ExpiryJobWorkflowName, workflowInput); err != nil {
		return errors.WrapIfWithDetails(err, "failed to start the expiry workflow", "workflowId", getWorkflowID(clusterID))
	}

	a.logger.Info("expiry workflow successfully started", map[string]interface{}{"workflowID": getWorkflowID(clusterID)})
	return nil
}

func (a asyncExpiryService) CancelExpiry(ctx context.Context, clusterID uint) error {

	if err := a.cadenceClient.CancelWorkflow(ctx, getWorkflowID(clusterID), ""); err != nil {
		var enfe *shared.EntityNotExistsError
		if errors.As(err, &enfe) {
			return nil
		}

		return errors.WrapIfWithDetails(err, "failed to cancel the expiry workflow", "workflowId", getWorkflowID(clusterID))
	}

	a.logger.Info("expiry workflow successfully cancelled", map[string]interface{}{"workflowID": getWorkflowID(clusterID)})
	return nil
}

// computes the unique workflow id for the cluster (clusterID is unique in the system)
func getWorkflowID(clusterID uint) string {

	return fmt.Sprintf("%s-%d-%s", workflow.ExpiryJobWorkflowName, clusterID, expiry.InternalServiceName)
}

func startToCloseDuration(expiryDate string) (time.Duration, error) {
	expiryTime, err := time.ParseInLocation(time.RFC3339, expiryDate, time.Now().Location())
	if err != nil {
		return 0, errors.WrapIf(err, "failed to parse the expiry date")
	}

	// add extra 24 hours to the scheduled expiry
	startToCloseTimeout := expiryTime.Add(startToCloseDurationOffset).Sub(time.Now())
	return startToCloseTimeout, nil
}
