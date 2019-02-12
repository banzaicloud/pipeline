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
	"github.com/goph/emperror"
	"go.uber.org/cadence/worker"
	"go.uber.org/zap"
)

// NewWorker returns a new Cadence worker.
func NewWorker(config Config, taskList string, logger *zap.Logger) (worker.Worker, error) {
	serviceClient, err := newServiceClient("cadence-worker", config, logger)
	if err != nil {
		return nil, emperror.Wrap(err, "could not create cadence worker client")
	}

	return worker.New(
		serviceClient,
		config.Domain,
		taskList,
		worker.Options{
			Logger: logger,
		},
	), nil
}
