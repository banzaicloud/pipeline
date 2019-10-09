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

package anchore

import (
	"context"
	"fmt"

	"emperror.dev/errors"
	securityV1Alpha "github.com/banzaicloud/anchore-image-validator/pkg/apis/security/v1alpha1"
	"github.com/banzaicloud/anchore-image-validator/pkg/clientset/v1alpha1"
	securityClientV1Alpha "github.com/banzaicloud/anchore-image-validator/pkg/clientset/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"

	"github.com/banzaicloud/pipeline/internal/common"
	"github.com/banzaicloud/pipeline/pkg/k8sclient"
	"github.com/banzaicloud/pipeline/pkg/security"
)

// SecurityResourceService gathers operations for managing security (anchore) related resources
type SecurityResourceService interface {
	WhitelistService
}

// WhitelistService whitelist management operations
type WhitelistService interface {
	GetWhitelists(ctx context.Context, cluster Cluster) ([]securityV1Alpha.WhiteListItem, error)
	CreateWhitelist(ctx context.Context, cluster Cluster, whitelistItem security.ReleaseWhiteListItem) (interface{}, error)
	DeleteWhitelist(ctx context.Context, cluster Cluster, whitelistItemName string) error
}

// PolicyService policy management operations
type PolicyService interface {
}

type securityResourceService struct {
	logger common.Logger
}

func NewSecurityResourceService(logger common.Logger) SecurityResourceService {
	_ = scheme.AddToScheme(scheme.Scheme)

	return securityResourceService{
		logger: logger,
	}
}

func (s securityResourceService) GetWhitelists(ctx context.Context, cluster Cluster) ([]securityV1Alpha.WhiteListItem, error) {
	logCtx := map[string]interface{}{"clusterID": cluster.GetID()}
	s.logger.Info("retrieving whitelist ...", logCtx)

	wlClient, err := s.getWhiteListsClient(ctx, cluster)
	if err != nil {
		s.logger.Debug("failed to create whitelist client", logCtx)

		return nil, err
	}

	whitelist, err := wlClient.List(metav1.ListOptions{})
	if err != nil {
		s.logger.Debug("failed to retrieve whitelist", logCtx)

		return nil, errors.WrapIf(err, "failed to retrieve current whitelist")
	}

	s.logger.Info("whitelist successfully retrieved", logCtx)
	return whitelist.Items, nil
}

func (s securityResourceService) CreateWhitelist(ctx context.Context, cluster Cluster, whitelistItem security.ReleaseWhiteListItem) (interface{}, error) {
	logCtx := map[string]interface{}{"clusterID": cluster.GetID(), "whiteListItem": whitelistItem.Name}
	s.logger.Info("creating whitelist item ...", logCtx)

	wlClient, err := s.getWhiteListsClient(ctx, cluster)
	if err != nil {
		s.logger.Debug("failed to create whitelist client", logCtx)

		return nil, err
	}

	wlItem, err := wlClient.Create(s.assembleWhiteListItem(whitelistItem))
	if err != nil {
		s.logger.Debug("failed to create whitelist item", logCtx)

		return nil, errors.WrapIf(err, "failed to create whitelist item")
	}

	s.logger.Info("whitelist item successfully created", logCtx)
	return wlItem, nil
}

func (s securityResourceService) DeleteWhitelist(ctx context.Context, cluster Cluster, whitelistItemName string) error {
	logCtx := map[string]interface{}{"clusterID": cluster.GetID(), "whiteListItem": whitelistItemName}
	s.logger.Info("creating whitelist item ...", logCtx)

	wlClient, err := s.getWhiteListsClient(ctx, cluster)
	if err != nil {
		s.logger.Debug("failed to create whitelist client", logCtx)

		return err
	}
	if err := wlClient.Delete(whitelistItemName, metav1.NewDeleteOptions(0)); err != nil {
		s.logger.Debug("failed to delete whitelist", logCtx)

		return errors.WrapIf(err, "failed to delete whitelist")
	}

	s.logger.Info("whitelist item successfully deleted", logCtx)
	return nil

}

func (s securityResourceService) getWhiteListsClient(ctx context.Context, cluster Cluster) (securityClientV1Alpha.WhiteListInterface, error) {

	kubeConfig, err := cluster.GetK8sConfig()
	if err != nil {
		return nil, errors.WrapIf(err, "failed to get k8s config for the cluster")
	}

	config, err := k8sclient.NewClientConfig(kubeConfig)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to create k8s client config")
	}

	securityClientSet, err := v1alpha1.SecurityConfig(config)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to create security config")
	}

	return securityClientSet.Whitelists(), nil
}

func (s securityResourceService) assembleWhiteListItem(whitelistItem security.ReleaseWhiteListItem) *securityV1Alpha.WhiteListItem {
	return &securityV1Alpha.WhiteListItem{
		TypeMeta: metav1.TypeMeta{
			Kind:       "WhiteListItem",
			APIVersion: fmt.Sprintf("%v/%v", securityV1Alpha.GroupName, securityV1Alpha.GroupVersion),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: whitelistItem.Name,
		},
		Spec: securityV1Alpha.WhiteListSpec{
			Creator: whitelistItem.Owner,
			Reason:  whitelistItem.Reason,
			Regexp:  whitelistItem.Regexp,
		},
	}
}

// Cluster defines operations that can be performed on a k8s cluster
type Cluster interface {
	GetK8sConfig() ([]byte, error)
	GetName() string
	GetOrganizationId() uint
	GetUID() string
	GetID() uint
	IsReady() (bool, error)
	NodePoolExists(nodePoolName string) bool
	RbacEnabled() bool
}
