package main

import (
	"fmt"
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
	"os"
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

	// Initialise config
	config.Init()
	// Ensure DB connection
	db := model.GetDB()
	// Initialise auth
	//auth.Init()

	// Creating tables if not exists
	logger.Info("Create table(s):",
		model.ClusterModel.TableName(model.ClusterModel{}),
		model.AmazonClusterModel.TableName(model.AmazonClusterModel{}),
		model.AzureClusterModel.TableName(model.AzureClusterModel{}),
		model.GoogleClusterModel.TableName(model.GoogleClusterModel{}))

	// Create tables
	db.AutoMigrate(
		&model.ClusterModel{},
		&model.AmazonClusterModel{},
		&model.AzureClusterModel{},
		&model.GoogleClusterModel{},
		&auth_identity.AuthIdentity{},
		//&auth.User{},
		&defaults.AWSProfile{},
		&defaults.AKSProfile{},
		&defaults.GKEProfile{},
	)

	defaults.SetDefaultValues()

	router := gin.Default()

	router.Use(cors.New(config.GetCORS()))

	if auth.IsEnabled() {
		authHandler := gin.WrapH(auth.Auth.NewServeMux())

		// We have to make the raw net/http handlers a bit Gin-ish
		router.Use(gin.WrapH(sessionManager.SessionManager.Middleware(utils.NopHandler{})))
		router.Use(gin.WrapH(auth.RedirectBack.Middleware(utils.NopHandler{})))

		authGroup := router.Group("/auth/")
		{
			authGroup.GET("/*w", authHandler)
			authGroup.GET("/*w/*w", authHandler)
		}
	}

	v1 := router.Group("/api/v1/")
	{
		if auth.IsEnabled() {
			v1.Use(auth.Auth0Handler)
		}
		v1.POST("/clusters", api.CreateCluster)
		//v1.GET("/status", api.Status)
		v1.GET("/clusters", api.FetchClusters)
		v1.GET("/clusters/:id", api.FetchCluster)
		v1.PUT("/clusters/:id", api.UpdateCluster)
		v1.DELETE("/clusters/:id", api.DeleteCluster)
		v1.HEAD("/clusters/:id", api.GetClusterStatus)
		v1.GET("/clusters/:id/config", api.GetClusterConfig)
		v1.GET("/clusters/:id/endpoints", api.ListEndpoints)
		v1.GET("/clusters/:id/deployments", api.ListDeployments)
		v1.POST("/clusters/:id/deployments", api.CreateDeployment)
		v1.HEAD("/clusters/:id/deployments", api.GetTillerStatus)
		v1.DELETE("/clusters/:id/deployments/:name", api.DeleteDeployment)
		v1.PUT("/clusters/:id/deployments/:name", api.UpgradeDeployment)
		v1.HEAD("/clusters/:id/deployments/:name", api.HelmDeploymentStatus)
		v1.POST("/clusters/:id/helminit", api.InitHelmOnCluster)
		v1.GET("/cluster/profiles/:type", api.GetClusterProfiles)
		v1.POST("/cluster/profiles", api.AddClusterProfile)
		v1.PUT("/cluster/profiles/:type", api.UpdateClusterProfile)
		v1.DELETE("/cluster/profiles/:type/:name", api.DeleteClusterProfile)
		v1.GET("/token", auth.GenerateToken)
	}
	notify.SlackNotify("API is already running")
	router.Run(":9090")
}
