package config

import (
	"context"
	"sync"

	"github.com/banzaicloud/pipeline/internal/platform/cadence"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"go.uber.org/cadence/.gen/go/shared"
	"go.uber.org/cadence/client"
	"go.uber.org/cadence/worker"
)

var cadenceClient client.Client
var cadenceOnce sync.Once

func newCadenceConfig() cadence.Config {
	return cadence.Config{
		Host:   viper.GetString("cadence.host"),
		Port:   viper.GetInt("cadence.port"),
		Domain: viper.GetString("cadence.domain"),
	}
}

// CadenceTaskList returns the used task list name.
// TODO: this should be separated later
func CadenceTaskList() string {
	return "pipeline"
}

// CadenceClient returns a cadence client.
func CadenceClient() client.Client {
	cadenceOnce.Do(func() {
		cadenceClient = newCadence()
	})

	return cadenceClient
}

func newCadence() client.Client {
	c, err := cadence.NewClient(newCadenceConfig(), ZapLogger())
	if err != nil {
		panic(err)
	}

	return c
}

// CadenceWorker returns a cadence worker.
func CadenceWorker() worker.Worker {
	w, err := cadence.NewWorker(newCadenceConfig(), CadenceTaskList(), ZapLogger())
	if err != nil {
		panic(err)
	}

	return w
}

func RegisterCadenceDomain(logger logrus.FieldLogger) {
	config := newCadenceConfig()
	client, err := cadence.NewDomainClient(config, ZapLogger())
	if err != nil {
		panic(err)
	}

	logger = logger.WithField("domain", config.Domain)

	domainRequest := &shared.RegisterDomainRequest{Name: &config.Domain}

	client.Register(context.Background(), domainRequest)
	if err != nil {
		if _, ok := err.(*shared.DomainAlreadyExistsError); !ok {
			panic(err)
		}
		logger.Info("domain already registered")
	} else {
		logger.Info("domain succeesfully registered")
	}
}
