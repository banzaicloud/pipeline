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
