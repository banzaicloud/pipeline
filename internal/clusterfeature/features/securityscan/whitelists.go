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
	"fmt"
	"time"

	"emperror.dev/errors"
	"github.com/banzaicloud/anchore-image-validator/pkg/apis/security/v1alpha1"
	securityV1Alpha "github.com/banzaicloud/anchore-image-validator/pkg/apis/security/v1alpha1"
	securityClientV1Alpha "github.com/banzaicloud/anchore-image-validator/pkg/clientset/v1alpha1"
	"github.com/banzaicloud/pipeline/internal/clusterfeature/clusterfeatureadapter"
	"github.com/banzaicloud/pipeline/internal/common"
	"github.com/banzaicloud/pipeline/pkg/backoff"
	"github.com/banzaicloud/pipeline/pkg/k8sclient"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
)

// WhiteListService handles whitelist creation and removal
type WhiteListService interface {
	// EnsureReleaseWhiteList makes sure that the passed whitelist is applied to the cluster
	EnsureReleaseWhiteList(ctx context.Context, clusterID uint, items []releaseSpec) error
}

type whiteListService struct {
	clusterGetter clusterfeatureadapter.ClusterGetter
	logger        common.Logger
}

func NewWhiteListService(clusterGetter clusterfeatureadapter.ClusterGetter, logger common.Logger) WhiteListService {
	svc := new(whiteListService)
	svc.clusterGetter = clusterGetter
	svc.logger = logger
	return svc
}

func (wls *whiteListService) getWhiteListsClient(ctx context.Context, clusterID uint) (securityClientV1Alpha.WhiteListInterface, error) {
	cl, err := wls.clusterGetter.GetClusterByIDOnly(ctx, clusterID)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to get cluster")
	}

	kubeConfig, err := cl.GetK8sConfig()
	if err != nil {
		return nil, errors.WrapIf(err, "failed to get k8s config for the cluster")
	}

	config, err := k8sclient.NewClientConfig(kubeConfig)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to create k8s client config")
	}

	securityClientSet, err := securityClientV1Alpha.SecurityConfig(config)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to create security config")
	}

	return securityClientSet.Whitelists(), nil
}

func (wls *whiteListService) EnsureReleaseWhiteList(ctx context.Context, clusterID uint, items []releaseSpec) error {
	// todo should this be here?
	_ = v1alpha1.AddToScheme(scheme.Scheme)

	wlClient, err := wls.getWhiteListsClient(ctx, clusterID)
	if err != nil {
		return errors.WrapIf(err, "failed to get security client")
	}

	installedItems, err := wlClient.List(metav1.ListOptions{})
	if err != nil {
		return errors.WrapIf(err, "failed to retrieve current whitelist")
	}

	installedItemsMap := make(map[string]v1alpha1.WhiteListItem)
	for _, installedItem := range installedItems.Items {
		installedItemsMap[installedItem.Name] = installedItem
	}

	toBeAdded := make([]releaseSpec, 0)

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
	toBeRemoved := make([]string, 0)
	for itemName := range installedItemsMap {
		toBeRemoved = append(toBeRemoved, itemName)
	}

	if err := wls.runWithBackoff(func() error { return wls.removeItems(ctx, wlClient, toBeRemoved) }); err != nil {
		return errors.WrapIf(err, "failed to remove whitelist items")
	}

	if err := wls.runWithBackoff(func() error { return wls.installItems(ctx, wlClient, toBeAdded) }); err != nil {
		return errors.WrapIf(err, "failed to install whitelist items")
	}

	return nil
}

func (wls *whiteListService) removeItems(ctx context.Context, whitelistCli securityClientV1Alpha.WhiteListInterface, itemNames []string) error {
	var collectedErrors error
	for _, itemName := range itemNames {
		if err := whitelistCli.Delete(itemName, &metav1.DeleteOptions{}); err != nil {
			collectedErrors = errors.Append(collectedErrors, errors.WrapIff(err, "failed to remove whitelist item %s", itemName))
			continue
		}
	}
	return collectedErrors
}

func (wls *whiteListService) installItems(ctx context.Context, whitelistCli securityClientV1Alpha.WhiteListInterface, items []releaseSpec) error {
	var collectedErrors error
	for _, item := range items {
		if _, err := whitelistCli.Create(wls.assembleWhiteLisItem(item)); err != nil {
			collectedErrors = errors.Append(collectedErrors, errors.WrapIff(err, "failed to azdd whitelist item %s", item.Name))
			continue
		}
	}
	return collectedErrors
}

func (wls *whiteListService) assembleWhiteLisItem(releaseItem releaseSpec) *securityV1Alpha.WhiteListItem {
	return &securityV1Alpha.WhiteListItem{
		TypeMeta: metav1.TypeMeta{
			Kind:       "WhiteListItem",
			APIVersion: fmt.Sprintf("%v/%v", securityV1Alpha.GroupName, securityV1Alpha.GroupVersion),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: releaseItem.Name,
		},
		Spec: securityV1Alpha.WhiteListSpec{
			Creator: "pipeline",
			Reason:  releaseItem.Reason,
			Regexp:  releaseItem.Regexp,
		},
	}
}

func (wls *whiteListService) runWithBackoff(f func() error) error {
	// it may take some time until the WhiteListItem CRD is created, thus the first attempt to create
	// a whitelist cr may fail. Retry the whitelist creation in case of failure
	var backoffConfig = backoff.ConstantBackoffConfig{
		Delay:      time.Duration(5) * time.Second,
		MaxRetries: 3,
	}
	var backoffPolicy = backoff.NewConstantBackoffPolicy(backoffConfig)

	return backoff.Retry(f, backoffPolicy)

}
