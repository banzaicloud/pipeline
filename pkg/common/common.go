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

package common

import "time"

// BanzaiResponse describes Pipeline's responses
type BanzaiResponse struct {
	StatusCode int    `json:"status_code,omitempty"`
	Message    string `json:"message,omitempty"`
}

// ErrorResponse describes Pipeline's responses when an error occurred
type ErrorResponse struct {
	Code    int    `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
}

// CreatorBaseFields describes all field which contains info about who created the cluster/application etc
type CreatorBaseFields struct {
	CreatedAt   time.Time `json:"createdAt,omitempty"`
	CreatorName string    `json:"creatorName,omitempty"`
	CreatorId   uint      `json:"creatorId,omitempty"`
}

// NodeNames describes node names
type NodeNames map[string][]string

// ### [ Constants to common cluster default values ] ### //
const (
	DefaultNodeMinCount = 0
	DefaultNodeMaxCount = 2
)

// Constants for labeling cluster nodes
const (
	LabelKey         = "nodepool.banzaicloud.io/name"
	OnDemandLabelKey = "node.banzaicloud.io/ondemand"
)

// Constant for tainting head node
const (
	HeadNodeTaintKey = "nodepool.banzaicloud.io/name"
)

const (
	SpotConfigMapKey = "spot-deploy-config"
)
