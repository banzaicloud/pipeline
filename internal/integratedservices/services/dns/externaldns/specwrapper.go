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

package externaldns

import (
	"context"
	"encoding/json"

	"emperror.dev/errors"

	"github.com/banzaicloud/pipeline/internal/integratedservices"
)

// valuesSpecWrapper (temporarily) encodes and decodes the original specification into the CR configuration
// This serves the backwards compatibility
type valuesSpecWrapper struct {
	*ChartValues
	ApiSpec *integratedservices.IntegratedServiceSpec `json:"apiSpec,omitempty"`
}

func NewSpecWrapper() integratedservices.SpecWrapper {
	return valuesSpecWrapper{}
}

func (c valuesSpecWrapper) Wrap(ctx context.Context, values interface{}, spec integratedservices.IntegratedServiceSpec) ([]byte, error) {
	dnsVals := values.(ChartValues)
	wrapped, err := json.Marshal(valuesSpecWrapper{&dnsVals, &spec})
	if err != nil {
		return nil, errors.WrapIf(err, "failed to encode values")
	}

	return wrapped, nil
}

func (c valuesSpecWrapper) Unwrap(ctx context.Context, instanceConfig []byte) (integratedservices.IntegratedServiceSpec, error) {
	var wrapped valuesSpecWrapper
	if err := json.Unmarshal(instanceConfig, &wrapped); err != nil {
		return nil, errors.WrapIf(err, "failed to decode values")
	}

	return *wrapped.ApiSpec, nil
}
