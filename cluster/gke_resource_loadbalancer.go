package cluster

import (
	"context"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
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

	log := log.WithFields(logrus.Fields{"project": lb.project, "region": lb.region, "zone": lb.zone})

	if lb.targetPools == nil {

		log.Info("List target pools")
		pools, err := lb.csv.TargetPools.List(lb.project, lb.region).Context(context.Background()).Do()
		if err != nil {
			return nil, errors.Wrap(err, "error during listing target pools")
		}

		log.Info("List instances")
		instance, err := findInstanceByClusterName(lb.csv, lb.project, lb.zone, lb.clusterName)
		if err != nil {
			return nil, errors.Wrap(err, "error during listing instances")
		}

		log.Infof("Find target pool(s) by instance[%s]", instance.Name)
		lb.targetPools = findTargetPoolsByInstances(pools.Items, instance.SelfLink)

	}

	return lb.targetPools, nil
}
