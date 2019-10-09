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
	securityClientV1Alpha "github.com/banzaicloud/anchore-image-validator/pkg/clientset/v1alpha1"
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
	GetWhitelists(ctx context.Context, cluster Cluster) (interface{}, error)
	CreateWhitelist(ctx context.Context, cluster Cluster, whitelistItem security.ReleaseWhiteListItem) (interface{}, error)
	DeleteWhitelist(ctx context.Context, cluster Cluster, whitelistItemName string) (interface{}, error)
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

func (s securityResourceService) GetWhitelists(ctx context.Context, cluster Cluster) (interface{}, error) {
	panic("implement me")
}

func (s securityResourceService) CreateWhitelist(ctx context.Context, cluster Cluster, whitelistItem security.ReleaseWhiteListItem) (interface{}, error) {
	panic("implement me")
}

func (s securityResourceService) DeleteWhitelist(ctx context.Context, cluster Cluster, whitelistItemName string) (interface{}, error) {
	panic("implement me")
}

func (wls securityResourceService) getWhiteListsClient(ctx context.Context, cluster Cluster) (securityClientV1Alpha.WhiteListInterface, error) {

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
