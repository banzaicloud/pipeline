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

package backup

import (
	"context"

	"github.com/banzaicloud/integrated-service-sdk/api/v1alpha1/backup"

	"github.com/banzaicloud/pipeline/internal/integratedservices"
)

// Manager component implementing input / output manipulation as required for the backup integrated service
type Manager struct {
	version string
}

func NewManager(version string) Manager {
	return Manager{
		version: version,
	}
}

func (m Manager) GetOutput(ctx context.Context, clusterID uint, spec integratedservices.IntegratedServiceSpec) (integratedservices.IntegratedServiceOutput, error) {
	return map[string]interface{}{
		"backup": map[string]interface{}{
			"version": m.version,
		},
	}, nil
}

func (m Manager) ValidateSpec(ctx context.Context, spec integratedservices.IntegratedServiceSpec) error {
	backupSpec, err := backup.BindIntegratedServiceSpec(spec)
	if err != nil {
		return integratedservices.InvalidIntegratedServiceSpecError{
			IntegratedServiceName: IntegratedServiceName,
			Problem:               err.Error(),
		}
	}

	if err := backupSpec.Validate(); err != nil {
		return integratedservices.InvalidIntegratedServiceSpecError{
			IntegratedServiceName: IntegratedServiceName,
			Problem:               err.Error(),
		}
	}

	return nil
}

func (m Manager) PrepareSpec(ctx context.Context, clusterID uint, spec integratedservices.IntegratedServiceSpec) (integratedservices.IntegratedServiceSpec, error) {
	return spec, nil
}

func (m Manager) Name() string {
	return IntegratedServiceName
}
