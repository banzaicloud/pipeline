// Copyright © 2020 Banzai Cloud
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

package ingress

import "context"

// ServiceName is the unique name of the integrated service
const ServiceName = "ingress"

const (
	ControllerTraefik = "traefik"
)

const (
	ServiceTypeClusterIP    = "ClusterIP"
	ServiceTypeLoadBalancer = "LoadBalancer"
	ServiceTypeNodePort     = "NodePort"
)

type OperatorClusterStore interface {
	Get(ctx context.Context, clusterID uint) (OperatorCluster, error)
}

type OperatorCluster struct {
	OrganizationID uint
	Cloud          string
}

type OrgDomainService interface {
	GetOrgDomain(ctx context.Context, orgID uint) (OrgDomain, error)
}

type OrgDomain struct {
	Name         string
	WildcardName string
}
