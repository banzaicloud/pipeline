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
	"strconv"

	"emperror.dev/errors"

	"github.com/banzaicloud/pipeline/.gen/cloudinfo"
)

// SpotPriceValidator provides an interface to validating node pool node
// instance spot prices.
type SpotPriceValidator interface {
	// ValidateSpotPrice returns an error if the specified spot price would not be
	// valid in the provided context.
	ValidateSpotPrice(
		ctx context.Context,
		cloud string,
		service string,
		region string,
		productType string,
		location string,
		spotPrice string,
	) error
}

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

// ValidateSpotPrice returns an error if the specified spot price would not be
// valid in the provided context.
func (c *Client) ValidateSpotPrice(
	ctx context.Context,
	cloud string,
	service string,
	region string,
	productType string,
	location string,
	spotPrice string,
) error {
	if spotPrice == "" ||
		spotPrice == "0.0" { // Note: on-demand instances.
		return nil
	}

	spotPriceValue, err := strconv.ParseFloat(spotPrice, 64)
	if err != nil {
		return errors.Errorf("invalid non-float spot price value '%s'", spotPrice)
	}

	productDetails, err := c.GetProductDetails(ctx, cloud, service, region, productType)
	if err != nil {
		return errors.WrapWithDetails(
			err,
			"retrieving product details failed",
			"cloud", cloud,
			"service", service,
			"region", region,
			"productType", productType,
		)
	} else if len(productDetails.SpotPrice) == 0 {
		return errors.WithDetails(
			errors.New("invalid product details, empty zone spot prices"),
			"cloud", cloud,
			"service", service,
			"region", region,
			"productType", productType,
		)
	}

	isValid := true

	for _, zoneSpotPrice := range productDetails.SpotPrice {
		isValid = isValid && spotPriceValue >= zoneSpotPrice.Price

		if location == zoneSpotPrice.Zone { // Note: zone specified.
			isValid = spotPriceValue >= zoneSpotPrice.Price

			break
		}
	}

	if !isValid {
		return errors.WithDetails(
			errors.Errorf("invalid spot price %s", spotPrice),
			"zoneSpotPrices", productDetails.SpotPrice,
		)
	}

	return nil
}
