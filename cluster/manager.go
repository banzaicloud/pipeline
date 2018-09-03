package cluster

import (
	"context"

	pipelineContext "github.com/banzaicloud/pipeline/internal/platform/context"
	"github.com/banzaicloud/pipeline/model"
	"github.com/goph/emperror"
	"github.com/sirupsen/logrus"
)

type clusterRepository interface {
	Exists(organizationID uint, name string) (bool, error)
	All() ([]*model.ClusterModel, error)
	FindByOrganization(organizationID uint) ([]*model.ClusterModel, error)
	FindOneByID(organizationID uint, clusterID uint) (*model.ClusterModel, error)
	FindOneByName(organizationID uint, clusterName string) (*model.ClusterModel, error)
}

type secretValidator interface {
	ValidateSecretType(organizationID uint, secretID string, cloud string) error
}

type Manager struct {
	clusters clusterRepository
	secrets  secretValidator

	logger       logrus.FieldLogger
	errorHandler emperror.Handler
}

func NewManager(clusters clusterRepository, secrets secretValidator, logger logrus.FieldLogger, errorHandler emperror.Handler) *Manager {
	return &Manager{
		clusters:     clusters,
		secrets:      secrets,
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
