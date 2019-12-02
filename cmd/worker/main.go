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
	"text/template"

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

	anchore2 "github.com/banzaicloud/pipeline/internal/anchore"
	intClusterAuth "github.com/banzaicloud/pipeline/internal/cluster/auth"
	"github.com/banzaicloud/pipeline/internal/cluster/clusteradapter"
	"github.com/banzaicloud/pipeline/internal/cluster/clustersecret"
	"github.com/banzaicloud/pipeline/internal/cluster/clustersecret/clustersecretadapter"
	"github.com/banzaicloud/pipeline/internal/cluster/clustersetup"
	intClusterDNS "github.com/banzaicloud/pipeline/internal/cluster/dns"
	"github.com/banzaicloud/pipeline/internal/cluster/endpoints"
	intClusterK8s "github.com/banzaicloud/pipeline/internal/cluster/kubernetes"
	intClusterWorkflow "github.com/banzaicloud/pipeline/internal/cluster/workflow"
	"github.com/banzaicloud/pipeline/internal/clusterfeature"
	"github.com/banzaicloud/pipeline/internal/clusterfeature/clusterfeatureadapter"
	featureDns "github.com/banzaicloud/pipeline/internal/clusterfeature/features/dns"
	"github.com/banzaicloud/pipeline/internal/clusterfeature/features/dns/dnsadapter"
	featureLogging "github.com/banzaicloud/pipeline/internal/clusterfeature/features/logging"
	featureMonitoring "github.com/banzaicloud/pipeline/internal/clusterfeature/features/monitoring"
	"github.com/banzaicloud/pipeline/internal/clusterfeature/features/securityscan"
	"github.com/banzaicloud/pipeline/internal/clusterfeature/features/securityscan/securityscanadapter"
	featureVault "github.com/banzaicloud/pipeline/internal/clusterfeature/features/vault"
	"github.com/banzaicloud/pipeline/internal/common/commonadapter"
	"github.com/banzaicloud/pipeline/internal/global"
	"github.com/banzaicloud/pipeline/internal/helm"
	"github.com/banzaicloud/pipeline/internal/helm/helmadapter"
	"github.com/banzaicloud/pipeline/internal/kubernetes"
	"github.com/banzaicloud/pipeline/internal/kubernetes/kubernetesadapter"
	intpkeworkflowadapter "github.com/banzaicloud/pipeline/internal/pke/workflow/adapter"
	"github.com/banzaicloud/pipeline/internal/platform/buildinfo"
	"github.com/banzaicloud/pipeline/internal/platform/cadence"
	"github.com/banzaicloud/pipeline/internal/platform/database"
	"github.com/banzaicloud/pipeline/internal/platform/errorhandler"
	"github.com/banzaicloud/pipeline/internal/platform/log"
	azurePKEAdapter "github.com/banzaicloud/pipeline/internal/providers/azure/pke/adapter"
	"github.com/banzaicloud/pipeline/internal/providers/pke/pkeworkflow"
	"github.com/banzaicloud/pipeline/internal/providers/pke/pkeworkflow/pkeworkflowadapter"
	intSecret "github.com/banzaicloud/pipeline/internal/secret"
	anchore "github.com/banzaicloud/pipeline/internal/security"
	pkgAuth "github.com/banzaicloud/pipeline/pkg/auth"
	"github.com/banzaicloud/pipeline/src/auth"
	"github.com/banzaicloud/pipeline/src/auth/authdriver"
	"github.com/banzaicloud/pipeline/src/cluster"
	"github.com/banzaicloud/pipeline/src/dns"
	"github.com/banzaicloud/pipeline/src/secret"
	"github.com/banzaicloud/pipeline/src/spotguide"
	"github.com/banzaicloud/pipeline/src/spotguide/scm"
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

	var config configuration
	err = v.Unmarshal(&config)
	emperror.Panic(errors.Wrap(err, "failed to unmarshal configuration"))

	err = config.Process()
	emperror.Panic(errors.WithMessage(err, "failed to process configuration"))

	err = v.Unmarshal(&global.Config)
	emperror.Panic(errors.Wrap(err, "failed to unmarshal global configuration"))

	err = global.Config.Process()
	emperror.Panic(errors.WithMessage(err, "failed to process global configuration"))

	// Create logger (first thing after configuration loading)
	logger := log.NewLogger(config.Log)

	// Legacy logger instance
	logrusLogger := log.NewLogrusLogger(log.Config{
		Level:  config.Log.Level,
		Format: config.Log.Format,
	})
	global.SetLogrusLogger(logrusLogger)

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

	err = global.Config.Validate()
	if err != nil {
		logger.Error(err.Error(), map[string]interface{}{"config": "global"})

		os.Exit(3)
	}

	// Configure error handler
	errorHandler, err := errorhandler.New(config.Errors, logger)
	if err != nil {
		logger.Error(err.Error())

		os.Exit(1)
	}
	defer errorHandler.Close()
	defer emperror.HandleRecover(errorHandler)
	global.SetErrorHandler(errorHandler)

	buildInfo := buildinfo.New(version, commitHash, buildDate)

	logger.Info("starting application", buildInfo.Fields())

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
		global.SetDB(db)

		clusterManager := cluster.NewManager(
			clusteradapter.NewClusters(db),
			nil,
			nil,
			nil,
			nil,
			nil,
			logrusLogger,
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

		helmService := helm.NewHelmService(helmadapter.NewClusterService(clusterManager), commonadapter.NewLogger(logger))

		clusters := pkeworkflowadapter.NewClusterManagerAdapter(clusterManager)
		secretStore := pkeworkflowadapter.NewSecretStore(secret.Store)

		clusterSecretStore := clustersecret.NewStore(
			clustersecretadapter.NewClusterManagerAdapter(clusterManager),
			clustersecretadapter.NewSecretStore(secret.Store),
		)

		clusterAuthService, err := intClusterAuth.NewDexClusterAuthService(clusterSecretStore)
		emperror.Panic(errors.Wrap(err, "failed to create DexClusterAuthService"))

		scmProvider := global.Config.CICD.SCM
		var scmToken string
		switch scmProvider {
		case "github":
			scmToken = global.Config.Github.Token
		case "gitlab":
			scmToken = global.Config.Gitlab.Token
		default:
			emperror.Panic(fmt.Errorf("Unknown SCM provider configured: %s", scmProvider))
		}

		scmFactory, err := scm.NewSCMFactory(scmProvider, scmToken, auth.SCMTokenStore{})
		emperror.Panic(errors.WrapIf(err, "failed to create SCMFactory"))

		spotguideManager := spotguide.NewSpotguideManager(
			db,
			version,
			scmFactory,
			nil,
			spotguide.PlatformData{},
		)

		commonSecretStore := commonadapter.NewSecretStore(secret.Store, commonadapter.OrgIDContextExtractorFunc(auth.GetCurrentOrganizationID))
		configFactory := kubernetes.NewConfigFactory(commonSecretStore)

		// Cluster setup
		{
			wf := clustersetup.Workflow{
				InstallInitManifest: config.Cluster.Manifest != "",
			}
			workflow.RegisterWithOptions(wf.Execute, workflow.RegisterOptions{Name: clustersetup.WorkflowName})

			initManifestTemplate := template.New("")
			if config.Cluster.Manifest != "" {
				initManifestTemplate = template.Must(template.ParseFiles(config.Cluster.Manifest))
			}

			initManifestActivity := clustersetup.NewInitManifestActivity(
				initManifestTemplate,
				kubernetes.NewDynamicFileClientFactory(configFactory),
			)
			activity.RegisterWithOptions(initManifestActivity.Execute, activity.RegisterOptions{Name: clustersetup.InitManifestActivityName})

			createPipelineNamespaceActivity := clustersetup.NewCreatePipelineNamespaceActivity(
				config.Cluster.Namespace,
				kubernetes.NewClientFactory(configFactory),
			)
			activity.RegisterWithOptions(createPipelineNamespaceActivity.Execute, activity.RegisterOptions{Name: clustersetup.CreatePipelineNamespaceActivityName})

			labelKubeSystemNamespaceActivity := clustersetup.NewLabelKubeSystemNamespaceActivity(
				kubernetes.NewClientFactory(configFactory),
			)
			activity.RegisterWithOptions(labelKubeSystemNamespaceActivity.Execute, activity.RegisterOptions{Name: clustersetup.LabelKubeSystemNamespaceActivityName})

			installTillerActivity := clustersetup.NewInstallTillerActivity(
				config.Helm.Tiller.Version,
				kubernetes.NewClientFactory(configFactory),
			)
			activity.RegisterWithOptions(installTillerActivity.Execute, activity.RegisterOptions{Name: clustersetup.InstallTillerActivityName})

			installTillerWaitActivity := clustersetup.NewInstallTillerWaitActivity(
				config.Helm.Tiller.Version,
				kubernetes.NewHelmClientFactory(configFactory, commonadapter.NewLogger(logger)),
			)
			activity.RegisterWithOptions(installTillerWaitActivity.Execute, activity.RegisterOptions{Name: clustersetup.InstallTillerWaitActivityName})

			installNodePoolLabelSetOperatorActivity := clustersetup.NewInstallNodePoolLabelSetOperatorActivity(
				config.Cluster.Labels,
				helmService,
			)
			activity.RegisterWithOptions(installNodePoolLabelSetOperatorActivity.Execute, activity.RegisterOptions{Name: clustersetup.InstallNodePoolLabelSetOperatorActivityName})

			configureNodePoolLabelsActivity := clustersetup.NewConfigureNodePoolLabelsActivity(
				config.Cluster.Labels.Namespace,
				kubernetes.NewDynamicClientFactory(configFactory),
			)
			activity.RegisterWithOptions(configureNodePoolLabelsActivity.Execute, activity.RegisterOptions{Name: clustersetup.ConfigureNodePoolLabelsActivityName})
		}

		workflow.RegisterWithOptions(cluster.CreateClusterWorkflow, workflow.RegisterOptions{Name: cluster.CreateClusterWorkflowName})

		downloadK8sConfigActivity := cluster.NewDownloadK8sConfigActivity(clusterManager)
		activity.RegisterWithOptions(downloadK8sConfigActivity.Execute, activity.RegisterOptions{Name: cluster.DownloadK8sConfigActivityName})

		setupPrivilegesActivity := cluster.NewSetupPrivilegesActivity(kubernetes.NewClientFactory(configFactory), clusterManager)
		activity.RegisterWithOptions(setupPrivilegesActivity.Execute, activity.RegisterOptions{Name: cluster.SetupPrivilegesActivityName})

		labelNodesWithNodepoolNameActivity := cluster.NewLabelNodesWithNodepoolNameActivity(kubernetes.NewClientFactory(configFactory), clusterManager)
		activity.RegisterWithOptions(labelNodesWithNodepoolNameActivity.Execute, activity.RegisterOptions{Name: cluster.LabelNodesWithNodepoolNameActivityName})

		workflow.RegisterWithOptions(cluster.RunPostHooksWorkflow, workflow.RegisterOptions{Name: cluster.RunPostHooksWorkflowName})

		runPostHookActivity := cluster.NewRunPostHookActivity(clusterManager)
		activity.RegisterWithOptions(runPostHookActivity.Execute, activity.RegisterOptions{Name: cluster.RunPostHookActivityName})

		updateClusterStatusActivity := cluster.NewUpdateClusterStatusActivity(clusterManager)
		activity.RegisterWithOptions(updateClusterStatusActivity.Execute, activity.RegisterOptions{Name: cluster.UpdateClusterStatusActivityName})

		// Register amazon specific workflows and activities
		registerAwsWorkflows(clusters, tokenGenerator)

		azurePKEClusterStore := azurePKEAdapter.NewGORMAzurePKEClusterStore(db, commonadapter.NewLogger(logger))

		{
			passwordSecrets := intpkeworkflowadapter.NewPasswordSecretStore(commonSecretStore)
			registerPKEWorkflows(passwordSecrets)
		}

		// Register azure specific workflows
		registerAzureWorkflows(secretStore, tokenGenerator, azurePKEClusterStore)

		// Register EKS specific workflows
		err = registerEKSWorkflows(secret.Store)
		if err != nil {
			emperror.Panic(errors.WrapIf(err, "failed to register EKS workflows"))
		}

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

		deleteUnusedClusterSecretsActivity := intClusterWorkflow.MakeDeleteUnusedClusterSecretsActivity(secret.Store)
		activity.RegisterWithOptions(deleteUnusedClusterSecretsActivity.Execute, activity.RegisterOptions{Name: intClusterWorkflow.DeleteUnusedClusterSecretsActivityName})

		workflow.RegisterWithOptions(intClusterWorkflow.DeleteK8sResourcesWorkflow, workflow.RegisterOptions{Name: intClusterWorkflow.DeleteK8sResourcesWorkflowName})

		k8sConfigGetter := intSecret.MakeKubeSecretStore(secret.Store)

		deleteHelmDeploymentsActivity := intClusterWorkflow.MakeDeleteHelmDeploymentsActivity(k8sConfigGetter, logrusLogger)
		activity.RegisterWithOptions(deleteHelmDeploymentsActivity.Execute, activity.RegisterOptions{Name: intClusterWorkflow.DeleteHelmDeploymentsActivityName})

		deleteUserNamespacesActivity := intClusterWorkflow.MakeDeleteUserNamespacesActivity(intClusterK8s.MakeUserNamespaceDeleter(logrusLogger), k8sConfigGetter)
		activity.RegisterWithOptions(deleteUserNamespacesActivity.Execute, activity.RegisterOptions{Name: intClusterWorkflow.DeleteUserNamespacesActivityName})

		deleteNamespaceResourcesActivity := intClusterWorkflow.MakeDeleteNamespaceResourcesActivity(intClusterK8s.MakeNamespaceResourcesDeleter(logrusLogger), k8sConfigGetter)
		activity.RegisterWithOptions(deleteNamespaceResourcesActivity.Execute, activity.RegisterOptions{Name: intClusterWorkflow.DeleteNamespaceResourcesActivityName})

		deleteNamespaceServicesActivity := intClusterWorkflow.MakeDeleteNamespaceServicesActivity(intClusterK8s.MakeNamespaceServicesDeleter(logrusLogger), k8sConfigGetter)
		activity.RegisterWithOptions(deleteNamespaceServicesActivity.Execute, activity.RegisterOptions{Name: intClusterWorkflow.DeleteNamespaceServicesActivityName})

		clusterDNSRecordsDeleter, err := intClusterDNS.MakeDefaultRecordsDeleter()
		emperror.Panic(emperror.Wrap(err, "failed to create default cluster DNS records deleter"))

		deleteClusterDNSRecordsActivity := intClusterWorkflow.MakeDeleteClusterDNSRecordsActivity(clusterDNSRecordsDeleter)
		activity.RegisterWithOptions(deleteClusterDNSRecordsActivity.Execute, activity.RegisterOptions{Name: intClusterWorkflow.DeleteClusterDNSRecordsActivityName})

		waitPersistentVolumesDeletionActivity := intClusterWorkflow.MakeWaitPersistentVolumesDeletionActivity(k8sConfigGetter, logrusLogger)
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

			orgGetter := authdriver.NewOrganizationGetter(db)

			logger := commonadapter.NewLogger(logger) // TODO: make this a context aware logger
			featureRepository := clusterfeatureadapter.NewGormFeatureRepository(db, logger)
			kubernetesService := kubernetes.NewService(
				kubernetesadapter.NewConfigSecretGetter(clusteradapter.NewClusters(db)),
				kubernetes.NewConfigFactory(commonSecretStore),
				logger,
			)

			clusterGetter := clusterfeatureadapter.MakeClusterGetter(clusterManager)
			clusterService := clusterfeatureadapter.NewClusterService(clusterManager)
			endpointManager := endpoints.NewEndpointManager(logger)
			orgDomainService := dnsadapter.NewOrgDomainService(
				config.Cluster.DNS.BaseDomain,
				dnsSvc,
				dnsadapter.NewClusterOrgGetter(clusterManager, orgGetter),
				logger,
			)

			customAnchoreConfigProvider := securityscan.NewCustomAnchoreConfigProvider(
				featureRepository,
				commonSecretStore,
				logger,
			)
			configProvider := anchore2.ConfigProviderChain{customAnchoreConfigProvider}

			if config.Cluster.SecurityScan.Anchore.Enabled {
				configProvider = append(configProvider, anchore2.StaticConfigProvider{
					Config: config.Cluster.SecurityScan.Anchore.Config,
				})
			}

			anchoreUserService := anchore.MakeAnchoreUserService(
				configProvider,
				securityscanadapter.NewUserNameGenerator(securityscanadapter.NewClusterService(clusterManager)),
				commonSecretStore,
				logger,
			)
			featureAnchoreService := securityscan.NewFeatureAnchoreService(anchoreUserService, logger)
			featureWhitelistService := securityscan.NewFeatureWhitelistService(clusterGetter, anchore.NewSecurityResourceService(logger), logger)

			featureOperatorRegistry := clusterfeature.MakeFeatureOperatorRegistry([]clusterfeature.FeatureOperator{
				featureDns.MakeFeatureOperator(
					clusterGetter,
					clusterService,
					helmService,
					logger,
					orgDomainService,
					commonSecretStore,
					config.Cluster.DNS.Config,
				),
				securityscan.MakeFeatureOperator(
					config.Cluster.SecurityScan.Anchore.Enabled,
					config.Cluster.SecurityScan.Anchore.Endpoint,
					clusterGetter,
					clusterService,
					helmService,
					commonSecretStore,
					featureAnchoreService,
					featureWhitelistService,
					emperror.MakeContextAware(errorHandler),
					logger,
				),
				featureVault.MakeFeatureOperator(clusterGetter,
					clusterService,
					helmService,
					kubernetesService,
					commonSecretStore,
					config.Cluster.Vault.Config,
					logger,
				),
				featureMonitoring.MakeFeatureOperator(
					clusterGetter,
					clusterService,
					helmService,
					kubernetesService,
					config.Cluster.Monitoring.Config,
					logger,
					commonSecretStore,
				),
				featureLogging.MakeFeatureOperator(
					clusterGetter,
					clusterService,
					helmService,
					kubernetesService,
					endpointManager,
					config.Cluster.Logging.Config,
					logger,
					commonSecretStore,
				),
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
