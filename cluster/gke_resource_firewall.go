package cluster

import (
	"context"
	"strings"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	gkeCompute "google.golang.org/api/compute/v1"
)

type firewallsChecker struct {
	csv         *gkeCompute.Service
	project     string
	clusterName string
}

func newFirewallChecker(csv *gkeCompute.Service, project, clusterName string) *firewallsChecker {
	return &firewallsChecker{
		csv:         csv,
		project:     project,
		clusterName: clusterName,
	}
}

func (fc *firewallsChecker) getType() string {
	return firewall
}

func (fc *firewallsChecker) list() (resourceNames []string, err error) {

	log := log.WithFields(logrus.Fields{"checker": "firewall", "project": fc.project, "cluster": fc.clusterName})

	log.Info("List firewalls")
	firewalls, err := fc.csv.Firewalls.List(fc.project).Context(context.Background()).Do()
	if err != nil {
		return nil, errors.Wrap(err, "Error during listing firewalls")
	}

	log.Info("Find firewall(s) by target")
	k8sFirewalls := findFirewallRulesByTarget(firewalls.Items, fc.clusterName)
	for _, f := range k8sFirewalls {
		resourceNames = append(resourceNames, f.Name)
	}

	return
}

func (fc *firewallsChecker) isResourceDeleted(resourceName string) error {
	return isFirewallDeleted(fc.csv, fc.project, resourceName)
}

func (fc *firewallsChecker) forceDelete(resourceName string) error {
	return deleteFirewall(fc.csv, fc.project, resourceName)
}

// findFirewallRulesByTarget returns all firewalls which created by Kubernetes
func findFirewallRulesByTarget(rules []*gkeCompute.Firewall, clusterName string) []*gkeCompute.Firewall {

	var firewalls []*gkeCompute.Firewall
	for _, r := range rules {
		if r != nil {

			if strings.Contains(r.Description, kubernetesIO) {

				for _, tag := range r.TargetTags {
					log.Debugf("Firewall rule[%s] target tag: %s", r.Name, tag)
					if strings.HasPrefix(tag, targetPrefix+clusterName) {
						log.Debugf("Append firewall list[%s]", r.Name)
						firewalls = append(firewalls, r)
					}
				}

			}
		}
	}

	return firewalls
}

// isFirewallDeleted checks the given firewall is deleted by Kubernetes
func isFirewallDeleted(csv *gkeCompute.Service, project, firewall string) error {

	log := log.WithFields(logrus.Fields{"firewall": firewall, "project": project})

	log.Info("get firewall")

	_, err := csv.Firewalls.Get(project, firewall).Context(context.Background()).Do()
	if err != nil {
		return notFoundGoogleError(err)
	}

	return errors.New("firewall is still alive")
}

func deleteFirewall(csv *gkeCompute.Service, project, firewallName string) error {

	log := log.WithFields(logrus.Fields{"project": project, "firewall": firewallName})

	log.Info("delete firewall")

	operation, err := csv.Firewalls.Delete(project, firewallName).Context(context.Background()).Do()
	if err != nil {
		return notFoundGoogleError(err)
	}

	log.Info("wait for operation complete")

	return waitForOperation(newComputeOperation(csv), "", project, operation.Name)
}
