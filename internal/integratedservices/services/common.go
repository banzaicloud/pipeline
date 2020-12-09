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

package services

import (
	"emperror.dev/errors"
	"github.com/mitchellh/mapstructure"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/banzaicloud/pipeline/internal/integratedservices"
)

// BindIntegratedServiceSpec binds an incoming integrated service specific raw spec (json) into the appropriate struct
func BindIntegratedServiceSpec(inputSpec integratedservices.IntegratedServiceSpec, boundSpec interface{}) error {
	if err := mapstructure.Decode(inputSpec, &boundSpec); err != nil {
		return errors.WrapIf(err, "failed to decode integrated service specification")
	}
	return nil
}

func SetManagedByPipeline(meta *v1.ObjectMeta) {
	if meta.Annotations == nil {
		meta.Annotations = make(map[string]string)
	}
	meta.Annotations["app.kubernetes.io/managed-by"] = "banzaicloud.io/pipeline"
}

func IsManagedByPipeline(meta v1.ObjectMeta) bool {
	return meta.Annotations["app.kubernetes.io/managed-by"] == "banzaicloud.io/pipeline"
}
