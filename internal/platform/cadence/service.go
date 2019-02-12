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
	"github.com/pkg/errors"
	"go.uber.org/cadence/.gen/go/cadence/workflowserviceclient"
	"go.uber.org/yarpc"
	"go.uber.org/yarpc/transport/tchannel"
	"go.uber.org/zap"
)

const cadenceService = "cadence-frontend"

// newServiceClient returns a new Cadence service client instance.
func newServiceClient(name string, config Config, logger *zap.Logger) (workflowserviceclient.Interface, error) {
	ch, err := tchannel.NewChannelTransport(
		tchannel.ServiceName(name),
		tchannel.Logger(logger),
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to setup tchannel")
	}

	dispatcher := yarpc.NewDispatcher(yarpc.Config{
		Name: name,
		Outbounds: yarpc.Outbounds{
			cadenceService: {Unary: ch.NewSingleOutbound(config.Addr())},
		},
	})

	if err := dispatcher.Start(); err != nil { // TODO: dispatcher.Stop() when the application exits?
		return nil, errors.Wrap(err, "failed to start dispatcher")
	}

	return workflowserviceclient.New(dispatcher.ClientConfig(cadenceService)), nil
}
