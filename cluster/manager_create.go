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
	stderrors "errors"

	pkgAuth "github.com/banzaicloud/pipeline/pkg/auth"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	secretTypes "github.com/banzaicloud/pipeline/pkg/secret"
	"github.com/banzaicloud/pipeline/secret"
	"github.com/goph/emperror"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// CreationContext represents the data necessary to do generic cluster creation steps/checks.
type CreationContext struct {
	OrganizationID  pkgAuth.OrganizationID
	UserID          pkgAuth.UserID
	ExternalBaseURL string
	Name            string
	Provider        string
	SecretID        secretTypes.SecretID
	SecretIDs       []secretTypes.SecretID
	PostHooks       []PostFunctioner
}

var ErrAlreadyExists = stderrors.New("cluster already exists with this name")

type clusterCreator interface {
	// Validate validates the cluster creation context.
	Validate(ctx context.Context) error

	// Prepare prepares a cluster to be created.
	Prepare(ctx context.Context) (CommonCluster, error)

	// Create creates a cluster.
	Create(ctx context.Context) error
}

// CreateCluster creates a new cluster in the background.
func (m *Manager) CreateCluster(ctx context.Context, creationCtx CreationContext, creator clusterCreator) (CommonCluster, error) {
	logger := m.getLogger(ctx).WithFields(logrus.Fields{
		"organization": creationCtx.OrganizationID,
		"user":         creationCtx.UserID,
		"cluster":      creationCtx.Name,
	})

	errorHandler := emperror.HandlerWith(
		m.getErrorHandler(ctx),
		"organization", creationCtx.OrganizationID,
		"user", creationCtx.UserID,
		"cluster", creationCtx.Name,
	)

	logger.Debug("looking for existing cluster")
	if err := m.assertNotExists(creationCtx); err != nil {
		return nil, err
	}

	logger.Info("validating secret")
	if len(creationCtx.SecretIDs) > 0 {
		var err error
		for _, secretID := range creationCtx.SecretIDs {
			err = m.secrets.ValidateSecretType(creationCtx.OrganizationID, secretID, creationCtx.Provider)
			if err == nil {
				creationCtx.SecretID = secretID
				break
			}
		}
		if err != nil {
			return nil, err
		}
	} else {
		if err := m.secrets.ValidateSecretType(creationCtx.OrganizationID, creationCtx.SecretID, creationCtx.Provider); err != nil {
			return nil, err
		}
	}

	logger.Debug("validating creation context")
	if err := creator.Validate(ctx); err != nil {
		return nil, errors.Wrap(&invalidError{err}, "validation failed")
	}

	logger.Debug("preparing cluster creation")
	cluster, err := creator.Prepare(ctx)
	if err != nil {
		return nil, err
	}

	switch c := cluster.(type) {
	case *EC2ClusterPKE:
		for _, secretID := range creationCtx.SecretIDs {
			if m.secrets.ValidateSecretType(creationCtx.OrganizationID, secretID, pkgCluster.Amazon) == nil {
				c.model.Cluster.SecretID = secretID
			}
			if m.secrets.ValidateSecretType(creationCtx.OrganizationID, secretID, secretTypes.SSHSecretType) == nil {
				c.model.Cluster.SSHSecretID = secretID
			}
			if m.secrets.ValidateSecretType(creationCtx.OrganizationID, secretID, pkgCluster.Kubernetes) == nil {
				c.model.Cluster.ConfigSecretID = secretID
			}
		}
	}

	m.clusterTotalMetric.WithLabelValues(cluster.GetCloud(), cluster.GetLocation()).Inc()

	timer, err := m.getPrometheusTimer(cluster.GetCloud(), cluster.GetLocation(), pkgCluster.Creating, cluster.GetOrganizationId(), cluster.GetName())
	if err != nil {
		return nil, err
	}

	if err := cluster.UpdateStatus(pkgCluster.Creating, pkgCluster.CreatingMessage); err != nil {
		return nil, err
	}

	logger.Infof("creating cluster")

	go func() {
		defer emperror.HandleRecover(m.errorHandler)
		ctx = context.WithValue(ctx, "ExternalBaseURL", creationCtx.ExternalBaseURL)
		err := m.createCluster(ctx, cluster, creator, creationCtx.PostHooks, logger)
		if err != nil {
			errorHandler.Handle(err)
			return
		}
		timer.ObserveDuration()
	}()

	return cluster, nil
}

func (m *Manager) assertNotExists(ctx CreationContext) error {
	exists, err := m.clusters.Exists(ctx.OrganizationID, ctx.Name)
	if err != nil {
		return err
	}

	if exists {
		return ErrAlreadyExists
	}

	return nil
}

// createCluster creates the cluster blockingly given an initially validated context
// updates cluster status, but the caller logs the returned error
func (m *Manager) createCluster(
	ctx context.Context,
	cluster CommonCluster,
	creator clusterCreator,
	postHooks []PostFunctioner,
	logger logrus.FieldLogger,
) error {
	// Check if public ssh key is needed for the cluster. If so and there is generate one and store it Vault
	if len(cluster.GetSshSecretId()) == 0 && cluster.RequiresSshPublicKey() {
		logger.Debug("generating SSH Key for the cluster")

		sshKey, err := secret.GenerateSSHKeyPair()
		if err != nil {
			cluster.UpdateStatus(pkgCluster.Error, "internal error")
			return emperror.Wrap(err, "failed to generate SSH key")
		}

		sshSecretId, err := secret.StoreSSHKeyPair(sshKey, cluster.GetOrganizationId(), cluster.GetID(), cluster.GetName(), cluster.GetUID())
		if err != nil {
			cluster.UpdateStatus(pkgCluster.Error, "internal error")
			return emperror.Wrap(err, "failed to store SSH key")
		}

		if err := cluster.SaveSshSecretId(sshSecretId); err != nil {
			cluster.UpdateStatus(pkgCluster.Error, "internal error")
			return emperror.Wrap(err, "failed to save SSH key secret ID")
		}
	}
	if err := creator.Create(ctx); err != nil {
		cluster.UpdateStatus(pkgCluster.Error, err.Error())
		return err
	}

	postHookFunctions := BasePostHookFunctions
	if postHooks != nil && len(postHooks) != 0 {
		postHookFunctions = append(postHookFunctions, postHooks...)
	}

	if err := RunPostHooks(postHookFunctions, cluster); err != nil {
		return emperror.Wrap(err, "posthook failed")
	}

	m.events.ClusterCreated(cluster.GetID())

	return nil
}
