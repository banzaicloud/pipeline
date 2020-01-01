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

package cloudinfo

import (
	"context"

	"emperror.dev/errors"

	"github.com/banzaicloud/pipeline/.gen/cloudinfo"
)

type productCacheKey struct {
	cloud       string
	service     string
	region      string
	productType string
}

// GetProductDetails returns details for a single product.
func (c *Client) GetProductDetails(
	ctx context.Context,
	cloud string,
	service string,
	region string,
	productType string,
) (cloudinfo.ProductDetails, error) {
	key := productCacheKey{
		cloud:       cloud,
		service:     service,
		region:      region,
		productType: productType,
	}

	cachedProduct, ok := c.productCache.Load(key)
	if !ok {
		err := c.warmProductCache(ctx, cloud, service, region)
		if err != nil {
			return cloudinfo.ProductDetails{}, err
		}

		cachedProduct, ok := c.productCache.Load(key)
		if !ok {
			return cloudinfo.ProductDetails{}, errors.NewWithDetails(
				"no product info found",
				"cloud", cloud,
				"region", region,
				"service", service,
				"instanceType", productType,
			)
		}

		return cachedProduct.(cloudinfo.ProductDetails), nil
	}

	return cachedProduct.(cloudinfo.ProductDetails), nil
}

func (c *Client) warmProductCache(ctx context.Context, cloud string, service string, region string) error {
	response, _, err := c.apiClient.ProductsApi.GetProducts(ctx, cloud, service, region)
	if err != nil {
		return errors.WithStack(err)
	}

	for _, product := range response.Products {
		c.productCache.Store(
			productCacheKey{
				cloud:       cloud,
				service:     service,
				region:      region,
				productType: product.Type,
			},
			product,
		)
	}

	return nil
}
