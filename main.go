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

	evbus "github.com/asaskevich/EventBus"
	"github.com/banzaicloud/go-gin-prometheus"
	"github.com/banzaicloud/pipeline/api"
	"github.com/banzaicloud/pipeline/api/ark/backups"
	"github.com/banzaicloud/pipeline/api/ark/backupservice"
	"github.com/banzaicloud/pipeline/api/ark/buckets"
	"github.com/banzaicloud/pipeline/api/ark/restores"
	"github.com/banzaicloud/pipeline/api/ark/schedules"
	"github.com/banzaicloud/pipeline/auth"
	"github.com/banzaicloud/pipeline/cluster"
	"github.com/banzaicloud/pipeline/config"
	"github.com/banzaicloud/pipeline/dns"
	arkSync "github.com/banzaicloud/pipeline/internal/ark/sync"
	"github.com/banzaicloud/pipeline/internal/audit"
	intCluster "github.com/banzaicloud/pipeline/internal/cluster"
	"github.com/banzaicloud/pipeline/internal/dashboard"
	"github.com/banzaicloud/pipeline/internal/monitor"
	ginternal "github.com/banzaicloud/pipeline/internal/platform/gin"
	"github.com/banzaicloud/pipeline/internal/platform/gin/correlationid"
	ginlog "github.com/banzaicloud/pipeline/internal/platform/gin/log"
	platformlog "github.com/banzaicloud/pipeline/internal/platform/log"
	"github.com/banzaicloud/pipeline/model/defaults"
	"github.com/banzaicloud/pipeline/notify"
	"github.com/banzaicloud/pipeline/pkg/k8sclient"
	"github.com/banzaicloud/pipeline/pkg/providers"
	"github.com/banzaicloud/pipeline/secret"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/goph/emperror"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// Provisioned by ldflags
var (
	Version    string
	CommitHash string
	BuildDate  string
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
	logger.Info("Pipeline initialization")
	errorHandler := config.ErrorHandler()

	// Connect to database
	db := config.DB()
	droneDb, err := config.DroneDB()
	if err != nil {
		logger.Panic(err.Error())
	}

	casbinDSN, err := config.CasbinDSN()
	if err != nil {
		logger.Panic(err.Error())
	}

	// Initialize auth
	auth.Init(droneDb)

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

	clusterEventBus := evbus.New()
	clusterEvents := cluster.NewClusterEvents(clusterEventBus)
	clusters := intCluster.NewClusters(db)
	secretValidator := providers.NewSecretValidator(secret.Store)
	clusterManager := cluster.NewManager(clusters, secretValidator, clusterEvents, log, errorHandler)

	if viper.GetBool(config.MonitorEnabled) {
		client, err := k8sclient.NewInClusterClient()
		if err != nil {
			errorHandler.Handle(emperror.Wrap(err, "failed to enable monitoring"))
		} else {
			monitorClusterSubscriber := monitor.NewClusterSubscriber(
				client,
				clusterManager,
				db,
				viper.GetString(config.ControlPlaneNamespace),
				viper.GetString(config.PipelineSystemNamespace),
				viper.GetString(config.MonitorConfigMap),
				viper.GetString(config.MonitorConfigMapPrometheusKey),
				viper.GetString(config.MonitorCertSecret),
				viper.GetString(config.MonitorCertMountPath),
				errorHandler,
			)
			monitorClusterSubscriber.Register(monitor.NewClusterEvents(clusterEventBus))
		}
	}

	clusterAPI := api.NewClusterAPI(clusterManager, log, errorHandler)

	//Initialise Gin router
	router := gin.New()

	// These two paths can contain sensitive information, so it is advised not to log them out.
	skipPaths := viper.GetStringSlice("audit.skippaths")
	router.Use(correlationid.Middleware())
	router.Use(ginlog.Middleware(log, skipPaths...))
	router.Use(gin.Recovery())
	router.Use(ginternal.NewDrainModeMiddleware(errorHandler).Middleware)
	router.Use(cors.New(config.GetCORS()))
	if viper.GetBool("audit.enabled") {
		log.Infoln("Audit enabled, installing Gin audit middleware")
		router.Use(audit.LogWriter(skipPaths, viper.GetStringSlice("audit.headers"), db, log))
	}

	root := router.Group("/")
	{
		root.GET("/", api.RedirectRoot)
	}
	// Add prometheus metric endpoint
	if viper.GetBool("metrics.enabled") {
		p := ginprometheus.NewPrometheus("http", []string{})
		p.SetListenAddress(viper.GetString("metrics.port"))
		p.Use(router)
	}

	auth.Install(router)

	basePath := viper.GetString("pipeline.basepath")

	authorizer := auth.NewAuthorizer(casbinDSN)

	dgroup := router.Group(path.Join(basePath, "dashboard", "orgs"))
	dgroup.Use(auth.Handler)
	dgroup.Use(authorizer)
	dgroup.Use(api.OrganizationMiddleware)
	dgroup.GET("/:orgid/clusters", dashboard.GetDashboard)

	v1 := router.Group(path.Join(basePath, "api", "v1/"))
	v1.GET("/functions", api.ListFunctions)
	{
		v1.Use(auth.Handler)
		v1.Use(authorizer)
		orgs := v1.Group("/orgs")
		{
			orgs.Use(api.OrganizationMiddleware)

			orgs.GET("/:orgid/spotguides", api.GetSpotguides)
			orgs.PUT("/:orgid/spotguides", api.SyncSpotguides)
			orgs.POST("/:orgid/spotguides", api.LaunchSpotguide)
			// Spotguide name may contain '/'s so we have to use *name
			orgs.GET("/:orgid/spotguides/*name", api.GetSpotguide)
			orgs.HEAD("/:orgid/spotguides/*name", api.GetSpotguide)

			orgs.POST("/:orgid/clusters", clusterAPI.CreateClusterRequest)
			//v1.GET("/status", api.Status)
			orgs.GET("/:orgid/clusters", clusterAPI.GetClusters)
			orgs.GET("/:orgid/clusters/:id", api.GetClusterStatus)
			orgs.GET("/:orgid/clusters/:id/details", api.GetClusterDetails)
			orgs.GET("/:orgid/clusters/:id/pods", api.GetPodDetails)
			orgs.PUT("/:orgid/clusters/:id", clusterAPI.UpdateCluster)
			orgs.PUT("/:orgid/clusters/:id/posthooks", api.ReRunPostHooks)
			orgs.POST("/:orgid/clusters/:id/secrets", api.InstallSecretsToCluster)
			orgs.POST("/:orgid/clusters/:id/secrets/:secretName", api.InstallSecretToCluster)
			orgs.PATCH("/:orgid/clusters/:id/secrets/:secretName", api.MergeSecretInCluster)
			orgs.Any("/:orgid/clusters/:id/proxy/*path", api.ProxyToCluster)
			orgs.DELETE("/:orgid/clusters/:id", clusterAPI.DeleteCluster)
			orgs.HEAD("/:orgid/clusters/:id", api.ClusterHEAD)
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
			orgs.GET("/:orgid/users", api.GetUsers)
			orgs.GET("/:orgid/users/:id", api.GetUsers)
			orgs.POST("/:orgid/users/:id", api.AddUser)
			orgs.DELETE("/:orgid/users/:id", api.RemoveUser)

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

			orgs.GET("/:orgid", api.GetOrganizations)
			orgs.DELETE("/:orgid", api.DeleteOrganization)
		}
		v1.GET("/orgs", api.GetOrganizations)
		v1.POST("/orgs", api.CreateOrganization)
		v1.GET("/token", auth.GenerateToken) // TODO Deprecated, should be removed once the UI has support.
		v1.POST("/tokens", auth.GenerateToken)
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

	notify.SlackNotify("API is already running")
	var listenPort string
	port := viper.GetInt("pipeline.listenport")
	if port != 0 {
		listenPort = fmt.Sprintf(":%d", port)
	}

	certFile, keyFile := viper.GetString("pipeline.certfile"), viper.GetString("pipeline.keyfile")
	if certFile != "" && keyFile != "" {
		logger.Info("Pipeline API listening on TLS port ", listenPort)
		router.RunTLS(listenPort, certFile, keyFile)
	} else {
		logger.Info("Pipeline API listening on port ", listenPort)
		router.Run(listenPort)
	}
}
