package cluster

import (
	"context"

	pipelineContext "github.com/banzaicloud/pipeline/internal/platform/context"
	"github.com/sirupsen/logrus"
)

type clusterRepository interface {
	Exists(organizationID uint, name string) (bool, error)
}

type secretValidator interface {
	ValidateSecretType(organizationID uint, secretID string, cloud string) error
}

type Manager struct {
	clusters clusterRepository
	secrets  secretValidator

	logger logrus.FieldLogger
}

func NewManager(clusters clusterRepository, secrets secretValidator, logger logrus.FieldLogger) *Manager {
	return &Manager{
		clusters: clusters,
		secrets:  secrets,
		logger:   logger,
	}
}

func (m *Manager) getLogger(ctx context.Context) logrus.FieldLogger {
	return pipelineContext.LoggerWithCorrelationID(ctx, m.logger)
}
