// Copyright © 2018 Banzai Cloud
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

package v1alpha1

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"

	"github.com/banzaicloud/anchore-image-validator/pkg/apis/security/v1alpha1"
)

// SecurityV1Alpha1Interface interface for audit
type SecurityV1Alpha1Interface interface {
	Audits(namespace string) AuditInterface
	Whitelists(namespace string) WhiteListInterface
}

// SecurityV1Alpha1Client client for crd
type SecurityV1Alpha1Client struct {
	restClient rest.Interface
}

// SecurityConfig for admission hook configuration
func SecurityConfig(c *rest.Config) (*SecurityV1Alpha1Client, error) {
	config := *c
	config.ContentConfig.GroupVersion = &schema.GroupVersion{Group: v1alpha1.GroupName, Version: v1alpha1.GroupVersion}
	config.APIPath = "/apis"
	config.NegotiatedSerializer = serializer.WithoutConversionCodecFactory{CodecFactory: scheme.Codecs}
	config.UserAgent = rest.DefaultKubernetesUserAgent()

	client, err := rest.RESTClientFor(&config)
	if err != nil {
		return nil, err
	}

	return &SecurityV1Alpha1Client{restClient: client}, nil
}

// Audits returns Audits for client
func (c *SecurityV1Alpha1Client) Audits() AuditInterface {
	return &auditClient{
		restClient: c.restClient,
	}
}

// Whitelists return WhiteLists for client
func (c *SecurityV1Alpha1Client) Whitelists() WhiteListInterface {
	return &whitelistClient{
		restClient: c.restClient,
	}
}
