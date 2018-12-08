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
	"encoding/json"
	"strconv"
	"time"

	arkAPI "github.com/heptio/ark/pkg/apis/ark/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/kubernetes/pkg/apis/core"
)

const (
	// LabelKeyCloud label key used for cloud information
	LabelKeyCloud = "pipeline-cloud"
	// LabelKeyDistribution label key used for distribution information
	LabelKeyDistribution = "pipeline-distribution"
	// LabelKeyNodeCount label key is used for node count
	LabelKeyNodeCount = "pipeline-nodecount"
)

// PersistBackupRequest describes a backup persisting request
type PersistBackupRequest struct {
	BucketID uint

	Cloud          string
	Distribution   string
	NodeCount      uint
	ContentChecked bool

	ClusterID    uint
	DeploymentID uint

	Nodes  *core.NodeList
	Backup *arkAPI.Backup
}

// Backup describes an ARK backup
type Backup struct {
	ID               uint                                `json:"id"`
	UID              string                              `json:"uid"`
	Name             string                              `json:"name"`
	TTL              metav1.Duration                     `json:"ttl"`
	Labels           labels.Set                          `json:"labels"`
	Cloud            string                              `json:"cloud"`
	Distribution     string                              `json:"distribution"`
	Options          BackupOptions                       `json:"options,omitempty"`
	Status           string                              `json:"status"`
	StartAt          time.Time                           `json:"startAt"`
	ExpireAt         time.Time                           `json:"expireAt"`
	VolumeBackups    map[string]*arkAPI.VolumeBackupInfo `json:"volumeBackups,omitempty"`
	ValidationErrors []string                            `json:"validationErrors,omitempty"`

	ClusterID       uint    `json:"clusterId,omitempty"`
	ActiveClusterID uint    `json:"activeClusterId,omitempty"`
	Bucket          *Bucket `json:"-"`
}

// DeleteBackupResponse describes a delete backup response
type DeleteBackupResponse struct {
	ID     uint `json:"id"`
	Status int  `json:"status"`
}

// CreateBackupRequest descibes a create backup request
type CreateBackupRequest struct {
	Name    string          `json:"name" binding:"required"`
	TTL     metav1.Duration `json:"ttl" binding:"required"`
	Labels  labels.Set      `json:"labels"`
	Options BackupOptions   `json:"options"`
}

// CreateBackupResponse describes a create backup response
type CreateBackupResponse struct {
	Name   string `json:"name"`
	Status int    `json:"status"`
}

// BackupOptions defines options specification for an Ark backup
type BackupOptions struct {
	// IncludedNamespaces is a slice of namespace names to include objects
	// from. If empty, all namespaces are included.
	IncludedNamespaces []string `json:"includedNamespaces,omitempty"`

	// ExcludedNamespaces contains a list of namespaces that are not
	// included in the backup.
	ExcludedNamespaces []string `json:"excludedNamespaces,omitempty"`

	// IncludedResources is a slice of resource names to include
	// in the backup. If empty, all resources are included.
	IncludedResources []string `json:"includedResources,omitempty"`

	// ExcludedResources is a slice of resource names that are not
	// included in the backup.
	ExcludedResources []string `json:"excludedResources,omitempty"`

	// LabelSelector is a metav1.LabelSelector to filter with
	// when adding individual objects to the backup. If empty
	// or nil, all objects are included. Optional.
	LabelSelector *metav1.LabelSelector `json:"labelSelector,omitempty"`

	// SnapshotVolumes specifies whether to take cloud snapshots
	// of any PV's referenced in the set of objects included
	// in the Backup.
	SnapshotVolumes *bool `json:"snapshotVolumes,omitempty"`

	// IncludeClusterResources specifies whether cluster-scoped resources
	// should be included for consideration in the backup.
	IncludeClusterResources *bool `json:"includeClusterResources,omitempty"`
}

// UnmarshalJSON is a custom JSON unmarshal function for labelSelector parsing
func (bo *BackupOptions) UnmarshalJSON(data []byte) error {

	type Alias BackupOptions
	aux := &struct {
		LabelSelector string `json:"labelSelector"`
		*Alias
	}{
		Alias: (*Alias)(bo),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	if aux.LabelSelector == "" {
		bo.LabelSelector = nil
	} else {
		parsed, err := metav1.ParseToLabelSelector(aux.LabelSelector)
		if err != nil {
			return err
		}
		bo.LabelSelector = parsed
	}

	return nil
}

// ExtendFromLabels used for set information to the request from the labels of the Backup
func (req *PersistBackupRequest) ExtendFromLabels() {

	if len(req.Backup.Labels) < 1 {
		return
	}

	if req.Cloud == "" {
		req.Cloud = req.Backup.Labels[LabelKeyCloud]
	}

	if req.Distribution == "" {
		req.Distribution = req.Backup.Labels[LabelKeyDistribution]
	}

	if count, err := strconv.Atoi(req.Backup.Labels[LabelKeyNodeCount]); req.NodeCount == 0 && err == nil {
		req.NodeCount = uint(count)
	}
}
