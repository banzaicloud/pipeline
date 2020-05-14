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

package kubernetes

import (
	loggingV1beta1 "github.com/banzaicloud/logging-operator/pkg/sdk/api/v1beta1"
	elasticsearchV1 "github.com/elastic/cloud-on-k8s/pkg/apis/elasticsearch/v1"
	kibanaV1 "github.com/elastic/cloud-on-k8s/pkg/apis/kibana/v1"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/client-go/kubernetes/scheme"
)

func init() {
	if err := loggingV1beta1.AddToScheme(scheme.Scheme); err != nil {
		panic("failed to add logging scheme: " + err.Error())
	}

	if err := elasticsearchV1.AddToScheme(scheme.Scheme); err != nil {
		panic("failed to add elastic scheme: " + err.Error())
	}

	if err := kibanaV1.AddToScheme(scheme.Scheme); err != nil {
		panic("failed to add kibana scheme: " + err.Error())
	}

	if err := apiextensionsv1beta1.AddToScheme(scheme.Scheme); err != nil {
		panic("failed to add apiextensionsv1beta1 scheme: " + err.Error())
	}
}
