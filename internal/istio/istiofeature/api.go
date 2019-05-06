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

package istiofeature

import (
	"github.com/goph/emperror"
	"github.com/sirupsen/logrus"

	"github.com/banzaicloud/pipeline/cluster"
	"github.com/banzaicloud/pipeline/internal/clustergroup/api"
)

type Reconciler func(desiredState DesiredState) error
type ReconcilerWithCluster func(desiredState DesiredState, c cluster.CommonCluster) error

type DesiredState string

const (
	DesiredStatePresent DesiredState = "present"
	DesiredStateAbsent  DesiredState = "absent"
)

type Config struct {
	// MasterClusterID contains the cluster ID where the control plane and the operator runs
	MasterClusterID uint `json:"masterClusterID"`
	// AutoSidecarInjectNamespaces list of namespaces that will be labelled with istio-injection=enabled
	AutoSidecarInjectNamespaces []string `json:"autoSidecarInjectNamespaces,omitempty"`
	// BypassEgressTraffic prevents Envoy sidecars from intercepting external requests
	BypassEgressTraffic bool `json:"bypassEgressTraffic,omitempty"`
	// EnableMTLS signals if mutual TLS is enabled in the service mesh
	EnableMTLS bool `json:"enableMTLS,omitempty"`

	name         string
	enabled      bool
	clusterGroup api.ClusterGroup
}

type MeshReconciler struct {
	Configuration Config
	Master        cluster.CommonCluster
	Remotes       []cluster.CommonCluster

	clusterGetter api.ClusterGetter
	logger        logrus.FieldLogger
	errorHandler  emperror.Handler
}
