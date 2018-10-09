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
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

const (
	BaseScheduleName = "cluster-backup"
)

// CreateScheduleRequest describes a create schedule request
type CreateScheduleRequest struct {
	Name     string          `json:"name" binding:"required"`
	TTL      metav1.Duration `json:"ttl" binding:"required"`
	Schedule string          `json:"schedule" binding:"required"`
	Labels   labels.Set      `json:"labels"`
	Options  BackupOptions   `json:"options"`
}

// Schedule describes an ARK schedule
type Schedule struct {
	UID              string          `json:"uid"`
	Name             string          `json:"name"`
	Schedule         string          `json:"schedule"`
	TTL              metav1.Duration `json:"ttl"`
	Labels           labels.Set      `json:"labels"`
	Options          BackupOptions   `json:"options,omitempty"`
	Status           string          `json:"status"`
	LastBackup       time.Time       `json:"lastBackup"`
	ValidationErrors []string        `json:"validationErrors,omitempty"`
}

// CreateScheduleResponse describes a create schedule response
type CreateScheduleResponse struct {
	Name   string `json:"name"`
	Status int    `json:"status"`
}

// DeleteScheduleResponse describes a delete schedule response
type DeleteScheduleResponse struct {
	Name   string `json:"name"`
	Status int    `json:"status"`
}
