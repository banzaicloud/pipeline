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
	"time"

	"github.com/jinzhu/gorm"

	evbus "github.com/asaskevich/EventBus"
	ginprometheus "github.com/banzaicloud/go-gin-prometheus"
	"github.com/banzaicloud/pipeline/api"
	"github.com/banzaicloud/pipeline/api/ark/backups"
	"github.com/banzaicloud/pipeline/api/ark/backupservice"
	"github.com/banzaicloud/pipeline/api/ark/buckets"
	"github.com/banzaicloud/pipeline/api/ark/restores"
	"github.com/banzaicloud/pipeline/api/ark/schedules"
	"github.com/banzaicloud/pipeline/api/cluster/namespace"
	"github.com/banzaicloud/pipeline/api/cluster/pke"
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
	intCluster "github.com/banzaicloud/pipeline/internal/cluster"
	intClusterAuth "github.com/banzaicloud/pipeline/internal/cluster/auth"
	"github.com/banzaicloud/pipeline/internal/cluster/clustersecret"
	"github.com/banzaicloud/pipeline/internal/cluster/clustersecret/clustersecretadapter"
	"github.com/banzaicloud/pipeline/internal/dashboard"
	"github.com/banzaicloud/pipeline/internal/monitor"
	"github.com/banzaicloud/pipeline/internal/notification"
	ginternal "github.com/banzaicloud/pipeline/internal/platform/gin"
	"github.com/banzaicloud/pipeline/internal/platform/gin/correlationid"
	ginlog "github.com/banzaicloud/pipeline/internal/platform/gin/log"
	platformlog "github.com/banzaicloud/pipeline/internal/platform/log"
	azurePKEAdapter "github.com/banzaicloud/pipeline/internal/providers/azure/pke/adapter"
	azurePKEDriver "github.com/banzaicloud/pipeline/internal/providers/azure/pke/driver"
	"github.com/banzaicloud/pipeline/model/defaults"
	"github.com/banzaicloud/pipeline/pkg/k8sclient"
	"github.com/banzaicloud/pipeline/pkg/providers"
	"github.com/banzaicloud/pipeline/secret"
	"github.com/banzaicloud/pipeline/spotguide"
	"github.com/banzaicloud/pipeline/spotguide/scm"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/goph/emperror"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

//Common logger for package
// nolint: gochecknoglobals
var log *logrus.Logger

// nolint: gochecknoglobals
var logger *logrus.Entry

func initLog() *logrus.Entry {
	log = config.Logger()
	logger := log.WithFields(logrus.Fields{"state": "init"})
	return logger
}

func main() {
	if len(os.Args) > 1 && os.Args[1] == "--version" {
		if commitHash == "" {
			fmt.Println("version: ", version, " built on ", buildDate)
		} else {
			fmt.Printf("version: %s-%s built on %s\n", version, commitHash, buildDate)
		}
		os.Exit(0)
	}

	logger = initLog()
	logger.WithFields(logrus.Fields{
		"version":     version,
		"commit_hash": commitHash,
		"build_date":  buildDate,
	}).Info("Pipeline initialization")
	errorHandler := config.ErrorHandler()

	// Connect to database
	db := config.DB()
	cicdDB, err := config.CICDDB()
	if err != nil {
		logger.Panic(err.Error())
	}

	basePath := viper.GetString("pipeline.basepath")

	enforcer := intAuth.NewEnforcer(db)
	accessManager := intAuth.NewAccessManager(enforcer, basePath)
	accessManager.AddDefaultPolicies()

	orgImporter := auth.NewOrgImporter(db, accessManager, config.EventBus)
	tokenHandler := auth.NewTokenHandler(accessManager)

	// Initialize auth
	auth.Init(cicdDB, accessManager, orgImporter)

	if viper.GetBool(config.DBAutoMigrateEnabled) {
		log.Info("running automatic schema migrations")

		err = Migrate(db, logger)
		if err != nil {
			panic(err)
		}
	}

	err = defaults.SetDefaultValues()
	if err != nil {
		panic(err)
	}

	// External DNS service
	dnsSvc, err := dns.GetExternalDnsServiceClient()
	if err != nil {
		log.Errorf("Getting external dns service client failed: %s", err.Error())
		panic(err)
	}

	if dnsSvc == nil {
		log.Infoln("External dns service functionality is not enabled")
	}

	prometheus.MustRegister(cluster.NewExporter())

	clusterEventBus := evbus.New()
	clusterEvents := cluster.NewClusterEvents(clusterEventBus)
	clusters := intCluster.NewClusters(db)
	secretValidator := providers.NewSecretValidator(secret.Store)
	statusChangeDurationMetric := prometheus.NewSummaryVec(prometheus.SummaryOpts{
		Namespace: "pipeline",
		Name:      "cluster_status_change_duration",
		Help:      "Cluster status change duration in seconds",
	},
		[]string{"provider", "location", "status", "orgName", "clusterName"},
	)
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
	//SELECT count(id) as count, location, cloud FROM clusters GROUP BY location, cloud; (init values)
	if err := db.Raw("SELECT count(id) as count, location, cloud FROM clusters GROUP BY location, cloud").Scan(&totalClusters).Error; err != nil {
		logger.Error(err)
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
		log.Errorf("no pipeline.external_url set. falling back to %q", externalBaseURL)
	}

	workflowClient, err := config.CadenceClient()
	if err != nil {
		errorHandler.Handle(emperror.Wrap(err, "Failed to configure Cadence client"))
	}

	clusterManager := cluster.NewManager(clusters, secretValidator, clusterEvents, statusChangeDurationMetric, clusterTotalMetric, workflowClient, log, errorHandler)
	clusterGetter := common.NewClusterGetter(clusterManager, logger, errorHandler)

	clusterTTLController := cluster.NewTTLController(clusterManager, clusterEventBus, log.WithField("subsystem", "ttl-controller"), errorHandler)
	defer clusterTTLController.Stop()
	err = clusterTTLController.Start()
	if err != nil {
		logger.Panic(err)
	}

	if viper.GetBool(config.MonitorEnabled) {
		client, err := k8sclient.NewInClusterClient()
		if err != nil {
			errorHandler.Handle(emperror.Wrap(err, "failed to enable monitoring"))
		} else {
			dnsBaseDomain, err := dns.GetBaseDomain()
			if err != nil {
				errorHandler.Handle(emperror.Wrap(err, "failed to enable monitoring"))
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
		go monitor.NewSpotMetricsExporter(context.Background(), clusterManager, log.WithField("subsystem", "spot-metrics-exporter")).Run(viper.GetDuration(config.SpotMetricsCollectionInterval))
	}

	clusterAPI := api.NewClusterAPI(clusterManager, clusterGetter, workflowClient, log, errorHandler, externalBaseURL, api.ClusterCreators{
		PKEOnAzure: azurePKEDriver.NewAzurePKEClusterCreator(
			log,
			azurePKEAdapter.NewGORMAzurePKEClusterStore(db),
			workflowClient,
		),
	})

	nplsApi := api.NewNodepoolManagerAPI(clusterGetter, log, errorHandler)

	//Initialise Gin router
	router := gin.New()

	// These two paths can contain sensitive information, so it is advised not to log them out.
	skipPaths := viper.GetStringSlice("audit.skippaths")
	router.Use(correlationid.Middleware())
	router.Use(ginlog.Middleware(log, skipPaths...))

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
		log.Infoln("Audit enabled, installing Gin audit middleware")
		router.Use(audit.LogWriter(skipPaths, viper.GetStringSlice("audit.headers"), db, log))
	}

	router.GET("/", api.RedirectRoot)

	base := router.Group(basePath)
	base.GET("notifications", notification.GetNotifications)
	base.GET("version", VersionHandler)

	auth.Install(router, tokenHandler.GenerateToken)
	auth.StartTokenStoreGC()

	authorizationMiddleware := intAuth.NewMiddleware(enforcer, basePath, errorHandler)

	dgroup := base.Group(path.Join("dashboard", "orgs"))
	dgroup.Use(auth.Handler)
	dgroup.Use(api.OrganizationMiddleware)
	dgroup.Use(authorizationMiddleware)
	dgroup.GET("/:orgid/clusters", dashboard.GetDashboard)

	domainAPI := api.NewDomainAPI(clusterManager, log, errorHandler)
	organizationAPI := api.NewOrganizationAPI(orgImporter)
	userAPI := api.NewUserAPI(accessManager, db, log, errorHandler)
	networkAPI := api.NewNetworkAPI(log)

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
	emperror.Panic(emperror.Wrap(err, "failed to create SCMFactory"))

	sharedSpotguideOrg, err := spotguide.CreateSharedSpotguideOrganization(config.DB(), scmProvider, viper.GetString(config.SpotguideSharedLibraryGitHubOrganization))
	if err != nil {
		log.Errorf("failed to create shared Spotguide organization: %s", err)
	}

	spotguideManager := spotguide.NewSpotguideManager(config.DB(), version, scmFactory, sharedSpotguideOrg)

	// subscribe to organization creations and sync spotguides into the newly created organizations
	spotguide.AuthEventEmitter.NotifyOrganizationRegistered(func(orgID uint, userID uint) {
		if err := spotguideManager.ScrapeSpotguides(orgID, userID); err != nil {
			log.Warnf("failed to scrape Spotguide repositories for org [%d]: %s", orgID, err)
		}
	})

	// periodically sync shared spotguides
	syncTicker := time.NewTicker(viper.GetDuration(config.SpotguideSyncInterval))
	go func() {
		if err := spotguideManager.ScrapeSharedSpotguides(); err != nil {
			log.Errorf("failed to sync shared spotguides: %v", err)
		}

		for range syncTicker.C {
			if err := spotguideManager.ScrapeSharedSpotguides(); err != nil {
				log.Errorf("failed to sync shared spotguides: %v", err)
			}
		}
	}()

	spotguideAPI := api.NewSpotguideAPI(logger, errorHandler, spotguideManager)

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
			//v1.GET("/status", api.Status)
			orgs.GET("/:orgid/clusters", clusterAPI.GetClusters)
			orgs.GET("/:orgid/clusters/:id", clusterAPI.GetCluster)
			orgs.GET("/:orgid/clusters/:id/pods", api.GetPodDetails)
			orgs.GET("/:orgid/clusters/:id/bootstrap", clusterAPI.GetBootstrapInfo)
			orgs.PUT("/:orgid/clusters/:id", clusterAPI.UpdateCluster)

			orgs.PUT("/:orgid/clusters/:id/posthooks", clusterAPI.ReRunPostHooks)
			orgs.POST("/:orgid/clusters/:id/secrets", api.InstallSecretsToCluster)
			orgs.POST("/:orgid/clusters/:id/secrets/:secretName", api.InstallSecretToCluster)
			orgs.PATCH("/:orgid/clusters/:id/secrets/:secretName", api.MergeSecretInCluster)
			orgs.Any("/:orgid/clusters/:id/proxy/*path", clusterAPI.ProxyToCluster)
			orgs.DELETE("/:orgid/clusters/:id", clusterAPI.DeleteCluster)
			orgs.HEAD("/:orgid/clusters/:id", clusterAPI.ClusterCheck)
			orgs.GET("/:orgid/clusters/:id/config", api.GetClusterConfig)
			orgs.GET("/:orgid/clusters/:id/apiendpoint", api.GetApiEndpoint)
			orgs.GET("/:orgid/clusters/:id/nodes", api.GetClusterNodes)
			orgs.GET("/:orgid/clusters/:id/endpoints", api.ListEndpoints)
			orgs.GET("/:orgid/clusters/:id/secrets", api.ListClusterSecrets)
			orgs.GET("/:orgid/clusters/:id/deployments", api.ListDeployments)
			orgs.POST("/:orgid/clusters/:id/deployments", api.CreateDeployment)
			orgs.GET("/:orgid/clusters/:id/deployments/:name", api.GetDeployment)
			orgs.GET("/:orgid/clusters/:id/deployments/:name/resources", api.GetDeploymentResources)
			orgs.GET("/:orgid/clusters/:id/hpa", api.GetHpaResource)
			orgs.PUT("/:orgid/clusters/:id/hpa", api.PutHpaResource)
			orgs.DELETE("/:orgid/clusters/:id/hpa", api.DeleteHpaResource)
			orgs.HEAD("/:orgid/clusters/:id/deployments", api.GetTillerStatus)
			orgs.DELETE("/:orgid/clusters/:id/deployments/:name", api.DeleteDeployment)
			orgs.PUT("/:orgid/clusters/:id/deployments/:name", api.UpgradeDeployment)
			orgs.HEAD("/:orgid/clusters/:id/deployments/:name", api.HelmDeploymentStatus)
			orgs.POST("/:orgid/clusters/:id/helminit", api.InitHelmOnCluster)

			orgs.GET("/:orgid/clusters/:id/scanlog", api.GetScanLog)
			orgs.GET("/:orgid/clusters/:id/scanlog/:releaseName", api.GetScanLog)
			orgs.GET("/:orgid/clusters/:id/whitelists", api.GetWhiteLists)
			orgs.POST("/:orgid/clusters/:id/whitelists", api.CreateWhiteList)
			orgs.DELETE("/:orgid/clusters/:id/whitelists/:name", api.DeleteWhiteList)
			orgs.GET("/:orgid/clusters/:id/policies", api.GetPolicies)
			orgs.GET("/:orgid/clusters/:id/policies/:policyId", api.GetPolicies)
			orgs.POST("/:orgid/clusters/:id/policies", api.CreatePolicy)
			orgs.PUT("/:orgid/clusters/:id/policies/:policyId", api.UpdatePolicies)
			orgs.DELETE("/:orgid/clusters/:id/policies/:policyId", api.DeletePolicy)

			orgs.GET("/:orgid/clusters/:id/images", api.ListImages)
			orgs.GET("/:orgid/clusters/:id/images/:imageDigest/deployments", api.GetImageDeployments)
			orgs.GET("/:orgid/clusters/:id/deployments/:name/images", api.GetDeploymentImages)

			orgs.POST("/:orgid/clusters/:id/imagescan", api.ScanImages)
			orgs.GET("/:orgid/clusters/:id/imagescan/:imagedigest", api.GetScanResult)
			orgs.GET("/:orgid/clusters/:id/imagescan/:imagedigest/vuln", api.GetImageVulnerabilities)

			clusters := orgs.Group("/:orgid/clusters/:id")

			clusters.GET("/nodepools/labels", nplsApi.GetNodepoolLabelSets)
			clusters.POST("/nodepools/labels", nplsApi.SetNodepoolLabelSets)

			namespaceAPI := namespace.NewAPI(clusterGetter, errorHandler)
			namespaceAPI.RegisterRoutes(clusters.Group("/namespaces/:namespace"))

			pkeGroup := clusters.Group("/pke")

			leaderRepository, err := pke.NewVaultLeaderRepository()
			emperror.Panic(emperror.Wrap(err, "failed to create Vault leader repository"))

			pkeAPI := pke.NewAPI(clusterGetter, errorHandler, tokenHandler, externalBaseURL, workflowClient, leaderRepository)
			pkeAPI.RegisterRoutes(pkeGroup)

			clusterSecretStore := clustersecret.NewStore(
				clustersecretadapter.NewClusterManagerAdapter(clusterManager),
				clustersecretadapter.NewSecretStore(secret.Store),
			)

			clusterAuthService, err := intClusterAuth.NewDexClusterAuthService(clusterSecretStore)
			emperror.Panic(emperror.Wrap(err, "failed to create DexClusterAuthService"))

			pipelineExternalURL, err := url.Parse(externalBaseURL)
			emperror.Panic(emperror.Wrap(err, "failed to parse pipeline externalBaseURL"))

			pipelineExternalURL.Path = "/auth/dex/cluster/callback"

			clusterAuthAPI, err := api.NewClusterAuthAPI(
				clusterGetter,
				clusterAuthService,
				viper.GetString("auth.tokensigningkey"),
				viper.GetString("auth.dexURL"),
				pipelineExternalURL.String(),
			)
			emperror.Panic(emperror.Wrap(err, "failed to create ClusterAuthAPI"))

			clusterAuthAPI.RegisterRoutes(pkeGroup, router)

			orgs.GET("/:orgid/helm/repos", api.HelmReposGet)
			orgs.POST("/:orgid/helm/repos", api.HelmReposAdd)
			orgs.PUT("/:orgid/helm/repos/:name", api.HelmReposModify)
			orgs.PUT("/:orgid/helm/repos/:name/update", api.HelmReposUpdate)
			orgs.DELETE("/:orgid/helm/repos/:name", api.HelmReposDelete)
			orgs.GET("/:orgid/helm/charts", api.HelmCharts)
			orgs.GET("/:orgid/helm/chart/:reponame/:name", api.HelmChart)
			orgs.GET("/:orgid/profiles/cluster/:distribution", api.GetClusterProfiles)
			orgs.POST("/:orgid/profiles/cluster", api.AddClusterProfile)
			orgs.PUT("/:orgid/profiles/cluster", api.UpdateClusterProfile)
			orgs.DELETE("/:orgid/profiles/cluster/:distribution/:name", api.DeleteClusterProfile)
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
			orgs.POST("/:orgid/users/:id", userAPI.AddUser)
			orgs.DELETE("/:orgid/users/:id", userAPI.RemoveUser)

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

	arkEvents.NewClusterEventHandler(arkEvents.NewClusterEvents(clusterEventBus), config.DB(), logger)
	if viper.GetBool(config.ARKSyncEnabled) {
		go arkSync.RunSyncServices(
			context.Background(),
			config.DB(),
			arkClusterManager.New(clusterManager),
			platformlog.NewLogger(platformlog.Config{
				Level:  viper.GetString(config.ARKLogLevel),
				Format: viper.GetString(config.LoggingLogFormat),
			}).WithField("subsystem", "ark"),
			config.ErrorHandler(),
			viper.GetDuration(config.ARKBucketSyncInterval),
			viper.GetDuration(config.ARKRestoreSyncInterval),
			viper.GetDuration(config.ARKBackupSyncInterval),
		)
	}

	base.GET("api", api.MetaHandler(router, basePath+"/api"))

	issueHandler, err := api.NewIssueHandler(version, commitHash, buildDate)
	if err != nil {
		emperror.Panic(emperror.Wrap(err, "failed to create IssueHandler"))
	}
	base.POST("issues", auth.Handler, issueHandler)

	internalBindAddr := viper.GetString("pipeline.internalBindAddr")
	logger.Infof("Pipeline internal API listening on http://%s", internalBindAddr)
	go createInternalAPIRouter(skipPaths, db, basePath, clusterAPI).Run(internalBindAddr)

	bindAddr := viper.GetString("pipeline.bindaddr")
	if port := viper.GetInt("pipeline.listenport"); port != 0 {
		host := strings.Split(bindAddr, ":")[0]
		bindAddr = fmt.Sprintf("%s:%d", host, port)
		logger.Errorf("pipeline.listenport=%d setting is deprecated! Falling back to pipeline.bindaddr=%s", port, bindAddr)
	}
	certFile, keyFile := viper.GetString("pipeline.certfile"), viper.GetString("pipeline.keyfile")
	if certFile != "" && keyFile != "" {
		logger.Infof("Pipeline API listening on https://%s", bindAddr)
		router.RunTLS(bindAddr, certFile, keyFile)
	} else {
		logger.Infof("Pipeline API listening on http://%s", bindAddr)
		router.Run(bindAddr)
	}
}

func createInternalAPIRouter(skipPaths []string, db *gorm.DB, basePath string, clusterAPI *api.ClusterAPI) *gin.Engine {
	//Initialise Gin router for Internal API
	internalRouter := gin.New()
	internalRouter.Use(correlationid.Middleware())
	internalRouter.Use(ginlog.Middleware(log, skipPaths...))
	internalRouter.Use(gin.Recovery())
	if viper.GetBool("audit.enabled") {
		log.Infoln("Audit enabled, installing Gin audit middleware to internal router")
		internalRouter.Use(audit.LogWriter(skipPaths, viper.GetStringSlice("audit.headers"), db, log))
	}
	internalGroup := internalRouter.Group(path.Join(basePath, "api", "v1/", "orgs"))
	internalGroup.Use(auth.InternalUserHandler)
	internalGroup.Use(api.OrganizationMiddleware)
	internalGroup.GET("/:orgid/clusters/:id/nodepools", api.GetNodePools)
	internalGroup.PUT("/:orgid/clusters/:id/nodepools", clusterAPI.UpdateNodePools)
	return internalRouter
}
