// Copyright © 2019 Banzai Cloud
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

package monitoring

import (
	"context"
	"fmt"

	"emperror.dev/errors"

	pkgCluster "github.com/banzaicloud/pipeline/cluster"
	"github.com/banzaicloud/pipeline/internal/clusterfeature/clusterfeatureadapter"
	"github.com/banzaicloud/pipeline/internal/common"
	"github.com/banzaicloud/pipeline/internal/secret/secrettype"
	"github.com/banzaicloud/pipeline/secret"
)

type secretManager struct {
	operator FeatureOperator
	cluster  clusterfeatureadapter.Cluster
	tags     []string
	infoer   secretComponentInfoer
}

type secretComponentInfoer interface {
	name() string
	generatedSecretName() string
}

func (m secretManager) generateHTPasswordSecret(ctx context.Context) error {
	clusterNameSecretTag := getClusterNameSecretTag(m.cluster.GetName())
	clusterUidSecretTag := getClusterUIDSecretTag(m.cluster.GetUID())
	releaseSecretTag := getReleaseSecretTag()

	var secretTags = []string{
		clusterNameSecretTag,
		clusterUidSecretTag,
		releaseSecretTag,
		secret.TagBanzaiReadonly,
		featureSecretTag,
	}

	secretTags = append(secretTags, m.tags...)

	adminPass, err := secret.RandomString("randAlphaNum", 12)
	if err != nil {
		return errors.WrapIf(err, fmt.Sprintf("%s password generation failed", m.infoer.name()))
	}

	secretRequest := &secret.CreateSecretRequest{
		Name: m.infoer.generatedSecretName(),
		Type: secrettype.HtpasswdSecretType,
		Values: map[string]string{
			secrettype.Username: generatedSecretUsername,
			secrettype.Password: adminPass,
		},
		Tags: secretTags,
	}
	_, err = secret.Store.CreateOrUpdate(m.cluster.GetOrganizationId(), secretRequest)
	if err != nil {
		return errors.WrapIf(err, fmt.Sprintf("failed to store %s secret", m.infoer.name()))
	}

	return nil
}

func (m secretManager) getComponentSecret(
	ctx context.Context,
	ingress ingressSpecWithSecret,
	logger common.Logger,
) (string, error) {
	var secretName string
	if ingress.SecretId == "" {
		// get secret by name, this necessary in case of feature update
		var secretName = m.infoer.generatedSecretName()
		existingSecretID, err := m.operator.secretStore.GetIDByName(ctx, secretName)
		if existingSecretID != "" {
			logger.Debug(fmt.Sprintf("%s secret already exists", m.infoer.name()))
			return secretName, nil
		} else if isSecretNotFoundError(err) {
			// generate and store secret
			err = m.generateHTPasswordSecret(ctx)
			if err != nil {
				return "", errors.WrapIf(err, fmt.Sprintf("failed to generate %s secret", m.infoer.name()))
			}
		} else {
			return "", errors.WrapIf(err, fmt.Sprintf("error during getting %s secret", m.infoer.name()))
		}
	} else {
		var err error
		secretName, err = m.operator.secretStore.GetNameByID(ctx, ingress.SecretId)
		if err != nil {
			return "", errors.WrapIfWithDetails(err, "failed to get secret",
				"secretID", ingress.SecretId, "component", m.infoer.name())
		}
	}
	return secretName, nil
}

func (m secretManager) installSecret(ctx context.Context, clusterID uint, secretName string) error {
	pipelineSystemNamespace := m.operator.config.pipelineSystemNamespace

	installSecretRequest := pkgCluster.InstallSecretRequest{
		SourceSecretName: secretName,
		Namespace:        pipelineSystemNamespace,
		Spec: map[string]pkgCluster.InstallSecretRequestSpecItem{
			"auth": {Source: secrettype.HtpasswdFile},
		},
		Update: true,
	}

	if _, err := m.operator.installSecret(ctx, clusterID, secretName, installSecretRequest); err != nil {
		return errors.WrapIfWithDetails(err, fmt.Sprintf("failed to install %s secret to cluster", m.infoer.name()), "clusterID", clusterID)
	}

	return nil
}

func generateAndInstallSecret(
	ctx context.Context,
	ingressSpec ingressSpecWithSecret,
	manager secretManager,
	logger common.Logger,
) (string, error) {
	var secretName string
	var err error
	if ingressSpec.Enabled {
		// get secret from spec or generate
		secretName, err = manager.getComponentSecret(ctx, ingressSpec, logger)
		if err != nil {
			return "", errors.WrapIfWithDetails(err, "failed to get secret",
				"component", manager.infoer.name())
		}

		// install secret
		if err := manager.installSecret(ctx, manager.cluster.GetID(), secretName); err != nil {
			return "", errors.WrapIfWithDetails(err, "failed to install secret to cluster",
				"clusterID", manager.cluster.GetID(), "component", manager.infoer.name())
		}
	}
	return secretName, err
}
