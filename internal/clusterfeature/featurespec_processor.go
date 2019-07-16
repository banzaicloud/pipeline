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

package clusterfeature

import (
	"encoding/json"

	"github.com/banzaicloud/pipeline/dns"
	"github.com/goph/emperror"
	"github.com/goph/logur"
	"github.com/mitchellh/mapstructure"
)

// FeatureSpecProcessor component interface for processing FeatureSpecs
type FeatureSpecProcessor interface {
	//Process processes (transforms) the passed in FeatureSpec to produce the feature specific representation
	Process(spec FeatureSpec) (interface{}, error)
}

type externalDnsFeatureSpecProcessor struct {
	logger logur.Logger
}

// Process method for assembling the "values" for the helm deployment
func (p *externalDnsFeatureSpecProcessor) Process(spec FeatureSpec) (interface{}, error) {

	rawValues := dns.ExternalDnsChartValues{}
	if err := mapstructure.Decode(spec, &rawValues); err != nil {

		return nil, emperror.Wrap(err, "could not process feature spec")
	}

	values, err := json.Marshal(rawValues)
	if err != nil {

		return nil, emperror.Wrap(err, "failed to decode values")
	}

	return values, nil
}

func NewExternalDnsFeatureProcessor(logger logur.Logger) FeatureSpecProcessor {

	return &externalDnsFeatureSpecProcessor{
		logger: logur.WithFields(logger, map[string]interface{}{"feature-processor": "comp"}),
	}
}
