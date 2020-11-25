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

package services

import (
	"github.com/banzaicloud/integrated-service-sdk/api/v1alpha1"

	"github.com/banzaicloud/pipeline/internal/integratedservices"
)

// ServiceNameMapper maps integrated service names that are different in external / legacy systems
type ServiceNameMapper interface {
	MapServiceName(serviceName string) string
}

type svcNameMapper struct {
	mappings map[string]string
}

func NewServiceNameMapper() ServiceNameMapper {
	return svcNameMapper{
		map[string]string{
			"dns":          "external-dns",
			"external-dns": "dns",
		},
	}
}

func (s svcNameMapper) MapServiceName(serviceName string) string {
	if mapped, ok := s.mappings[serviceName]; ok {
		return mapped
	}

	return serviceName
}

type StatusMapper interface {
	MapStatus(serviceInstance v1alpha1.ServiceInstance) integratedservices.IntegratedServiceStatus
}

type svcStatusMapper struct {
	mappings map[v1alpha1.Phase]integratedservices.IntegratedServiceStatus
}

func NewServiceStatusMapper() StatusMapper {
	return svcStatusMapper{
		mappings: map[v1alpha1.Phase]integratedservices.IntegratedServiceStatus{
			// changing statuses
			v1alpha1.PreInstalling:     integratedservices.IntegratedServiceStatusPending,
			v1alpha1.PreInstallFailed:  integratedservices.IntegratedServiceStatusPending,
			v1alpha1.PreInstallSuccess: integratedservices.IntegratedServiceStatusPending,
			v1alpha1.Installing:        integratedservices.IntegratedServiceStatusPending,
			v1alpha1.InstallSuccess:    integratedservices.IntegratedServiceStatusPending,
			v1alpha1.PostInstall:       integratedservices.IntegratedServiceStatusPending,
			v1alpha1.PostInstallFailed: integratedservices.IntegratedServiceStatusPending,
			v1alpha1.Uninstalling:      integratedservices.IntegratedServiceStatusPending,

			// failed final statuses
			v1alpha1.InstallFailed:   integratedservices.IntegratedServiceStatusError,
			v1alpha1.UninstallFailed: integratedservices.IntegratedServiceStatusError,

			// Final, stable phases
			v1alpha1.Uninstalled: integratedservices.IntegratedServiceStatusInactive,
			v1alpha1.Installed:   integratedservices.IntegratedServiceStatusActive,
		},
	}
}

func (s svcStatusMapper) MapStatus(serviceInstance v1alpha1.ServiceInstance) integratedservices.IntegratedServiceStatus {
	if serviceInstance.Status.Status == v1alpha1.StatusUnmanaged {
		return integratedservices.IntegratedServiceStatusInactive
	}

	if serviceInstance.Status.Status == v1alpha1.StatusInvalid {
		return integratedservices.IntegratedServiceStatusError
	}

	if mapped, ok := s.mappings[serviceInstance.Status.Phase]; ok {
		return mapped
	}

	return integratedservices.IntegratedServiceStatusPending
}
