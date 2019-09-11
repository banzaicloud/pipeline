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

	"github.com/banzaicloud/pipeline/internal/clusterfeature/clusterfeatureadapter"
)

type arr = []interface{}
type obj = map[string]interface{}

type clusterGetterMock struct {
}

func (c clusterGetterMock) GetClusterByIDOnly(ctx context.Context, clusterID uint) (clusterfeatureadapter.Cluster, error) {
	panic("implement me")
}

type clusterServiceMock struct {
}

func (c clusterServiceMock) CheckClusterReady(ctx context.Context, clusterID uint) error {
	panic("implement me")
}

type helmServiceMock struct {
}

func (h helmServiceMock) ApplyDeployment(
	ctx context.Context,
	clusterID uint,
	namespace string,
	deploymentName string,
	releaseName string,
	values []byte,
	chartVersion string,
) error {
	panic("implement me")
}

func (h helmServiceMock) DeleteDeployment(ctx context.Context, clusterID uint, releaseName string) error {
	panic("implement me")
}

type secretStoreMock struct {
}

func (s secretStoreMock) GetSecretValues(ctx context.Context, secretID string) (map[string]string, error) {
	panic("implement me")
}
