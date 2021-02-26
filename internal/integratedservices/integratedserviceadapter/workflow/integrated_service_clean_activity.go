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

package workflow

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/banzaicloud/pipeline/internal/integratedservices"
)

const IntegratedServiceCleanActivityName = "integrated-service-clean"

type IntegratedServiceCleanActivityInput struct {
	ClusterID uint
	Force     bool
}

type IntegratedServiceCleanActivity struct {
	integratedCleaner integratedservices.IntegratedServiceCleaner
	logger            logrus.FieldLogger
}

func MakeIntegratedServiceCleanActivity(integratedCleaner integratedservices.IntegratedServiceCleaner, logger logrus.FieldLogger) IntegratedServiceCleanActivity {
	return IntegratedServiceCleanActivity{
		integratedCleaner: integratedCleaner,
		logger:            logger,
	}
}

func (a IntegratedServiceCleanActivity) Execute(ctx context.Context, input IntegratedServiceCleanActivityInput) error {
	err := a.integratedCleaner.DisableServiceInstance(ctx, input.ClusterID)
	if err != nil && input.Force {
		logger := a.logger.WithFields(logrus.Fields{
			"clusterID": input.ClusterID,
		})
		logger.Error(err)

		return nil
	}

	return err
}
