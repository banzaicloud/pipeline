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
	"github.com/banzaicloud/pipeline/dns"
	"github.com/ghodss/yaml"
	"github.com/goph/emperror"
	"github.com/goph/logur"
	v1 "k8s.io/api/core/v1"
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

	// todo check what values exactly should / must be passed in the spec - implement validate!
	// todo some entries come from secrets - access the secret store from here!
	rbacEnabled, _ := spec["rbac-enabled"]
	imageVersion, _ := spec["external-dns-image-version"]
	awsSecretKey, _ := spec["aws-secret-access-key"]
	awsAccessKey, _ := spec["aws-access-key-id"]
	region, _ := spec["region"]
	txtOwner, _ := spec["txt-owner"]
	domainFilters, _ := spec["domain-filters"]

	externalDnsValues := dns.ExternalDnsChartValues{
		Rbac: dns.ExternalDnsRbacSettings{
			Create: rbacEnabled.(bool),
		},
		Sources: []string{"service", "ingress"},
		Image: dns.ExternalDnsImageSettings{
			Tag: imageVersion.(string),
		},
		Aws: dns.ExternalDnsAwsSettings{
			Credentials: dns.ExternalDnsAwsCredentials{
				SecretKey: awsSecretKey.(string),
				AccessKey: awsAccessKey.(string),
			},
			Region: region.(string),
		},
		DomainFilters: domainFilters.([]string),
		Policy:        "sync",
		TxtOwnerId:    txtOwner.(string),
		Affinity:      v1.Affinity{},     // todo process this based on the cluster? (check it - hooks)
		Tolerations:   []v1.Toleration{}, // todo process this based on the cluster? (check it - hooks)
	}

	values, err := yaml.Marshal(externalDnsValues)
	if err != nil {
		return nil, emperror.Wrap(err, "Json Convert Failed")
	}

	return values, nil
}

func NewExternalDnsFeatureProcessor(logger logur.Logger) FeatureSpecProcessor {

	return &externalDnsFeatureSpecProcessor{
		logger: logur.WithFields(logger, map[string]interface{}{"feature-processor": "comp"}),
	}
}
