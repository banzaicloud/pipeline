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

package pkeaws

import (
	"fmt"

	"github.com/banzaicloud/pipeline/internal/common"
)

// These interfaces are aliased so that the module code is separated from the rest of the application.
// If the module is moved out of the app, copy the aliased interfaces here.

// Logger is the fundamental interface for all log operations.
type Logger = common.Logger

// NoopLogger is a logger that discards every log event.
type NoopLogger = common.NoopLogger

// ErrorHandler handles an error.
type ErrorHandler = common.ErrorHandler

// NoopErrorHandler is an error handler that discards every error.
type NoopErrorHandler = common.NoopErrorHandler

// TODO: this is temporary
func GenerateNodePoolStackName(clusterName string, poolName string) string {
	if poolName == "master" {
		return fmt.Sprintf("pke-master-%s", clusterName)
	}
	return fmt.Sprintf("pke-pool-%s-worker-%s", clusterName, poolName)
}
