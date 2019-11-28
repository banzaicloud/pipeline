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

package monitor

import (
	"context"
	"time"

	"emperror.dev/emperror"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	pkgEC2 "github.com/banzaicloud/pipeline/pkg/providers/amazon/ec2"
	"github.com/banzaicloud/pipeline/src/cluster"
	"github.com/banzaicloud/pipeline/src/secret/verify"
)

const metricsNamesapce = "pipeline"

type spotMetricsExporter struct {
	ctx          context.Context
	manager      *cluster.Manager
	logger       logrus.FieldLogger
	errorHandler emperror.Handler

	ec2Clients map[string]*ec2.EC2
	exporter   *pkgEC2.SpotMetricsExporter
}

// NewSpotMetricsExporter gives back an initialized spotMetricsExporter
func NewSpotMetricsExporter(ctx context.Context, manager *cluster.Manager, logger logrus.FieldLogger) *spotMetricsExporter {
	return &spotMetricsExporter{
		ctx:          ctx,
		manager:      manager,
		logger:       logger,
		errorHandler: NewSpotMetricsErrorHandler(logger),
		exporter:     pkgEC2.NewSpotMetricsExporter(logger, metricsNamesapce),
		ec2Clients:   make(map[string]*ec2.EC2),
	}
}

// Run runs the metrics collections with the given interval
func (e *spotMetricsExporter) Run(interval time.Duration) {
	e.logger.WithField("interval", interval.String()).Debug("collecting spot request metrics from EKS clusters")
	err := e.collectMetrics()
	if err != nil {
		e.errorHandler.Handle(emperror.Wrap(err, "could not collect spot metrics"))
	}

	ticker := time.NewTicker(interval)
	for {
		select {
		case <-ticker.C:
			e.logger.WithField("interval", interval.String()).Debug("collecting spot request metrics from EKS clusters")
			err := e.collectMetrics()
			if err != nil {
				e.errorHandler.Handle(emperror.Wrap(err, "could not collect spot metrics"))
			}
		case <-e.ctx.Done():
			e.logger.Debug("closing ticker")
			ticker.Stop()
			return
		}
	}
}

func (e *spotMetricsExporter) collectMetrics() error {
	clusters, err := e.manager.GetAllClusters(e.ctx)
	if err != nil {
		return emperror.Wrap(err, "could not get clusters from cluster manager")
	}

	requests := make(map[string]*pkgEC2.SpotInstanceRequest)
	for _, cluster := range clusters {
		clusterName := cluster.GetName()
		clusterID := cluster.GetID()
		log := e.logger.WithField("cluster", clusterName)

		status, err := cluster.GetStatus()
		if err != nil {
			e.errorHandler.Handle(emperror.WrapWith(err, "could not get cluster status", "clusterID", clusterID, "clusterName", clusterName))
		}
		if status.Status != pkgCluster.Running || cluster.GetDistribution() != pkgCluster.EKS {
			continue
		}

		log.Debug("collecting metrics from cluster")
		clusterSecret, err := cluster.GetSecretWithValidation()
		if err != nil {
			e.errorHandler.Handle(emperror.WrapWith(err, "could not get secret", "clusterID", clusterID, "clusterName", clusterName))
			continue
		}

		client, err := e.getEC2Client(aws.Config{
			Region:      aws.String(cluster.GetLocation()),
			Credentials: verify.CreateAWSCredentials(clusterSecret.Values),
		})
		if err != nil {
			e.errorHandler.Handle(emperror.WrapWith(err, "could not get EC2 service", "clusterID", clusterID, "clusterName", clusterName))
			continue
		}

		srs, err := e.exporter.GetSpotRequests(client)
		if err != nil {
			e.errorHandler.Handle(emperror.WrapWith(err, "could not get spot requests", "clusterID", clusterID, "clusterName", clusterName))
			continue
		}
		for key, request := range srs {
			if requests[key] == nil {
				requests[key] = request
			}
		}
	}

	e.exporter.SetSpotRequestMetrics(requests)

	return nil
}

func (e *spotMetricsExporter) getEC2Client(config aws.Config) (*ec2.EC2, error) {
	credentials, err := config.Credentials.Get()
	if err != nil {
		return nil, errors.WithStack(err)
	}

	key := *config.Region + "-" + credentials.AccessKeyID
	if e.ec2Clients[key] == nil {
		sess, err := session.NewSession(&config)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		e.ec2Clients[key] = ec2.New(sess)
	}

	return e.ec2Clients[key], nil
}
