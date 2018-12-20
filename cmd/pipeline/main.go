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
	"os"
	"path"
	"strings"
	"time"

	"github.com/banzaicloud/pipeline/spotguide"

	evbus "github.com/asaskevich/EventBus"
	"github.com/banzaicloud/go-gin-prometheus"
	"github.com/banzaicloud/pipeline/api"
	"github.com/banzaicloud/pipeline/api/ark/backups"
	"github.com/banzaicloud/pipeline/api/ark/backupservice"
	"github.com/banzaicloud/pipeline/api/ark/buckets"
	"github.com/banzaicloud/pipeline/api/ark/restores"
	"github.com/banzaicloud/pipeline/api/ark/schedules"
	"github.com/banzaicloud/pipeline/api/cluster/namespace"
	"github.com/banzaicloud/pipeline/api/common"
	"github.com/banzaicloud/pipeline/api/middleware"
	"github.com/banzaicloud/pipeline/auth"
	"github.com/banzaicloud/pipeline/cluster"
	"github.com/banzaicloud/pipeline/config"
	"github.com/banzaicloud/pipeline/dns"
	arkEvents "github.com/banzaicloud/pipeline/internal/ark/events"
	arkSync "github.com/banzaicloud/pipeline/internal/ark/sync"
	"github.com/banzaicloud/pipeline/internal/audit"
	intAuth "github.com/banzaicloud/pipeline/internal/auth"
	intCluster "github.com/banzaicloud/pipeline/internal/cluster"
	"github.com/banzaicloud/pipeline/internal/dashboard"
	"github.com/banzaicloud/pipeline/internal/monitor"
	"github.com/banzaicloud/pipeline/internal/notification"
	ginternal "github.com/banzaicloud/pipeline/internal/platform/gin"
	"github.com/banzaicloud/pipeline/internal/platform/gin/correlationid"
	ginlog "github.com/banzaicloud/pipeline/internal/platform/gin/log"
	platformlog "github.com/banzaicloud/pipeline/internal/platform/log"
	"github.com/banzaicloud/pipeline/model/defaults"
	"github.com/banzaicloud/pipeline/pkg/k8sclient"
	"github.com/banzaicloud/pipeline/pkg/providers"
	"github.com/banzaicloud/pipeline/secret"
	"github.com/casbin/gorm-adapter"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/goph/emperror"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

//Common logger for package
var log *logrus.Logger
var logger *logrus.Entry

func initLog() *logrus.Entry {
	log = config.Logger()
	logger := log.WithFields(logrus.Fields{"state": "init"})
	return logger
}

func main() {
	if len(os.Args) > 1 && os.Args[1] == "--version" {
		if CommitHash == "" {
			fmt.Println("version: ", Version, " built on ", BuildDate)
		} else {
			fmt.Printf("version: %s-%s built on %s\n", Version, CommitHash, BuildDate)
		}
		os.Exit(0)
	}

	logger = initLog()
	logger.WithFields(logrus.Fields{
		"version":     Version,
		"commit_hash": CommitHash,
		"build_date":  BuildDate,
	}).Info("Pipeline initialization")
	errorHandler := config.ErrorHandler()

	// Connect to database
	db := config.DB()
	cicdDB, err := config.CICDDB()
	if err != nil {
		logger.Panic(err.Error())
	}

	casbinDSN, err := config.CasbinDSN()
	if err != nil {
		logger.Panic(err.Error())
	}

	basePath := viper.GetString("pipeline.basepath")

	casbinAdapter := gormadapter.NewAdapter("mysql", casbinDSN, true)
	enforcer := intAuth.NewEnforcer(casbinAdapter)
	enforcer.StartAutoLoadPolicy(10 * time.Second)
	accessManager := intAuth.NewAccessManager(enforcer, basePath)

	accessManager.AddDefaultPolicies()

	githubImporter := auth.NewGithubImporter(db, accessManager, config.EventBus)

	// Initialize auth
	auth.Init(cicdDB, accessManager, githubImporter)

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
	clusterManager := cluster.NewManager(clusters, secretValidator, clusterEvents, statusChangeDurationMetric, clusterTotalMetric, log, errorHandler)
	clusterGetter := common.NewClusterGetter(clusterManager, logger, errorHandler)

	if viper.GetBool(config.MonitorEnabled) {
		client, err := k8sclient.NewInClusterClient()
		if err != nil {
			errorHandler.Handle(emperror.Wrap(err, "failed to enable monitoring"))
		} else {
			monitorClusterSubscriber := monitor.NewClusterSubscriber(
				client,
				clusterManager,
				db,
				viper.GetString(config.DNSBaseDomain),
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

	clusterAPI := api.NewClusterAPI(clusterManager, clusterGetter, log, errorHandler)

	//Initialise Gin router
	router := gin.New()

	// These two paths can contain sensitive information, so it is advised not to log them out.
	skipPaths := viper.GetStringSlice("audit.skippaths")
	router.Use(correlationid.Middleware())
	router.Use(ginlog.Middleware(log, skipPaths...))
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

	router.GET("/version", VersionHandler)
	router.GET(path.Join(basePath, "notifications"), notification.GetNotifications)

	root := router.Group("/")
	{
		root.GET("/", api.RedirectRoot)
	}
	// Add prometheus metric endpoint
	if viper.GetBool(config.MetricsEnabled) {
		p := ginprometheus.NewPrometheus("pipeline", []string{})
		p.SetListenAddress(viper.GetString(config.MetricsPort))
		p.Use(router, "/metrics")
	}

	generateTokenHandler := auth.NewTokenHandler(accessManager)

	auth.Install(router, generateTokenHandler)
	auth.StartTokenStoreGC()

	authorizationMiddleware := intAuth.NewMiddleware(enforcer, basePath)

	dgroup := router.Group(path.Join(basePath, "dashboard", "orgs"))
	dgroup.Use(auth.Handler)
	dgroup.Use(authorizationMiddleware)
	dgroup.Use(api.OrganizationMiddleware)
	dgroup.GET("/:orgid/clusters", dashboard.GetDashboard)

	domainAPI := api.NewDomainAPI(clusterManager, log, errorHandler)
	organizationAPI := api.NewOrganizationAPI(githubImporter)
	userAPI := api.NewUserAPI(accessManager)
	spotguideManager := spotguide.NewSpotguideManager(config.DB(), Version, viper.GetString("github.token"), viper.GetString(config.SpotguideSharedLibraryGitHubOrganization))

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

	v1 := router.Group(path.Join(basePath, "api", "v1/"))
	v1.GET("/functions", api.ListFunctions)
	v1.GET("/securityscan", api.SecurytiScanEnabled)
	{
		v1.Use(auth.Handler)
		v1.GET("/me", userAPI.GetCurrentUser)
		v1.Use(authorizationMiddleware)
		orgs := v1.Group("/orgs")
		{
			orgs.Use(api.OrganizationMiddleware)

			orgs.GET("/:orgid/spotguides", spotguideAPI.GetSpotguides)
			orgs.PUT("/:orgid/spotguides", middleware.NewRateLimiterByOrgID(api.SyncSpotguidesRateLimit), spotguideAPI.SyncSpotguides)
			orgs.POST("/:orgid/spotguides", spotguideAPI.LaunchSpotguide)
			// Spotguide name may contain '/'s so we have to use :owner/:name
			orgs.GET("/:orgid/spotguides/:owner/:name", spotguideAPI.GetSpotguide)
			orgs.HEAD("/:orgid/spotguides/:owner/:name", spotguideAPI.GetSpotguide)
			orgs.GET("/:orgid/spotguides/:owner/:name/icon", spotguideAPI.GetSpotguideIcon)

			orgs.GET("/:orgid/domain", domainAPI.GetDomain)

			orgs.POST("/:orgid/clusters", clusterAPI.CreateClusterRequest)
			//v1.GET("/status", api.Status)
			orgs.GET("/:orgid/clusters", clusterAPI.GetClusters)
			orgs.GET("/:orgid/clusters/:id", clusterAPI.GetCluster)
			orgs.GET("/:orgid/clusters/:id/details", clusterAPI.GetCluster) // Deprecated: use /:orgid/clusters/:id instead
			orgs.GET("/:orgid/clusters/:id/pods", api.GetPodDetails)
			orgs.PUT("/:orgid/clusters/:id", clusterAPI.UpdateCluster)
			orgs.PUT("/:orgid/clusters/:id/posthooks", api.ReRunPostHooks)
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
			namespaceAPI := namespace.NewAPI(clusterGetter, errorHandler)
			namespaceAPI.RegisterRoutes(clusters.Group("/namespaces/:namespace"))

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

			orgs.GET("/:orgid/cloudinfo", api.GetSupportedClusterList)
			orgs.GET("/:orgid/cloudinfo/:cloudtype", api.GetCloudInfo)

			orgs.GET("/:orgid/azure/resourcegroups", api.GetResourceGroups)
			orgs.POST("/:orgid/azure/resourcegroups", api.AddResourceGroups)
			orgs.DELETE("/:orgid/azure/resourcegroups/:name", api.DeleteResourceGroups)

			orgs.GET("/:orgid/google/projects", api.GetProjects)

			orgs.GET("/:orgid", organizationAPI.GetOrganizations)
			orgs.DELETE("/:orgid", organizationAPI.DeleteOrganization)
		}
		v1.GET("/orgs", organizationAPI.GetOrganizations)
		v1.PUT("/orgs", organizationAPI.SyncOrganizations)
		v1.GET("/token", generateTokenHandler) // TODO Deprecated, should be removed once the UI has support.
		v1.POST("/tokens", generateTokenHandler)
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
		backups.AddOrgRoutes(orgs.Group("/:orgid/backups"))
	}

	if viper.GetBool(config.ARKSyncEnabled) {
		arkEvents.NewClusterEventHandler(arkEvents.NewClusterEvents(clusterEventBus), config.DB(), logger)
		go arkSync.RunSyncServices(
			context.Background(),
			config.DB(),
			clusterManager,
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

	router.GET(basePath+"/api", api.MetaHandler(router, basePath+"/api"))

	issueHandler, err := api.NewIssueHandler(Version, CommitHash, BuildDate)
	if err != nil {
		panic(err)
	}
	router.POST(basePath+"/issues", auth.Handler, issueHandler)

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
