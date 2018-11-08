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

	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	"github.com/banzaicloud/pipeline/secret"
	"github.com/goph/emperror"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

// CreationContext represents the data necessary to do generic cluster creation steps/checks.
type CreationContext struct {
	OrganizationID uint
	UserID         uint
	Name           string
	Provider       string
	SecretID       string
	PostHooks      []PostFunctioner
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

// CreateCluster creates a new cluster.
func (m *Manager) CreateCluster(ctx context.Context, creationCtx CreationContext, creator clusterCreator) (CommonCluster, error) {
	logger := m.getLogger(ctx).WithFields(logrus.Fields{
		"organization": creationCtx.OrganizationID,
		"user":         creationCtx.UserID,
		"cluster":      creationCtx.Name,
	})

	logger.Info("looking for existing cluster")
	if err := m.assertNotExists(creationCtx); err != nil {
		return nil, err
	}

	logger.Info("validating secret")
	err := m.secrets.ValidateSecretType(creationCtx.OrganizationID, creationCtx.SecretID, creationCtx.Provider)
	if err != nil {
		return nil, err
	}

	logger.Info("validating creation context")

	if err := creator.Validate(ctx); err != nil {
		return nil, errors.Wrap(&invalidError{err}, "validation failed")
	}

	logger.Info("creation context is valid")
	logger.Info("preparing cluster creation")

	cluster, err := creator.Prepare(ctx)
	if err != nil {
		return nil, err
	}
	timer := prometheus.NewTimer(StatusChangeDuration.WithLabelValues(cluster.GetCloud(), cluster.GetLocation(), pkgCluster.Creating))

	if err := cluster.UpdateStatus(pkgCluster.Creating, pkgCluster.CreatingMessage); err != nil {
		return nil, err
	}

	logger.Info("creating cluster")

	go func() {
		defer emperror.HandleRecover(m.errorHandler)
		defer timer.ObserveDuration()

		err := m.createCluster(ctx, cluster, creator, creationCtx.PostHooks, logger)
		if err != nil {
			logger.Errorf("failed to create cluster: %s", err.Error())
		}
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

func (m *Manager) createCluster(
	ctx context.Context,
	cluster CommonCluster,
	creator clusterCreator,
	postHooks []PostFunctioner,
	logger logrus.FieldLogger,
) error {
	// Check if public ssh key is needed for the cluster. If so and there is generate one and store it Vault
	if len(cluster.GetSshSecretId()) == 0 && cluster.RequiresSshPublicKey() {
		logger.Info("generating SSH Key for the cluster")

		sshKey, err := secret.GenerateSSHKeyPair()
		if err != nil {
			return errors.Wrap(err, "key generator failed")
		}

		sshSecretId, err := secret.StoreSSHKeyPair(sshKey, cluster.GetOrganizationId(), cluster.GetID(), cluster.GetName(), cluster.GetUID())
		if err != nil {
			return errors.Wrap(err, "key store failed")
		}

		if err := cluster.SaveSshSecretId(sshSecretId); err != nil {
			return errors.Wrap(err, "saving SSH key secret failed")
		}
	}

	err := creator.Create(ctx)
	if err != nil {
		cluster.UpdateStatus(pkgCluster.Error, err.Error())
		return err
	}

	// Apply PostHooks
	// These are hardcoded posthooks maybe we will want a bit more dynamic
	postHookFunctions := BasePostHookFunctions

	if postHooks != nil && len(postHooks) != 0 {
		postHookFunctions = append(postHookFunctions, postHooks...)
	}

	err = RunPostHooks(postHookFunctions, cluster)

	if err != nil {
		return errors.Wrap(err, "error during running cluster posthooks")
	}

	m.events.ClusterCreated(cluster.GetID())

	return nil
}
