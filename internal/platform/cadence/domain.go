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

package cadence

import (
	"context"
	"time"

	"emperror.dev/errors"
	"go.uber.org/cadence/.gen/go/cadence/workflowserviceclient"
	"go.uber.org/cadence/.gen/go/shared"
	"go.uber.org/zap"
)

func createNonExistentDomain(serviceClient workflowserviceclient.Interface, config Config, logger *zap.Logger) error {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	err := serviceClient.RegisterDomain(ctx, &shared.RegisterDomainRequest{
		Name:                                   &config.Domain,
		WorkflowExecutionRetentionPeriodInDays: &config.WorkflowExecutionRetentionPeriodInDays,
	})
	if _, ok := err.(*shared.DomainAlreadyExistsError); ok {
		logger.Info("Cadence domain already exists")
	} else if err != nil {
		return errors.WrapIf(err, "failed to register Cadence domain")
	} else {
		logger.Info("Created Cadence domain")
	}
	return nil
}
