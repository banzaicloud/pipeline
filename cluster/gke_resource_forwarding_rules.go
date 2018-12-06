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

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	gkeCompute "google.golang.org/api/compute/v1"
)

type forwardingRulesChecker struct {
	csv     *gkeCompute.Service
	project string
	region  string
	lb      *loadBalancerHelper
}

func newForwardingRulesChecker(csv *gkeCompute.Service, project string, region string, lb *loadBalancerHelper) *forwardingRulesChecker {
	return &forwardingRulesChecker{
		csv:     csv,
		project: project,
		region:  region,
		lb:      lb,
	}
}

func (fc *forwardingRulesChecker) getType() string {
	return forwardingRule
}

func (fc *forwardingRulesChecker) list() (resourceNames []string, err error) {

	log.WithFields(logrus.Fields{"project": fc.project, "region": fc.region})

	log.Info("List forwarding rules")
	forwardingRules, err := fc.csv.ForwardingRules.List(fc.project, fc.region).Context(context.Background()).Do()
	if err != nil {
		return nil, errors.Wrap(err, "error during listing forwarding rules")
	}

	targetPools, err := fc.lb.listTargetPools()
	if err != nil {
		return nil, errors.Wrap(err, "error during listing target pools")
	}

	for _, rule := range forwardingRules.Items {
		if rule != nil && isClusterTarget(targetPools, rule.Target) {
			resourceNames = append(resourceNames, rule.Name)
		}
	}

	return
}

func (fc *forwardingRulesChecker) isResourceDeleted(resourceName string) error {
	return isForwardingRuleDeleted(fc.csv, fc.project, fc.region, resourceName)
}

func (fc *forwardingRulesChecker) forceDelete(resourceName string) error {
	return deleteForwardingRule(fc.csv, fc.project, fc.region, resourceName)
}

// isClusterTarget checks the target match with the deleting cluster
func isClusterTarget(targetPools []*gkeCompute.TargetPool, targetPoolName string) bool {
	for _, tp := range targetPools {
		if tp != nil && tp.SelfLink == targetPoolName {
			return true
		}
	}
	return false
}

// isForwardingRuleDeleted checks the given forwarding rule is deleted by Kubernetes
func isForwardingRuleDeleted(csv *gkeCompute.Service, project, region, forwardingRule string) error {

	log := log.WithFields(logrus.Fields{"rule": forwardingRule, "project": project, "region": region})

	log.Info("Get forwarding rule")
	_, err := csv.ForwardingRules.Get(project, region, forwardingRule).Context(context.Background()).Do()
	if err != nil {
		return isResourceNotFound(err)
	}

	return errors.New("forwarding rule is still alive")
}

func deleteForwardingRule(csv *gkeCompute.Service, project, region, ruleName string) error {

	log := log.WithFields(logrus.Fields{"project": project, "rule": ruleName, "region": region})

	log.Info("delete forwardingRule")

	operation, err := csv.ForwardingRules.Delete(project, region, ruleName).Context(context.Background()).Do()
	if err != nil {
		return isResourceNotFound(err)
	}

	log.Info("wait for operation complete")

	return waitForOperation(newComputeRegionOperation(csv, project, region), operation.Name, log)
}
