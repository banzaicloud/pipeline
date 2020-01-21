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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"

	"github.com/banzaicloud/anchore-image-validator/pkg/apis/security/v1alpha1"
)

// AuditInterface for audit
type AuditInterface interface {
	List(metav1.ListOptions) (*v1alpha1.AuditList, error)
	Get(string, metav1.GetOptions) (*v1alpha1.Audit, error)
	Create(*v1alpha1.Audit) (*v1alpha1.Audit, error)
	Update(string, []byte) (*v1alpha1.Audit, error)
}

type auditClient struct {
	restClient rest.Interface
}

func (c *auditClient) List(opts metav1.ListOptions) (*v1alpha1.AuditList, error) {
	result := v1alpha1.AuditList{}
	err := c.restClient.
		Get().
		Resource("audits").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(&result)

	return &result, err
}

func (c *auditClient) Get(name string, opts metav1.GetOptions) (*v1alpha1.Audit, error) {
	result := v1alpha1.Audit{}
	err := c.restClient.
		Get().
		Resource("audits").
		Name(name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(&result)

	return &result, err
}

func (c *auditClient) Create(audit *v1alpha1.Audit) (*v1alpha1.Audit, error) {
	result := v1alpha1.Audit{}
	err := c.restClient.
		Post().
		Resource("audits").
		Body(audit).
		Do().
		Into(&result)

	return &result, err
}

func (c *auditClient) Update(name string, auditPatch []byte) (*v1alpha1.Audit, error) {
	result := v1alpha1.Audit{}
	err := c.restClient.
		Patch(types.MergePatchType).
		Resource("audits").
		Name(name).
		Body(auditPatch).
		Do().
		Into(&result)

	return &result, err
}
