// Copyright © 2017 The Kubicorn Authors
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

package defaults

import (
	"github.com/banzaicloud/kubicorn/apis/cluster"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func NewClusterDefaults(base *cluster.Cluster) *cluster.Cluster {
	new := &cluster.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: base.ObjectMeta.Annotations,
		},
		Name:          base.Name,
		CloudId:       base.CloudId,
		Cloud:         base.Cloud,
		Location:      base.Location,
		Network:       base.Network,
		SSH:           base.SSH,
		Values:        base.Values,
		KubernetesAPI: base.KubernetesAPI,
		ServerPools:   base.ServerPools,
		Project:       base.Project,
		Components:    base.Components,
	}
	return new
}
