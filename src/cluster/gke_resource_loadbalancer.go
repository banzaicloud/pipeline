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

package cluster

import (
	"context"

	"emperror.dev/errors"
	gkeCompute "google.golang.org/api/compute/v1"
)

type loadBalancerHelper struct {
	csv         *gkeCompute.Service
	project     string
	region      string
	zone        string
	clusterName string
	targetPools []*gkeCompute.TargetPool
}

func newLoadBalancerHelper(csv *gkeCompute.Service, project, region, zone, clusterName string) *loadBalancerHelper {
	return &loadBalancerHelper{
		csv:         csv,
		project:     project,
		region:      region,
		zone:        zone,
		clusterName: clusterName,
	}
}

func (lb *loadBalancerHelper) listTargetPools() ([]*gkeCompute.TargetPool, error) {
	if lb.targetPools == nil {
		pools, err := lb.csv.TargetPools.List(lb.project, lb.region).Context(context.Background()).Do()
		if err != nil {
			return nil, errors.WrapIf(err, "error during listing target pools")
		}

		instance, err := findInstanceByClusterName(lb.csv, lb.project, lb.zone, lb.clusterName)
		if err != nil {
			return nil, errors.WrapIf(err, "couldn't check if cluster exists")
		}

		if pools != nil && instance != nil {
			lb.targetPools = findTargetPoolsByInstances(pools.Items, instance.SelfLink)
		}
	}

	return lb.targetPools, nil
}
