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

const InternalServiceName = "expiry"

type Expirer interface {
	Expire(ctx context.Context, clusterID uint, expiryDate string) error
}

type ExpiryCanceller interface {
	CancelExpiry(ctx context.Context, clusterID uint) error
}

type ExpiryService interface {
	Expirer
	ExpiryCanceller
}

// Synchronous no - op ExpirationService implementation
type syncExpiryService struct {
	logger common.Logger
}

func (s syncExpiryService) CancelExpiry(ctx context.Context, clusterID uint) error {
	return nil
}

func NewSyncExpiryService(log common.Logger) syncExpiryService {
	return syncExpiryService{
		logger: log,
	}
}

func (s syncExpiryService) Expire(ctx context.Context, clusterID uint, expiryDate string) error {

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
