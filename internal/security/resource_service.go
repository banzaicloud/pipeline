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

	"emperror.dev/errors"
	"github.com/banzaicloud/anchore-image-validator/pkg/clientset/v1alpha1"
	"k8s.io/client-go/kubernetes/scheme"

	"github.com/banzaicloud/pipeline/internal/clusterfeature/clusterfeatureadapter"
	"github.com/banzaicloud/pipeline/internal/common"
	"github.com/banzaicloud/pipeline/pkg/k8sclient"
	"github.com/banzaicloud/pipeline/pkg/security"

	securityClientV1Alpha "github.com/banzaicloud/anchore-image-validator/pkg/clientset/v1alpha1"
)

type SecurityResourceService interface {
	WhitelistService
}
type WhitelistService interface {
	GetWhitelists(ctx context.Context, clusterID uint) (interface{}, error)
	CreateWhitelist(ctx context.Context, clusterID uint, whitelistItem security.WhitelistItem) (interface{}, error)
	DeleteWhitelist(ctx context.Context, clusterID uint, whitelistItemName string) (interface{}, error)
}

type PolicyService interface {
}

type securityResourceService struct {
	clusterGetter clusterfeatureadapter.ClusterGetter
	logger        common.Logger
}

func NewSecurityResourceService(clusterGetter clusterfeatureadapter.ClusterGetter, logger common.Logger) SecurityResourceService {
	_ = scheme.AddToScheme(scheme.Scheme)
	return securityResourceService{
		clusterGetter: clusterGetter,
		logger:        logger,
	}
}

func (s securityResourceService) GetWhitelists(ctx context.Context, clusterID uint) (interface{}, error) {
	panic("implement me")
}

func (s securityResourceService) CreateWhitelist(ctx context.Context, clusterID uint, whitelistItem security.WhitelistItem) (interface{}, error) {
	panic("implement me")
}

func (s securityResourceService) DeleteWhitelist(ctx context.Context, clusterID uint, whitelistItemName string) (interface{}, error) {
	panic("implement me")
}

func (wls securityResourceService) getWhiteListsClient(ctx context.Context, clusterID uint) (securityClientV1Alpha.WhiteListInterface, error) {
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

	securityClientSet, err := v1alpha1.SecurityConfig(config)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to create security config")
	}

	return securityClientSet.Whitelists(), nil
}
