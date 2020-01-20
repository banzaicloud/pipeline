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
)

const ServiceName = "expiry"

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

func CalculateDuration(now time.Time, tillDate string) (time.Duration, error) {
	expiryTime, err := time.Parse(time.RFC3339, tillDate)
	if err != nil {
		return 0, errors.WrapIf(err, "failed to parse the expiry date")
	}

	return expiryTime.Sub(now), nil
}
