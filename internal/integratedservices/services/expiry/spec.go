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

package expiry

import (
	"time"

	"github.com/banzaicloud/pipeline/internal/integratedservices"
)

type binderFunc = func(inputSpec integratedservices.IntegratedServiceSpec, boundSpec interface{}) error

type ServiceSpec struct {
	Date string `json:"date" mapstructure:"date"`
}

// https://www.ietf.org/rfc/rfc3339.txt
func (s ServiceSpec) Validate() error {
	t, err := time.Parse(time.RFC3339, s.Date)
	if err != nil {
		return integratedservices.InvalidIntegratedServiceSpecError{
			IntegratedServiceName: ServiceName,
			Problem:               "date must be in RFC3339 format",
		}
	}

	if !t.After(time.Now()) {
		return integratedservices.InvalidIntegratedServiceSpecError{
			IntegratedServiceName: ServiceName,
			Problem:               "the provided date must be in the future",
		}
	}

	return nil
}
