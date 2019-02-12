package cadence

import (
	"github.com/goph/emperror"
	"go.uber.org/cadence/client"
	"go.uber.org/zap"
)

// NewClient returns a new Cadence client.
func NewClient(config Config, logger *zap.Logger) (client.Client, error) {
	serviceClient, err := newServiceClient("cadence-client", config, logger)
	if err != nil {
		return nil, emperror.Wrap(err, "could not create cadence client")
	}

	return client.NewClient(
		serviceClient,
		config.Domain,
		&client.Options{
			Identity: config.Identity,
		},
	), nil
}
