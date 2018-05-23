package main

import (
	"fmt"
	"os"

	"github.com/banzaicloud/pipeline/api"
	"github.com/banzaicloud/pipeline/auth"
	"github.com/banzaicloud/pipeline/config"
	"github.com/banzaicloud/pipeline/model"
	"github.com/banzaicloud/pipeline/model/defaults"
	"github.com/banzaicloud/pipeline/notify"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/qor/auth/auth_identity"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

//Version of Pipeline
var Version string

//GitRev of Pipeline
var GitRev string

//Common logger for package
var logger *logrus.Entry

// Initialize database

//This WILL be move to CommonTypes!!!

func initLog() *logrus.Entry {
	log := config.Logger()
	logger := log.WithFields(logrus.Fields{"state": "init"})
	return logger
}

func main() {

	if len(os.Args) > 1 && os.Args[1] == "--version" {
		if GitRev == "" {
			fmt.Println("version:", Version)
		} else {
			fmt.Printf("version: %s-%s\n", Version, GitRev)
		}
		os.Exit(0)
	}

	logger = initLog()
	logger.Info("Pipeline initialization")

	// Ensure DB connection
	db := model.GetDB()
	// Initialise auth
	auth.Init()

	// Creating tables if not exists
	logger.Info("Create table(s):",
		model.ClusterModel.TableName(model.ClusterModel{}),
		model.AmazonClusterModel.TableName(model.AmazonClusterModel{}),
		model.AzureClusterModel.TableName(model.AzureClusterModel{}),
		model.AzureNodePoolModel.TableName(model.AzureNodePoolModel{}),
		model.GoogleClusterModel.TableName(model.GoogleClusterModel{}),
		model.GoogleNodePoolModel.TableName(model.GoogleNodePoolModel{}),
	)

	// Create tables
	if err := db.AutoMigrate(
		&model.ClusterModel{},
		&model.AmazonClusterModel{},
		&model.AmazonNodePoolsModel{},
		&model.AzureClusterModel{},
		&model.AzureNodePoolModel{},
		&model.GoogleClusterModel{},
		&model.GoogleNodePoolModel{},
		&model.DummyClusterModel{},
		&model.KubernetesClusterModel{},
		&auth_identity.AuthIdentity{},
		&auth.User{},
		&auth.UserOrganization{},
		&auth.Organization{},
		&defaults.AWSProfile{},
		&defaults.AWSNodePoolProfile{},
		&defaults.AKSProfile{},
		&defaults.AKSNodePoolProfile{},
		&defaults.GKEProfile{},
		&defaults.GKENodePoolProfile{},
	).Error; err != nil {

		panic(err)
	}

	defaults.SetDefaultValues()

	router := gin.New()

	// These two paths can contain sensitive information, so it is advised not to log them out.
	router.Use(gin.LoggerWithWriter(gin.DefaultWriter, "/auth/tokens", "/auth/github/callback"))
	router.Use(gin.Recovery())
	router.Use(cors.New(config.GetCORS()))

	auth.Install(router)

	root := router.Group("/")
	{
		root.GET("/", api.RedirectRoot)
	}

	basePath := viper.GetString("pipeline.basepath")
	v1 := router.Group(basePath + "/api/v1/")
	{
		v1.Use(auth.Handler)
		v1.Use(auth.NewAuthorizer())
		orgs := v1.Group("/orgs")
		{
			orgs.Use(api.OrganizationMiddleware)
			orgs.POST("/:orgid/clusters", api.CreateCluster)
			//v1.GET("/status", api.Status)
			orgs.GET("/:orgid/clusters", api.FetchClusters)
			orgs.GET("/:orgid/clusters/:id", api.GetClusterStatus)
			orgs.GET("/:orgid/clusters/:id/details", api.FetchCluster)
			orgs.PUT("/:orgid/clusters/:id", api.UpdateCluster)
			orgs.POST("/:orgid/clusters/:id/secrets", api.InstallSecretsToCluster)
			orgs.DELETE("/:orgid/clusters/:id", api.DeleteCluster)
			orgs.HEAD("/:orgid/clusters/:id", api.FetchCluster)
			orgs.GET("/:orgid/clusters/:id/config", api.GetClusterConfig)
			orgs.GET("/:orgid/clusters/:id/apiendpoint", api.GetApiEndpoint)
			orgs.GET("/:orgid/clusters/:id/nodes", api.GetClusterNodes)
			orgs.POST("/:orgid/clusters/:id/monitoring", api.UpdateMonitoring)
			orgs.GET("/:orgid/clusters/:id/endpoints", api.ListEndpoints)
			orgs.GET("/:orgid/clusters/:id/deployments", api.ListDeployments)
			orgs.POST("/:orgid/clusters/:id/deployments", api.CreateDeployment)
			orgs.HEAD("/:orgid/clusters/:id/deployments", api.GetTillerStatus)
			orgs.DELETE("/:orgid/clusters/:id/deployments/:name", api.DeleteDeployment)
			orgs.PUT("/:orgid/clusters/:id/deployments/:name", api.UpgradeDeployment)
			orgs.HEAD("/:orgid/clusters/:id/deployments/:name", api.HelmDeploymentStatus)
			orgs.POST("/:orgid/clusters/:id/helminit", api.InitHelmOnCluster)
			orgs.GET("/:orgid/clusters/:id/helm/repos", api.HelmReposGet)
			orgs.POST("/:orgid/clusters/:id/helm/repos", api.HelmReposAdd)
			orgs.PUT("/:orgid/clusters/:id/helm/repos/:name", api.HelmReposModify)
			orgs.PUT("/:orgid/clusters/:id/helm/repos/:name/update", api.HelmReposUpdate)
			orgs.DELETE("/:orgid/clusters/:id/helm/repos/:name", api.HelmReposDelete)
			orgs.GET("/:orgid/clusters/:id/helm/charts", api.HelmCharts)
			orgs.GET("/:orgid/clusters/:id/helm/chart/:reponame/:name/:version", api.HelmChart)
			orgs.GET("/:orgid/clusters/:id/helm/chart/:reponame/:name", api.HelmChart)
			orgs.GET("/:orgid/profiles/cluster/:type", api.GetClusterProfiles)
			orgs.POST("/:orgid/profiles/cluster", api.AddClusterProfile)
			orgs.PUT("/:orgid/profiles/cluster", api.UpdateClusterProfile)
			orgs.DELETE("/:orgid/profiles/cluster/:type/:name", api.DeleteClusterProfile)
			orgs.GET("/:orgid/secrets", api.ListSecrets)
			orgs.POST("/:orgid/secrets", api.AddSecrets)
			orgs.DELETE("/:orgid/secrets/:secretid", api.DeleteSecrets)
			orgs.GET("/:orgid/users", api.GetUsers)
			orgs.GET("/:orgid/users/:id", api.GetUsers)
			orgs.POST("/:orgid/users/:id", api.AddUser)
			orgs.DELETE("/:orgid/users/:id", api.RemoveUser)

			orgs.GET("/:orgid/cloudinfo", api.GetSupportedClusterList)
			orgs.GET("/:orgid/cloudinfo/filters", api.GetSupportedFilters)
			orgs.POST("/:orgid/cloudinfo/:cloudtype", api.GetCloudInfo)

			orgs.GET("/:orgid/allowed/secrets/", api.ListAllowedSecretTypes)
			orgs.GET("/:orgid/allowed/secrets/:type", api.ListAllowedSecretTypes)

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
	}

	router.GET(basePath+"/api", api.MetaHandler(router, basePath+"/api"))

	notify.SlackNotify("API is already running")
	var listenPort string
	port := viper.GetInt("pipeline.listenport")
	if port != 0 {
		listenPort = fmt.Sprintf(":%d", port)
		logger.Info("Pipeline API listening on port ", listenPort)
	}
	router.Run(listenPort)
}
