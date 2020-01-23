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
	"github.com/antihax/optional"

	"github.com/banzaicloud/pipeline/.gen/cloudinfo"
)

// GetServiceRegions returns the cloud provider regions where the specified service is available.
func (c *Client) GetServiceRegions(ctx context.Context, cloudProvider string, service string) ([]string, error) {
	regions, _, err := c.apiClient.RegionsApi.GetRegions(ctx, cloudProvider, service)
	if err != nil {
		return nil, errors.WrapWithDetails(
			err, "could not get service availability regions",
			"cloudProvider", cloudProvider,
			"service", service,
		)
	}

	regionIds := make([]string, len(regions))
	for idx, region := range regions {
		regionIds[idx] = region.Id
	}

	return regionIds, nil
}

func (c *Client) PKEImageName(cloudProvider, service, os, kubeVersion, pkeVersion, region string) (string, error) {
	opts := &cloudinfo.GetImagesOpts{
		Version:    optional.NewString(kubeVersion),
		Os:         optional.NewString(os),
		PkeVersion: optional.NewString(pkeVersion),
		LatestOnly: optional.NewString("true"),
	}
	images, _, err := c.apiClient.ImagesApi.GetImages(context.Background(), cloudProvider, service, region, opts)
	if err != nil {
		return "", errors.WrapIfWithDetails(
			err, "couldn't get PKE images",
			"cloudProvider", cloudProvider,
			"service", service,
			"region", region,
			"getImagesOpts", opts,
		)
	}
	if len(images) <= 0 {
		return "", errors.NewWithDetails(
			"no PKE image found",
			"cloudProvider", cloudProvider,
			"service", service,
			"region", region,
			"getImagesOpts", opts,
		)
	}

	return images[0].Name, nil
}
