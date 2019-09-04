// Copyright © 2018 Banzai Cloud
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
	"fmt"
	"net/url"
	"os"
	"path"
	"strings"

	"emperror.dev/emperror"
	"emperror.dev/errors"
	evbus "github.com/asaskevich/EventBus"
	ginprometheus "github.com/banzaicloud/go-gin-prometheus"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"logur.dev/logur"

	"github.com/banzaicloud/pipeline/api"
	"github.com/banzaicloud/pipeline/api/ark/backups"
	"github.com/banzaicloud/pipeline/api/ark/backupservice"
	"github.com/banzaicloud/pipeline/api/ark/buckets"
	"github.com/banzaicloud/pipeline/api/ark/restores"
	"github.com/banzaicloud/pipeline/api/ark/schedules"
	"github.com/banzaicloud/pipeline/api/cluster/namespace"
	"github.com/banzaicloud/pipeline/api/cluster/pke"
	cgroupAPI "github.com/banzaicloud/pipeline/api/clustergroup"
	"github.com/banzaicloud/pipeline/api/common"
	"github.com/banzaicloud/pipeline/api/middleware"
	"github.com/banzaicloud/pipeline/auth"
	"github.com/banzaicloud/pipeline/cluster"
	"github.com/banzaicloud/pipeline/config"
	"github.com/banzaicloud/pipeline/dns"
	arkClusterManager "github.com/banzaicloud/pipeline/internal/ark/clustermanager"
	arkEvents "github.com/banzaicloud/pipeline/internal/ark/events"
	arkSync "github.com/banzaicloud/pipeline/internal/ark/sync"
	"github.com/banzaicloud/pipeline/internal/audit"
	intAuth "github.com/banzaicloud/pipeline/internal/auth"
	"github.com/banzaicloud/pipeline/internal/cloudinfo"
	intCluster "github.com/banzaicloud/pipeline/internal/cluster"
	intClusterAuth "github.com/banzaicloud/pipeline/internal/cluster/auth"
	"github.com/banzaicloud/pipeline/internal/cluster/clustersecret"
	"github.com/banzaicloud/pipeline/internal/cluster/clustersecret/clustersecretadapter"
	prometheusMetrics "github.com/banzaicloud/pipeline/internal/cluster/metrics/adapters/prometheus"
	"github.com/banzaicloud/pipeline/internal/clusterfeature"
	"github.com/banzaicloud/pipeline/internal/clusterfeature/clusterfeatureadapter"
	"github.com/banzaicloud/pipeline/internal/clusterfeature/clusterfeaturedriver"
	featureDns "github.com/banzaicloud/pipeline/internal/clusterfeature/features/dns"
	"github.com/banzaicloud/pipeline/internal/clustergroup"
	cgroupAdapter "github.com/banzaicloud/pipeline/internal/clustergroup/adapter"
	"github.com/banzaicloud/pipeline/internal/clustergroup/deployment"
	"github.com/banzaicloud/pipeline/internal/common/commonadapter"
	"github.com/banzaicloud/pipeline/internal/dashboard"
	"github.com/banzaicloud/pipeline/internal/federation"
	"github.com/banzaicloud/pipeline/internal/global"
	"github.com/banzaicloud/pipeline/internal/helm"
	"github.com/banzaicloud/pipeline/internal/helm/helmadapter"
	cgFeatureIstio "github.com/banzaicloud/pipeline/internal/istio/istiofeature"
	"github.com/banzaicloud/pipeline/internal/monitor"
	"github.com/banzaicloud/pipeline/internal/notification"
	"github.com/banzaicloud/pipeline/internal/platform/buildinfo"
	"github.com/banzaicloud/pipeline/internal/platform/errorhandler"
	ginternal "github.com/banzaicloud/pipeline/internal/platform/gin"
	"github.com/banzaicloud/pipeline/internal/platform/gin/correlationid"
	ginlog "github.com/banzaicloud/pipeline/internal/platform/gin/log"
	ginutils "github.com/banzaicloud/pipeline/internal/platform/gin/utils"
	"github.com/banzaicloud/pipeline/internal/platform/log"
	platformlog "github.com/banzaicloud/pipeline/internal/platform/log"
	azurePKEAdapter "github.com/banzaicloud/pipeline/internal/providers/azure/pke/adapter"
	azurePKEDriver "github.com/banzaicloud/pipeline/internal/providers/azure/pke/driver"
	anchore "github.com/banzaicloud/pipeline/internal/security"
	"github.com/banzaicloud/pipeline/pkg/ctxutil"
	"github.com/banzaicloud/pipeline/pkg/k8sclient"
	"github.com/banzaicloud/pipeline/pkg/providers"
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
	v, p := viper.GetViper(), pflag.NewFlagSet(friendlyAppName, pflag.ExitOnError)

	configure(v, p)

	p.Bool("version", false, "Show version information")

	_ = p.Parse(os.Args[1:])

	if v, _ := p.GetBool("version"); v {
		fmt.Printf("%s version %s (%s) built on %s\n", friendlyAppName, version, commitHash, buildDate)

		os.Exit(0)
	}

	var conf configuration
	err := viper.Unmarshal(&conf)
	emperror.Panic(errors.Wrap(err, "failed to unmarshal configuration"))

	// Create logger (first thing after configuration loading)
	logger := log.NewLogger(conf.Log)

	// Legacy logger instance
	logrusLogger := config.Logger()

	// Provide some basic context to all log lines
	logger = log.WithFields(logger, map[string]interface{}{"application": appName})

	log.SetStandardLogger(logger)
	log.SetK8sLogger(logger)

	err = conf.Validate()
	if err != nil {
		logger.Error(err.Error())

		os.Exit(3)
	}

	errorHandler, err := errorhandler.New(conf.ErrorHandler, logger)
	if err != nil {
		logger.Error(err.Error())

		os.Exit(1)
	}
	defer errorHandler.Close()
	defer emperror.HandleRecover(errorHandler)
	global.SetErrorHandler(errorHandler)

	buildInfo := buildinfo.New(version, commitHash, buildDate)

	logger.Info("starting application", buildInfo.Fields())

	// Connect to database
	db := config.DB()
	cicdDB, err := config.CICDDB()
	emperror.Panic(err)

	basePath := viper.GetString("pipeline.basepath")

	enforcer := intAuth.NewEnforcer(db)

	orgImporter := auth.NewOrgImporter(db, config.EventBus)
	tokenHandler := auth.NewTokenHandler()

	// Initialize auth
	auth.Init(cicdDB, orgImporter)

	if viper.GetBool(config.DBAutoMigrateEnabled) {
		logger.Info("running automatic schema migrations")

		err = Migrate(db, logrusLogger)
		if err != nil {
			panic(err)
		}
	}

	// External DNS service
	dnsSvc, err := dns.GetExternalDnsServiceClient()
	if err != nil {
		emperror.Panic(errors.WithMessage(err, "getting external dns service client failed"))
	}

	if dnsSvc == nil {
		logger.Info("external dns service functionality is not enabled")
	}

	prometheus.MustRegister(cluster.NewExporter())

	clusterEventBus := evbus.New()
	clusterEvents := cluster.NewClusterEvents(clusterEventBus)
	clusters := intCluster.NewClusters(db)
	secretValidator := providers.NewSecretValidator(secret.Store)
	statusChangeDurationMetric := prometheusMetrics.MakePrometheusClusterStatusChangeDurationMetric()
	// Initialise cluster total metric
	clusterTotalMetric := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "pipeline",
		Name:      "cluster_total",
		Help:      "the number of clusters launched",
	},
		[]string{"provider", "location"},
	)
	type totalClusterMetric struct {
		Location string
		Cloud    string
		Count    int
	}
	totalClusters := make([]totalClusterMetric, 0)
	// SELECT count(id) as count, location, cloud FROM clusters GROUP BY location, cloud; (init values)
	if err := db.Raw("SELECT count(id) as count, location, cloud FROM clusters GROUP BY location, cloud").Scan(&totalClusters).Error; err != nil {
		logger.Error(err.Error())
		// TODO: emperror.Panic?
	}
	for _, row := range totalClusters {
		clusterTotalMetric.With(
			map[string]string{
				"location": row.Location,
				"provider": row.Cloud,
			}).Add(float64(row.Count))
	}
	prometheus.MustRegister(statusChangeDurationMetric, clusterTotalMetric)

	externalBaseURL := viper.GetString("pipeline.externalURL")
	if externalBaseURL == "" {
		externalBaseURL = "http://" + viper.GetString("pipeline.bindaddr")
		logger.Warn("no pipeline.external_url set, falling back to bind address", map[string]interface{}{
			"fallback": externalBaseURL,
		})
	}

	externalURLInsecure := viper.GetBool(config.PipelineExternalURLInsecure)

	oidcIssuerURL := viper.GetString(config.OIDCIssuerURL)

	workflowClient, err := config.CadenceClient()
	if err != nil {
		errorHandler.Handle(errors.WrapIf(err, "Failed to configure Cadence client"))
	}

	clusterManager := cluster.NewManager(clusters, secretValidator, clusterEvents, statusChangeDurationMetric, clusterTotalMetric, workflowClient, logrusLogger, errorHandler)
	clusterGetter := common.NewClusterGetter(clusterManager, logrusLogger, errorHandler)

	clusterTTLController := cluster.NewTTLController(clusterManager, clusterEventBus, logrusLogger.WithField("subsystem", "ttl-controller"), errorHandler)
	defer clusterTTLController.Stop()
	err = clusterTTLController.Start()
	emperror.Panic(err)

	if viper.GetBool(config.MonitorEnabled) {
		client, err := k8sclient.NewInClusterClient()
		if err != nil {
			errorHandler.Handle(errors.WrapIf(err, "failed to enable monitoring"))
		} else {
			dnsBaseDomain, err := dns.GetBaseDomain()
			if err != nil {
				errorHandler.Handle(errors.WrapIf(err, "failed to enable monitoring"))
			}

			monitorClusterSubscriber := monitor.NewClusterSubscriber(
				client,
				clusterManager,
				db,
				dnsBaseDomain,
				viper.GetString(config.ControlPlaneNamespace),
				viper.GetString(config.PipelineSystemNamespace),
				viper.GetString(config.MonitorConfigMap),
				viper.GetString(config.MonitorConfigMapPrometheusKey),
				viper.GetString(config.MonitorCertSecret),
				viper.GetString(config.MonitorCertMountPath),
				errorHandler,
			)
			monitorClusterSubscriber.Init()
			monitorClusterSubscriber.Register(monitor.NewClusterEvents(clusterEventBus))
		}
	}

	if viper.GetBool(config.SpotMetricsEnabled) {
		go monitor.NewSpotMetricsExporter(context.Background(), clusterManager, logrusLogger.WithField("subsystem", "spot-metrics-exporter")).Run(viper.GetDuration(config.SpotMetricsCollectionInterval))
	}

	cloudInfoEndPoint := viper.GetString(config.CloudInfoEndPoint)
	if cloudInfoEndPoint == "" {
		errorHandler.Handle(errors.New("missing CloudInfo endpoint"))
		return
	}
	cloudInfoClient := cloudinfo.NewClient(cloudInfoEndPoint, logrusLogger)

	gormAzurePKEClusterStore := azurePKEAdapter.NewGORMAzurePKEClusterStore(db)
	clusterCreators := api.ClusterCreators{
		PKEOnAzure: azurePKEDriver.MakeAzurePKEClusterCreator(
			logrusLogger,
			gormAzurePKEClusterStore,
			workflowClient,
			externalBaseURL,
			externalURLInsecure,
			oidcIssuerURL,
		),
	}
	clusterDeleters := api.ClusterDeleters{
		PKEOnAzure: azurePKEDriver.MakeAzurePKEClusterDeleter(
			clusterEvents,
			clusterManager.GetKubeProxyCache(),
			logrusLogger,
			secret.Store,
			statusChangeDurationMetric,
			gormAzurePKEClusterStore,
			workflowClient,
		),
	}

	cgroupAdapter := cgroupAdapter.NewClusterGetter(clusterManager)
	clusterGroupManager := clustergroup.NewManager(cgroupAdapter, clustergroup.NewClusterGroupRepository(db, logrusLogger), logrusLogger, errorHandler)
	infraNamespace := viper.GetString(config.PipelineSystemNamespace)
	federationHandler := federation.NewFederationHandler(cgroupAdapter, infraNamespace, logrusLogger, errorHandler)
	deploymentManager := deployment.NewCGDeploymentManager(db, cgroupAdapter, logrusLogger, errorHandler)
	serviceMeshFeatureHandler := cgFeatureIstio.NewServiceMeshFeatureHandler(cgroupAdapter, logrusLogger, errorHandler)
	clusterGroupManager.RegisterFeatureHandler(federation.FeatureName, federationHandler)
	clusterGroupManager.RegisterFeatureHandler(deployment.FeatureName, deploymentManager)
	clusterGroupManager.RegisterFeatureHandler(cgFeatureIstio.FeatureName, serviceMeshFeatureHandler)
	clusterUpdaters := api.ClusterUpdaters{
		PKEOnAzure: azurePKEDriver.MakeAzurePKEClusterUpdater(
			logrusLogger,
			externalBaseURL,
			externalURLInsecure,
			secret.Store,
			gormAzurePKEClusterStore,
			workflowClient,
		),
	}

	clusterAPI := api.NewClusterAPI(clusterManager, clusterGetter, workflowClient, cloudInfoClient, clusterGroupManager, logrusLogger, errorHandler, externalBaseURL, externalURLInsecure, clusterCreators, clusterDeleters, clusterUpdaters)

	nplsApi := api.NewNodepoolManagerAPI(clusterGetter, logrusLogger, errorHandler)

	// Initialise Gin router
	router := gin.New()

	// These two paths can contain sensitive information, so it is advised not to log them out.
	skipPaths := viper.GetStringSlice("audit.skippaths")
	router.Use(correlationid.Middleware())
	router.Use(ginlog.Middleware(logrusLogger, skipPaths...))

	// Add prometheus metric endpoint
	if viper.GetBool(config.MetricsEnabled) {
		p := ginprometheus.NewPrometheus("pipeline", []string{})
		p.SetListenAddress(viper.GetString(config.MetricsAddress) + ":" + viper.GetString(config.MetricsPort))
		p.Use(router, "/metrics")
	}

	router.Use(gin.Recovery())
	drainModeMetric := prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "pipeline",
		Name:      "drain_mode",
		Help:      "read only mode is on/off",
	})
	prometheus.MustRegister(drainModeMetric)
	router.Use(ginternal.NewDrainModeMiddleware(drainModeMetric, errorHandler).Middleware)
	router.Use(cors.New(config.GetCORS()))
	if viper.GetBool("audit.enabled") {
		logger.Info("Audit enabled, installing Gin audit middleware")
		router.Use(audit.LogWriter(skipPaths, viper.GetStringSlice("audit.headers"), db, logrusLogger))
	}
	router.Use(func(c *gin.Context) { // TODO: move to middleware
		c.Request = c.Request.WithContext(ctxutil.WithParams(c.Request.Context(), ginutils.ParamsToMap(c.Params)))
	})

	router.GET("/", api.RedirectRoot)

	base := router.Group(basePath)
	base.GET("notifications", notification.GetNotifications)
	base.GET("version", gin.WrapH(buildinfo.Handler(buildInfo)))

	auth.Install(router, tokenHandler.GenerateToken)
	auth.StartTokenStoreGC()

	authorizationMiddleware := intAuth.NewMiddleware(enforcer, basePath, errorHandler)

	dashboardAPI := dashboard.NewDashboardAPI(clusterManager, clusterGroupManager, logrusLogger, errorHandler)
	dgroup := base.Group(path.Join("dashboard", "orgs"))
	dgroup.Use(auth.Handler)
	dgroup.Use(api.OrganizationMiddleware)
	dgroup.Use(authorizationMiddleware)
	dgroup.GET("/:orgid/clusters", dashboardAPI.GetDashboard)

	{
		// Cluster details dashboard
		dcGroup := dgroup.Group("/:orgid/clusters/:id")
		dcGroup.Use(cluster.NewClusterCheckMiddleware(clusterManager, errorHandler))
		dcGroup.GET("", dashboardAPI.GetClusterDashboard)
	}

	domainAPI := api.NewDomainAPI(clusterManager, logrusLogger, errorHandler)
	organizationAPI := api.NewOrganizationAPI(orgImporter)
	userAPI := api.NewUserAPI(db, logrusLogger, errorHandler)
	networkAPI := api.NewNetworkAPI(logrusLogger)

	switch viper.GetString(config.DNSBaseDomain) {
	case "", "example.com", "example.org":
		global.AutoDNSEnabled = false
	default:
		global.AutoDNSEnabled = true
	}

	spotguidePlatformData := spotguide.PlatformData{
		AutoDNSEnabled: global.AutoDNSEnabled,
	}

	scmProvider := viper.GetString("cicd.scm")
	var scmToken string
	switch scmProvider {
	case "github":
		scmToken = viper.GetString("github.token")
	case "gitlab":
		scmToken = viper.GetString("gitlab.token")
	default:
		emperror.Panic(fmt.Errorf("Unknown SCM provider configured: %s", scmProvider))
	}

	scmFactory, err := scm.NewSCMFactory(scmProvider, scmToken)
	emperror.Panic(errors.WrapIf(err, "failed to create SCMFactory"))

	sharedSpotguideOrg, err := spotguide.EnsureSharedSpotguideOrganization(config.DB(), scmProvider, viper.GetString(config.SpotguideSharedLibraryGitHubOrganization))
	if err != nil {
		errorHandler.Handle(errors.WrapIf(err, "failed to create shared Spotguide organization"))
	}

	spotguideManager := spotguide.NewSpotguideManager(
		config.DB(),
		version,
		scmFactory,
		sharedSpotguideOrg,
		spotguidePlatformData,
	)

	// subscribe to organization creations and sync spotguides into the newly created organizations
	spotguide.AuthEventEmitter.NotifyOrganizationRegistered(func(orgID uint, userID uint) {
		if err := spotguideManager.ScrapeSpotguides(orgID, userID); err != nil {
			logger.Warn(
				errors.WithMessage(err, "failed to scrape Spotguide repositories").Error(),
				map[string]interface{}{
					"organizationId": orgID,
				},
			)
		}
	})

	// periodically sync shared spotguides
	if err := spotguide.ScheduleScrapingSharedSpotguides(workflowClient); err != nil {
		errorHandler.Handle(errors.WrapIf(err, "failed to schedule syncing shared spotguides"))
	}

	spotguideAPI := api.NewSpotguideAPI(logrusLogger, errorHandler, spotguideManager)

	v1 := base.Group("api/v1")
	{
		v1.Use(auth.Handler)
		v1.GET("/functions", api.ListFunctions)
		v1.GET("/securityscan", api.SecurityScanEnabled)
		v1.GET("/me", userAPI.GetCurrentUser)
		v1.PATCH("/me", userAPI.UpdateCurrentUser)
		orgs := v1.Group("/orgs")
		{
			orgs.Use(api.OrganizationMiddleware)
			orgs.Use(authorizationMiddleware)

			orgs.GET("/:orgid/spotguides", spotguideAPI.GetSpotguides)
			orgs.PUT("/:orgid/spotguides", middleware.NewRateLimiterByOrgID(api.SyncSpotguidesRateLimit), spotguideAPI.SyncSpotguides)
			orgs.POST("/:orgid/spotguides", spotguideAPI.LaunchSpotguide)
			// Spotguide name may contain '/'s so we have to use :owner/:name
			orgs.GET("/:orgid/spotguides/:owner/:name", spotguideAPI.GetSpotguide)
			orgs.HEAD("/:orgid/spotguides/:owner/:name", spotguideAPI.GetSpotguide)
			orgs.GET("/:orgid/spotguides/:owner/:name/icon", spotguideAPI.GetSpotguideIcon)

			orgs.GET("/:orgid/domain", domainAPI.GetDomain)
			orgs.POST("/:orgid/clusters", clusterAPI.CreateCluster)
			// v1.GET("/status", api.Status)
			orgs.GET("/:orgid/clusters", clusterAPI.GetClusters)

			// cluster API
			cRouter := orgs.Group("/:orgid/clusters/:id")
			{
				cRouter.Use(cluster.NewClusterCheckMiddleware(clusterManager, errorHandler))

				cRouter.GET("", clusterAPI.GetCluster)
				cRouter.GET("/pods", api.GetPodDetails)
				cRouter.GET("/bootstrap", clusterAPI.GetBootstrapInfo)
				cRouter.PUT("", clusterAPI.UpdateCluster)

				cRouter.PUT("/posthooks", clusterAPI.ReRunPostHooks)
				cRouter.POST("/secrets", api.InstallSecretsToCluster)
				cRouter.POST("/secrets/:secretName", api.InstallSecretToCluster)
				cRouter.PATCH("/secrets/:secretName", api.MergeSecretInCluster)
				cRouter.Any("/proxy/*path", clusterAPI.ProxyToCluster)
				cRouter.DELETE("", clusterAPI.DeleteCluster)
				cRouter.HEAD("", clusterAPI.ClusterCheck)
				cRouter.GET("/config", api.GetClusterConfig)
				cRouter.GET("/apiendpoint", api.GetApiEndpoint)
				cRouter.GET("/nodes", api.GetClusterNodes)
				cRouter.GET("/endpoints", api.ListEndpoints)
				cRouter.GET("/secrets", api.ListClusterSecrets)
				cRouter.GET("/deployments", api.ListDeployments)
				cRouter.POST("/deployments", api.CreateDeployment)
				cRouter.GET("/deployments/:name", api.GetDeployment)
				cRouter.GET("/deployments/:name/resources", api.GetDeploymentResources)
				cRouter.GET("/hpa", api.GetHpaResource)
				cRouter.PUT("/hpa", api.PutHpaResource)
				cRouter.DELETE("/hpa", api.DeleteHpaResource)
				cRouter.HEAD("/deployments", api.GetTillerStatus)
				cRouter.DELETE("/deployments/:name", api.DeleteDeployment)
				cRouter.PUT("/deployments/:name", api.UpgradeDeployment)
				cRouter.HEAD("/deployments/:name", api.HelmDeploymentStatus)
				cRouter.POST("/helminit", api.InitHelmOnCluster)

				cRouter.GET("/images", api.ListImages)
				cRouter.GET("/images/:imageDigest/deployments", api.GetImageDeployments)
				cRouter.GET("/deployments/:name/images", api.GetDeploymentImages)

				if anchore.AnchoreEnabled {
					cRouter.GET("/scanlog", api.GetScanLog)
					cRouter.GET("/scanlog/:releaseName", api.GetScanLog)
					cRouter.GET("/whitelists", api.GetWhiteLists)
					cRouter.POST("/whitelists", api.CreateWhiteList)
					cRouter.DELETE("/whitelists/:name", api.DeleteWhiteList)
					cRouter.GET("/policies", api.GetPolicies)
					cRouter.GET("/policies/:policyId", api.GetPolicies)
					cRouter.POST("/policies", api.CreatePolicy)
					cRouter.PUT("/policies/:policyId", api.UpdatePolicies)
					cRouter.DELETE("/policies/:policyId", api.DeletePolicy)

					cRouter.POST("/imagescan", api.ScanImages)
					cRouter.GET("/imagescan/:imagedigest", api.GetScanResult)
					cRouter.GET("/imagescan/:imagedigest/vuln", api.GetImageVulnerabilities)
				}

			}

			clusterSecretStore := clustersecret.NewStore(
				clustersecretadapter.NewClusterManagerAdapter(clusterManager),
				clustersecretadapter.NewSecretStore(secret.Store),
			)

			// ClusterInfo Feature API
			{
				logger := commonadapter.NewLogger(logger) // TODO: make this a context aware logger
				featureRepository := clusterfeatureadapter.NewGormFeatureRepository(db, logger)
				helmService := helm.NewHelmService(helmadapter.NewClusterService(clusterManager), logger)
				secretStore := commonadapter.NewSecretStore(secret.Store, commonadapter.OrgIDContextExtractorFunc(auth.GetCurrentOrganizationID))
				clusterService := clusterfeatureadapter.NewClusterService(clusterManager)
				orgDomainService := featureDns.NewOrgDomainService(clusterManager, dnsSvc, logger)
				dnsFeatureManager := featureDns.NewDnsFeatureManager(featureRepository, secretStore, clusterService, clusterManager, helmService, orgDomainService, logger)
				featureRegistry := clusterfeature.NewFeatureRegistry(map[string]clusterfeature.FeatureManager{
					dnsFeatureManager.Name(): clusterfeatureadapter.NewAsyncFeatureManagerStub(dnsFeatureManager, featureRepository, workflowClient, logger),
				})

				service := clusterfeature.NewFeatureService(featureRegistry, featureRepository, logger)
				endpoints := clusterfeaturedriver.MakeEndpoints(service)
				handlers := clusterfeaturedriver.MakeHTTPHandlers(endpoints, errorHandler)

				router := cRouter.Group("/features")

				router.GET("", ginutils.HTTPHandlerToGinHandlerFunc(handlers.List))
				router.GET("/:featureName", ginutils.HTTPHandlerToGinHandlerFunc(handlers.Details))
				router.DELETE("/:featureName", ginutils.HTTPHandlerToGinHandlerFunc(handlers.Deactivate))
				router.POST("/:featureName", ginutils.HTTPHandlerToGinHandlerFunc(handlers.Activate))
				router.PUT("/:featureName", ginutils.HTTPHandlerToGinHandlerFunc(handlers.Update))
			}

			// ClusterGroupAPI
			cgroupsAPI := cgroupAPI.NewAPI(clusterGroupManager, deploymentManager, logrusLogger, errorHandler)
			cgroupsAPI.AddRoutes(orgs.Group("/:orgid/clustergroups"))

			cRouter.GET("/nodepools/labels", nplsApi.GetNodepoolLabelSets)
			cRouter.POST("/nodepools/labels", nplsApi.SetNodepoolLabelSets)

			namespaceAPI := namespace.NewAPI(clusterGetter, errorHandler)
			namespaceAPI.RegisterRoutes(cRouter.Group("/namespaces/:namespace"))

			pkeGroup := cRouter.Group("/pke")

			leaderRepository, err := pke.NewVaultLeaderRepository()
			emperror.Panic(errors.WrapIf(err, "failed to create Vault leader repository"))

			pkeAPI := pke.NewAPI(clusterGetter, errorHandler, tokenHandler, externalBaseURL, workflowClient, leaderRepository)
			pkeAPI.RegisterRoutes(pkeGroup)

			clusterAuthService, err := intClusterAuth.NewDexClusterAuthService(clusterSecretStore)
			emperror.Panic(errors.WrapIf(err, "failed to create DexClusterAuthService"))

			pipelineExternalURL, err := url.Parse(externalBaseURL)
			emperror.Panic(errors.WrapIf(err, "failed to parse pipeline externalBaseURL"))

			pipelineExternalURL.Path = "/auth/dex/cluster/callback"

			clusterAuthAPI, err := api.NewClusterAuthAPI(
				clusterGetter,
				clusterAuthService,
				viper.GetString("auth.tokensigningkey"),
				oidcIssuerURL,
				viper.GetBool(config.OIDCIssuerInsecure),
				pipelineExternalURL.String(),
			)
			emperror.Panic(errors.WrapIf(err, "failed to create ClusterAuthAPI"))

			clusterAuthAPI.RegisterRoutes(cRouter, router)

			orgs.GET("/:orgid/helm/repos", api.HelmReposGet)
			orgs.POST("/:orgid/helm/repos", api.HelmReposAdd)
			orgs.PUT("/:orgid/helm/repos/:name", api.HelmReposModify)
			orgs.PUT("/:orgid/helm/repos/:name/update", api.HelmReposUpdate)
			orgs.DELETE("/:orgid/helm/repos/:name", api.HelmReposDelete)
			orgs.GET("/:orgid/helm/charts", api.HelmCharts)
			orgs.GET("/:orgid/helm/chart/:reponame/:name", api.HelmChart)
			orgs.GET("/:orgid/secrets", api.ListSecrets)
			orgs.GET("/:orgid/secrets/:id", api.GetSecret)
			orgs.POST("/:orgid/secrets", api.AddSecrets)
			orgs.PUT("/:orgid/secrets/:id", api.UpdateSecrets)
			orgs.DELETE("/:orgid/secrets/:id", api.DeleteSecrets)
			orgs.GET("/:orgid/secrets/:id/validate", api.ValidateSecret)
			orgs.GET("/:orgid/secrets/:id/tags", api.GetSecretTags)
			orgs.PUT("/:orgid/secrets/:id/tags/*tag", api.AddSecretTag)
			orgs.DELETE("/:orgid/secrets/:id/tags/*tag", api.DeleteSecretTag)
			orgs.GET("/:orgid/users", userAPI.GetUsers)
			orgs.GET("/:orgid/users/:id", userAPI.GetUsers)

			orgs.GET("/:orgid/buckets", api.ListAllBuckets)
			orgs.POST("/:orgid/buckets", api.CreateBucket)
			orgs.HEAD("/:orgid/buckets/:name", api.CheckBucket)
			orgs.GET("/:orgid/buckets/:name", api.GetBucket)
			orgs.DELETE("/:orgid/buckets/:name", api.DeleteBucket)

			orgs.GET("/:orgid/networks", networkAPI.ListVPCNetworks)
			orgs.GET("/:orgid/networks/:id/subnets", networkAPI.ListVPCSubnets)
			orgs.GET("/:orgid/networks/:id/routeTables", networkAPI.ListRouteTables)

			orgs.GET("/:orgid/azure/resourcegroups", api.GetResourceGroups)
			orgs.POST("/:orgid/azure/resourcegroups", api.AddResourceGroups)
			orgs.DELETE("/:orgid/azure/resourcegroups/:name", api.DeleteResourceGroups)

			orgs.GET("/:orgid/google/projects", api.GetProjects)

			orgs.GET("/:orgid", organizationAPI.GetOrganizations)
			orgs.DELETE("/:orgid", organizationAPI.DeleteOrganization)
		}
		v1.GET("/orgs", organizationAPI.GetOrganizations)
		v1.PUT("/orgs", organizationAPI.SyncOrganizations)
		v1.GET("/token", tokenHandler.GenerateToken) // TODO Deprecated, should be removed once the UI has support.
		v1.POST("/tokens", tokenHandler.GenerateToken)
		v1.GET("/tokens", auth.GetTokens)
		v1.GET("/tokens/:id", auth.GetTokens)
		v1.DELETE("/tokens/:id", auth.DeleteToken)

		v1.GET("/allowed/secrets", api.ListAllowedSecretTypes)
		v1.GET("/allowed/secrets/:type", api.ListAllowedSecretTypes)

		backups.AddRoutes(orgs.Group("/:orgid/clusters/:id/backups"))
		backupservice.AddRoutes(orgs.Group("/:orgid/clusters/:id/backupservice"))
		restores.AddRoutes(orgs.Group("/:orgid/clusters/:id/restores"))
		schedules.AddRoutes(orgs.Group("/:orgid/clusters/:id/schedules"))
		buckets.AddRoutes(orgs.Group("/:orgid/backupbuckets"))
		backups.AddOrgRoutes(orgs.Group("/:orgid/backups"), clusterManager)
	}

	arkEvents.NewClusterEventHandler(arkEvents.NewClusterEvents(clusterEventBus), config.DB(), logrusLogger)
	if viper.GetBool(config.ARKSyncEnabled) {
		go arkSync.RunSyncServices(
			context.Background(),
			config.DB(),
			arkClusterManager.New(clusterManager),
			platformlog.NewLogrusLogger(platformlog.Config{
				Level:  viper.GetString(config.ARKLogLevel),
				Format: viper.GetString(conf.Log.Format),
			}).WithField("subsystem", "ark"),
			errorHandler,
			viper.GetDuration(config.ARKBucketSyncInterval),
			viper.GetDuration(config.ARKRestoreSyncInterval),
			viper.GetDuration(config.ARKBackupSyncInterval),
		)
	}

	base.GET("api", api.MetaHandler(router, basePath+"/api"))

	issueHandler, err := api.NewIssueHandler(version, commitHash, buildDate)
	if err != nil {
		emperror.Panic(errors.WrapIf(err, "failed to create IssueHandler"))
	}
	base.POST("issues", auth.Handler, issueHandler)

	internalBindAddr := viper.GetString("pipeline.internalBindAddr")
	logger.Info("Pipeline internal API listening", map[string]interface{}{"address": "http://" + internalBindAddr})

	go createInternalAPIRouter(skipPaths, db, basePath, clusterAPI, logger, logrusLogger).Run(internalBindAddr) // nolint: errcheck

	bindAddr := viper.GetString("pipeline.bindaddr")
	if port := viper.GetInt("pipeline.listenport"); port != 0 { // TODO: remove deprecated option
		host := strings.Split(bindAddr, ":")[0]
		bindAddr = fmt.Sprintf("%s:%d", host, port)
		logger.Warn(fmt.Sprintf(
			"pipeline.listenport=%d setting is deprecated! Falling back to pipeline.bindaddr=%s",
			port,
			bindAddr,
		))
	}
	certFile, keyFile := viper.GetString("pipeline.certfile"), viper.GetString("pipeline.keyfile")
	if certFile != "" && keyFile != "" {
		logger.Info("Pipeline API listening", map[string]interface{}{"address": "https://" + bindAddr})
		_ = router.RunTLS(bindAddr, certFile, keyFile)
	} else {
		logger.Info("Pipeline API listening", map[string]interface{}{"address": "http://" + bindAddr})
		_ = router.Run(bindAddr)
	}
}

func createInternalAPIRouter(skipPaths []string, db *gorm.DB, basePath string, clusterAPI *api.ClusterAPI, logger logur.Logger, logrusLogger logrus.FieldLogger) *gin.Engine {
	// Initialise Gin router for Internal API
	internalRouter := gin.New()
	internalRouter.Use(correlationid.Middleware())
	internalRouter.Use(ginlog.Middleware(logrusLogger, skipPaths...))
	internalRouter.Use(gin.Recovery())
	if viper.GetBool("audit.enabled") {
		logger.Info("Audit enabled, installing Gin audit middleware to internal router")
		internalRouter.Use(audit.LogWriter(skipPaths, viper.GetStringSlice("audit.headers"), db, logrusLogger))
	}
	internalGroup := internalRouter.Group(path.Join(basePath, "api", "v1/", "orgs"))
	internalGroup.Use(auth.InternalUserHandler)
	internalGroup.Use(api.OrganizationMiddleware)
	internalGroup.GET("/:orgid/clusters/:id/nodepools", api.GetNodePools)
	internalGroup.PUT("/:orgid/clusters/:id/nodepools", clusterAPI.UpdateNodePools)
	return internalRouter
}
