package cluster

import (
	"context"
	"fmt"

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

	log.Debugf("Forwarding rules: %d", len(forwardingRules.Items))

	for _, rule := range forwardingRules.Items {
		if rule != nil && isClusterTarget(targetPools, fc.project, fc.region, rule.Target) {
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
func isClusterTarget(targetPools []*gkeCompute.TargetPool, project, region, targetPoolName string) bool {
	for _, tp := range targetPools {
		log.Info("target url: ", getTargetUrl(project, region, targetPoolName))
		log.Info("tp name: ", tp.Name)
		if tp != nil && tp.Name == getTargetUrl(project, region, targetPoolName) {
			return true
		}
	}
	return false
}

// getTargetUrl returns target url for gke cluster
func getTargetUrl(project, region, targetPoolName string) string {
	return fmt.Sprintf("https://www.googleapis.com/compute/v1/projects/%s/regions/%s/targetPools/%s", project, region, targetPoolName)
}

// isForwardingRuleDeleted checks the given forwarding rule is deleted by Kubernetes
func isForwardingRuleDeleted(csv *gkeCompute.Service, project, region, forwardingRule string) error {

	log := log.WithFields(logrus.Fields{"rule": forwardingRule, "project": project, "region": region})

	log.Info("Get forwarding rule", forwardingRule)
	_, err := csv.ForwardingRules.Get(project, region, forwardingRule).Context(context.Background()).Do()
	if err != nil {
		return notFoundGoogleError(err)
	}

	return errors.New("forwarding rule is still alive")
}

func deleteForwardingRule(csv *gkeCompute.Service, project, region, ruleName string) error {

	log := log.WithFields(logrus.Fields{"project": project, "rule": ruleName, "region": region})

	log.Info("delete forwardingRule")

	operation, err := csv.ForwardingRules.Delete(project, region, ruleName).Context(context.Background()).Do()
	if err != nil {
		return notFoundGoogleError(err)
	}

	log.Info("wait for operation complete")

	return waitForOperation(newComputeOperation(csv), "", project, operation.Name)
}
