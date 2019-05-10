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

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/goph/emperror"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/validation"
)

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

// Validate checks whether the node pool labels collide with labels
// set by Pipeline and also if these are valid Kubernetes labels
func ValidateNodePoolLabels(labels map[string]string) error {
	for name, value := range labels {
		// validate node label name
		errs := validation.IsQualifiedName(name)
		if len(errs) > 0 {
			return emperror.WrapWith(errors.New(strings.Join(errs, "\n")), "invalid node label name", "labelName", name)
		}

		// validate node label value
		errs = validation.IsValidLabelValue(value)
		if len(errs) > 0 {
			return emperror.WrapWith(errors.New(strings.Join(errs, "\n")), "invalid node label value", "labelValue", value)
		}
	}

	return nil
}

// ### [ Constants to common cluster default values ] ### //
const (
	DefaultNodeMinCount = 0
	DefaultNodeMaxCount = 2
)

// Constant for the common part of all possible Pipeline specific label name
const (
	PipelineSpecificLabelsCommonPart = "banzaicloud.io"
)

// Constants for labeling cluster nodes
const (
	LabelKey                = "nodepool.banzaicloud.io/name"
	OnDemandLabelKey        = "node.banzaicloud.io/ondemand"
	CloudInfoLabelKeyPrefix = "node.banzaicloud.io/"
	HeadNodeLabelKey        = "nodepool.banzaicloud.io/head"
)

// Constant for tainting head node
const (
	NodePoolNameTaintKey = "nodepool.banzaicloud.io/name"
)

const (
	SpotConfigMapKey = "spot-deploy-config"
)

// ErrorResponseWithStatus aborts the http request with a JSON error response with the given status code and error
func ErrorResponseWithStatus(c *gin.Context, status int, err error) {

	if c.Writer.Status() != http.StatusOK {
		return
	}

	c.AbortWithStatusJSON(status, ErrorResponse{
		Code:    status,
		Message: err.Error(),
		Error:   errors.Cause(err).Error(),
	})
}
