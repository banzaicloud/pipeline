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

package internal

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/request"
	"go.uber.org/cadence/activity"
)

// SetClientToken assigns idempotency token to the same value across retries
func SetClientToken(ctx context.Context, clientToken **string) {
	if clientToken == nil {
		info := activity.GetInfo(ctx)
		token := info.WorkflowExecution.RunID + "-" + info.ActivityID
		*clientToken = &token
	}
}

// WaitUntilActivity calls function until it returns an error which is not request.WaiterResourceNotReadyErrorCode.
func WaitUntilActivity(ctx context.Context, f func(context.Context, ...request.WaiterOption) error) error {
	// Do not rely on the waiter for retries to support heartbeating
	for {
		err := f(ctx, request.WithWaiterMaxAttempts(1))
		aerr, ok := err.(awserr.Error)
		if !ok || request.WaiterResourceNotReadyErrorCode != aerr.Code() {
			return err
		}
		activity.RecordHeartbeat(ctx)
		// TODO(maxim): Configurable poll interval
		time.Sleep(10 * time.Second)
	}
}
