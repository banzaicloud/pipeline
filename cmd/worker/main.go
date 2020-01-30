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

	cloudinfoapi "github.com/banzaicloud/pipeline/.gen/cloudinfo"
	anchore2 "github.com/banzaicloud/pipeline/internal/anchore"
	cluster2 "github.com/banzaicloud/pipeline/internal/cluster"
	intClusterAuth "github.com/banzaicloud/pipeline/internal/cluster/auth"
	"github.com/banzaicloud/pipeline/internal/cluster/clusteradapter"
	"github.com/banzaicloud/pipeline/internal/cluster/clustersecret"
	"github.com/banzaicloud/pipeline/internal/cluster/clustersecret/clustersecretadapter"
	"github.com/banzaicloud/pipeline/internal/cluster/clustersetup"
	"github.com/banzaicloud/pipeline/internal/cluster/clusterworkflow"
	"github.com/banzaicloud/pipeline/internal/cluster/distributionadapter"
	intClusterDNS "github.com/banzaicloud/pipeline/internal/cluster/dns"
	"github.com/banzaicloud/pipeline/internal/cluster/endpoints"
	intClusterK8s "github.com/banzaicloud/pipeline/internal/cluster/kubernetes"
	intClusterWorkflow "github.com/banzaicloud/pipeline/internal/cluster/workflow"
	"github.com/banzaicloud/pipeline/internal/clustergroup"
	cgroupAdapter "github.com/banzaicloud/pipeline/internal/clustergroup/adapter"
	"github.com/banzaicloud/pipeline/internal/clustergroup/deployment"
	"github.com/banzaicloud/pipeline/internal/common/commonadapter"
	"github.com/banzaicloud/pipeline/internal/federation"
	"github.com/banzaicloud/pipeline/internal/global"
	"github.com/banzaicloud/pipeline/internal/helm"
	"github.com/banzaicloud/pipeline/internal/helm/helmadapter"
	"github.com/banzaicloud/pipeline/internal/integratedservices"
	"github.com/banzaicloud/pipeline/internal/integratedservices/integratedserviceadapter"
	"github.com/banzaicloud/pipeline/internal/integratedservices/services"
	integratedServiceDNS "github.com/banzaicloud/pipeline/internal/integratedservices/services/dns"
	"github.com/banzaicloud/pipeline/internal/integratedservices/services/dns/dnsadapter"
	"github.com/banzaicloud/pipeline/internal/integratedservices/services/expiry"
	"github.com/banzaicloud/pipeline/internal/integratedservices/services/expiry/adapter"
	expiryWorkflow "github.com/banzaicloud/pipeline/internal/integratedservices/services/expiry/adapter/workflow"
	integratedServiceLogging "github.com/banzaicloud/pipeline/internal/integratedservices/services/logging"
	integratedServiceMonitoring "github.com/banzaicloud/pipeline/internal/integratedservices/services/monitoring"
	"github.com/banzaicloud/pipeline/internal/integratedservices/services/securityscan"
	"github.com/banzaicloud/pipeline/internal/integratedservices/services/securityscan/securityscanadapter"
	integratedServiceVault "github.com/banzaicloud/pipeline/internal/integratedservices/services/vault"
	cgFeatureIstio "github.com/banzaicloud/pipeline/internal/istio/istiofeature"
	"github.com/banzaicloud/pipeline/internal/kubernetes"
	"github.com/banzaicloud/pipeline/internal/kubernetes/kubernetesadapter"
	intpkeworkflowadapter "github.com/banzaicloud/pipeline/internal/pke/workflow/adapter"
	"github.com/banzaicloud/pipeline/internal/platform/buildinfo"
	"github.com/banzaicloud/pipeline/internal/platform/cadence"
	"github.com/banzaicloud/pipeline/internal/platform/database"
	"github.com/banzaicloud/pipeline/internal/platform/errorhandler"
	"github.com/banzaicloud/pipeline/internal/platform/log"
	eksClusterAdapter "github.com/banzaicloud/pipeline/internal/providers/amazon/eks/adapter"
	eksClusterDriver "github.com/banzaicloud/pipeline/internal/providers/amazon/eks/driver"
	eksworkflow "github.com/banzaicloud/pipeline/internal/providers/amazon/eks/workflow"
	azurePKEAdapter "github.com/banzaicloud/pipeline/internal/providers/azure/pke/adapter"
	azurepkedriver "github.com/banzaicloud/pipeline/internal/providers/azure/pke/driver"
	"github.com/banzaicloud/pipeline/internal/providers/pke/pkeworkflow"
	"github.com/banzaicloud/pipeline/internal/providers/pke/pkeworkflow/pkeworkflowadapter"
	intSecret "github.com/banzaicloud/pipeline/internal/secret"
	anchore "github.com/banzaicloud/pipeline/internal/security"
	pkgAuth "github.com/banzaicloud/pipeline/pkg/auth"
	"github.com/banzaicloud/pipeline/pkg/cloudinfo"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	"github.com/banzaicloud/pipeline/src/auth"
	"github.com/banzaicloud/pipeline/src/auth/authdriver"
	"github.com/banzaicloud/pipeline/src/cluster"
	legacyclusteradapter "github.com/banzaicloud/pipeline/src/cluster/clusteradapter"
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
	v := viper.NewWithOptions(
		viper.KeyDelimiter("::"),
	)
	p := pflag.NewFlagSet(friendlyAppName, pflag.ExitOnError)

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

		workflowClient, err := cadence.NewClient(config.Cadence, zaplog.New(logur.WithFields(logger, map[string]interface{}{"component": "cadence-client"})))
		if err != nil {
			errorHandler.Handle(errors.WrapIf(err, "Failed to configure Cadence client"))
		}

		clusterRepo := clusteradapter.NewClusters(db)
		clusterManager := cluster.NewManager(
			clusterRepo,
			nil,
			nil,
			nil,
			nil,
			workflowClient,
			logrusLogger,
			errorHandler,
			clusteradapter.NewStore(db, clusterRepo),
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

		eksClusters := eksClusterAdapter.NewClusterManagerAdapter(clusterManager)

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

		runPostHookActivity := cluster.NewRunPostHookActivity(clusterManager, config.Cluster.PostHook)
		activity.RegisterWithOptions(runPostHookActivity.Execute, activity.RegisterOptions{Name: cluster.RunPostHookActivityName})

		updateClusterStatusActivity := cluster.NewUpdateClusterStatusActivity(clusterManager)
		activity.RegisterWithOptions(updateClusterStatusActivity.Execute, activity.RegisterOptions{Name: cluster.UpdateClusterStatusActivityName})

		cloudinfoClient := cloudinfo.NewClient(cloudinfoapi.NewAPIClient(&cloudinfoapi.Configuration{
			BasePath:      config.Cloudinfo.Endpoint,
			DefaultHeader: make(map[string]string),
			UserAgent:     fmt.Sprintf("Pipeline/%s", version),
		}))

		// Register amazon specific workflows and activities
		registerAwsWorkflows(clusters, tokenGenerator, secretStore, cloudinfoClient)

		azurePKEClusterStore := azurePKEAdapter.NewClusterStore(db, commonadapter.NewLogger(logger))

		{
			passwordSecrets := intpkeworkflowadapter.NewPasswordSecretStore(commonSecretStore)
			registerPKEWorkflows(passwordSecrets)
		}

		// Register azure specific workflows
		registerAzureWorkflows(secretStore, tokenGenerator, azurePKEClusterStore)

		// Register EKS specific workflows
		err = registerEKSWorkflows(secret.Store, eksClusters)
		if err != nil {
			emperror.Panic(errors.WrapIf(err, "failed to register EKS workflows"))
		}

		clusterStore := clusteradapter.NewStore(db, clusteradapter.NewClusters(db))

		{
			workflow.RegisterWithOptions(clusterworkflow.DeleteClusterWorkflow, workflow.RegisterOptions{Name: clusterworkflow.DeleteClusterWorkflowName})

			cgroupAdapter := cgroupAdapter.NewClusterGetter(clusterManager)
			clusterGroupManager := clustergroup.NewManager(cgroupAdapter, clustergroup.NewClusterGroupRepository(db, logrusLogger), logrusLogger, errorHandler)
			federationHandler := federation.NewFederationHandler(cgroupAdapter, global.Config.Cluster.Namespace, logrusLogger, errorHandler)
			deploymentManager := deployment.NewCGDeploymentManager(db, cgroupAdapter, logrusLogger, errorHandler)
			serviceMeshFeatureHandler := cgFeatureIstio.NewServiceMeshFeatureHandler(cgroupAdapter, logrusLogger, errorHandler)
			clusterGroupManager.RegisterFeatureHandler(federation.FeatureName, federationHandler)
			clusterGroupManager.RegisterFeatureHandler(deployment.FeatureName, deploymentManager)
			clusterGroupManager.RegisterFeatureHandler(cgFeatureIstio.FeatureName, serviceMeshFeatureHandler)

			removeClusterFromGroupActivity := clusterworkflow.MakeRemoveClusterFromGroupActivity(clusterGroupManager)
			activity.RegisterWithOptions(removeClusterFromGroupActivity.Execute, activity.RegisterOptions{Name: clusterworkflow.RemoveClusterFromGroupActivityName})

			commonClusterDeleter := legacyclusteradapter.NewCommonClusterDeleterAdapter(
				clusterManager,
				clusterManager,
			)
			deleteClusterActivity := clusterworkflow.MakeDeleteClusterActivity(
				clusteradapter.NewPolyClusterDeleter(
					clusterStore,
					clusteradapter.ClusterDeleterEntry{
						Key:     clusteradapter.MakeClusterDeleterKey(pkgCluster.Alibaba, pkgCluster.ACK),
						Deleter: commonClusterDeleter,
					},
					clusteradapter.ClusterDeleterEntry{
						Key: clusteradapter.MakeClusterDeleterKey(pkgCluster.Amazon, pkgCluster.EKS),
						Deleter: eksClusterDriver.NewEKSClusterDeleter(
							nil,
							clusterManager.GetKubeProxyCache(),
							logrusLogger,
							secret.Store,
							nil,
							workflowClient,
							clusterManager,
						),
					},
					clusteradapter.ClusterDeleterEntry{
						Key:     clusteradapter.MakeClusterDeleterKey(pkgCluster.Amazon, pkgCluster.PKE),
						Deleter: commonClusterDeleter,
					},
					clusteradapter.ClusterDeleterEntry{
						Key:     clusteradapter.MakeClusterDeleterKey(pkgCluster.Azure, pkgCluster.AKS),
						Deleter: commonClusterDeleter,
					},
					clusteradapter.ClusterDeleterEntry{
						Key: clusteradapter.MakeClusterDeleterKey(pkgCluster.Azure, pkgCluster.PKE),
						Deleter: azurepkedriver.MakeClusterDeleter(
							nil,
							clusterManager.GetKubeProxyCache(),
							logrusLogger,
							secret.Store,
							nil,
							azurePKEClusterStore,
							workflowClient,
						),
					},
					clusteradapter.ClusterDeleterEntry{
						Key:     clusteradapter.MakeClusterDeleterKey(pkgCluster.Google, pkgCluster.GKE),
						Deleter: commonClusterDeleter,
					},
					clusteradapter.ClusterDeleterEntry{
						Key:     clusteradapter.MakeClusterDeleterKey(pkgCluster.Kubernetes, pkgCluster.Unknown),
						Deleter: commonClusterDeleter,
					},
					clusteradapter.ClusterDeleterEntry{
						Key:     clusteradapter.MakeClusterDeleterKey(pkgCluster.Oracle, pkgCluster.OKE),
						Deleter: commonClusterDeleter,
					},
				),
			)
			activity.RegisterWithOptions(deleteClusterActivity.Execute, activity.RegisterOptions{Name: clusterworkflow.DeleteClusterActivityName})
		}

		{
			workflow.RegisterWithOptions(clusterworkflow.DeleteNodePoolWorkflow, workflow.RegisterOptions{Name: clusterworkflow.DeleteNodePoolWorkflowName})

			createNodePoolActivity := clusterworkflow.NewCreateNodePoolActivity(
				clusterStore,
				db,
				clusteradapter.NewNodePoolStore(db, clusterStore),
				distributionadapter.NewEKSNodePoolStore(db),
				eksworkflow.NewAWSSessionFactory(secret.Store),
			)
			activity.RegisterWithOptions(createNodePoolActivity.Execute, activity.RegisterOptions{Name: clusterworkflow.CreateNodePoolActivityName})

			createNodePoolLabelSetActivity := clusterworkflow.NewCreateNodePoolLabelSetActivity(
				cluster2.NewDynamicClientFactory(clusterStore, kubernetes.NewDynamicClientFactory(configFactory)),
				config.Cluster.Labels.Namespace,
			)
			activity.RegisterWithOptions(createNodePoolLabelSetActivity.Execute, activity.RegisterOptions{Name: clusterworkflow.CreateNodePoolLabelSetActivityName})

			workflow.RegisterWithOptions(clusterworkflow.CreateNodePoolWorkflow, workflow.RegisterOptions{Name: clusterworkflow.CreateNodePoolWorkflowName})

			deleteNodePoolActivity := clusterworkflow.NewDeleteNodePoolActivity(
				clusterStore,
				clusteradapter.NewNodePoolStore(db, clusterStore),
				eksworkflow.NewAWSSessionFactory(secret.Store),
			)
			activity.RegisterWithOptions(deleteNodePoolActivity.Execute, activity.RegisterOptions{Name: clusterworkflow.DeleteNodePoolActivityName})

			deleteNodePoolLabelSetActivity := clusterworkflow.NewDeleteNodePoolLabelSetActivity(
				cluster2.NewDynamicClientFactory(clusterStore, kubernetes.NewDynamicClientFactory(configFactory)),
				config.Cluster.Labels.Namespace,
			)
			activity.RegisterWithOptions(deleteNodePoolLabelSetActivity.Execute, activity.RegisterOptions{Name: clusterworkflow.DeleteNodePoolLabelSetActivityName})

			setClusterStatusActivity := clusterworkflow.NewSetClusterStatusActivity(clusterStore)
			activity.RegisterWithOptions(setClusterStatusActivity.Execute, activity.RegisterOptions{Name: clusterworkflow.SetClusterStatusActivityName})
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
		emperror.Panic(errors.WrapIf(err, "failed to create default cluster DNS records deleter"))

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
			featureRepository := integratedserviceadapter.NewGormIntegratedServiceRepository(db, logger)
			kubernetesService := kubernetes.NewService(
				kubernetesadapter.NewConfigSecretGetter(clusteradapter.NewClusters(db)),
				kubernetes.NewConfigFactory(commonSecretStore),
				logger,
			)

			clusterGetter := integratedserviceadapter.MakeClusterGetter(clusterManager)
			clusterService := integratedserviceadapter.NewClusterService(clusterManager)
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
			featureAnchoreService := securityscan.NewIntegratedServiceAnchoreService(anchoreUserService, logger)
			featureWhitelistService := securityscan.NewIntegratedServiceWhitelistService(clusterGetter, anchore.NewSecurityResourceService(logger), logger)

			// expiry integrated service
			workflow.RegisterWithOptions(expiryWorkflow.ExpiryJobWorkflow, workflow.RegisterOptions{Name: expiryWorkflow.ExpiryJobWorkflowName})

			clusterDeleter := clusteradapter.NewCadenceClusterManager(workflowClient)
			expiryActivity := expiryWorkflow.NewExpiryActivity(clusterDeleter)
			activity.RegisterWithOptions(expiryActivity.Execute, activity.RegisterOptions{Name: expiryWorkflow.ExpireActivityName})

			expirerService := adapter.NewAsyncExpiryService(workflowClient, logger)

			featureOperatorRegistry := integratedservices.MakeIntegratedServiceOperatorRegistry([]integratedservices.IntegratedServiceOperator{
				integratedServiceDNS.MakeIntegratedServiceOperator(
					clusterGetter,
					clusterService,
					helmService,
					logger,
					orgDomainService,
					commonSecretStore,
					config.Cluster.DNS.Config,
				),
				securityscan.MakeIntegratedServiceOperator(
					config.Cluster.SecurityScan.Anchore.Enabled,
					config.Cluster.SecurityScan.Anchore.Endpoint,
					clusterGetter,
					clusterService,
					helmService,
					commonSecretStore,
					featureAnchoreService,
					featureWhitelistService,
					errorHandler,
					logger,
				),
				integratedServiceVault.MakeIntegratedServicesOperator(clusterGetter,
					clusterService,
					helmService,
					kubernetesService,
					commonSecretStore,
					config.Cluster.Vault.Config,
					logger,
				),
				integratedServiceMonitoring.MakeIntegratedServiceOperator(
					clusterGetter,
					clusterService,
					helmService,
					kubernetesService,
					config.Cluster.Monitoring.Config,
					logger,
					commonSecretStore,
				),
				integratedServiceLogging.MakeIntegratedServicesOperator(
					clusterGetter,
					clusterService,
					helmService,
					kubernetesService,
					endpointManager,
					config.Cluster.Logging.Config,
					logger,
					commonSecretStore,
				),
				expiry.NewExpiryServiceOperator(expirerService, services.BindIntegratedServiceSpec, logger),
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
