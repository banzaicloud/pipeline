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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ServiceInstanceSpec defines the desired state of ServiceInstance.
type ServiceInstanceSpec struct {
	Service string `json:"service,omitempty"`
	Version string `json:"version,omitempty"`
	Enabled *bool  `json:"enabled,omitempty"`
	Config  string `json:"config,omitempty"`
}

type Status string

const (
	StatusUnmanaged Status = "Unmanaged"
	StatusManaged   Status = "Managed"
	StatusInvalid   Status = "Invalid"
)

type Phase string

const (
	PreInstalling    Phase = "Preinstalling"
	PreInstallFailed Phase = "PreinstallFailed"

	// Internal state for transitioning from PreInstalling to Installing
	PreInstallSuccess Phase = "Preinstalling"

	Installing    Phase = "Installing"
	InstallFailed Phase = "InstallFailed"

	// Internal state for transitioning from Installing to PostInstall
	InstallSuccess Phase = "InstallSuccess"

	PostInstall       Phase = "Postinstall"
	PostInstallFailed Phase = "PostinstallFailed"

	Uninstalling    Phase = "Uninstalling"
	UninstallFailed Phase = "UninstallFailed"

	// Final, stable phases
	Uninstalled Phase = "Uninstalled"
	Installed   Phase = "Installed"
)

// ServiceInstanceStatus defines the observed state of ServiceInstance.
type ServiceInstanceStatus struct {
	// AvailableVersions defines the upgrade steps that need to be taken to reach a certain version
	AvailableVersions map[string][]string `json:"availableVersions,omitempty"`
	// Version is the last version that has successfully been installed
	Version string `json:"version,omitempty"`
	// Status represents the management state of the service
	// - Unmanaged if the service is not configured actively
	// - Managed if the service is actively configured and there are no problems found
	// - Invalid if reconciliation is blocked due to misconfiguration
	Status Status `json:"status,omitempty"`
	// Phase represents the internal state of the resource
	Phase Phase `json:"phase,omitempty"`
	// NextVersion represents the next version that the resource is converging to
	NextVersion string `json:"nextVersion,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// ServiceInstance is the Schema for the serviceinstances API.
type ServiceInstance struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ServiceInstanceSpec   `json:"spec,omitempty"`
	Status ServiceInstanceStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ServiceInstanceList contains a list of ServiceInstance.
type ServiceInstanceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ServiceInstance `json:"items"`
}

// nolint:gochecknoinits
func init() {
	SchemeBuilder.Register(&ServiceInstance{}, &ServiceInstanceList{})
}
