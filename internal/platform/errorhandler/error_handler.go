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

package errorhandler

import (
	"context"

	"cloud.google.com/go/errorreporting"
	"emperror.dev/errors"
	"emperror.dev/handler/stackdriver"
	"github.com/goph/logur"
)

// Config holds information for configuring an error handler.
type Config struct {
	Stackdriver struct {
		Enabled   bool
		ProjectId string
	}

	ServiceName    string
	ServiceVersion string
}

func (c Config) Validate() error {
	if c.Stackdriver.Enabled {
		if c.Stackdriver.ProjectId == "" {
			return errors.New("stackdriver project ID cannot be empty")
		}
	}

	return nil
}

// New returns a new error handler.
func New(config Config, logger logur.Logger) (Handlers, error) {
	logHandler := logur.NewErrorHandler(logger)
	handlers := Handlers{logHandler}

	if config.Stackdriver.Enabled {
		ctx := context.Background()
		client, err := errorreporting.NewClient(ctx, config.Stackdriver.ProjectId, errorreporting.Config{
			ServiceName:    config.ServiceName,
			ServiceVersion: config.ServiceVersion,
			OnError:        logHandler.Handle,
		})
		if err != nil {
			return nil, errors.Wrap(err, "failed to create stackdriver client")
		}

		handlers = append(handlers, stackdriver.New(client))
	}

	return handlers, nil
}
