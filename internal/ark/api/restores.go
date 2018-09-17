// Copyright © 2018 Banzai Cloud
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

package api

import (
	arkAPI "github.com/heptio/ark/pkg/apis/ark/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

// PersistRestoreRequest describes a persist restore request
type PersistRestoreRequest struct {
	BucketID  uint
	ClusterID uint

	Results *RestoreResults
	Restore *arkAPI.Restore
}

// RestoreResults describes a restore results
type RestoreResults struct {
	Errors   arkAPI.RestoreResult `json:"errors"`
	Warnings arkAPI.RestoreResult `json:"warnings"`
}

// Restore describes an ARK restore
type Restore struct {
	ID               uint           `json:"id"`
	UID              string         `json:"uid"`
	Name             string         `json:"name"`
	BackupName       string         `json:"backupName"`
	Status           string         `json:"status"`
	Warnings         uint           `json:"warnings"`
	Errors           uint           `json:"errors"`
	ValidationErrors []string       `json:"validationErrors,omitempty"`
	Options          RestoreOptions `json:"options,omitempty"`

	Results *RestoreResults `json:"-"`
	Bucket  *Bucket         `json:"-"`
}

// RestoreOptions defines options specification for an Ark restore
type RestoreOptions struct {
	// IncludedNamespaces is a slice of namespace names to include objects
	// from. If empty, all namespaces are included.
	IncludedNamespaces []string `json:"includedNamespaces,omitempty"`

	// ExcludedNamespaces contains a list of namespaces that are not
	// included in the restore.
	ExcludedNamespaces []string `json:"excludedNamespaces,omitempty"`

	// IncludedResources is a slice of resource names to include
	// in the restore. If empty, all resources in the backup are included.
	IncludedResources []string `json:"includedResources,omitempty"`

	// ExcludedResources is a slice of resource names that are not
	// included in the restore.
	ExcludedResources []string `json:"excludedResources,omitempty"`

	// NamespaceMapping is a map of source namespace names
	// to target namespace names to restore into. Any source
	// namespaces not included in the map will be restored into
	// namespaces of the same name.
	NamespaceMapping map[string]string `json:"namespaceMapping,omitempty"`

	// LabelSelector is a metav1.LabelSelector to filter with
	// when restoring individual objects from the backup. If empty
	// or nil, all objects are included. Optional.
	LabelSelector *metav1.LabelSelector `json:"labelSelector,omitempty"`

	// RestorePVs specifies whether to restore all included
	// PVs from snapshot (via the cloudprovider).
	RestorePVs *bool `json:"restorePVs,omitempty"`

	// IncludeClusterResources specifies whether cluster-scoped resources
	// should be included for consideration in the restore. If null, defaults
	// to true.
	IncludeClusterResources *bool `json:"includeClusterResources,omitempty"`
}

// CreateRestoreRequest describes a create restore request
type CreateRestoreRequest struct {
	BackupName string         `json:"backupName" binding:"required"`
	Labels     labels.Set     `json:"labels"`
	Options    RestoreOptions `json:"options"`
}

// CreateRestoreResponse describes a create restore response
type CreateRestoreResponse struct {
	Restore *Restore `json:"restore"`
	Status  int      `json:"status"`
}

// DeleteRestoreResponse describes a delete restore response
type DeleteRestoreResponse struct {
	ID     uint `json:"id"`
	Status int  `json:"status"`
}
