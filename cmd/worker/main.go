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
	"context"
	"encoding/base32"
	"fmt"
	"os"
	"syscall"
	"text/template"

	"emperror.dev/emperror"
	"emperror.dev/errors"
	"emperror.dev/errors/match"
	bauth "github.com/banzaicloud/bank-vaults/pkg/sdk/auth"
	"github.com/banzaicloud/bank-vaults/pkg/sdk/vault"
	"github.com/mitchellh/mapstructure"
	"github.com/oklog/run"
	appkitrun "github.com/sagikazarmark/appkit/run"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"go.uber.org/cadence/activity"
	"go.uber.org/cadence/workflow"
	zaplog "logur.dev/integration/zap"
	"logur.dev/logur"

	"github.com/banzaicloud/pipeline/internal/helm/helmadapter"

	cloudinfoapi "github.com/banzaicloud/pipeline/.gen/cloudinfo"
	anchore2 "github.com/banzaicloud/pipeline/internal/anchore"
	"github.com/banzaicloud/pipeline/internal/app/pipeline/process"
	"github.com/banzaicloud/pipeline/internal/app/pipeline/process/processadapter"
	cluster2 "github.com/banzaicloud/pipeline/internal/cluster"
	intClusterAuth "github.com/banzaicloud/pipeline/internal/cluster/auth"
	"github.com/banzaicloud/pipeline/internal/cluster/clusteradapter"
	"github.com/banzaicloud/pipeline/internal/cluster/clustersecret"
	"github.com/banzaicloud/pipeline/internal/cluster/clustersecret/clustersecretadapter"
	"github.com/banzaicloud/pipeline/internal/cluster/clustersetup"
	"github.com/banzaicloud/pipeline/internal/cluster/clusterworkflow"
	"github.com/banzaicloud/pipeline/internal/cluster/distribution/eks/eksadapter"
	eksClusterAdapter "github.com/banzaicloud/pipeline/internal/cluster/distribution/eks/eksprovider/adapter"
	eksClusterDriver "github.com/banzaicloud/pipeline/internal/cluster/distribution/eks/eksprovider/driver"
	eksworkflow "github.com/banzaicloud/pipeline/internal/cluster/distribution/eks/eksprovider/workflow"
	intClusterDNS "github.com/banzaicloud/pipeline/internal/cluster/dns"
	"github.com/banzaicloud/pipeline/internal/cluster/endpoints"
	intClusterK8s "github.com/banzaicloud/pipeline/internal/cluster/kubernetes"
	intClusterWorkflow "github.com/banzaicloud/pipeline/internal/cluster/workflow"
	"github.com/banzaicloud/pipeline/internal/clustergroup"
	cgroupAdapter "github.com/banzaicloud/pipeline/internal/clustergroup/adapter"
	"github.com/banzaicloud/pipeline/internal/clustergroup/deployment"
	"github.com/banzaicloud/pipeline/internal/cmd"
	"github.com/banzaicloud/pipeline/internal/common/commonadapter"
	"github.com/banzaicloud/pipeline/internal/federation"
	"github.com/banzaicloud/pipeline/internal/global"
	"github.com/banzaicloud/pipeline/internal/helm"
	"github.com/banzaicloud/pipeline/internal/integratedservices"
	"github.com/banzaicloud/pipeline/internal/integratedservices/integratedserviceadapter"
	"github.com/banzaicloud/pipeline/internal/integratedservices/services"
	integratedServiceDNS "github.com/banzaicloud/pipeline/internal/integratedservices/services/dns"
	"github.com/banzaicloud/pipeline/internal/integratedservices/services/dns/dnsadapter"
	"github.com/banzaicloud/pipeline/internal/integratedservices/services/expiry"
	"github.com/banzaicloud/pipeline/internal/integratedservices/services/expiry/adapter"
	expiryWorkflow "github.com/banzaicloud/pipeline/internal/integratedservices/services/expiry/adapter/workflow"
	intsvcingress "github.com/banzaicloud/pipeline/internal/integratedservices/services/ingress"
	intsvcingressadapter "github.com/banzaicloud/pipeline/internal/integratedservices/services/ingress/ingressadapter"
	integratedServiceLogging "github.com/banzaicloud/pipeline/internal/integratedservices/services/logging"
	integratedServiceMonitoring "github.com/banzaicloud/pipeline/internal/integratedservices/services/monitoring"
	"github.com/banzaicloud/pipeline/internal/integratedservices/services/securityscan"
	"github.com/banzaicloud/pipeline/internal/integratedservices/services/securityscan/securityscanadapter"
	integratedServiceVault "github.com/banzaicloud/pipeline/internal/integratedservices/services/vault"
	cgFeatureIstio "github.com/banzaicloud/pipeline/internal/istio/istiofeature"
	"github.com/banzaicloud/pipeline/internal/kubernetes"
	"github.com/banzaicloud/pipeline/internal/kubernetes/kubernetesadapter"
	intpkeworkflowadapter "github.com/banzaicloud/pipeline/internal/pke/workflow/adapter"
	"github.com/banzaicloud/pipeline/internal/platform/appkit"
	"github.com/banzaicloud/pipeline/internal/platform/buildinfo"
	"github.com/banzaicloud/pipeline/internal/platform/cadence"
	"github.com/banzaicloud/pipeline/internal/platform/database"
	"github.com/banzaicloud/pipeline/internal/platform/errorhandler"
	"github.com/banzaicloud/pipeline/internal/platform/log"
	azurePKEAdapter "github.com/banzaicloud/pipeline/internal/providers/azure/pke/adapter"
	azurepkedriver "github.com/banzaicloud/pipeline/internal/providers/azure/pke/driver"
	"github.com/banzaicloud/pipeline/internal/providers/pke/pkeworkflow"
	"github.com/banzaicloud/pipeline/internal/providers/pke/pkeworkflow/pkeworkflowadapter"
	vsphereadapter "github.com/banzaicloud/pipeline/internal/providers/vsphere/pke/adapter"
	vspheredriver "github.com/banzaicloud/pipeline/internal/providers/vsphere/pke/driver"
	"github.com/banzaicloud/pipeline/internal/secret/kubesecret"
	"github.com/banzaicloud/pipeline/internal/secret/pkesecret"
	"github.com/banzaicloud/pipeline/internal/secret/restricted"
	"github.com/banzaicloud/pipeline/internal/secret/secretadapter"
	"github.com/banzaicloud/pipeline/internal/secret/types"
	anchore "github.com/banzaicloud/pipeline/internal/security"
	pkgAuth "github.com/banzaicloud/pipeline/pkg/auth"
	"github.com/banzaicloud/pipeline/pkg/cloudinfo"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	"github.com/banzaicloud/pipeline/pkg/hook"
	"github.com/banzaicloud/pipeline/src/auth"
	"github.com/banzaicloud/pipeline/src/auth/authdriver"
	"github.com/banzaicloud/pipeline/src/cluster"
	legacyclusteradapter "github.com/banzaicloud/pipeline/src/cluster/clusteradapter"
	"github.com/banzaicloud/pipeline/src/dns"
	"github.com/banzaicloud/pipeline/src/secret"
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
	err = v.Unmarshal(&config, hook.DecodeHookWithDefaults())
	emperror.Panic(errors.Wrap(err, "failed to unmarshal configuration"))

	err = config.Process()
	emperror.Panic(errors.WithMessage(err, "failed to process configuration"))

	err = mapstructure.Decode(config, &global.Config)
	emperror.Panic(errors.Wrap(err, "failed to bind configuration to global configuration"))

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

	commonLogger := commonadapter.NewContextAwareLogger(logger, appkit.ContextExtractor)

	vaultClient, err := vault.NewClient("pipeline")
	emperror.Panic(err)
	global.SetVault(vaultClient)

	secretStore := secretadapter.NewVaultStore(vaultClient, "secret")
	pkeSecreter := pkesecret.NewPkeSecreter(vaultClient, commonLogger)
	secretTypes := types.NewDefaultTypeList(types.DefaultTypeListConfig{
		TLSDefaultValidity: config.Secret.TLS.DefaultValidity,
		PkeSecreter:        pkeSecreter,
	})
	secret.InitSecretStore(secretStore, secretTypes)
	restricted.InitSecretStore(secret.Store)

	var group run.Group

	// Configure Cadence worker
	{
		const taskList = "pipeline"
		worker, err := cadence.NewWorker(config.Cadence, taskList, zaplog.New(logur.WithFields(logger, map[string]interface{}{"component": "cadence-worker"})))
		emperror.Panic(err)

		db, err := database.Connect(config.Database.Config)
		if err != nil {
			emperror.Panic(err)
		}
		global.SetDB(db)

		workflowClient, err := cadence.NewClient(config.Cadence, zaplog.New(logur.WithFields(logger, map[string]interface{}{"component": "cadence-client"})))
		if err != nil {
			errorHandler.Handle(errors.WrapIf(err, "Failed to configure Cadence client"))
		}

		commonSecretStore := commonadapter.NewSecretStore(secret.Store, commonadapter.OrgIDContextExtractorFunc(auth.GetCurrentOrganizationID))

		releaseDeleter := cmd.CreateReleaseDeleter(config.Helm, db, commonSecretStore, commonLogger)
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
			releaseDeleter,
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

		orgService := helmadapter.NewOrgService(commonLogger)
		unifiedHelmReleaser, helmFacade := cmd.CreateUnifiedHelmReleaser(
			config.Helm,
			db,
			commonSecretStore,
			helm.ClusterKubeConfigFunc(clusterManager.KubeConfigFunc()),
			orgService,
			commonLogger,
		)

		clusters := pkeworkflowadapter.NewClusterManagerAdapter(clusterManager)
		secretStore := pkeworkflowadapter.NewSecretStore(secret.Store)

		clusterSecretStore := clustersecret.NewStore(
			clustersecretadapter.NewClusterManagerAdapter(clusterManager),
			clustersecretadapter.NewSecretStore(secret.Store),
		)

		eksClusters := eksClusterAdapter.NewClusterManagerAdapter(clusterManager)

		clusterAuthService, err := intClusterAuth.NewDexClusterAuthService(clusterSecretStore)
		emperror.Panic(errors.Wrap(err, "failed to create DexClusterAuthService"))

		configFactory := kubernetes.NewConfigFactory(commonSecretStore)

		processService := process.NewService(processadapter.NewGormStore(db), workflowClient)
		processActivity := process.NewProcessActivity(processService)

		activity.RegisterWithOptions(processActivity.ExecuteProcess, activity.RegisterOptions{Name: process.ProcessActivityName})
		activity.RegisterWithOptions(processActivity.ExecuteProcessEvent, activity.RegisterOptions{Name: process.ProcessEventActivityName})

		// Cluster setup
		{
			wf := clustersetup.Workflow{
				InstallInitManifest: config.Cluster.Manifest != "",
				HelmV3:              config.Helm.V3,
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

			if !config.Helm.V3 {
				activity.RegisterWithOptions(installTillerActivity.Execute, activity.RegisterOptions{Name: clustersetup.InstallTillerActivityName})

				installTillerWaitActivity := clustersetup.NewInstallTillerWaitActivity(
					config.Helm.Tiller.Version,
					kubernetes.NewHelmClientFactory(configFactory, commonadapter.NewLogger(logger)),
				)
				activity.RegisterWithOptions(installTillerWaitActivity.Execute, activity.RegisterOptions{Name: clustersetup.InstallTillerWaitActivityName})
			}
			installNodePoolLabelSetOperatorActivity := clustersetup.NewInstallNodePoolLabelSetOperatorActivity(
				config.Cluster.Labels,
				unifiedHelmReleaser,
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

		runPostHookActivity := cluster.NewRunPostHookActivity(clusterManager, unifiedHelmReleaser)
		activity.RegisterWithOptions(runPostHookActivity.Execute, activity.RegisterOptions{Name: cluster.RunPostHookActivityName})

		updateClusterStatusActivity := cluster.NewUpdateClusterStatusActivity(clusterManager)
		activity.RegisterWithOptions(updateClusterStatusActivity.Execute, activity.RegisterOptions{Name: cluster.UpdateClusterStatusActivityName})

		cloudinfoClient := cloudinfo.NewClient(cloudinfoapi.NewAPIClient(&cloudinfoapi.Configuration{
			BasePath:      config.Cloudinfo.Endpoint,
			DefaultHeader: make(map[string]string),
			UserAgent:     fmt.Sprintf("Pipeline/%s", version),
		}))

		// Register amazon specific workflows and activities
		registerAwsWorkflows(clusters, tokenGenerator, secretStore, cloudinfoClient, config.Distribution.PKE.Amazon.GlobalRegion)

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
		vsphereClusterStore := vsphereadapter.NewClusterStore(db)

		cgroupAdapter := cgroupAdapter.NewClusterGetter(clusterManager)
		clusterGroupManager := clustergroup.NewManager(cgroupAdapter, clustergroup.NewClusterGroupRepository(db, logrusLogger), logrusLogger, errorHandler)
		{
			workflow.RegisterWithOptions(clusterworkflow.DeleteClusterWorkflow, workflow.RegisterOptions{Name: clusterworkflow.DeleteClusterWorkflowName})

			federationHandler := federation.NewFederationHandler(cgroupAdapter, config.Cluster.Namespace, logrusLogger, errorHandler, config.Cluster.Federation, config.Cluster.DNS.Config, unifiedHelmReleaser)
			deploymentManager := deployment.NewCGDeploymentManager(db, cgroupAdapter, logrusLogger, errorHandler, deployment.NewHelmService(helmFacade, unifiedHelmReleaser))
			serviceMeshFeatureHandler := cgFeatureIstio.NewServiceMeshFeatureHandler(cgroupAdapter, logrusLogger, errorHandler, config.Cluster.Backyards, unifiedHelmReleaser)
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
					clusteradapter.ClusterDeleterEntry{
						Key: clusteradapter.MakeClusterDeleterKey(pkgCluster.Vsphere, pkgCluster.PKE),
						Deleter: vspheredriver.MakeClusterDeleter(
							nil,
							clusterManager.GetKubeProxyCache(),
							commonLogger,
							secret.Store,
							nil,
							vsphereClusterStore,
							workflowClient,
						),
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
				eksadapter.NewNodePoolStore(db),
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

		k8sConfigGetter := kubesecret.MakeKubeSecretStore(secret.Store)

		// Register vsphere specific workflows

		registerVsphereWorkflows(secretStore, tokenGenerator, vsphereClusterStore, k8sConfigGetter)

		generateCertificatesActivity := pkeworkflow.NewGenerateCertificatesActivity(clusterSecretStore)
		activity.RegisterWithOptions(generateCertificatesActivity.Execute, activity.RegisterOptions{Name: pkeworkflow.GenerateCertificatesActivityName})

		createDexClientActivity := pkeworkflow.NewCreateDexClientActivity(clusters, clusterAuthService)
		activity.RegisterWithOptions(createDexClientActivity.Execute, activity.RegisterOptions{Name: pkeworkflow.CreateDexClientActivityName})

		deleteDexClientActivity := pkeworkflow.NewDeleteDexClientActivity(clusters, clusterAuthService)
		activity.RegisterWithOptions(deleteDexClientActivity.Execute, activity.RegisterOptions{Name: pkeworkflow.DeleteDexClientActivityName})

		setMasterTaintActivity := pkeworkflow.NewSetMasterTaintActivity(clusters)
		activity.RegisterWithOptions(setMasterTaintActivity.Execute, activity.RegisterOptions{Name: pkeworkflow.SetMasterTaintActivityName})

		deleteUnusedClusterSecretsActivity := intClusterWorkflow.MakeDeleteUnusedClusterSecretsActivity(secret.Store)
		activity.RegisterWithOptions(deleteUnusedClusterSecretsActivity.Execute, activity.RegisterOptions{Name: intClusterWorkflow.DeleteUnusedClusterSecretsActivityName})

		workflow.RegisterWithOptions(intClusterWorkflow.DeleteK8sResourcesWorkflow, workflow.RegisterOptions{Name: intClusterWorkflow.DeleteK8sResourcesWorkflowName})

		deleteHelmDeploymentsActivity := intClusterWorkflow.MakeDeleteHelmDeploymentsActivity(k8sConfigGetter, releaseDeleter, logrusLogger)
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
					unifiedHelmReleaser,
					logger,
					orgDomainService,
					commonSecretStore,
					config.Cluster.DNS.Config,
				),
				securityscan.MakeIntegratedServiceOperator(
					config.Cluster.SecurityScan.Config,
					clusterGetter,
					clusterService,
					unifiedHelmReleaser,
					commonSecretStore,
					featureAnchoreService,
					featureWhitelistService,
					errorHandler,
					logger,
				),
				integratedServiceVault.MakeIntegratedServicesOperator(clusterGetter,
					clusterService,
					unifiedHelmReleaser,
					kubernetesService,
					commonSecretStore,
					config.Cluster.Vault.Config,
					logger,
				),
				integratedServiceMonitoring.MakeIntegratedServiceOperator(
					clusterGetter,
					clusterService,
					unifiedHelmReleaser,
					kubernetesService,
					config.Cluster.Monitoring.Config,
					logger,
					commonSecretStore,
				),
				integratedServiceLogging.MakeIntegratedServicesOperator(
					clusterGetter,
					clusterService,
					unifiedHelmReleaser,
					kubernetesService,
					endpointManager,
					config.Cluster.Logging.Config,
					logger,
					commonSecretStore,
				),
				expiry.NewExpiryServiceOperator(expirerService, services.BindIntegratedServiceSpec, logger),
				intsvcingress.NewOperator(
					intsvcingressadapter.NewOperatorClusterStore(clusterStore),
					clusterService,
					config.Cluster.Ingress.Config,
					unifiedHelmReleaser,
					intsvcingressadapter.NewOrgDomainService(config.Cluster.DNS.BaseDomain, orgGetter),
				),
			})

			registerClusterFeatureWorkflows(featureOperatorRegistry, featureRepository)
		}

		group.Add(appkitrun.CadenceWorkerRun(worker))
	}

	// Setup signal handler
	group.Add(run.SignalHandler(context.Background(), syscall.SIGINT, syscall.SIGTERM))

	err = group.Run()
	emperror.WithFilter(errorHandler, match.As(&run.SignalError{}).MatchError).Handle(err)
}
