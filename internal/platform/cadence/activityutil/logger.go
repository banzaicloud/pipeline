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

package activityutil

import (
	"context"

	pipelineCtx "github.com/banzaicloud/pipeline/internal/platform/context"
	"go.uber.org/cadence/activity"
	"go.uber.org/zap"
)

// GetLogger returns a logger that can be used in an activity.
func GetLogger(ctx context.Context) *zap.Logger {
	logger := activity.GetLogger(ctx)

	cid := pipelineCtx.CorrelationID(ctx)
	if cid != "" {
		logger = logger.With(zap.String("correlation-id", cid))
	}

	return logger
}
