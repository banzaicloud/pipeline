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

package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/banzaicloud/pipeline/auth"
	"github.com/banzaicloud/pipeline/cluster"
	conf "github.com/banzaicloud/pipeline/config"
	intAuth "github.com/banzaicloud/pipeline/internal/auth"
	intCluster "github.com/banzaicloud/pipeline/internal/cluster"
	"github.com/banzaicloud/pipeline/internal/platform/buildinfo"
	"github.com/banzaicloud/pipeline/internal/platform/cadence"
	"github.com/banzaicloud/pipeline/internal/platform/database"
	"github.com/banzaicloud/pipeline/internal/platform/errorhandler"
	"github.com/banzaicloud/pipeline/internal/platform/log"
	"github.com/banzaicloud/pipeline/internal/platform/zaplog"
	"github.com/banzaicloud/pipeline/internal/providers/pke/pkeworkflow"
	"github.com/banzaicloud/pipeline/internal/providers/pke/pkeworkflow/pkeworkflowadapter"
	"github.com/goph/emperror"
	"github.com/oklog/run"
	"github.com/pkg/errors"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"go.uber.org/cadence/activity"
	"go.uber.org/cadence/workflow"
)

// nolint: gochecknoinits
func init() {
	pflag.Bool("version", false, "Show version information")
	pflag.Bool("dump-config", false, "Dump configuration to the console (and exit)")
}

func main() {
	Configure(viper.GetViper(), pflag.CommandLine)

	pflag.Parse()

	if viper.GetBool("version") {
		fmt.Printf("%s version %s (%s) built on %s\n", FriendlyServiceName, version, commitHash, buildDate)

		os.Exit(0)
	}

	err := viper.ReadInConfig()
	_, configFileNotFound := err.(viper.ConfigFileNotFoundError)
	if !configFileNotFound {
		emperror.Panic(errors.Wrap(err, "failed to read configuration"))
	}

	var config Config
	err = viper.Unmarshal(&config)
	emperror.Panic(errors.Wrap(err, "failed to unmarshal configuration"))

	// Create logger (first thing after configuration loading)
	logger := log.NewLogurLogger(config.Log)

	// Provide some basic context to all log lines
	logger = log.WithFields(logger, map[string]interface{}{"environment": config.Environment, "service": ServiceName})

	if configFileNotFound {
		logger.Warn("configuration file not found", nil)
	}

	err = config.Validate()
	if err != nil {
		logger.Error(err.Error(), nil)

		os.Exit(3)
	}

	if viper.GetBool("dump-config") {
		fmt.Printf("%+v\n", config)

		os.Exit(0)
	}

	// Configure error handler
	errorHandler := errorhandler.New(logger)
	defer emperror.HandleRecover(errorHandler)

	buildInfo := buildinfo.New(version, commitHash, buildDate)

	logger.Info("starting application", buildInfo.Fields())

	var group run.Group

	// Configure Cadence worker
	{
		const taskList = "pipeline"
		zapLogger := zaplog.NewLogger(zaplog.Config{
			Level:  config.Log.Level,
			Format: config.Log.Format,
		})
		worker, err := cadence.NewWorker(config.Cadence, taskList, zapLogger)
		emperror.Panic(err)

		workflow.RegisterWithOptions(pkeworkflow.CreateClusterWorkflow, workflow.RegisterOptions{Name: pkeworkflow.CreateClusterWorkflowName})
		workflow.RegisterWithOptions(pkeworkflow.DeleteClusterWorkflow, workflow.RegisterOptions{Name: pkeworkflow.DeleteClusterWorkflowName})

		db, err := database.Connect(config.Database)
		if err != nil {
			emperror.Panic(err)
		}

		clusterManager := cluster.NewManager(
			intCluster.NewClusters(db),
			nil,
			nil,
			nil,
			nil,
			nil,
			conf.Logger(),
			errorHandler,
		)
		enforcer := intAuth.NewEnforcer(db)
		accessManager := intAuth.NewAccessManager(enforcer, config.Pipeline.BasePath)
		tokenGenerator := pkeworkflowadapter.NewTokenGenerator(auth.NewTokenHandler(accessManager))
		auth.Init(nil, accessManager, nil)
		auth.InitTokenStore()

		clusters := pkeworkflowadapter.NewClusterManagerAdapter(clusterManager)

		generateCertificatesActivity := pkeworkflow.NewGenerateCertificatesActivity(clusters)
		activity.RegisterWithOptions(generateCertificatesActivity.Execute, activity.RegisterOptions{Name: pkeworkflow.GenerateCertificatesActivityName})

		createAWSRolesActivity := pkeworkflow.NewCreateAWSRolesActivity(clusters)
		activity.RegisterWithOptions(createAWSRolesActivity.Execute, activity.RegisterOptions{Name: pkeworkflow.CreateAWSRolesActivityName})

		waitCFCompletionActivity := pkeworkflow.NewWaitCFCompletionActivity(clusters)
		activity.RegisterWithOptions(waitCFCompletionActivity.Execute, activity.RegisterOptions{Name: pkeworkflow.WaitCFCompletionActivityName})

		createPKEVPCActivity := pkeworkflow.NewCreateVPCActivity(clusters)
		activity.RegisterWithOptions(createPKEVPCActivity.Execute, activity.RegisterOptions{Name: pkeworkflow.CreateVPCActivityName})

		updateClusterStatusActivitiy := pkeworkflow.NewUpdateClusterStatusActivity(clusters)
		activity.RegisterWithOptions(updateClusterStatusActivitiy.Execute, activity.RegisterOptions{Name: pkeworkflow.UpdateClusterStatusActivityName})

		updateClusterNetworkActivitiy := pkeworkflow.NewUpdateClusterNetworkActivity(clusters)
		activity.RegisterWithOptions(updateClusterNetworkActivitiy.Execute, activity.RegisterOptions{Name: pkeworkflow.UpdateClusterNetworkActivityName})

		createElasticIPActivity := pkeworkflow.NewCreateElasticIPActivity(clusters)
		activity.RegisterWithOptions(createElasticIPActivity.Execute, activity.RegisterOptions{Name: pkeworkflow.CreateElasticIPActivityName})

		createMasterActivity := pkeworkflow.NewCreateMasterActivity(clusters, tokenGenerator)
		activity.RegisterWithOptions(createMasterActivity.Execute, activity.RegisterOptions{Name: pkeworkflow.CreateMasterActivityName})

		listNodePoolsActivity := pkeworkflow.NewListNodePoolsActivity(clusters)
		activity.RegisterWithOptions(listNodePoolsActivity.Execute, activity.RegisterOptions{Name: pkeworkflow.ListNodePoolsActivityName})

		createWorkerPoolActivity := pkeworkflow.NewCreateWorkerPoolActivity(clusters, tokenGenerator)
		activity.RegisterWithOptions(createWorkerPoolActivity.Execute, activity.RegisterOptions{Name: pkeworkflow.CreateWorkerPoolActivityName})

		deleteWorkerPoolActivity := pkeworkflow.NewDeleteWorkerPoolActivity(clusters)
		activity.RegisterWithOptions(deleteWorkerPoolActivity.Execute, activity.RegisterOptions{Name: pkeworkflow.DeleteWorkerPoolActivityName})

		uploadSshKeyPairActivity := pkeworkflow.NewUploadSSHKeyPairActivity(clusters)
		activity.RegisterWithOptions(uploadSshKeyPairActivity.Execute, activity.RegisterOptions{Name: pkeworkflow.UploadSSHKeyPairActivityName})

		var closeCh = make(chan struct{})

		group.Add(
			func() error {
				err := worker.Start()
				if err != nil {
					return err
				}

				<-closeCh

				return nil
			},
			func(e error) {
				worker.Stop()
				close(closeCh)
			},
		)
	}

	// Setup signal handler
	{
		var (
			cancelInterrupt = make(chan struct{})
			ch              = make(chan os.Signal, 2)
		)
		defer close(ch)

		group.Add(
			func() error {
				signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)

				select {
				case sig := <-ch:
					logger.Info("captured signal", map[string]interface{}{"signal": sig})
				case <-cancelInterrupt:
				}

				return nil
			},
			func(e error) {
				close(cancelInterrupt)
				signal.Stop(ch)
			},
		)
	}

	err = group.Run()
	emperror.Handle(errorHandler, err)
}
