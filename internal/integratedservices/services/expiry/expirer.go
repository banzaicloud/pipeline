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

package expiry

import (
	"context"
	"time"

	"emperror.dev/errors"

	"github.com/banzaicloud/pipeline/internal/common"
)

type Expirer interface {
	// Expires a resource at a given date
	Expire(ctx context.Context, clusterID uint, expiryDate string) error
}

type ExpiryCanceler interface {
	CancelExpiry(ctx context.Context, clusterID uint) error
}

type ExpiryService interface {
	Expirer
	ExpiryCanceler
}

// No-op Expirer implementation
type noOpExpirer struct {
	logger common.Logger
}

func NewNoOpExpirer(log common.Logger) noOpExpirer {
	return noOpExpirer{
		logger: log,
	}
}

// Non-blocker no-op implementation
func (n noOpExpirer) Expire(ctx context.Context, clusterID uint, expiryDate string) error {
	n.logger.Info("noOpExpirer called", map[string]interface{}{"date": expiryDate})
	return nil
}

// Synchronous no - op Expirer implementation
type syncExpirer struct {
	logger common.Logger
}

func NewSyncNoOpExpirer(log common.Logger) syncExpirer {
	return syncExpirer{
		logger: log,
	}
}

func (s syncExpirer) Expire(ctx context.Context, clusterID uint, expiryDate string) error {

	t, err := time.ParseInLocation(time.RFC3339, expiryDate, time.Now().Location())
	if err != nil {
		return errors.WrapIf(err, "failed to parse the expiry date")
	}

	// get the duration
	duration := t.Sub(time.Now())
	time.Sleep(duration)

	s.logger.Info("expirer logic triggered")
	return nil
}
