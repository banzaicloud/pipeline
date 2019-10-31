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

package cadence

import (
	"emperror.dev/errors"
	"go.uber.org/cadence/client"
	"go.uber.org/zap"
)

// NewClient returns a new Cadence client.
func NewClient(config Config, logger *zap.Logger) (client.Client, error) {
	serviceClient, err := newServiceClient("cadence-client", config, logger)
	if err != nil {
		return nil, errors.WithMessage(err, "could not create cadence client")
	}

	return client.NewClient(
		serviceClient,
		config.Domain,
		&client.Options{
			Identity: config.Identity,
		},
	), nil
}
