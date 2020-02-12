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

package dns

import (
	"emperror.dev/errors"

	"github.com/banzaicloud/pipeline/pkg/validation"
)

// Config contains configuration for the dns integrated service.
type Config struct {
	Namespace      string
	BaseDomain     string
	ProviderSecret string
	Charts         ChartsConfig
}

func (c Config) Validate() error {
	if c.Namespace == "" {
		return errors.New("dns namespace is required")
	}

	if c.BaseDomain != "" {
		if err := validation.ValidateSubdomain(c.BaseDomain); err != nil {
			return errors.WithMessage(err, "invalid dns base domain")
		}
	}

	// TODO: validate chart config: image and tag are not empty!

	return nil
}

type ChartsConfig struct {
	ExternalDNS ExternalDNSChartConfig
}

type ChartConfigBase struct {
	Chart   string
	Version string
}

type ExternalDNSChartConfig struct {
	ChartConfigBase `mapstructure:",squash"`
	Values          ExternalDNSChartValuesConfig
}

type ExternalDNSChartValuesConfig struct {
	Image ExternalDNSChartValuesImageConfig
}

type ExternalDNSChartValuesImageConfig struct {
	Repository string
	Tag        string
}
