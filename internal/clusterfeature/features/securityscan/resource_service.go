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
	"time"

	"emperror.dev/errors"
	"github.com/banzaicloud/anchore-image-validator/pkg/apis/security/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"

	"github.com/banzaicloud/pipeline/internal/clusterfeature/clusterfeatureadapter"
	"github.com/banzaicloud/pipeline/internal/common"
	anchore "github.com/banzaicloud/pipeline/internal/security"
	"github.com/banzaicloud/pipeline/pkg/backoff"
	"github.com/banzaicloud/pipeline/pkg/k8sclient"
	"github.com/banzaicloud/pipeline/pkg/security"
)

// FeatureWhiteListService handles whitelist creation and removal
type FeatureWhiteListService interface {
	// EnsureReleaseWhiteList makes sure that the passed whitelist is applied to the cluster
	EnsureReleaseWhiteList(ctx context.Context, clusterID uint, items []releaseSpec) error
}

type featureWhiteListService struct {
	clusterGetter    clusterfeatureadapter.ClusterGetter
	whiteListService anchore.WhitelistService
	logger           common.Logger
}

func NewFeatureWhitelistService(clusterGetter clusterfeatureadapter.ClusterGetter, whiteListService anchore.WhitelistService, logger common.Logger) FeatureWhiteListService {
	_ = v1alpha1.AddToScheme(scheme.Scheme)

	return &featureWhiteListService{
		clusterGetter:    clusterGetter,
		whiteListService: whiteListService,
		logger:           logger,
	}
}

func (wls *featureWhiteListService) EnsureReleaseWhiteList(ctx context.Context, clusterID uint, items []releaseSpec) error {
	logCtx := map[string]interface{}{"clusterID": clusterID}
	cluster, err := wls.clusterGetter.GetClusterByIDOnly(ctx, clusterID)
	if err != nil {
		wls.logger.Debug("failed to get the cluster", logCtx)

		return errors.WrapIf(err, "failed to get cluster")
	}

	installedItems, err := wls.whiteListService.GetWhitelists(ctx, cluster)
	if err != nil {
		wls.logger.Debug("failed to retrieve whitelist", logCtx)

		return errors.WrapIf(err, "failed to retrieve whitelist")
	}

	installedItemsMap := make(map[string]v1alpha1.WhiteListItem)
	for _, installedItem := range installedItems {
		installedItemsMap[installedItem.Name] = installedItem
	}

	var toBeAdded []releaseSpec

	// find items to be installed
	for _, releaseItem := range items {
		installed, ok := installedItemsMap[releaseItem.Name]
		if !ok {
			// the release is not installed
			toBeAdded = append(toBeAdded, releaseItem)
			continue
		}
		// remove the existing releas from the map
		delete(installedItemsMap, installed.Name)
	}

	// items to be removed are left in the map at this point
	toBeRemoved := make([]string, 0, len(installedItemsMap))
	for itemName := range installedItemsMap {
		toBeRemoved = append(toBeRemoved, itemName)
	}

	if err := wls.runWithBackoff(func() error { return wls.removeItems(ctx, cluster, toBeRemoved) }); err != nil {
		return errors.WrapIf(err, "failed to remove whitelist items")
	}

	if err := wls.runWithBackoff(func() error { return wls.installItems(ctx, cluster, toBeAdded) }); err != nil {
		return errors.WrapIf(err, "failed to install whitelist items")
	}

	return nil
}

func (wls *featureWhiteListService) removeItems(ctx context.Context, cluster clusterfeatureadapter.Cluster, itemNames []string) error {
	var collectedErrors error
	for _, itemName := range itemNames {
		if err := wls.whiteListService.DeleteWhitelist(ctx, cluster, itemName); err != nil {
			collectedErrors = errors.Append(collectedErrors, errors.WrapIff(err, "failed to remove whitelist item %s", itemName))
		}
	}
	return collectedErrors
}

func (wls *featureWhiteListService) installItems(ctx context.Context, cluster clusterfeatureadapter.Cluster, items []releaseSpec) error {
	var collectedErrors error
	for _, item := range items {
		wlItem := security.ReleaseWhiteListItem{
			Name:   item.Name,
			Owner:  "pipeline",
			Reason: item.Reason,
			Regexp: item.Regexp,
		}

		if _, err := wls.whiteListService.CreateWhitelist(ctx, cluster, wlItem); err != nil {
			collectedErrors = errors.Append(collectedErrors, errors.WrapIff(err, "failed to add whitelist item %s", item.Name))
		}
	}
	return collectedErrors
}

func (wls *featureWhiteListService) runWithBackoff(f func() error) error {
	// it may take some time until the WhiteListItem CRD is created, thus the first attempt to create
	// a whitelist cr may fail. Retry the whitelist creation in case of failure
	var backoffConfig = backoff.ConstantBackoffConfig{
		Delay:      5 * time.Second,
		MaxRetries: 3,
	}
	var backoffPolicy = backoff.NewConstantBackoffPolicy(backoffConfig)

	return backoff.Retry(f, backoffPolicy)
}

type NamespaceService interface {
	// LabelNamespaces add the passed map of labels to the slice of namespaces
	LabelNamespaces(ctx context.Context, clusterID uint, namespaces []string, labels map[string]string) error

	// RemoveLabels removes the labels from the slice of namespaces
	RemoveLabels(ctx context.Context, clusterID uint, namespaces []string, labels []string) error
}

type namespaceService struct {
	clusterGetter clusterfeatureadapter.ClusterGetter
	logger        common.Logger
}

func NewNamespacesService(getter clusterfeatureadapter.ClusterGetter, log common.Logger) NamespaceService {
	return &namespaceService{
		clusterGetter: getter,
		logger:        log,
	}
}

func (nss *namespaceService) LabelNamespaces(ctx context.Context, clusterID uint, namespaces []string, newLabels map[string]string) error {

	namespacesCli, err := nss.getNamespacesCli(ctx, clusterID)
	if err != nil {
		return errors.WrapIf(err, "failed to get namespaces client")
	}

	var combinedErr error
	for _, namespace := range namespaces {

		nss.logger.Debug("label namespace", map[string]interface{}{"namespace": namespace})
		ns, err := namespacesCli.Get(namespace, metav1.GetOptions{})
		if err != nil {
			nss.logger.Debug("failed to retrieve namespace", map[string]interface{}{"namespace": namespace})
			// todo should we report error if an invalid namespace is passed in? if so uncomment the line below
			//combinedErr = errors.Append(combinedErr, errors.WrapIff(err, "failed to get namespace %s", ns))
			continue
		}

		// merge ns labels
		freshLabels := nss.mergeLabels(ns.GetLabels(), newLabels)

		// update
		ns.SetLabels(freshLabels)
		ns, err = namespacesCli.Update(ns)
		if err != nil {
			nss.logger.Debug("failed to label namespace", map[string]interface{}{"namespace": namespace, "labels": freshLabels})
			combinedErr = errors.Append(combinedErr, errors.WrapIff(err, "failed to get namespace %s", ns))
		}
		nss.logger.Debug("namespace labeled", map[string]interface{}{"namespace": namespace, "labels": freshLabels})
	}

	return combinedErr
}

func (nss *namespaceService) mergeLabels(currentLabels map[string]string, newLabels map[string]string) map[string]string {
	mergedLabels := currentLabels
	if mergedLabels == nil {
		mergedLabels = make(map[string]string)
	}

	for lName, lValue := range newLabels {
		mergedLabels[lName] = lValue
	}

	return mergedLabels
}

func (nss *namespaceService) RemoveLabels(ctx context.Context, clusterID uint, namespaces []string, labels []string) error {
	nss.logger.Info("remove labels from namespaces", map[string]interface{}{"namespaces": namespaces, "labels": labels})
	var combinedErr error
	namespacesCli, err := nss.getNamespacesCli(ctx, clusterID)
	if err != nil {
		return errors.WrapIf(err, "failed to get namespaces client")
	}

	for _, namespace := range namespaces {
		nss.logger.Debug("remove labels from namespace", map[string]interface{}{"namespace": namespace})
		ns, err := namespacesCli.Get(namespace, metav1.GetOptions{})
		if err != nil {
			// record error, step forward
			nss.logger.Debug("failed to get namespace", map[string]interface{}{"namespace": namespace})
			combinedErr = errors.Append(combinedErr, errors.WrapIff(err, "failed to get namespace %s", ns))
			continue
		}

		freshLabels := ns.GetLabels()
		for _, labelName := range labels {
			delete(freshLabels, labelName)
		}

		ns.SetLabels(freshLabels)
		ns, err = namespacesCli.Update(ns)
		if err != nil {
			nss.logger.Debug("failed to remove labels form namespace", map[string]interface{}{"namespace": namespace, "labels": freshLabels})
			combinedErr = errors.Append(combinedErr, errors.WrapIff(err, "failed to get namespace %s", ns))
		}
		nss.logger.Debug("namespace labeled", map[string]interface{}{"namespace": namespace, "labels": freshLabels})
	}
	nss.logger.Info("removed labels from namespaces", map[string]interface{}{"namespaces": namespaces, "labels": labels})
	return combinedErr
}

func (nss *namespaceService) getNamespacesCli(ctx context.Context, clusterID uint) (v1.NamespaceInterface, error) {
	cl, err := nss.clusterGetter.GetClusterByIDOnly(ctx, clusterID)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to get cluster")
	}

	kubeConfig, err := cl.GetK8sConfig()
	if err != nil {
		return nil, errors.WrapIf(err, "failed to get k8s config for the cluster")
	}

	cli, err := k8sclient.NewClientFromKubeConfig(kubeConfig)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to create k8s client")
	}

	return cli.CoreV1().Namespaces(), nil
}
