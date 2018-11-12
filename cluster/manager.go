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

	"github.com/banzaicloud/pipeline/auth"
	"github.com/banzaicloud/pipeline/config"
	pipelineContext "github.com/banzaicloud/pipeline/internal/platform/context"
	"github.com/banzaicloud/pipeline/model"
	"github.com/goph/emperror"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type clusterRepository interface {
	Exists(organizationID uint, name string) (bool, error)
	All() ([]*model.ClusterModel, error)
	FindByOrganization(organizationID uint) ([]*model.ClusterModel, error)
	FindOneByID(organizationID uint, clusterID uint) (*model.ClusterModel, error)
	FindOneByName(organizationID uint, clusterName string) (*model.ClusterModel, error)
	FindBySecret(organizationID uint, secretID string) ([]*model.ClusterModel, error)
}

type secretValidator interface {
	ValidateSecretType(organizationID uint, secretID string, cloud string) error
}

type Manager struct {
	clusters                   clusterRepository
	secrets                    secretValidator
	events                     clusterEvents
	statusChangeDurationMetric *prometheus.SummaryVec
	clusterTotalMetric         *prometheus.CounterVec

	logger       logrus.FieldLogger
	errorHandler emperror.Handler
}

func NewManager(clusters clusterRepository,
	secrets secretValidator,
	events clusterEvents,
	statusChangeDurationMetric *prometheus.SummaryVec,
	clusterTotalMetric *prometheus.CounterVec,
	logger logrus.FieldLogger,
	errorHandler emperror.Handler) *Manager {
	return &Manager{
		clusters:                   clusters,
		secrets:                    secrets,
		events:                     events,
		statusChangeDurationMetric: statusChangeDurationMetric,
		clusterTotalMetric:         clusterTotalMetric,

		logger:       logger,
		errorHandler: errorHandler,
	}
}

func (m *Manager) getLogger(ctx context.Context) logrus.FieldLogger {
	return pipelineContext.LoggerWithCorrelationID(ctx, m.logger)
}

func (m *Manager) getErrorHandler(ctx context.Context) emperror.Handler {
	return pipelineContext.ErrorHandlerWithCorrelationID(ctx, m.errorHandler)
}

func (m *Manager) getPrometheusTimer(provider, location, status string, orgId uint, clusterName string) (*prometheus.Timer, error) {
	if viper.GetBool(config.MetricsDebug) {
		org, err := auth.GetOrganizationById(orgId)
		if err != nil {
			return nil, err
		}

		return prometheus.NewTimer(m.statusChangeDurationMetric.WithLabelValues(provider, location, status, org.Name, clusterName)), nil
	}
	return prometheus.NewTimer(m.statusChangeDurationMetric.WithLabelValues(provider, location, status, "", "")), nil
}
