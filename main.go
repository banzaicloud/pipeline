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
	"fmt"
	"os"
	"strings"

	"github.com/banzaicloud/go-gin-prometheus"
	"github.com/banzaicloud/pipeline/api"
	"github.com/banzaicloud/pipeline/auth"
	"github.com/banzaicloud/pipeline/config"
	"github.com/banzaicloud/pipeline/dns"
	"github.com/banzaicloud/pipeline/dns/route53/model"
	"github.com/banzaicloud/pipeline/internal/audit"
	"github.com/banzaicloud/pipeline/internal/platform/gin/correlationid"
	ginlog "github.com/banzaicloud/pipeline/internal/platform/gin/log"
	"github.com/banzaicloud/pipeline/model"
	"github.com/banzaicloud/pipeline/model/defaults"
	"github.com/banzaicloud/pipeline/notify"
	"github.com/banzaicloud/pipeline/spotguide"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// Provisioned by ldflags
var (
	Version   string
	GitRev    string
	BuildDate string
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
		if GitRev == "" {
			fmt.Println("version: ", Version, " built on ", BuildDate)
		} else {
			fmt.Printf("version: %s-%s built on %s\n", Version, GitRev, BuildDate)
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

	var tables = []interface{}{&model.ClusterModel{},
		&model.ACSKClusterModel{},
		&model.ACSKNodePoolModel{},
		&model.AmazonNodePoolsModel{},
		&model.EC2ClusterModel{},
		&model.EKSClusterModel{},
		&model.AKSClusterModel{},
		&model.AKSNodePoolModel{},
		&model.DummyClusterModel{},
		&model.KubernetesClusterModel{},
		&auth.AuthIdentity{},
		&auth.User{},
		&auth.UserOrganization{},
		&auth.Organization{},
		&defaults.EC2Profile{},
		&defaults.EC2NodePoolProfile{},
		&defaults.EKSProfile{},
		&defaults.EKSNodePoolProfile{},
		&defaults.AKSProfile{},
		&defaults.AKSNodePoolProfile{},
		&defaults.GKEProfile{},
		&defaults.GKENodePoolProfile{},
		&route53model.Route53Domain{},
		&spotguide.Repo{},
	}

	var tableNames string
	for _, table := range tables {
		tableNames += fmt.Sprintf(" %s", db.NewScope(table).TableName())
	}

	logger.WithFields(logrus.Fields{
		"table_names": strings.TrimSpace(tableNames),
	}).Info("migrating tables")

	// Create tables
	if err := db.AutoMigrate(tables...).Error; err != nil {
		panic(err)
	}

	err = Migrate(db, logger)
	if err != nil {
		panic(err)
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

	// Spotguides
	go func() {
		err := spotguide.ScrapeSpotguides()
		if err != nil {
			errorHandler.Handle(errors.Wrap(err, "failed to scrape Spotguide repositories"))
		}
	}()

	//Initialise Gin router
	router := gin.New()

	// These two paths can contain sensitive information, so it is advised not to log them out.
	skipPaths := viper.GetStringSlice("audit.skippaths")
	router.Use(correlationid.Middleware())
	router.Use(ginlog.Middleware(log, skipPaths...))
	router.Use(gin.Recovery())
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
	v1 := router.Group(basePath + "/api/v1/")
	v1.GET("/functions", api.ListFunctions)
	{
		v1.Use(auth.Handler)
		v1.Use(auth.NewAuthorizer(casbinDSN))
		orgs := v1.Group("/orgs")
		{
			orgs.Use(api.OrganizationMiddleware)

			orgs.GET("/:orgid/spotguides", api.GetSpotguides)
			orgs.PUT("/:orgid/spotguides", api.SyncSpotguides)
			orgs.POST("/:orgid/spotguides", api.LaunchSpotguide)
			// Spotguide name may contain '/'s so we have to use *name
			orgs.GET("/:orgid/spotguides/*name", api.GetSpotguide)
			orgs.HEAD("/:orgid/spotguides/*name", api.GetSpotguide)

			orgs.POST("/:orgid/clusters", api.CreateClusterRequest)
			//v1.GET("/status", api.Status)
			orgs.GET("/:orgid/clusters", api.GetClusters)
			orgs.GET("/:orgid/clusters/:id", api.GetClusterStatus)
			orgs.GET("/:orgid/clusters/:id/details", api.GetClusterDetails)
			orgs.GET("/:orgid/clusters/:id/pods", api.GetPodDetails)
			orgs.PUT("/:orgid/clusters/:id", api.UpdateCluster)
			orgs.PUT("/:orgid/clusters/:id/posthooks", api.ReRunPostHooks)
			orgs.POST("/:orgid/clusters/:id/secrets", api.InstallSecretsToCluster)
			orgs.Any("/:orgid/clusters/:id/proxy/*path", api.ProxyToCluster)
			orgs.DELETE("/:orgid/clusters/:id", api.DeleteCluster)
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

			orgs.GET("/:orgid/buckets", api.ListBuckets)
			orgs.POST("/:orgid/buckets", api.CreateBucket)
			orgs.HEAD("/:orgid/buckets/:name", api.CheckBucket)
			orgs.DELETE("/:orgid/buckets/:name", api.DeleteBucket)

			orgs.GET("/:orgid/cloudinfo", api.GetSupportedClusterList)
			orgs.GET("/:orgid/cloudinfo/:cloudtype", api.GetCloudInfo)

			orgs.GET("/:orgid/azure/resourcegroups", api.GetResourceGroups)
			orgs.POST("/:orgid/azure/resourcegroups", api.AddResourceGroups)
			orgs.DELETE("/:orgid/azure/resourcegroups/:name", api.DeleteResourceGroups)

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
	}

	router.GET(basePath+"/api", api.MetaHandler(router, basePath+"/api"))

	notify.SlackNotify("API is already running")
	var listenPort string
	port := viper.GetInt("pipeline.listenport")
	if port != 0 {
		listenPort = fmt.Sprintf(":%d", port)
		logger.Info("Pipeline API listening on port ", listenPort)
	}

	certFile, keyFile := viper.GetString("pipeline.certfile"), viper.GetString("pipeline.keyfile")
	if certFile != "" && keyFile != "" {
		router.RunTLS(listenPort, certFile, keyFile)
	} else {
		router.Run(listenPort)
	}
}
