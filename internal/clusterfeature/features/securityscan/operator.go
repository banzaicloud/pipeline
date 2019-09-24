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

package securityscan

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"emperror.dev/errors"
	securityV1Alpha "github.com/banzaicloud/anchore-image-validator/pkg/apis/security/v1alpha1"
	securityClientV1Alpha "github.com/banzaicloud/anchore-image-validator/pkg/clientset/v1alpha1"
	"github.com/banzaicloud/pipeline/auth"
	"github.com/banzaicloud/pipeline/cluster"
	"github.com/banzaicloud/pipeline/internal/clusterfeature"
	"github.com/banzaicloud/pipeline/internal/clusterfeature/clusterfeatureadapter"
	"github.com/banzaicloud/pipeline/internal/clusterfeature/features"
	"github.com/banzaicloud/pipeline/internal/common"
	"github.com/banzaicloud/pipeline/pkg/backoff"
	"github.com/banzaicloud/pipeline/pkg/k8sclient"
	"github.com/banzaicloud/pipeline/secret"
	"github.com/mitchellh/mapstructure"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	securityScanChartVersion = "0.4.0"
	// todo read this from the chart possibly
	imageValidatorVersion = "0.3.3"
	securityScanChartName = "banzaicloud-stable/anchore-policy-validator"
	securityScanNamespace = "pipeline-system"
	securityScanRelease   = "anchore"
)

type featureOperator struct {
	clusterGetter  clusterfeatureadapter.ClusterGetter
	clusterService clusterfeature.ClusterService
	helmService    features.HelmService
	secretStore    features.SecretStore
	anchoreService features.AnchoreService
	logger         common.Logger
}

func MakeFeatureOperator(
	clusterGetter clusterfeatureadapter.ClusterGetter,
	clusterService clusterfeature.ClusterService,
	helmService features.HelmService,
	secretStore features.SecretStore,
	logger common.Logger,

) featureOperator {
	return featureOperator{
		clusterGetter:  clusterGetter,
		clusterService: clusterService,
		helmService:    helmService,
		secretStore:    secretStore,
		anchoreService: features.NewAnchoreService(), //wired service
		logger:         logger,
	}
}

// Name returns the name of the feature
func (op featureOperator) Name() string {
	return FeatureName
}

func (op featureOperator) Apply(ctx context.Context, clusterID uint, spec clusterfeature.FeatureSpec) error {
	logger := op.logger.WithContext(ctx).WithFields(map[string]interface{}{"cluster": clusterID, "feature": FeatureName})
	logger.Info("start to apply feature")

	ctx, err := op.ensureOrgIDInContext(ctx, clusterID)
	if err != nil {
		return errors.WrapIf(err, "failed to apply feature")
	}

	if err := op.clusterService.CheckClusterReady(ctx, clusterID); err != nil {
		return errors.WrapIf(err, "failed to apply feature")
	}

	boundSpec, err := bindFeatureSpec(spec)
	if err != nil {
		return errors.WrapIf(err, "failed to apply feature")
	}

	var anchoreValues *AnchoreValues
	if boundSpec.CustomAnchore.Enabled {
		anchoreValues, err = op.getCustomAnchoreValues(ctx, boundSpec.CustomAnchore)
		if err != nil {
			return errors.WrapIf(err, "failed to get default anchore values")
		}
	} else {
		anchoreValues, err = op.getDefaultAnchoreValues(ctx, clusterID)
		if err != nil {
			return errors.WrapIf(err, "failed to get default anchore values")
		}
	}

	if err := op.setSecurityScan(ctx, clusterID, true); err != nil {
		return errors.WrapIf(err, "failed to set security scan flag on cluster")
	}

	values, err := op.processChartValues(ctx, clusterID, *anchoreValues)
	if err != nil {
		return errors.WrapIf(err, "failed to assemble chart values")
	}

	if err = op.helmService.ApplyDeployment(ctx, clusterID, securityScanNamespace, securityScanChartName, securityScanRelease,
		values, securityScanChartVersion); err != nil {
		return errors.WrapIf(err, "failed to deploy feature")
	}

	if len(boundSpec.ReleaseWhiteList) > 0 {
		if err = op.installWhiteList(ctx, clusterID, boundSpec.ReleaseWhiteList); err != nil {
			return errors.WrapIf(err, "failed to install release white list")
		}
	}

	if boundSpec.WebhookConfig.Enabled {
		if err = op.configureWebHook(ctx, clusterID, boundSpec.WebhookConfig); err != nil {
			return errors.WrapIf(err, "failed to configure webhook")
		}
	}
	return nil
}

func (op featureOperator) Deactivate(ctx context.Context, clusterID uint) error {
	ctx, err := op.ensureOrgIDInContext(ctx, clusterID)
	if err != nil {
		return errors.WrapIf(err, "failed to deactivate feature")
	}

	if err := op.clusterService.CheckClusterReady(ctx, clusterID); err != nil {
		return errors.WrapIf(err, "failed to deactivate feature")
	}

	cl, err := op.clusterGetter.GetClusterByIDOnly(ctx, clusterID)
	if err != nil {
		return errors.WrapIf(err, "failed to get cluster by ID")
	}

	if err = op.anchoreService.DeleteUser(ctx, cl.GetOrganizationId(), cl.GetUID()); err != nil {
		return errors.WrapIf(err, "failed to deactivate")
	}

	if err := op.helmService.DeleteDeployment(ctx, clusterID, securityScanRelease); err != nil {
		return errors.WrapIfWithDetails(err, "failed to uninstall feature", "feature", FeatureName,
			"clusterID", clusterID)
	}

	if err := op.setSecurityScan(ctx, clusterID, false); err != nil {
		return errors.WrapIf(err, "failed to set security scan flag to false")
	}

	return nil
}

func (op featureOperator) ensureOrgIDInContext(ctx context.Context, clusterID uint) (context.Context, error) {
	if _, ok := auth.GetCurrentOrganizationID(ctx); !ok {
		cl, err := op.clusterGetter.GetClusterByIDOnly(ctx, clusterID)
		if err != nil {
			return ctx, errors.WrapIf(err, "failed to get cluster by ID")
		}
		ctx = auth.SetCurrentOrganizationID(ctx, cl.GetOrganizationId())
	}
	return ctx, nil
}

func (op featureOperator) createAnchoreUserForCluster(ctx context.Context, clusterID uint) (string, error) {
	cl, err := op.clusterGetter.GetClusterByIDOnly(ctx, clusterID)
	if err != nil {
		return "", errors.WrapIf(err, "error retrieving cluster")
	}

	usr, err := op.anchoreService.GenerateUser(ctx, cl.GetOrganizationId(), cl.GetUID())
	if err != nil {
		return "", errors.WrapIf(err, "error creating anchore user")
	}

	return usr, nil
}

func (op featureOperator) getDefaultValues(ctx context.Context, clusterID uint) (*SecurityScanChartValues, error) {

	cl, err := op.clusterGetter.GetClusterByIDOnly(ctx, clusterID)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to get cluster")
	}

	return getDefaultValues(cl), nil
}

func getDefaultValues(cl clusterfeatureadapter.Cluster) *SecurityScanChartValues {
	chartValues := new(SecurityScanChartValues)

	if headNodeAffinity := cluster.GetHeadNodeAffinity(cl); headNodeAffinity != (corev1.Affinity{}) {
		chartValues.Affinity = &headNodeAffinity
	}

	chartValues.Tolerations = cluster.GetHeadNodeTolerations()

	return chartValues
}

func (op featureOperator) processChartValues(ctx context.Context, clusterID uint, anchoreValues AnchoreValues) ([]byte, error) {
	securityScanValues, err := op.getDefaultValues(ctx, clusterID)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to get defaults for chart values")
	}

	securityScanValues.Anchore = anchoreValues

	values, err := json.Marshal(securityScanValues)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to marshal chart values")
	}

	return values, nil
}

func (op featureOperator) getCustomAnchoreValues(ctx context.Context, customAnchore anchoreSpec) (*AnchoreValues, error) {
	if !customAnchore.Enabled { // this is already checked
		return nil, errors.NewWithDetails("custom anchore disabled")
	}

	anchoreUserSecret, err := op.secretStore.GetSecretValues(ctx, customAnchore.SecretID)
	if err != nil {
		return nil, errors.WrapWithDetails(err, "failed to get anchore secret", "secretId", customAnchore.SecretID)
	}

	var anchoreValues AnchoreValues
	if err := mapstructure.Decode(anchoreUserSecret, &anchoreValues); err != nil {
		return nil, errors.WrapIf(err, "failed to extract anchore secret values")
	}

	anchoreValues.Host = customAnchore.Url

	return &anchoreValues, nil

}

func (op featureOperator) getDefaultAnchoreValues(ctx context.Context, clusterID uint) (*AnchoreValues, error) {
	// default (pipeline hosted) anchore
	if !op.anchoreService.AnchoreConfig().AnchoreEnabled {
		return nil, errors.NewWithDetails("default anchore is not enabled")
	}

	secretName, err := op.createAnchoreUserForCluster(ctx, clusterID)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to create anchore user")
	}

	anchoreSecretID := secret.GenerateSecretIDFromName(secretName)
	anchoreUserSecret, err := op.secretStore.GetSecretValues(ctx, anchoreSecretID)
	if err != nil {
		return nil, errors.WrapWithDetails(err, "failed to get anchore secret", "secretId", anchoreSecretID)
	}

	var anchoreValues AnchoreValues
	if err := mapstructure.Decode(anchoreUserSecret, &anchoreValues); err != nil {
		return nil, errors.WrapIf(err, "failed to extract anchore secret values")
	}

	anchoreValues.Host = op.anchoreService.AnchoreConfig().AnchoreEndpoint

	return &anchoreValues, nil
}

func (op featureOperator) installWhiteList(ctx context.Context, clusterID uint, releases []releaseSpec) error {

	cl, err := op.clusterGetter.GetClusterByIDOnly(ctx, clusterID)
	if err != nil {
		return errors.WrapIf(err, "failed to get cluster")
	}

	kubeConfig, err := cl.GetK8sConfig()
	if err != nil {
		return errors.WrapIf(err, "failed to get k8s config for the cluster")
	}

	config, err := k8sclient.NewClientConfig(kubeConfig)
	if err != nil {
		return errors.WrapIf(err, "failed to create k8s client config")
	}

	securityClientSet, err := securityClientV1Alpha.SecurityConfig(config)
	if err != nil {
		return errors.WrapIf(err, "failed to create security config")
	}

	for _, relSpec := range releases {

		whitelist := securityV1Alpha.WhiteListItem{
			TypeMeta: metav1.TypeMeta{
				Kind:       "WhiteListItem",
				APIVersion: fmt.Sprintf("%v/%v", securityV1Alpha.GroupName, securityV1Alpha.GroupVersion),
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: relSpec.Name,
			},
			Spec: securityV1Alpha.WhiteListSpec{
				Creator: "pipeline",
				Reason:  relSpec.Reason,
				Regexp:  relSpec.Regexp,
			},
		}

		// it may take some time until the WhiteListItem CRD is created, thus the first attempt to create
		// a whitelist cr may fail. Retry the whitelist creation in case of failure
		var backoffConfig = backoff.ConstantBackoffConfig{
			Delay:      time.Duration(5) * time.Second,
			MaxRetries: 3,
		}
		var backoffPolicy = backoff.NewConstantBackoffPolicy(backoffConfig)

		err = backoff.Retry(func() error {
			if _, err = securityClientSet.Whitelists().Create(&whitelist); err != nil {
				return errors.WrapIf(err, "failed to create whitelist item")
			}

			return nil

		}, backoffPolicy)

		return err
	}
	return nil
}

// setSecurityScan temporary workaround for signaling the security scan enablement
func (op *featureOperator) setSecurityScan(ctx context.Context, clusterID uint, enabled bool) error {
	cl, err := op.clusterGetter.GetClusterByIDOnly(ctx, clusterID)
	if err != nil {
		return errors.WrapIf(err, "failed to get cluster")
	}

	cl.SetSecurityScan(enabled)

	return nil
}

func (op *featureOperator) configureWebHook(ctx context.Context, clusterID uint, whConfig webHookConfigSpec) error {

	const labelKey = "scan"
	var combinedError error

	labelValue := "noscan"
	if whConfig.Selector == "include" {
		labelValue = "scan"
	}

	cl, err := op.clusterGetter.GetClusterByIDOnly(ctx, clusterID)
	if err != nil {
		return errors.WrapIf(err, "failed to get cluster")
	}

	kubeConfig, err := cl.GetK8sConfig()
	if err != nil {
		return errors.WrapIf(err, "failed to get k8s config for the cluster")
	}

	cli, err := k8sclient.NewClientFromKubeConfig(kubeConfig)
	if err != nil {
		return errors.WrapIf(err, "failed to create k8s client")
	}

	for _, namespace := range whConfig.Namespaces {
		// get the namespace
		ns, err := cli.CoreV1().Namespaces().Get(namespace, metav1.GetOptions{})
		if err != nil {
			combinedError = errors.Append(combinedError, errors.WrapIff(err, "failed to retrieve namespace: %s", namespace))
			continue
		}

		// merge ns labels
		labels := ns.GetLabels()
		if labels == nil {
			labels = make(map[string]string)
		}

		labels[labelKey] = labelValue
		ns.SetLabels(labels)

		if _, err = cli.CoreV1().Namespaces().Update(ns); err != nil {
			combinedError = errors.Append(combinedError, errors.WrapIff(err, "failed to label namespace: %s", namespace))
		}
	}

	return combinedError
}
