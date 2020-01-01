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
