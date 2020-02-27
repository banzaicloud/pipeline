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
	"emperror.dev/errors"

	"github.com/banzaicloud/pipeline/pkg/values"
)

type Config struct {
	Namespace   string
	ReleaseName string
	Controllers []string
	Charts      ChartsConfig
}

func (c Config) Validate() error {
	var errs error

	if c.Namespace == "" {
		errs = errors.Append(errs, errors.New("ingress namespace is required"))
	}

	if len(c.Controllers) == 0 {
		errs = errors.Append(errs, errors.New("at least 1 controller type must be configured"))
	}

	for _, ctrl := range c.Controllers {
		switch ctrl {
		case ControllerTraefik:
			// ok
		default:
			errs = errors.Append(errs, unsupportedControllerError{
				Controller: ctrl,
			})
		}
	}

	return errs
}

type ChartsConfig struct {
	Traefik TraefikChartConfig
}

type TraefikChartConfig struct {
	Chart   string
	Version string
	Values  values.Config
}
