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

package driver

import (
	"emperror.dev/errors"

	"github.com/banzaicloud/pipeline/internal/cluster/metrics"
	"github.com/banzaicloud/pipeline/internal/global"
	"github.com/banzaicloud/pipeline/src/auth"
)

func getClusterStatusChangeMetricTimer(provider, location, status string, orgId uint, clusterName string, statusChangeDurationMetric metrics.ClusterStatusChangeDurationMetric) (metrics.DurationMetricTimer, error) {
	if statusChangeDurationMetric == nil {
		return metrics.NoopDurationMetricTimer{}, nil
	}

	values := metrics.ClusterStatusChangeDurationMetricValues{
		ProviderName: provider,
		LocationName: location,
		Status:       status,
	}
	if global.Config.Telemetry.Debug {
		org, err := auth.GetOrganizationById(orgId)
		if err != nil {
			return nil, errors.WrapIf(err, "Error during getting organization. ")
		}

		values.OrganizationName = org.Name
		values.ClusterName = clusterName
	}
	return statusChangeDurationMetric.StartTimer(values), nil
}
