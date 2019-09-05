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

package watermill

import (
	"time"

	"emperror.dev/errors"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	watermilllog "logur.dev/integration/watermill"
	"logur.dev/logur"
)

// RouterConfig holds information for configuring Watermill router.
type RouterConfig struct {
	CloseTimeout time.Duration
}

// NewRouter returns a new message router for message subscription logic.
func NewRouter(config RouterConfig, logger logur.Logger) (*message.Router, error) {
	h, err := message.NewRouter(
		message.RouterConfig{
			CloseTimeout: config.CloseTimeout,
		},
		watermilllog.New(logur.WithFields(logger, map[string]interface{}{"component": "watermill"})),
	)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to create message router")
	}

	retryMiddleware := middleware.Retry{}
	retryMiddleware.MaxRetries = 1
	retryMiddleware.MaxInterval = time.Millisecond * 10

	h.AddMiddleware(
		// if retries limit was exceeded, message is sent to poison queue (poison_queue topic)
		retryMiddleware.Middleware,

		// recovered recovers panic from handlers
		middleware.Recoverer,

		// correlation ID middleware adds to every produced message correlation id of consumed message,
		// useful for debugging
		middleware.CorrelationID,
	)

	return h, nil
}
