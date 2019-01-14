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

package backoff

import (
	"context"

	"github.com/goph/emperror"
	"github.com/lestrrat-go/backoff"
)

// NewConstantBackoffPolicy creates a new constant backoff policy
func NewConstantBackoffPolicy(config *ConstantBackoffConfig) *backoff.Constant {
	return backoff.NewConstant(config.Delay, backoff.WithMaxRetries(config.MaxRetries), backoff.WithMaxElapsedTime(config.MaxElapsedTime))
}

// RetryConstant retries the given function using constant backoff policy
func RetryConstant(function func() error, backoffPolicy *backoff.Constant) (err error) {
	b, cancel := backoffPolicy.Start(context.Background())

	defer cancel()
	for {
		select {
		case <-b.Done():
			return emperror.Wrap(err, "all attempts failed")
		case <-b.Next():
			err = function()
			if err == nil {
				return nil
			}
			if backoff.IsPermanentError(err) {
				return emperror.Wrap(err, "permanent error happened during retrying")
			}
		}
	}
}

// MarkErrorPermanent marks an error permanent error so the retry will bail when this error happens
func MarkErrorPermanent(err error) error {
	return backoff.MarkPermanent(err)
}
