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
	"fmt"
	"strings"
	"time"

	"emperror.dev/emperror"
	"github.com/patrickmn/go-cache"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"go.uber.org/cadence/client"

	"github.com/banzaicloud/pipeline/internal/cluster/metrics"
	"github.com/banzaicloud/pipeline/internal/global"
	pipelineContext "github.com/banzaicloud/pipeline/internal/platform/context"
	"github.com/banzaicloud/pipeline/src/auth"
	"github.com/banzaicloud/pipeline/src/model"
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

type KubeProxyCache interface {
	Get(clusterUID string) (*KubeAPIProxy, bool)
	Put(clusterUID string, proxy *KubeAPIProxy)
	Delete(clusterUID string)
}

type goCacheKubeProxyCache struct {
	cache *cache.Cache
}

type Manager struct {
	clusters                   clusterRepository
	secrets                    secretValidator
	events                     clusterEvents
	statusChangeDurationMetric metrics.ClusterStatusChangeDurationMetric
	clusterTotalMetric         *prometheus.CounterVec
	kubeProxyCache             KubeProxyCache
	workflowClient             client.Client
	logger                     logrus.FieldLogger
	errorHandler               emperror.Handler
	clusterStore               interface {
		SetStatus(ctx context.Context, id uint, status, message string) error
	}
}

func NewManager(
	clusters clusterRepository,
	secrets secretValidator,
	events clusterEvents,
	statusChangeDurationMetric metrics.ClusterStatusChangeDurationMetric,
	clusterTotalMetric *prometheus.CounterVec,
	workflowClient client.Client,
	logger logrus.FieldLogger,
	errorHandler emperror.Handler,
	clusterStore interface {
		SetStatus(ctx context.Context, id uint, status, message string) error
	},
) *Manager {
	return &Manager{
		clusters:                   clusters,
		secrets:                    secrets,
		events:                     events,
		statusChangeDurationMetric: statusChangeDurationMetric,
		clusterTotalMetric:         clusterTotalMetric,
		kubeProxyCache:             &goCacheKubeProxyCache{cache: cache.New(defaultProxyExpirationMinutes*time.Minute, 1*time.Minute)},
		workflowClient:             workflowClient,
		logger:                     logger,
		errorHandler:               errorHandler,
		clusterStore:               clusterStore,
	}
}

func (m *Manager) getLogger(ctx context.Context) logrus.FieldLogger {
	return pipelineContext.LoggerWithCorrelationID(ctx, m.logger)
}

func (m *Manager) getErrorHandler(ctx context.Context) emperror.Handler {
	return pipelineContext.ErrorHandlerWithCorrelationID(ctx, m.errorHandler)
}

type clusterErrorHandler struct {
	handler       emperror.Handler
	status        string
	statusMessage string
	cluster       CommonCluster
}

func (c clusterErrorHandler) Handle(err error) {
	if c.statusMessage != "" {
		statusMessage := c.statusMessage
		if strings.Contains(statusMessage, "%") {
			statusMessage = fmt.Sprintf(statusMessage, err)
		}
		_ = c.cluster.SetStatus(c.status, statusMessage)
	}

	err = emperror.With(
		err,
		"clusterId", c.cluster.GetID(),
		"clusterName", c.cluster.GetName(),
	)

	c.handler.Handle(err)
}

func (c clusterErrorHandler) WithStatus(status, statusMessage string) clusterErrorHandler {
	return clusterErrorHandler{
		cluster:       c.cluster,
		status:        status,
		statusMessage: statusMessage,
		handler:       c.handler,
	}
}

func (m *Manager) getClusterErrorHandler(ctx context.Context, commonCluster CommonCluster) clusterErrorHandler {
	return clusterErrorHandler{
		handler: pipelineContext.ErrorHandlerWithCorrelationID(ctx, m.errorHandler),
		cluster: commonCluster,
	}
}

func (m *Manager) getClusterStatusChangeMetricTimer(provider, location, status string, orgId uint, clusterName string) (metrics.DurationMetricTimer, error) {
	values := metrics.ClusterStatusChangeDurationMetricValues{
		ProviderName: provider,
		LocationName: location,
		Status:       status,
	}
	if global.Config.Telemetry.Debug {
		org, err := auth.GetOrganizationById(orgId)
		if err != nil {
			return nil, emperror.Wrap(err, "Error during getting organization. ")
		}

		values.OrganizationName = org.Name
		values.ClusterName = clusterName
	}
	return m.statusChangeDurationMetric.StartTimer(values), nil
}

func (m *Manager) GetKubeProxy(requestSchema string, requestHost string, apiProxyPrefix string, commonCluster CommonCluster) (*KubeAPIProxy, error) {
	// Currently we do not lock this transaction of getting and optionally creating a KubeAPIProxy.
	// The worst thing that could happen is that for a short period (a Go GC period) there will be
	// an extra KubeAPIProxy object in memory, but we can keep this method lock-free I think this is a good trade-off.
	kubeProxy, found := m.kubeProxyCache.Get(commonCluster.GetUID())
	if !found {
		var err error

		kubeProxy, err = NewKubeAPIProxy(requestSchema, requestHost, apiProxyPrefix, commonCluster, defaultProxyExpirationMinutes*time.Minute)

		if err != nil {
			return nil, emperror.Wrap(err, "Error during creating cluster API proxy.")
		}

		m.kubeProxyCache.Put(commonCluster.GetUID(), kubeProxy)
	}
	return kubeProxy, nil
}

func (m *Manager) DeleteKubeProxy(commonCluster CommonCluster) {
	m.kubeProxyCache.Delete(commonCluster.GetUID())
}

func (c *goCacheKubeProxyCache) Get(clusterUID string) (*KubeAPIProxy, bool) {
	if kubeProxy, ok := c.cache.Get(clusterUID); ok {
		return kubeProxy.(*KubeAPIProxy), true
	}
	return nil, false
}

func (c *goCacheKubeProxyCache) Put(clusterUID string, kubeProxy *KubeAPIProxy) {
	c.cache.Set(clusterUID, kubeProxy, cache.DefaultExpiration)
}

func (c *goCacheKubeProxyCache) Delete(clusterUID string) {
	c.cache.Delete(clusterUID)
}

func (m *Manager) GetKubeProxyCache() KubeProxyCache {
	return m.kubeProxyCache
}
