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
