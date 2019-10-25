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

package workflow

import (
	"context"
	"errors"
	"time"

	"go.uber.org/cadence/activity"
)

const (
	shouldRetryReason = "should retry activity"
)

func shouldRetry(err error) bool {
	var sh interface {
		ShouldRetry() bool
	}
	if errors.As(err, &sh) {
		return sh.ShouldRetry()
	}
	return false
}

func startHeartbeat(ctx context.Context, interval time.Duration) *time.Ticker {
	heartbeat := time.NewTicker(interval)

	go func() {
		for {
			activity.RecordHeartbeat(ctx)

			if _, closed := <-heartbeat.C; closed {
				return
			}
		}
	}()

	return heartbeat
}

func wait(ctx context.Context, duration time.Duration) error {
	select {
	case <-time.NewTimer(duration).C:
	case <-ctx.Done():
	}
	return ctx.Err()
}
