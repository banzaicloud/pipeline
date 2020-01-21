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
	"fmt"
	"net/http"
	"strings"
	"time"

	"emperror.dev/errors"
	"github.com/gin-gonic/gin"

	"github.com/banzaicloud/pipeline/internal/global/nplabels"
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
	LabelKey = "nodepool.banzaicloud.io/name"
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

// ValidationError is returned when a request is semantically invalid.
type ValidationError struct {
	message    string
	violations []string
}

// Error implements the error interface.
func (e ValidationError) Error() string {
	errMsg := e.message
	if len(e.violations) > 0 {
		errMsg += ": " + strings.Join(e.violations, ", ")
	}
	return errMsg
}

// unwrapViolations is a helper func to unwrap violations from a validation error
func unwrapViolations(err error) []string {
	var verr interface {
		Violations() []string
	}

	if errors.As(err, &verr) {
		return verr.Violations()
	}

	return []string{err.Error()}
}

// Validate checks whether the node pool labels collide with labels
// set by Pipeline and also if these are valid Kubernetes labels
func ValidateNodePoolLabels(nodePoolName string, labels map[string]string) error {
	var violations []string

	for key, value := range labels {
		if err := nplabels.NodePoolLabelValidator().ValidateKey(key); err != nil {
			violations = append(violations, unwrapViolations(err)...)
		}

		if err := nplabels.NodePoolLabelValidator().ValidateValue(value); err != nil {
			violations = append(violations, unwrapViolations(err)...)
		}
	}

	if len(violations) > 0 {
		return errors.WithStack(ValidationError{
			message:    fmt.Sprintf("invalid labels on %s node pool", nodePoolName),
			violations: violations,
		})
	}

	return nil
}
