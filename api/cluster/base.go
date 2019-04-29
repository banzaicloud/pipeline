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

package api

import (
	"github.com/banzaicloud/pipeline/secret"
)

// CreateClusterRequestBase defines the common properties of cluster creation requests
type CreateClusterRequestBase struct {
	Name         string       `json:"name" binding:"required"`
	Features     []Feature    `json:"features"`
	SecretID     string       `json:"secretId"`
	SecretName   string       `json:"secretName"`
	SSHSecretID  string       `json:"sshSecretId"`
	ScaleOptions ScaleOptions `json:"scaleOptions,omitempty"`
	Type         string       `json:"type" binding:"required"`
}

// Feature defines a cluster feature's properties
type Feature struct {
	Kind   string                 `json:"kind"`
	Params map[string]interface{} `json:"params"`
}

// ScaleOptions describes scale options
type ScaleOptions struct {
	Enabled             bool     `json:"enabled"`
	DesiredCPU          float64  `json:"desiredCpu" binding:"min=1"`
	DesiredMEM          float64  `json:"desiredMem" binding:"min=1"`
	DesiredGPU          int      `json:"desiredGpu" binding:"min=0"`
	OnDemandPCT         int      `json:"onDemandPct,omitempty" binding:"min=0,max=100"`
	Excludes            []string `json:"excludes,omitempty"`
	KeepDesiredCapacity bool     `json:"keepDesiredCapacity"`
}

// GetSecretIDFromRequest returns the ID of the secret referenced in the request.
// If no valid secret reference is found, an error is returned instead.
func GetSecretIDFromRequest(req CreateClusterRequestBase) string {
	secretID := req.SecretID

	if secretID == "" && req.SecretName != "" {
		secretID = secret.GenerateSecretIDFromName(req.SecretName)
	}

	return secretID
}
