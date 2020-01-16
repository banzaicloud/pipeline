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

package k8sutil_test

import (
	"testing"

	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"

	"github.com/banzaicloud/pipeline/pkg/k8sutil"
)

func TestCreateConfigFromTemplateIsValid(t *testing.T) {
	configBase := &k8sutil.ConfigBase{
		ClusterName:              "cluster",
		APIEndpoint:              "https://asdf.as:123",
		CertificateAuthorityData: []byte("aasdasdasd"),
	}
	config := configBase.CreateConfigFromTemplate(
		k8sutil.CreateAuthInfoFunc(func(clusterName string) *api.AuthInfo {
			return &api.AuthInfo{}
		}))
	err := clientcmd.Validate(*config)
	if err != nil {
		t.Fatalf("%+v", err)
	}
}
