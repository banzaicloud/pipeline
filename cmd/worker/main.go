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
	"encoding/base32"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"emperror.dev/emperror"
	"emperror.dev/errors"
	bauth "github.com/banzaicloud/bank-vaults/pkg/sdk/auth"
	"github.com/oklog/run"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"go.uber.org/cadence/activity"
	"go.uber.org/cadence/workflow"
	zaplog "logur.dev/integration/zap"
	"logur.dev/logur"

	"github.com/banzaicloud/pipeline/internal/clusterfeature/features/securityscan"

	"github.com/banzaicloud/pipeline/auth"
	"github.com/banzaicloud/pipeline/cluster"
	conf "github.com/banzaicloud/pipeline/config"
	"github.com/banzaicloud/pipeline/dns"
	intCluster "github.com/banzaicloud/pipeline/internal/cluster"
	intClusterAuth "github.com/banzaicloud/pipeline/internal/cluster/auth"
	"github.com/banzaicloud/pipeline/internal/cluster/clustersecret"
	"github.com/banzaicloud/pipeline/internal/cluster/clustersecret/clustersecretadapter"
	intClusterDNS "github.com/banzaicloud/pipeline/internal/cluster/dns"
	intClusterK8s "github.com/banzaicloud/pipeline/internal/cluster/kubernetes"
	intClusterWorkflow "github.com/banzaicloud/pipeline/internal/cluster/workflow"
	"github.com/banzaicloud/pipeline/internal/clusterfeature"
	"github.com/banzaicloud/pipeline/internal/clusterfeature/clusterfeatureadapter"
	featureDns "github.com/banzaicloud/pipeline/internal/clusterfeature/features/dns"
	featureMonitoring "github.com/banzaicloud/pipeline/internal/clusterfeature/features/monitoring"
	featureVault "github.com/banzaicloud/pipeline/internal/clusterfeature/features/vault"
	"github.com/banzaicloud/pipeline/internal/common/commonadapter"
	"github.com/banzaicloud/pipeline/internal/global"
	"github.com/banzaicloud/pipeline/internal/helm"
	"github.com/banzaicloud/pipeline/internal/helm/helmadapter"
	"github.com/banzaicloud/pipeline/internal/kubernetes"
	"github.com/banzaicloud/pipeline/internal/platform/buildinfo"
	"github.com/banzaicloud/pipeline/internal/platform/cadence"
	"github.com/banzaicloud/pipeline/internal/platform/database"
	"github.com/banzaicloud/pipeline/internal/platform/errorhandler"
	"github.com/banzaicloud/pipeline/internal/platform/log"
	azurePKEAdapter "github.com/banzaicloud/pipeline/internal/providers/azure/pke/adapter"
	"github.com/banzaicloud/pipeline/internal/providers/pke/pkeworkflow"
	"github.com/banzaicloud/pipeline/internal/providers/pke/pkeworkflow/pkeworkflowadapter"
	intSecret "github.com/banzaicloud/pipeline/internal/secret"
	pkgAuth "github.com/banzaicloud/pipeline/pkg/auth"
	"github.com/banzaicloud/pipeline/secret"
	"github.com/banzaicloud/pipeline/spotguide"
	"github.com/banzaicloud/pipeline/spotguide/scm"
)

// Provisioned by ldflags
// nolint: gochecknoglobals
var (
	version    string
	commitHash string
	buildDate  string
)

func main() {
	v, p := viper.New(), pflag.NewFlagSet(friendlyAppName, pflag.ExitOnError)

	configure(v, p)

	p.String("config", "", "Configuration file")
	p.Bool("version", false, "Show version information")

	_ = p.Parse(os.Args[1:])

	if v, _ := p.GetBool("version"); v {
		fmt.Printf("%s version %s (%s) built on %s\n", friendlyAppName, version, commitHash, buildDate)

		os.Exit(0)
	}

	if c, _ := p.GetString("config"); c != "" {
		v.SetConfigFile(c)
	}

	err := v.ReadInConfig()
	_, configFileNotFound := err.(viper.ConfigFileNotFoundError)
	if !configFileNotFound {
		emperror.Panic(errors.Wrap(err, "failed to read configuration"))
	}

	registerAliases(v)

	var config configuration
	err = v.Unmarshal(&config)
	emperror.Panic(errors.Wrap(err, "failed to unmarshal configuration"))

	// Create logger (first thing after configuration loading)
	logger := log.NewLogger(config.Log)

	// Provide some basic context to all log lines
	logger = log.WithFields(logger, map[string]interface{}{"environment": config.Environment, "application": appName})

	log.SetStandardLogger(logger)
	log.SetK8sLogger(logger)

	if configFileNotFound {
		logger.Warn("configuration file not found")
	}

	err = config.Validate()
	if err != nil {
		logger.Error(err.Error())

		os.Exit(3)
	}

	// Configure error handler
	errorHandler, err := errorhandler.New(config.ErrorHandler, logger)
	if err != nil {
		logger.Error(err.Error())

		os.Exit(1)
	}
	defer errorHandler.Close()
	defer emperror.HandleRecover(errorHandler)
	global.SetErrorHandler(errorHandler)

	buildInfo := buildinfo.New(version, commitHash, buildDate)

	logger.Info("starting application", buildInfo.Fields())

	switch v.GetString(conf.DNSBaseDomain) {
	case "", "example.com", "example.org":
		global.AutoDNSEnabled = false
	default:
		global.AutoDNSEnabled = true
	}

	var group run.Group

	// Configure Cadence worker
	{
		const taskList = "pipeline"
		worker, err := cadence.NewWorker(config.Cadence, taskList, zaplog.New(logur.WithFields(logger, map[string]interface{}{"component": "cadence-worker"})))
		emperror.Panic(err)

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
		tokenStore := bauth.NewVaultTokenStore("pipeline")
		tokenManager := pkgAuth.NewTokenManager(
			pkgAuth.NewJWTTokenGenerator(
				config.Auth.Token.Issuer,
				config.Auth.Token.Audience,
				base32.StdEncoding.EncodeToString([]byte(config.Auth.Token.SigningKey)),
			),
			tokenStore,
		)
		tokenGenerator := auth.NewClusterTokenGenerator(tokenManager, tokenStore)

		clusters := pkeworkflowadapter.NewClusterManagerAdapter(clusterManager)
		secretStore := pkeworkflowadapter.NewSecretStore(secret.Store)

		clusterSecretStore := clustersecret.NewStore(
			clustersecretadapter.NewClusterManagerAdapter(clusterManager),
			clustersecretadapter.NewSecretStore(secret.Store),
		)

		clusterAuthService, err := intClusterAuth.NewDexClusterAuthService(clusterSecretStore)
		emperror.Panic(errors.Wrap(err, "failed to create DexClusterAuthService"))

		scmProvider := v.GetString("cicd.scm")
		var scmToken string
		switch scmProvider {
		case "github":
			scmToken = v.GetString("github.token")
		case "gitlab":
			scmToken = v.GetString("gitlab.token")
		default:
			emperror.Panic(fmt.Errorf("Unknown SCM provider configured: %s", scmProvider))
		}

		scmFactory, err := scm.NewSCMFactory(scmProvider, scmToken)
		emperror.Panic(errors.WrapIf(err, "failed to create SCMFactory"))

		spotguideManager := spotguide.NewSpotguideManager(
			db,
			version,
			scmFactory,
			nil,
			spotguide.PlatformData{},
		)

		// Register amazon specific workflows and activities
		registerAwsWorkflows(clusters, tokenGenerator)

		azurePKEClusterStore := azurePKEAdapter.NewGORMAzurePKEClusterStore(db)

		// Register azure specific workflows
		registerAzureWorkflows(secretStore, tokenGenerator, azurePKEClusterStore)

		generateCertificatesActivity := pkeworkflow.NewGenerateCertificatesActivity(clusterSecretStore)
		activity.RegisterWithOptions(generateCertificatesActivity.Execute, activity.RegisterOptions{Name: pkeworkflow.GenerateCertificatesActivityName})

		scrapeSharedSpotguidesActivity := spotguide.NewScrapeSharedSpotguidesActivity(spotguideManager)
		workflow.RegisterWithOptions(spotguide.ScrapeSharedSpotguidesWorkflow, workflow.RegisterOptions{Name: spotguide.ScrapeSharedSpotguidesWorkflowName})
		activity.RegisterWithOptions(scrapeSharedSpotguidesActivity.Execute, activity.RegisterOptions{Name: spotguide.ScrapeSharedSpotguidesActivityName})

		createDexClientActivity := pkeworkflow.NewCreateDexClientActivity(clusters, clusterAuthService)
		activity.RegisterWithOptions(createDexClientActivity.Execute, activity.RegisterOptions{Name: pkeworkflow.CreateDexClientActivityName})

		deleteDexClientActivity := pkeworkflow.NewDeleteDexClientActivity(clusters, clusterAuthService)
		activity.RegisterWithOptions(deleteDexClientActivity.Execute, activity.RegisterOptions{Name: pkeworkflow.DeleteDexClientActivityName})

		setMasterTaintActivity := pkeworkflow.NewSetMasterTaintActivity(clusters)
		activity.RegisterWithOptions(setMasterTaintActivity.Execute, activity.RegisterOptions{Name: pkeworkflow.SetMasterTaintActivityName})

		workflow.RegisterWithOptions(cluster.RunPostHooksWorkflow, workflow.RegisterOptions{Name: cluster.RunPostHooksWorkflowName})

		runPostHookActivity := cluster.NewRunPostHookActivity(clusterManager)
		activity.RegisterWithOptions(runPostHookActivity.Execute, activity.RegisterOptions{Name: cluster.RunPostHookActivityName})

		updateClusterStatusActivity := cluster.NewUpdateClusterStatusActivity(clusterManager)
		activity.RegisterWithOptions(updateClusterStatusActivity.Execute, activity.RegisterOptions{Name: cluster.UpdateClusterStatusActivityName})

		deleteUnusedClusterSecretsActivity := intClusterWorkflow.MakeDeleteUnusedClusterSecretsActivity(secret.Store)
		activity.RegisterWithOptions(deleteUnusedClusterSecretsActivity.Execute, activity.RegisterOptions{Name: intClusterWorkflow.DeleteUnusedClusterSecretsActivityName})

		workflow.RegisterWithOptions(intClusterWorkflow.DeleteK8sResourcesWorkflow, workflow.RegisterOptions{Name: intClusterWorkflow.DeleteK8sResourcesWorkflowName})

		k8sConfigGetter := intSecret.MakeKubeSecretStore(secret.Store)

		deleteHelmDeploymentsActivity := intClusterWorkflow.MakeDeleteHelmDeploymentsActivity(k8sConfigGetter, conf.Logger())
		activity.RegisterWithOptions(deleteHelmDeploymentsActivity.Execute, activity.RegisterOptions{Name: intClusterWorkflow.DeleteHelmDeploymentsActivityName})

		deleteUserNamespacesActivity := intClusterWorkflow.MakeDeleteUserNamespacesActivity(intClusterK8s.MakeUserNamespaceDeleter(conf.Logger()), k8sConfigGetter)
		activity.RegisterWithOptions(deleteUserNamespacesActivity.Execute, activity.RegisterOptions{Name: intClusterWorkflow.DeleteUserNamespacesActivityName})

		deleteNamespaceResourcesActivity := intClusterWorkflow.MakeDeleteNamespaceResourcesActivity(intClusterK8s.MakeNamespaceResourcesDeleter(conf.Logger()), k8sConfigGetter)
		activity.RegisterWithOptions(deleteNamespaceResourcesActivity.Execute, activity.RegisterOptions{Name: intClusterWorkflow.DeleteNamespaceResourcesActivityName})

		deleteNamespaceServicesActivity := intClusterWorkflow.MakeDeleteNamespaceServicesActivity(intClusterK8s.MakeNamespaceServicesDeleter(conf.Logger()), k8sConfigGetter)
		activity.RegisterWithOptions(deleteNamespaceServicesActivity.Execute, activity.RegisterOptions{Name: intClusterWorkflow.DeleteNamespaceServicesActivityName})

		clusterDNSRecordsDeleter, err := intClusterDNS.MakeDefaultRecordsDeleter()
		emperror.Panic(emperror.Wrap(err, "failed to create default cluster DNS records deleter"))

		deleteClusterDNSRecordsActivity := intClusterWorkflow.MakeDeleteClusterDNSRecordsActivity(clusterDNSRecordsDeleter)
		activity.RegisterWithOptions(deleteClusterDNSRecordsActivity.Execute, activity.RegisterOptions{Name: intClusterWorkflow.DeleteClusterDNSRecordsActivityName})

		waitPersistentVolumesDeletionActivity := intClusterWorkflow.MakeWaitPersistentVolumesDeletionActivity(k8sConfigGetter, conf.Logger())
		activity.RegisterWithOptions(waitPersistentVolumesDeletionActivity.Execute, activity.RegisterOptions{Name: intClusterWorkflow.WaitPersistentVolumesDeletionActivityName})

		{
			// External DNS service
			dnsSvc, err := dns.GetExternalDnsServiceClient()
			if err != nil {
				logger.Error("Getting external DNS service client failed", map[string]interface{}{"error": err.Error()})
				panic(err)
			}

			if dnsSvc == nil {
				logger.Info("External DNS service functionality is not enabled")
			}

			logger := commonadapter.NewLogger(logger) // TODO: make this a context aware logger
			featureRepository := clusterfeatureadapter.NewGormFeatureRepository(db, logger)
			helmService := helm.NewHelmService(helmadapter.NewClusterService(clusterManager), logger)
			kubernetesService := kubernetes.NewKubernetesService(helmadapter.NewClusterService(clusterManager), logger)
			secretStore := commonadapter.NewSecretStore(secret.Store, commonadapter.OrgIDContextExtractorFunc(auth.GetCurrentOrganizationID))
			clusterGetter := clusterfeatureadapter.MakeClusterGetter(clusterManager)
			clusterService := clusterfeatureadapter.NewClusterService(clusterGetter)
			orgDomainService := featureDns.NewOrgDomainService(clusterGetter, dnsSvc, logger)
			monitorConfiguration := featureMonitoring.NewFeatureConfiguration()
			featureOperatorRegistry := clusterfeature.MakeFeatureOperatorRegistry([]clusterfeature.FeatureOperator{
				featureDns.MakeFeatureOperator(clusterGetter, clusterService, helmService, logger, orgDomainService, secretStore),
				securityscan.MakeFeatureOperator(clusterGetter, clusterService, helmService, secretStore, logger),
				featureVault.MakeFeatureOperator(clusterGetter, clusterService, helmService, kubernetesService, secretStore, logger),
				featureMonitoring.MakeFeatureOperator(clusterGetter, clusterService, helmService, monitorConfiguration, logger, secretStore),
			})

			registerClusterFeatureWorkflows(featureOperatorRegistry, featureRepository)
		}

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
