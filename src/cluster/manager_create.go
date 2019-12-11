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
	"sort"
	"time"

	"emperror.dev/emperror"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"go.uber.org/cadence/client"

	"github.com/banzaicloud/pipeline/internal/secret/secrettype"
	"github.com/banzaicloud/pipeline/internal/secret/ssh"
	"github.com/banzaicloud/pipeline/internal/secret/ssh/sshdriver"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	"github.com/banzaicloud/pipeline/src/auth"
)

// CreationContext represents the data necessary to do generic cluster creation steps/checks.
type CreationContext struct {
	OrganizationID          uint
	UserID                  uint
	ExternalBaseURL         string
	ExternalBaseURLInsecure bool
	Name                    string
	Provider                string
	SecretID                string
	SecretIDs               []string
	PostHooks               pkgCluster.PostHooks
}

type contextKey string

const ExternalBaseURLKey = contextKey("ExternalBaseURL")
const ExternalBaseURLInsecureKey = contextKey("ExternalBaseURLInsecure")

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
func (m *Manager) CreateCluster(
	ctx context.Context,
	creationCtx CreationContext,
	creator clusterCreator,
) (CommonCluster, error) {
	logger := m.getLogger(ctx).WithFields(logrus.Fields{
		"organization": creationCtx.OrganizationID,
		"user":         creationCtx.UserID,
		"cluster":      creationCtx.Name,
	})

	if err := m.assertNotExists(creationCtx); err != nil {
		return nil, err
	}

	logger.Debug("validating secret")
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
		if creationCtx.SecretID != "" {
			c.model.Cluster.SecretID = creationCtx.SecretID
		}
		for _, secretID := range creationCtx.SecretIDs {
			if m.secrets.ValidateSecretType(creationCtx.OrganizationID, secretID, pkgCluster.Amazon) == nil {
				c.model.Cluster.SecretID = secretID
			}
			if m.secrets.ValidateSecretType(creationCtx.OrganizationID, secretID, secrettype.SSHSecretType) == nil {
				c.model.Cluster.SSHSecretID = secretID
			}
			if m.secrets.ValidateSecretType(creationCtx.OrganizationID, secretID, pkgCluster.Kubernetes) == nil {
				c.model.Cluster.ConfigSecretID = secretID
			}
		}
	}

	m.clusterTotalMetric.WithLabelValues(cluster.GetCloud(), cluster.GetLocation()).Inc()

	timer, err := m.getClusterStatusChangeMetricTimer(cluster.GetCloud(), cluster.GetLocation(), pkgCluster.Creating, cluster.GetOrganizationId(), cluster.GetName())
	if err != nil {
		return nil, err
	}

	if err := cluster.SetStatus(pkgCluster.Creating, pkgCluster.CreatingMessage); err != nil {
		return nil, err
	}

	logger.Info("creating cluster")

	errorHandler := m.getClusterErrorHandler(ctx, cluster)

	go func() {
		defer emperror.HandleRecover(errorHandler.WithStatus(pkgCluster.Error, "internal error while creating cluster"))

		ctx = context.WithValue(ctx, ExternalBaseURLKey, creationCtx.ExternalBaseURL)
		ctx = context.WithValue(ctx, ExternalBaseURLInsecureKey, creationCtx.ExternalBaseURLInsecure)
		err := m.createCluster(ctx, cluster, creator, creationCtx.PostHooks, logger)
		if err != nil {
			errorHandler.Handle(err)
			return
		}
		timer.RecordDuration()
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
	postHooks pkgCluster.PostHooks,
	logger logrus.FieldLogger,
) error {
	// Check if public ssh key is needed for the cluster. If so and there is generate one and store it Vault
	if len(cluster.GetSshSecretId()) == 0 && cluster.RequiresSshPublicKey() {
		logger.Debug("generating SSH Key for the cluster")

		sshKey, err := ssh.NewKeyPairGenerator().Generate()
		if err != nil {
			_ = cluster.SetStatus(pkgCluster.Error, "internal error")
			return emperror.Wrap(err, "failed to generate SSH key")
		}

		sshSecretId, err := sshdriver.StoreSSHKeyPair(sshKey, cluster.GetOrganizationId(), cluster.GetID(), cluster.GetName(), cluster.GetUID())
		if err != nil {
			_ = cluster.SetStatus(pkgCluster.Error, "internal error")
			return emperror.Wrap(err, "failed to store SSH key")
		}

		if err := cluster.SaveSshSecretId(sshSecretId); err != nil {
			_ = cluster.SetStatus(pkgCluster.Error, "internal error")
			return emperror.Wrap(err, "failed to save SSH key secret ID")
		}
	}
	if err := creator.Create(ctx); err != nil {
		_ = cluster.SetStatus(pkgCluster.Error, err.Error())
		return err
	}

	// We only set the status once here, because status setting overwrites modifications made by
	// the cluster create and the posthook run workflows.
	// Status setting should be updated everywhere to only change the status fields in the database.
	// (and/or reload the model between workflow executions which is probably a good idea anyway)
	err := cluster.SetStatus(pkgCluster.Creating, "running posthooks")
	if err != nil {
		return emperror.Wrap(err, "failed to update cluster status")
	}

	labelsMap, err := GetDesiredLabelsForCluster(ctx, cluster, nil, false)
	if err != nil {
		_ = cluster.SetStatus(pkgCluster.Error, "failed to get desired labels")

		return err
	}

	logger.WithField("workflowName", CreateClusterWorkflowName).Info("starting workflow")

	{
		org, err := auth.GetOrganizationById(cluster.GetOrganizationId())
		if err != nil {
			return emperror.Wrap(err, "failed to get organization name")
		}

		input := CreateClusterWorkflowInput{
			ClusterID:        cluster.GetID(),
			ClusterUID:       cluster.GetUID(),
			ClusterName:      cluster.GetName(),
			OrganizationID:   cluster.GetOrganizationId(),
			OrganizationName: org.Name,
			Distribution:     cluster.GetDistribution(),
			NodePoolLabels:   labelsMap,
		}

		workflowOptions := client.StartWorkflowOptions{
			TaskList:                     "pipeline",
			ExecutionStartToCloseTimeout: 2 * time.Hour, // TODO: lower timeout
		}

		exec, err := m.workflowClient.ExecuteWorkflow(ctx, workflowOptions, CreateClusterWorkflowName, input)
		if err != nil {
			_ = cluster.SetStatus(pkgCluster.Error, "failed to run setup jobs")

			return emperror.WrapWith(err, "failed to start workflow", "workflowName", CreateClusterWorkflowName)
		}

		logger.WithFields(logrus.Fields{
			"workflowName":  CreateClusterWorkflowName,
			"workflowID":    exec.GetID(),
			"workflowRunID": exec.GetRunID(),
		}).Info("workflow started successfully")

		err = exec.Get(ctx, nil)
		if err != nil {
			_ = m.clusterStore.SetStatus(ctx, cluster.GetID(), pkgCluster.Error, err.Error())

			return emperror.Wrap(err, "running setup jobs failed")
		}

		logger.WithFields(logrus.Fields{
			"workflowName":  CreateClusterWorkflowName,
			"workflowID":    exec.GetID(),
			"workflowRunID": exec.GetRunID(),
		}).Info("workflow finished successfully")
	}

	if postHooks == nil {
		postHooks = make(pkgCluster.PostHooks)
	}

	logger.WithField("workflowName", RunPostHooksWorkflowName).Info("starting workflow")

	{
		input := RunPostHooksWorkflowInput{
			ClusterID: cluster.GetID(),
			PostHooks: BuildWorkflowPostHookFunctions(postHooks, true),
		}

		workflowOptions := client.StartWorkflowOptions{
			TaskList:                     "pipeline",
			ExecutionStartToCloseTimeout: 2 * time.Hour, // TODO: lower timeout
		}

		exec, err := m.workflowClient.ExecuteWorkflow(ctx, workflowOptions, RunPostHooksWorkflowName, input)
		if err != nil {
			_ = cluster.SetStatus(pkgCluster.Error, "failed to run posthooks")

			return emperror.WrapWith(err, "failed to start workflow", "workflowName", RunPostHooksWorkflowName)
		}

		logger.WithFields(logrus.Fields{
			"workflowName":  RunPostHooksWorkflowName,
			"workflowID":    exec.GetID(),
			"workflowRunID": exec.GetRunID(),
		}).Info("workflow started successfully")

		err = exec.Get(ctx, nil)
		if err != nil {
			return emperror.Wrap(err, "running posthooks failed")
		}

		logger.WithFields(logrus.Fields{
			"workflowName":  RunPostHooksWorkflowName,
			"workflowID":    exec.GetID(),
			"workflowRunID": exec.GetRunID(),
		}).Info("workflow finished successfully")
	}

	m.events.ClusterCreated(cluster.GetID())

	return nil
}

// BuildWorkflowPostHookFunctions builds posthook workflow input.
func BuildWorkflowPostHookFunctions(postHooks pkgCluster.PostHooks, alwaysIncludeBasePostHooks bool) []RunPostHooksWorkflowInputPostHook {
	var workflowPostHooks []RunPostHooksWorkflowInputPostHook

	if len(postHooks) == 0 || alwaysIncludeBasePostHooks {
		for _, postHookName := range BasePostHookFunctions {
			workflowPostHooks = append(
				workflowPostHooks,
				RunPostHooksWorkflowInputPostHook{
					Name: postHookName,
				},
			)
		}
	}

	if len(postHooks) > 0 {
		// Fix base post hooks with parameters
		for key, existingPostHook := range workflowPostHooks {
			postHook, ok := postHooks[existingPostHook.Name]
			if ok {
				workflowPostHooks[key].Param = postHook

				delete(postHooks, existingPostHook.Name)
			}
		}

		var postHooksByPriority postHookFunctionByPriority

		for postHookName, param := range postHooks {
			postHook, ok := HookMap[postHookName]
			if !ok {
				log.Debugf("cannot find posthook function: %s", postHookName)

				continue
			}

			postHooksByPriority = append(
				postHooksByPriority,
				postHookFunctionSorter{
					Name:     postHookName,
					Param:    param,
					Priority: postHook.GetPriority(),
				},
			)
		}

		sort.Sort(postHooksByPriority)

		for _, postHookByPriority := range postHooksByPriority {
			workflowPostHooks = append(
				workflowPostHooks,
				RunPostHooksWorkflowInputPostHook{
					Name:  postHookByPriority.Name,
					Param: postHookByPriority.Param,
				},
			)
		}
	}

	return workflowPostHooks
}

type postHookFunctionSorter struct {
	Name     string
	Param    interface{}
	Priority int
}

type postHookFunctionByPriority []postHookFunctionSorter

func (p postHookFunctionByPriority) Len() int      { return len(p) }
func (p postHookFunctionByPriority) Swap(i, j int) { p[i], p[j] = p[j], p[i] }
func (p postHookFunctionByPriority) Less(i, j int) bool {
	return p[i].Priority < p[j].Priority
}
