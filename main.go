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
	"github.com/banzaicloud/pipeline/utils"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/qor/auth/auth_identity"
	sessionManager "github.com/qor/session/manager"
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
		model.GoogleClusterModel.TableName(model.GoogleClusterModel{}))

	// Create tables
	if err := db.AutoMigrate(
		&model.ClusterModel{},
		&model.AmazonClusterModel{},
		&model.AzureClusterModel{},
		&model.GoogleClusterModel{},
		&auth_identity.AuthIdentity{},
		&auth.User{},
		&auth.UserOrganization{},
		&auth.Organization{},
		&defaults.AWSProfile{},
		&defaults.AKSProfile{},
		&defaults.GKEProfile{}).Error; err != nil {

		panic(err)
	}

	defaults.SetDefaultValues()

	router := gin.Default()

	router.Use(cors.New(config.GetCORS()))

	authHandler := gin.WrapH(auth.Auth.NewServeMux())

	// We have to make the raw net/http handlers a bit Gin-ish
	router.Use(gin.WrapH(sessionManager.SessionManager.Middleware(utils.NopHandler{})))
	router.Use(gin.WrapH(auth.RedirectBack.Middleware(utils.NopHandler{})))

	authGroup := router.Group("/auth/")
	{
		authGroup.GET("/*w", authHandler)
		authGroup.GET("/*w/*w", authHandler)
	}

	v1 := router.Group("/api/v1/")
	{
		v1.Use(auth.Handler)
		orgs := v1.Group("/orgs")
		{
			orgs.Use(api.OrganizationMiddleware)
			orgs.POST("/:orgid/clusters", api.CreateCluster)
			//v1.GET("/status", api.Status)
			orgs.GET("/:orgid/clusters", api.FetchClusters)
			orgs.GET("/:orgid/clusters/:id", api.FetchCluster)
			orgs.PUT("/:orgid/clusters/:id", api.UpdateCluster)
			orgs.DELETE("/:orgid/clusters/:id", api.DeleteCluster)
			orgs.HEAD("/:orgid/clusters/:id", api.GetClusterStatus)
			orgs.GET("/:orgid/clusters/:id/config", api.GetClusterConfig)
			orgs.GET("/:orgid/clusters/:id/apiendpoint", api.GetApiEndpoint)
			orgs.POST("/:orgid/clusters/:id/monitoring", api.UpdateMonitoring)
			orgs.GET("/:orgid/clusters/:id/endpoints", api.ListEndpoints)
			orgs.GET("/:orgid/clusters/:id/deployments", api.ListDeployments)
			orgs.POST("/:orgid/clusters/:id/deployments", api.CreateDeployment)
			orgs.HEAD("/:orgid/clusters/:id/deployments", api.GetTillerStatus)
			orgs.DELETE("/:orgid/clusters/:id/deployments/:name", api.DeleteDeployment)
			orgs.PUT("/:orgid/clusters/:id/deployments/:name", api.UpgradeDeployment)
			orgs.HEAD("/:orgid/clusters/:id/deployments/:name", api.HelmDeploymentStatus)
			orgs.POST("/:orgid/clusters/:id/helminit", api.InitHelmOnCluster)
			orgs.GET("/:orgid/profiles/cluster/:type", api.GetClusterProfiles)
			orgs.POST("/:orgid/profiles/cluster", api.AddClusterProfile)
			orgs.PUT("/:orgid/profiles/cluster", api.UpdateClusterProfile)
			orgs.DELETE("/:orgid/profiles/cluster/:type/:name", api.DeleteClusterProfile)
			orgs.GET("/:orgid/secrets", api.ListSecrets)
			orgs.POST("/:orgid/secrets", api.AddSecrets)
			orgs.DELETE("/:orgid/secrets/:secretid", api.DeleteSecrets)
		}
		//v1.GET("/clusters/gke/:projectid/:zone/serverconf", cluster.GetGkeServerConfig) // todo think about it and move
		v1.GET("/token", auth.GenerateToken)
		v1.GET("/orgs", api.GetOrganizations)
		v1.GET("/orgs/:orgid", api.GetOrganizations)
		v1.POST("/orgs", api.CreateOrganization)
	}

	router.GET("/api", api.MetaHandler(router, "/api"))

	notify.SlackNotify("API is already running")
	var listenPort string
	port := viper.GetInt("pipeline.listenport")
	if port != 0 {
		listenPort = fmt.Sprintf(":%d", port)
		logger.Info("Pipeline API listening on port ", listenPort)
	}
	router.Run(listenPort)
}
