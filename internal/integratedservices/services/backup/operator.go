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

	"github.com/banzaicloud/pipeline/internal/integratedservices"
)

// Operator component implementing the operations related to the backup integrated service
type Operator struct {
	// todo add collaborators as required
}

// NewOperator constructs a new Operator instance
func NewOperator() Operator {
	// todo pass arguments as required to inject collaborators
	return Operator{}
}

func (o Operator) Apply(ctx context.Context, clusterID uint, spec integratedservices.IntegratedServiceSpec) error {
	panic("implement me")
}

func (o Operator) Deactivate(ctx context.Context, clusterID uint, spec integratedservices.IntegratedServiceSpec) error {
	panic("implement me")
}

func (o Operator) Name() string {
	return IntegratedServiceName
}
