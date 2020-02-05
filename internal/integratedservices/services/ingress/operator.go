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

package ingress

import (
	"context"

	"emperror.dev/errors"

	"github.com/banzaicloud/pipeline/internal/integratedservices"
	"github.com/banzaicloud/pipeline/internal/integratedservices/services"
)

type Operator struct {
	clusterService integratedservices.ClusterService
	traefikManager traefikManager
}

func NewOperator(
	clusters OperatorClusterStore,
	clusterService integratedservices.ClusterService,
	config Config,
	helmService services.HelmService,
	orgDomainService OrgDomainService,
) Operator {
	return Operator{
		clusterService: clusterService,
		traefikManager: traefikManager{
			clusters:         clusters,
			config:           config,
			helmService:      helmService,
			orgDomainService: orgDomainService,
		},
	}
}

// Name returns the integrated service's name.
func (Operator) Name() string {
	return ServiceName
}

// Apply applies a desired state for an integrated service on the given cluster.
func (op Operator) Apply(ctx context.Context, clusterID uint, spec integratedservices.IntegratedServiceSpec) error {
	if err := op.clusterService.CheckClusterReady(ctx, clusterID); err != nil {
		return err
	}

	var boundSpec Spec
	if err := services.BindIntegratedServiceSpec(spec, &boundSpec); err != nil {
		return errors.WrapIf(err, "failed to bind spec")
	}

	switch controllerType := boundSpec.Controller.Type; controllerType {
	case ControllerTraefik:
		if err := op.traefikManager.Deploy(ctx, clusterID, boundSpec); err != nil {
			return errors.WrapIf(err, "failed to deploy traefik")
		}
	default:
		return errors.Errorf("unhandled controller type %q", controllerType)
	}

	return nil
}

// Deactivate deactivates an integrated service on the given cluster.
func (op Operator) Deactivate(ctx context.Context, clusterID uint, spec integratedservices.IntegratedServiceSpec) error {
	var boundSpec Spec
	if err := services.BindIntegratedServiceSpec(spec, &boundSpec); err != nil {
		return errors.WrapIf(err, "failed to bind spec")
	}

	switch controllerType := boundSpec.Controller.Type; controllerType {
	case ControllerTraefik:
		if err := op.traefikManager.Remove(ctx, clusterID); err != nil {
			return errors.WrapIf(err, "failed to remove traefik")
		}
	default:
		return errors.Errorf("unhandled controller type %q", controllerType)
	}

	return nil
}
