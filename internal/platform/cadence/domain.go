package cadence

import (
	"github.com/goph/emperror"
	"go.uber.org/cadence/client"
	"go.uber.org/zap"
)

// NewDomainClient returns a new Cadence domain client.
func NewDomainClient(config Config, logger *zap.Logger) (client.DomainClient, error) {
	serviceClient, err := newServiceClient("cadence-domain-client", config, logger)
	if err != nil {
		return nil, emperror.Wrap(err, "could not create cadence domain client")
	}

	return client.NewDomainClient(
		serviceClient,
		&client.Options{
			Identity: config.Identity,
		},
	), nil
}
