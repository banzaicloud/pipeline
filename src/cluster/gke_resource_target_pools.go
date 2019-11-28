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
	"errors"
	"fmt"

	"github.com/sirupsen/logrus"
	gkeCompute "google.golang.org/api/compute/v1"
)

type targetPoolsChecker struct {
	csv         *gkeCompute.Service
	project     string
	clusterName string
	region      string
	zone        string
	lb          *loadBalancerHelper
}

func newTargetPoolsChecker(csv *gkeCompute.Service, project, clusterName, region, zone string, lb *loadBalancerHelper) *targetPoolsChecker {
	return &targetPoolsChecker{
		csv:         csv,
		project:     project,
		clusterName: clusterName,
		region:      region,
		zone:        zone,
		lb:          lb,
	}
}

func (tc *targetPoolsChecker) getType() string {
	return targetPool
}

func (tc *targetPoolsChecker) list() (resourceNames []string, err error) {

	var targetPools []*gkeCompute.TargetPool
	targetPools, err = tc.lb.listTargetPools()
	if err != nil {
		return
	}

	for _, tp := range targetPools {
		if tp != nil {
			resourceNames = append(resourceNames, tp.Name)
		}
	}

	return
}

func (tc *targetPoolsChecker) isResourceDeleted(resourceName string) error {
	return isTargetPoolDeleted(tc.csv, tc.project, tc.region, resourceName)
}

func (tc *targetPoolsChecker) forceDelete(resourceName string) error {
	return deleteTargetPool(tc.csv, tc.project, tc.region, resourceName)
}

// findInstanceByClusterName returns the cluster's instance
func findInstanceByClusterName(csv *gkeCompute.Service, project, zone, clusterName string) (*gkeCompute.Instance, error) {

	instances, err := csv.Instances.List(project, zone).Context(context.Background()).Do()
	if err != nil {
		return nil, err
	}

	for _, instance := range instances.Items {
		if instance != nil && instance.Metadata != nil {
			for _, item := range instance.Metadata.Items {
				if item != nil && item.Key == clusterNameKey && item.Value != nil && *item.Value == clusterName {
					return instance, nil
				}
			}
		}
	}

	return nil, fmt.Errorf("instance not found by cluster[%s]", clusterName)
}

// findTargetPoolsByInstances returns all target pools which created by Kubernetes
func findTargetPoolsByInstances(pools []*gkeCompute.TargetPool, instanceName string) []*gkeCompute.TargetPool {

	var filteredPools []*gkeCompute.TargetPool
	for _, p := range pools {
		if p != nil {
			for _, i := range p.Instances {
				if i == instanceName {
					filteredPools = append(filteredPools, p)
				}
			}
		}
	}

	return filteredPools
}

// isTargetPoolDeleted checks the given target pool is deleted by Kubernetes
func isTargetPoolDeleted(csv *gkeCompute.Service, project, region, targetPoolName string) error {

	log := log.WithFields(logrus.Fields{"project": project, "region": region, "target pool": targetPoolName})

	log.Info("Get target pool")
	_, err := csv.TargetPools.Get(project, region, targetPoolName).Context(context.Background()).Do()
	if err != nil {
		return isResourceNotFound(err)
	}

	return errors.New("target pool is still alive")
}

func deleteTargetPool(csv *gkeCompute.Service, project, region, poolName string) error {

	log := log.WithFields(logrus.Fields{"project": project, "pool": poolName, "region": region})

	log.Info("delete target pool")

	operation, err := csv.TargetPools.Delete(project, region, poolName).Context(context.Background()).Do()
	if err != nil {
		return isResourceNotFound(err)
	}

	log.Info("wait for operation complete")

	return waitForOperation(newComputeRegionOperation(csv, project, region), operation.Name, log)
}
