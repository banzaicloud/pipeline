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
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"

	"github.com/banzaicloud/anchore-image-validator/pkg/apis/security/v1alpha1"
)

// WhiteListInterface for whitelist
type WhiteListInterface interface {
	List(opts metav1.ListOptions) (*v1alpha1.WhiteListItemList, error)
	Get(name string, options metav1.GetOptions) (*v1alpha1.WhiteListItem, error)
	Create(*v1alpha1.WhiteListItem) (*v1alpha1.WhiteListItem, error)
	Delete(name string, options *metav1.DeleteOptions) error
}

type whitelistClient struct {
	restClient rest.Interface
}

func (c *whitelistClient) List(opts metav1.ListOptions) (*v1alpha1.WhiteListItemList, error) {
	result := v1alpha1.WhiteListItemList{}
	err := c.restClient.
		Get().
		Resource("whitelistitems").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(&result)

	return &result, err
}

func (c *whitelistClient) Get(name string, opts metav1.GetOptions) (*v1alpha1.WhiteListItem, error) {
	result := v1alpha1.WhiteListItem{}
	err := c.restClient.
		Get().
		Resource("whitelistitems").
		Name(name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(&result)

	return &result, err
}

func (c *whitelistClient) Create(whiteListItem *v1alpha1.WhiteListItem) (*v1alpha1.WhiteListItem, error) {
	result := v1alpha1.WhiteListItem{}
	err := c.restClient.
		Post().
		Resource("whitelistitems").
		Body(whiteListItem).
		Do().
		Into(&result)

	return &result, err
}

func (c *whitelistClient) Delete(name string, options *metav1.DeleteOptions) error {

	return c.restClient.
		Delete().
		Resource("whitelistitems").
		Name(name).
		Body(options).
		Do().
		Error()
}
