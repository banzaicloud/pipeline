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

package integratedservices

import (
	"context"

	"emperror.dev/errors"

	"github.com/banzaicloud/pipeline/internal/common"
)

// ISServiceV2 integrated service service implementation - V2
type ISServiceV2 struct {
	logger common.Logger
}

func NewISServiceV2(logger common.Logger) *ISServiceV2 {
	return &ISServiceV2{
		logger: logger,
	}
}

func (i ISServiceV2) List(ctx context.Context, clusterID uint) ([]IntegratedService, error) {
	// TODO implement me!
	i.logger.Info("operation not yet implemented", map[string]interface{}{"op": "List", "clusterId": clusterID})
	return nil, errors.NewWithDetails("Operation not, yet implemented!", "clusterID", clusterID)
}

func (i ISServiceV2) Details(ctx context.Context, clusterID uint, serviceName string) (IntegratedService, error) {
	// TODO implement me!
	i.logger.Info("operation not yet implemented", map[string]interface{}{"op": "Details", "clusterId": clusterID})
	return IntegratedService{}, errors.NewWithDetails("Operation not, yet implemented!", "clusterID", clusterID)
}

func (i ISServiceV2) Activate(ctx context.Context, clusterID uint, serviceName string, spec map[string]interface{}) error {
	// TODO implement me!
	i.logger.Info("operation not yet implemented", map[string]interface{}{"op": "Activate", "clusterId": clusterID})
	return errors.NewWithDetails("Operation not, yet implemented!", "clusterID", clusterID)
}

func (i ISServiceV2) Deactivate(ctx context.Context, clusterID uint, serviceName string) error {
	// TODO implement me!
	i.logger.Info("operation not yet implemented", map[string]interface{}{"op": "Deactivate", "clusterId": clusterID})
	return errors.NewWithDetails("Operation not, yet implemented!", "clusterID", clusterID)
}

func (i ISServiceV2) Update(ctx context.Context, clusterID uint, serviceName string, spec map[string]interface{}) error {
	// TODO implement me!
	i.logger.Info("operation not yet implemented", map[string]interface{}{"op": "Update", "clusterId": clusterID})
	return errors.NewWithDetails("Operation not, yet implemented!", "clusterID", clusterID)
}
