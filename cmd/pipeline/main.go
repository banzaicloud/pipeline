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
	"encoding/base32"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path"
	"regexp"

	"emperror.dev/emperror"
	"emperror.dev/errors"
	"github.com/ThreeDotsLabs/watermill/components/cqrs"
	"github.com/ThreeDotsLabs/watermill/message"
	watermillMiddleware "github.com/ThreeDotsLabs/watermill/message/router/middleware"
	evbus "github.com/asaskevich/EventBus"
	bauth "github.com/banzaicloud/bank-vaults/pkg/sdk/auth"
	ginprometheus "github.com/banzaicloud/go-gin-prometheus"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/go-kit/kit/endpoint"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/gorilla/mux"
	"github.com/jinzhu/gorm"
	"github.com/prometheus/client_golang/prometheus"
	auth2 "github.com/qor/auth"
	appkitendpoint "github.com/sagikazarmark/appkit/endpoint"
	appkiterrors "github.com/sagikazarmark/appkit/errors"
	"github.com/sagikazarmark/kitx/correlation"
	kitxendpoint "github.com/sagikazarmark/kitx/endpoint"
	kitxtransport "github.com/sagikazarmark/kitx/transport"
	kitxhttp "github.com/sagikazarmark/kitx/transport/http"
	"github.com/sagikazarmark/ocmux"
	"github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	zaplog "logur.dev/integration/zap"
	"logur.dev/logur"

	cloudinfoapi "github.com/banzaicloud/pipeline/.gen/cloudinfo"
	anchore2 "github.com/banzaicloud/pipeline/internal/anchore"
	"github.com/banzaicloud/pipeline/internal/app/frontend"
	"github.com/banzaicloud/pipeline/internal/app/pipeline/api/middleware/audit"
	"github.com/banzaicloud/pipeline/internal/app/pipeline/auth/token"
	"github.com/banzaicloud/pipeline/internal/app/pipeline/auth/token/tokenadapter"
	"github.com/banzaicloud/pipeline/internal/app/pipeline/auth/token/tokendriver"
	"github.com/banzaicloud/pipeline/internal/app/pipeline/cap/capdriver"
	googleproject "github.com/banzaicloud/pipeline/internal/app/pipeline/cloud/google/project"
	googleprojectdriver "github.com/banzaicloud/pipeline/internal/app/pipeline/cloud/google/project/projectdriver"
	"github.com/banzaicloud/pipeline/internal/app/pipeline/secrettype"
	"github.com/banzaicloud/pipeline/internal/app/pipeline/secrettype/secrettypedriver"
	arkClusterManager "github.com/banzaicloud/pipeline/internal/ark/clustermanager"
	arkEvents "github.com/banzaicloud/pipeline/internal/ark/events"
	arkSync "github.com/banzaicloud/pipeline/internal/ark/sync"
	intCluster "github.com/banzaicloud/pipeline/internal/cluster"
	intClusterAuth "github.com/banzaicloud/pipeline/internal/cluster/auth"
	"github.com/banzaicloud/pipeline/internal/cluster/clusteradapter"
	"github.com/banzaicloud/pipeline/internal/cluster/clusterdriver"
	"github.com/banzaicloud/pipeline/internal/cluster/clustersecret"
	"github.com/banzaicloud/pipeline/internal/cluster/clustersecret/clustersecretadapter"
	"github.com/banzaicloud/pipeline/internal/cluster/endpoints"
	prometheusMetrics "github.com/banzaicloud/pipeline/internal/cluster/metrics/adapters/prometheus"
	"github.com/banzaicloud/pipeline/internal/clustergroup"
	cgroupAdapter "github.com/banzaicloud/pipeline/internal/clustergroup/adapter"
	"github.com/banzaicloud/pipeline/internal/clustergroup/deployment"
	"github.com/banzaicloud/pipeline/internal/common/commonadapter"
	"github.com/banzaicloud/pipeline/internal/dashboard"
	"github.com/banzaicloud/pipeline/internal/federation"
	"github.com/banzaicloud/pipeline/internal/global"
	"github.com/banzaicloud/pipeline/internal/global/globalcluster"
	"github.com/banzaicloud/pipeline/internal/global/nplabels"
	"github.com/banzaicloud/pipeline/internal/helm"
	"github.com/banzaicloud/pipeline/internal/helm/helmadapter"
	"github.com/banzaicloud/pipeline/internal/integratedservices"
	"github.com/banzaicloud/pipeline/internal/integratedservices/integratedserviceadapter"
	"github.com/banzaicloud/pipeline/internal/integratedservices/integratedservicedriver"
	"github.com/banzaicloud/pipeline/internal/integratedservices/services"
	integratedServiceDNS "github.com/banzaicloud/pipeline/internal/integratedservices/services/dns"
	"github.com/banzaicloud/pipeline/internal/integratedservices/services/dns/dnsadapter"
	"github.com/banzaicloud/pipeline/internal/integratedservices/services/expiry"
	integratedServiceLogging "github.com/banzaicloud/pipeline/internal/integratedservices/services/logging"
	featureMonitoring "github.com/banzaicloud/pipeline/internal/integratedservices/services/monitoring"
	"github.com/banzaicloud/pipeline/internal/integratedservices/services/securityscan"
	"github.com/banzaicloud/pipeline/internal/integratedservices/services/securityscan/securityscanadapter"
	integratedServiceVault "github.com/banzaicloud/pipeline/internal/integratedservices/services/vault"
	cgFeatureIstio "github.com/banzaicloud/pipeline/internal/istio/istiofeature"
	"github.com/banzaicloud/pipeline/internal/kubernetes"
	"github.com/banzaicloud/pipeline/internal/monitor"
	"github.com/banzaicloud/pipeline/internal/platform/appkit"
	apphttp "github.com/banzaicloud/pipeline/internal/platform/appkit/transport/http"
	"github.com/banzaicloud/pipeline/internal/platform/buildinfo"
	"github.com/banzaicloud/pipeline/internal/platform/cadence"
	"github.com/banzaicloud/pipeline/internal/platform/database"
	"github.com/banzaicloud/pipeline/internal/platform/errorhandler"
	ginternal "github.com/banzaicloud/pipeline/internal/platform/gin"
	"github.com/banzaicloud/pipeline/internal/platform/gin/correlationid"
	"github.com/banzaicloud/pipeline/internal/platform/gin/ginauth"
	ginlog "github.com/banzaicloud/pipeline/internal/platform/gin/log"
	ginutils "github.com/banzaicloud/pipeline/internal/platform/gin/utils"
	"github.com/banzaicloud/pipeline/internal/platform/log"
	"github.com/banzaicloud/pipeline/internal/platform/watermill"
	eksDriver "github.com/banzaicloud/pipeline/internal/providers/amazon/eks/driver"
	azurePKEAdapter "github.com/banzaicloud/pipeline/internal/providers/azure/pke/adapter"
	azurePKEDriver "github.com/banzaicloud/pipeline/internal/providers/azure/pke/driver"
	"github.com/banzaicloud/pipeline/internal/providers/google"
	"github.com/banzaicloud/pipeline/internal/providers/google/googleadapter"
	pkgAuth "github.com/banzaicloud/pipeline/pkg/auth"
	"github.com/banzaicloud/pipeline/pkg/cloudinfo"
	"github.com/banzaicloud/pipeline/pkg/ctxutil"
	kubernetes2 "github.com/banzaicloud/pipeline/pkg/kubernetes"
	"github.com/banzaicloud/pipeline/pkg/problems"
	"github.com/banzaicloud/pipeline/pkg/providers"
	"github.com/banzaicloud/pipeline/src/api"
	"github.com/banzaicloud/pipeline/src/api/ark/backups"
	"github.com/banzaicloud/pipeline/src/api/ark/backupservice"
	"github.com/banzaicloud/pipeline/src/api/ark/buckets"
	"github.com/banzaicloud/pipeline/src/api/ark/restores"
	"github.com/banzaicloud/pipeline/src/api/ark/schedules"
	"github.com/banzaicloud/pipeline/src/api/cluster/namespace"
	"github.com/banzaicloud/pipeline/src/api/cluster/pke"
	cgroupAPI "github.com/banzaicloud/pipeline/src/api/clustergroup"
	"github.com/banzaicloud/pipeline/src/api/common"
	"github.com/banzaicloud/pipeline/src/auth"
	"github.com/banzaicloud/pipeline/src/auth/authadapter"
	"github.com/banzaicloud/pipeline/src/auth/authdriver"
	"github.com/banzaicloud/pipeline/src/auth/authgen"
	"github.com/banzaicloud/pipeline/src/cluster"
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
	logger = log.WithFields(logger, map[string]interface{}{"application": appName})

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

	// Connect to database
	db, err := database.Connect(config.Database.Config)
	emperror.Panic(errors.WithMessage(err, "failed to initialize db"))
	global.SetDB(db)

	// TODO: make this optional when CICD is disabled
	cicdDB, err := database.Connect(config.CICD.Database)
	emperror.Panic(errors.WithMessage(err, "failed to initialize CICD db"))

	commonLogger := commonadapter.NewContextAwareLogger(logger, appkit.ContextExtractor)
	commonErrorHandler := emperror.WithFilter(
		emperror.WithContextExtractor(errorHandler, appkit.ContextExtractor),
		appkiterrors.IsClientError, // filter out client errors
	)

	publisher, subscriber := watermill.NewPubSub(logger)
	defer publisher.Close()
	defer subscriber.Close()

	publisher, _ = message.MessageTransformPublisherDecorator(func(msg *message.Message) {
		if cid, ok := correlation.FromContext(msg.Context()); ok {
			watermillMiddleware.SetCorrelationID(cid, msg)
		}
	})(publisher)

	subscriber, _ = message.MessageTransformSubscriberDecorator(func(msg *message.Message) {
		if cid := watermillMiddleware.MessageCorrelationID(msg); cid != "" {
			msg.SetContext(correlation.ToContext(msg.Context(), cid))
		}
	})(subscriber)

	// Used internally to make sure every event/command bus uses the same one
	eventMarshaler := cqrs.JSONMarshaler{GenerateName: cqrs.StructName}

	secretStore := commonadapter.NewSecretStore(secret.Store, commonadapter.OrgIDContextExtractorFunc(auth.GetCurrentOrganizationID))

	organizationStore := authadapter.NewGormOrganizationStore(db)

	const organizationTopic = "organization"
	var organizationSyncer auth.OIDCOrganizationSyncer
	{
		eventBus, _ := cqrs.NewEventBus(
			publisher,
			func(eventName string) string { return organizationTopic },
			eventMarshaler,
		)
		eventDispatcher := authgen.NewOrganizationEventDispatcher(eventBus)

		roleBinder, err := auth.NewRoleBinder(config.Auth.Role.Default, config.Auth.Role.Binding)
		emperror.Panic(err)

		organizationSyncer = auth.NewOIDCOrganizationSyncer(
			auth.NewOrganizationSyncer(
				organizationStore,
				eventDispatcher,
				commonLogger.WithFields(map[string]interface{}{"component": "auth"}),
			),
			roleBinder,
		)
	}

	// Initialize auth
	tokenStore := bauth.NewVaultTokenStore("pipeline")
	tokenGenerator := pkgAuth.NewJWTTokenGenerator(
		config.Auth.Token.Issuer,
		config.Auth.Token.Audience,
		base32.StdEncoding.EncodeToString([]byte(config.Auth.Token.SigningKey)),
	)
	tokenManager := pkgAuth.NewTokenManager(tokenGenerator, tokenStore)
	auth.Init(db, cicdDB, config.Auth, tokenStore, tokenManager, organizationSyncer)

	if config.Database.AutoMigrate {
		logger.Info("running automatic schema migrations")

		err = Migrate(db, logrusLogger, commonLogger)
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
	clusters := clusteradapter.NewClusters(db)
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

	externalBaseURL := global.Config.Pipeline.External.URL
	if externalBaseURL == "" {
		externalBaseURL = "http://" + config.Pipeline.Addr
		logger.Warn("no pipeline.external_url set, falling back to bind address", map[string]interface{}{
			"fallback": externalBaseURL,
		})
	}

	externalURLInsecure := global.Config.Pipeline.External.Insecure

	workflowClient, err := cadence.NewClient(config.Cadence, zaplog.New(logur.WithFields(logger, map[string]interface{}{"component": "cadence-client"})))
	if err != nil {
		errorHandler.Handle(errors.WrapIf(err, "Failed to configure Cadence client"))
	}

	clusterManager := cluster.NewManager(clusters, secretValidator, clusterEvents, statusChangeDurationMetric, clusterTotalMetric, workflowClient, logrusLogger, errorHandler, clusteradapter.NewStore(db, clusters))
	commonClusterGetter := common.NewClusterGetter(clusterManager, logrusLogger, errorHandler)

	if config.SpotMetrics.Enabled {
		go monitor.NewSpotMetricsExporter(
			context.Background(),
			clusterManager,
			logrusLogger.WithField("subsystem", "spot-metrics-exporter"),
		).Run(config.SpotMetrics.CollectionInterval)
	}

	cloudinfoClient := cloudinfo.NewClient(cloudinfoapi.NewAPIClient(&cloudinfoapi.Configuration{
		BasePath:      config.Cloudinfo.Endpoint,
		DefaultHeader: make(map[string]string),
		UserAgent:     fmt.Sprintf("Pipeline/%s", version),
	}))

	azurePKEClusterStore := azurePKEAdapter.NewClusterStore(db, commonLogger)
	clusterCreators := api.ClusterCreators{
		PKEOnAzure: azurePKEDriver.MakeClusterCreator(
			azurePKEDriver.ClusterCreatorConfig{
				OIDCIssuerURL:               config.Auth.OIDC.Issuer,
				PipelineExternalURL:         externalBaseURL,
				PipelineExternalURLInsecure: externalURLInsecure,
			},
			logrusLogger,
			authdriver.NewOrganizationGetter(db),
			secret.Store,
			azurePKEClusterStore,
			workflowClient,
		),
		EKSAmazon: eksDriver.NewEksClusterCreator(
			logrusLogger,
			workflowClient,
			cloudinfoClient,
			clusters,
			secretValidator,
			statusChangeDurationMetric,
			clusterTotalMetric,
		),
	}

	cgroupAdapter := cgroupAdapter.NewClusterGetter(clusterManager)
	clusterGroupManager := clustergroup.NewManager(cgroupAdapter, clustergroup.NewClusterGroupRepository(db, logrusLogger), logrusLogger, errorHandler)
	federationHandler := federation.NewFederationHandler(cgroupAdapter, global.Config.Cluster.Namespace, logrusLogger, errorHandler)
	deploymentManager := deployment.NewCGDeploymentManager(db, cgroupAdapter, logrusLogger, errorHandler)
	serviceMeshFeatureHandler := cgFeatureIstio.NewServiceMeshFeatureHandler(cgroupAdapter, logrusLogger, errorHandler)
	clusterGroupManager.RegisterFeatureHandler(federation.FeatureName, federationHandler)
	clusterGroupManager.RegisterFeatureHandler(deployment.FeatureName, deploymentManager)
	clusterGroupManager.RegisterFeatureHandler(cgFeatureIstio.FeatureName, serviceMeshFeatureHandler)
	clusterUpdaters := api.ClusterUpdaters{
		PKEOnAzure: azurePKEDriver.MakeClusterUpdater(
			logrusLogger,
			externalBaseURL,
			externalURLInsecure,
			secret.Store,
			azurePKEClusterStore,
			workflowClient,
		),
		EKSAmazon: eksDriver.NewEksClusterUpdater(
			logrusLogger,
			workflowClient,
		),
	}

	configFactory := kubernetes.NewConfigFactory(secretStore)
	clientFactory := kubernetes.NewClientFactory(configFactory)
	dynamicClientFactory := kubernetes.NewDynamicClientFactory(configFactory)

	clusterAPI := api.NewClusterAPI(
		clusterManager,
		commonClusterGetter,
		workflowClient,
		logrusLogger,
		errorHandler,
		externalBaseURL,
		externalURLInsecure,
		clusterCreators,
		clusterUpdaters,
		dynamicClientFactory,
	)

	// Initialise Gin router
	engine := gin.New()

	router := mux.NewRouter()
	router.Use(ocmux.Middleware())

	// These two paths can contain sensitive information, so it is advised not to log them out.
	skipPaths := config.Audit.SkipPaths
	engine.Use(correlationid.Middleware())
	engine.Use(ginlog.Middleware(logrusLogger, skipPaths...))

	// Add prometheus metric endpoint
	if config.Telemetry.Enabled {
		p := ginprometheus.NewPrometheus("pipeline", []string{})
		p.SetListenAddress(config.Telemetry.Addr)
		p.Use(engine, "/metrics")
	}

	engine.Use(gin.Recovery())
	drainModeMetric := prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "pipeline",
		Name:      "drain_mode",
		Help:      "read only mode is on/off",
	})
	prometheus.MustRegister(drainModeMetric)
	engine.Use(ginternal.NewDrainModeMiddleware(config.Pipeline.BasePath, drainModeMetric, errorHandler).Middleware)

	corsConfig := cors.DefaultConfig()
	corsConfig.AllowHeaders = append(corsConfig.AllowHeaders, "Authorization", "secretId", "Banzai-Cloud-Pipeline-UUID")
	corsConfig.AllowCredentials = true
	corsConfig.AllowMethods = []string{"PUT", "DELETE", "GET", "POST", "OPTIONS", "PATCH"}
	corsConfig.ExposeHeaders = []string{"Content-Length"}

	corsConfig.AllowAllOrigins = config.CORS.AllowAllOrigins
	if !corsConfig.AllowAllOrigins {
		allowOriginsRegexp := config.CORS.AllowOriginsRegexp
		if allowOriginsRegexp != "" {
			originsRegexp, err := regexp.Compile(fmt.Sprintf("^(%s)$", allowOriginsRegexp))
			if err == nil {
				corsConfig.AllowOriginFunc = func(origin string) bool {
					return originsRegexp.Match([]byte(origin))
				}
			}
		} else if allowOrigins := config.CORS.AllowOrigins; len(allowOrigins) > 0 {
			corsConfig.AllowOrigins = allowOrigins
		}
	}

	engine.Use(cors.New(corsConfig))

	if config.Audit.Enabled {
		logger.Info("Audit enabled, installing Gin audit middleware")
		engine.Use(audit.LogWriter(skipPaths, config.Audit.Headers, db, logrusLogger))
	}
	engine.Use(func(c *gin.Context) { // TODO: move to middleware
		c.Request = c.Request.WithContext(ctxutil.WithParams(c.Request.Context(), ginutils.ParamsToMap(c.Params)))
	})

	router.Path("/").Methods(http.MethodGet).Handler(http.RedirectHandler(config.Auth.RedirectURL.Login, http.StatusTemporaryRedirect))
	engine.GET("/", gin.WrapH(router))

	basePath := config.Pipeline.BasePath

	base := engine.Group(basePath)
	router = router.PathPrefix(basePath).Subrouter()

	// Frontend service
	{
		err := frontend.RegisterApp(
			router.PathPrefix("/frontend").Subrouter(),
			config.Frontend,
			db,
			buildInfo,
			auth.UserExtractor{},
			commonLogger,
			commonErrorHandler,
		)
		emperror.Panic(err)

		// TODO: refactor authentication middleware
		base.Any("frontend/notifications", gin.WrapH(router))

		// TODO: return 422 unprocessable entity instead of 404
		if config.Frontend.Issue.Enabled {
			base.Any("frontend/issues", auth.Handler, gin.WrapH(router))
		}
	}

	base.GET("version", gin.WrapH(buildinfo.Handler(buildInfo)))

	auth.Install(engine)
	auth.StartTokenStoreGC(tokenStore)

	enforcer := auth.NewRbacEnforcer(organizationStore, commonLogger)
	authorizationMiddleware := ginauth.NewMiddleware(enforcer, basePath, errorHandler)

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

	scmTokenStore := auth.NewSCMTokenStore(tokenStore, global.Config.CICD.Enabled)

	organizationAPI := api.NewOrganizationAPI(organizationSyncer, auth.NewRefreshTokenStore(tokenStore))
	userAPI := api.NewUserAPI(db, scmTokenStore, logrusLogger, errorHandler)
	networkAPI := api.NewNetworkAPI(logrusLogger)

	var spotguideAPI *api.SpotguideAPI

	if global.Config.CICD.Enabled {
		spotguidePlatformData := spotguide.PlatformData{
			AutoDNSEnabled: global.Config.Cluster.DNS.BaseDomain != "",
		}

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

		scmFactory, err := scm.NewSCMFactory(scmProvider, scmToken, scmTokenStore)
		emperror.Panic(errors.WrapIf(err, "failed to create SCMFactory"))

		sharedSpotguideOrg, err := spotguide.EnsureSharedSpotguideOrganization(
			db,
			scmProvider,
			global.Config.Spotguide.SharedLibraryGitHubOrganization,
		)
		if err != nil {
			errorHandler.Handle(errors.WrapIf(err, "failed to create shared Spotguide organization"))
		}

		spotguideManager := spotguide.NewSpotguideManager(
			db,
			version,
			scmFactory,
			sharedSpotguideOrg,
			spotguidePlatformData,
		)

		// periodically sync shared spotguides
		if err := spotguide.ScheduleScrapingSharedSpotguides(workflowClient); err != nil {
			errorHandler.Handle(errors.WrapIf(err, "failed to schedule syncing shared spotguides"))
		}

		spotguideAPI = api.NewSpotguideAPI(logrusLogger, errorHandler, spotguideManager)
	}

	v1 := base.Group("api/v1")
	apiRouter := router.PathPrefix("/api/v1").Subrouter()
	{
		apiRouter.NotFoundHandler = problems.StatusProblemHandler(problems.NewStatusProblem(http.StatusNotFound))
		apiRouter.MethodNotAllowedHandler = problems.StatusProblemHandler(problems.NewStatusProblem(http.StatusMethodNotAllowed))

		v1.Use(auth.Handler)
		capdriver.RegisterHTTPHandler(mapCapabilities(config), commonErrorHandler, v1)
		v1.GET("/securityscan", api.SecurityScanEnabled)
		v1.GET("/me", userAPI.GetCurrentUser)
		v1.PATCH("/me", userAPI.UpdateCurrentUser)

		endpointMiddleware := []endpoint.Middleware{
			correlation.Middleware(),
			appkitendpoint.LoggingMiddleware(logger),
			appkitendpoint.ClientErrorMiddleware,
		}

		httpServerOptions := []kithttp.ServerOption{
			kithttp.ServerErrorHandler(kitxtransport.NewErrorHandler(commonErrorHandler)),
			kithttp.ServerErrorEncoder(kitxhttp.NewJSONProblemErrorEncoder(apphttp.NewDefaultProblemConverter())),
			kithttp.ServerBefore(correlation.HTTPToContext()),
		}

		orgs := v1.Group("/orgs")
		orgRouter := apiRouter.PathPrefix("/orgs/{orgId}").Subrouter()
		{
			orgs.Use(api.OrganizationMiddleware)
			orgs.Use(authorizationMiddleware)

			if global.Config.CICD.Enabled {
				spotguides := orgs.Group("/:orgid/spotguides")
				spotguideAPI.Install(spotguides)
			}

			orgs.POST("/:orgid/clusters", clusterAPI.CreateCluster)
			orgs.GET("/:orgid/clusters", clusterAPI.GetClusters)

			// cluster API
			cRouter := orgs.Group("/:orgid/clusters/:id")
			clusterRouter := orgRouter.PathPrefix("/clusters/{clusterId}").Subrouter()
			{
				logger := commonadapter.NewLogger(logger) // TODO: make this a context aware logger

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
				cRouter.HEAD("", clusterAPI.ClusterCheck)
				cRouter.GET("/config", api.GetClusterConfig)
				cRouter.GET("/apiendpoint", api.GetApiEndpoint)
				cRouter.GET("/nodes", api.GetClusterNodes)
				cRouter.GET("/endpoints", api.MakeEndpointLister(logger).ListEndpoints)
				cRouter.GET("/secrets", api.ListClusterSecrets)
				cRouter.GET("/deployments", api.ListDeployments)
				cRouter.POST("/deployments", api.CreateDeployment)
				cRouter.GET("/deployments/:name", api.GetDeployment)
				cRouter.GET("/deployments/:name/resources", api.GetDeploymentResources)
				cRouter.HEAD("/deployments", api.GetTillerStatus)
				cRouter.DELETE("/deployments/:name", api.DeleteDeployment)
				cRouter.PUT("/deployments/:name", api.UpgradeDeployment)
				cRouter.HEAD("/deployments/:name", api.HelmDeploymentStatus)

				cRouter.GET("/images", api.ListImages)
				cRouter.GET("/images/:imageDigest/deployments", api.GetImageDeployments)
				cRouter.GET("/deployments/:name/images", api.GetDeploymentImages)

				{
					service := intCluster.NewService(
						clusteradapter.NewStore(db, clusters),
						clusteradapter.NewCadenceClusterManager(workflowClient),
					)
					endpoints := clusterdriver.MakeClusterEndpoints(
						service,
						kitxendpoint.Combine(endpointMiddleware...),
					)
					clusterdriver.RegisterClusterHTTPHandlers(
						endpoints,
						clusterRouter,
						kitxhttp.ServerOptions(httpServerOptions),
					)
					cRouter.DELETE("", gin.WrapH(router))
				}

			}

			clusterSecretStore := clustersecret.NewStore(
				clustersecretadapter.NewClusterManagerAdapter(clusterManager),
				clustersecretadapter.NewSecretStore(secret.Store),
			)

			// Cluster IntegratedService API
			var featureService integratedservices.Service
			{
				featureRepository := integratedserviceadapter.NewGormIntegratedServiceRepository(db, commonLogger)
				clusterGetter := integratedserviceadapter.MakeClusterGetter(clusterManager)
				clusterPropertyGetter := dnsadapter.NewClusterPropertyGetter(clusterManager)
				endpointManager := endpoints.NewEndpointManager(commonLogger)
				integratedServiceManagers := []integratedservices.IntegratedServiceManager{
					securityscan.MakeIntegratedServiceManager(commonLogger),
				}

				if config.Cluster.DNS.Enabled {
					integratedServiceManagers = append(integratedServiceManagers, integratedServiceDNS.NewIntegratedServicesManager(clusterPropertyGetter, clusterPropertyGetter, config.Cluster.DNS.Config))
				}

				if config.Cluster.Vault.Enabled {
					integratedServiceManagers = append(integratedServiceManagers, integratedServiceVault.MakeIntegratedServiceManager(clusterGetter, secretStore, config.Cluster.Vault.Config, commonLogger))
				}

				if config.Cluster.Monitoring.Enabled {
					helmService := helm.NewHelmService(helmadapter.NewClusterService(clusterManager), commonLogger)
					integratedServiceManagers = append(integratedServiceManagers, featureMonitoring.MakeIntegratedServiceManager(
						clusterGetter,
						secretStore,
						endpointManager,
						helmService,
						config.Cluster.Monitoring.Config,
						commonLogger,
					))
				}

				if config.Cluster.Logging.Enabled {
					integratedServiceManagers = append(integratedServiceManagers, integratedServiceLogging.MakeIntegratedServiceManager(
						clusterGetter,
						secretStore,
						endpointManager,
						config.Cluster.Logging.Config,
						commonLogger,
					))
				}

				if config.Cluster.SecurityScan.Enabled {
					customAnchoreConfigProvider := securityscan.NewCustomAnchoreConfigProvider(
						featureRepository,
						secretStore,
						commonLogger,
					)

					configProvider := anchore2.ConfigProviderChain{customAnchoreConfigProvider}

					if config.Cluster.SecurityScan.Anchore.Enabled {
						configProvider = append(configProvider, securityscan.NewClusterAnchoreConfigProvider(
							config.Cluster.SecurityScan.Anchore.Endpoint,
							securityscanadapter.NewUserNameGenerator(securityscanadapter.NewClusterService(clusterManager)),
							securityscanadapter.NewUserSecretStore(secretStore),
						))
					}

					securityApiHandler := api.NewSecurityApiHandlers(commonClusterGetter, commonErrorHandler, commonLogger)

					anchoreProxy := api.NewAnchoreProxy(basePath, configProvider, commonErrorHandler, commonLogger)
					proxyHandler := anchoreProxy.Proxy()

					// forthcoming endpoint for all requests proxied to Anchore
					cRouter.Any("/anchore/*proxyPath", proxyHandler)

					// these are cluster resources
					cRouter.GET("/scanlog", securityApiHandler.ListScanLogs)
					cRouter.GET("/scanlog/:releaseName", securityApiHandler.GetScanLogs)

					cRouter.GET("/whitelists", securityApiHandler.GetWhiteLists)
					cRouter.POST("/whitelists", securityApiHandler.CreateWhiteList)
					cRouter.DELETE("/whitelists/:name", securityApiHandler.DeleteWhiteList)
				}

				if config.Cluster.Expiry.Enabled {
					integratedServiceManagers = append(integratedServiceManagers,
						expiry.NewExpiryServiceManager(services.BindIntegratedServiceSpec))
				}

				integratedServiceManagerRegistry := integratedservices.MakeIntegratedServiceManagerRegistry(integratedServiceManagers)
				integratedServiceOperationDispatcher := integratedserviceadapter.MakeCadenceIntegratedServiceOperationDispatcher(workflowClient, commonLogger)
				featureService = integratedservices.MakeIntegratedServiceService(integratedServiceOperationDispatcher, integratedServiceManagerRegistry, featureRepository, commonLogger)
				endpoints := integratedservicedriver.MakeEndpoints(
					featureService,
					kitxendpoint.Combine(endpointMiddleware...),
				)

				{
					integratedservicedriver.RegisterHTTPHandlers(
						endpoints,
						clusterRouter.PathPrefix("/services").Subrouter(),
						kitxhttp.ServerOptions(httpServerOptions),
					)

					cRouter.Any("/services", gin.WrapH(router))
					cRouter.Any("/services/:serviceName", gin.WrapH(router))
				}

				// set up legacy endpoint
				{
					integratedservicedriver.RegisterHTTPHandlers(
						endpoints,
						clusterRouter.PathPrefix("/features").Subrouter(),
						kitxhttp.ServerOptions(httpServerOptions),
					)

					cRouter.Any("/features", gin.WrapH(router))
					cRouter.Any("/features/:featureName", gin.WrapH(router))
				}
			}

			hpaApi := api.NewHPAAPI(featureService, clientFactory, configFactory, commonClusterGetter, errorHandler)
			cRouter.GET("/hpa", hpaApi.GetHpaResource)
			cRouter.PUT("/hpa", hpaApi.PutHpaResource)
			cRouter.DELETE("/hpa", hpaApi.DeleteHpaResource)

			// ClusterGroupAPI
			cgroupsAPI := cgroupAPI.NewAPI(clusterGroupManager, deploymentManager, logrusLogger, errorHandler)
			cgroupsAPI.AddRoutes(orgs.Group("/:orgid/clustergroups"))

			{
				clusterStore := clusteradapter.NewStore(db, clusters)

				labelValidator := kubernetes2.LabelValidator{
					ForbiddenDomains: append([]string{config.Cluster.Labels.Domain}, config.Cluster.Labels.ForbiddenDomains...),
				}

				nplabels.SetNodePoolLabelValidator(labelValidator)

				nplsApi := api.NewNodepoolManagerAPI(
					commonClusterGetter,
					dynamicClientFactory,
					labelValidator,
					logrusLogger,
					errorHandler,
				)
				cRouter.GET("/nodepool-labels", nplsApi.GetNodepoolLabelSets)

				labelSource := intCluster.NodePoolLabelSources{
					intCluster.NewCommonNodePoolLabelSource(),
					clusteradapter.NewCloudinfoNodePoolLabelSource(cloudinfoClient),
				}

				// Used by legacy node pool label code
				globalcluster.SetNodePoolLabelSource(intCluster.NodePoolLabelSources{
					intCluster.NewFilterValidNodePoolLabelSource(labelValidator),
					labelSource,
				})

				service := intCluster.NewNodePoolService(
					clusterStore,
					clusteradapter.NewNodePoolStore(db, clusterStore),
					intCluster.NodePoolValidators{
						intCluster.NewCommonNodePoolValidator(labelValidator),
						clusteradapter.NewDistributionNodePoolValidator(db),
					},
					intCluster.NodePoolProcessors{
						intCluster.NewCommonNodePoolProcessor(labelSource),
						clusteradapter.NewDistributionNodePoolProcessor(db),
					},
					clusteradapter.NewNodePoolManager(
						workflowClient,
						func(ctx context.Context) uint {
							if currentUser := ctx.Value(auth2.CurrentUser); currentUser != nil {
								return currentUser.(*auth.User).ID
							}

							return 0
						},
					),
				)
				endpoints := clusterdriver.TraceNodePoolEndpoints(clusterdriver.MakeNodePoolEndpoints(
					service,
					kitxendpoint.Combine(endpointMiddleware...),
				))

				clusterdriver.RegisterNodePoolHTTPHandlers(
					endpoints,
					clusterRouter.PathPrefix("/nodepools").Subrouter(),
					kitxhttp.ServerOptions(httpServerOptions),
				)

				cRouter.Any("/nodepools", gin.WrapH(router))
				cRouter.Any("/nodepools/:nodePoolName", gin.WrapH(router))
			}

			namespaceAPI := namespace.NewAPI(commonClusterGetter, clientFactory, errorHandler)
			namespaceAPI.RegisterRoutes(cRouter.Group("/namespaces"))

			pkeGroup := cRouter.Group("/pke")

			leaderRepository, err := pke.NewVaultLeaderRepository()
			emperror.Panic(errors.WrapIf(err, "failed to create Vault leader repository"))

			pkeAPI := pke.NewAPI(
				commonClusterGetter,
				errorHandler,
				auth.NewClusterTokenGenerator(tokenManager, tokenStore),
				externalBaseURL,
				workflowClient,
				leaderRepository,
			)
			pkeAPI.RegisterRoutes(pkeGroup)

			clusterAuthService, err := intClusterAuth.NewDexClusterAuthService(clusterSecretStore)
			emperror.Panic(errors.WrapIf(err, "failed to create DexClusterAuthService"))

			pipelineExternalURL, err := url.Parse(externalBaseURL)
			emperror.Panic(errors.WrapIf(err, "failed to parse pipeline externalBaseURL"))

			pipelineExternalURL.Path = "/auth/dex/cluster/callback"

			clusterAuthAPI, err := api.NewClusterAuthAPI(
				commonClusterGetter,
				clusterAuthService,
				config.Auth.Token.SigningKey,
				config.Auth.OIDC.Issuer,
				config.Auth.OIDC.Insecure,
				pipelineExternalURL.String(),
			)
			emperror.Panic(errors.WrapIf(err, "failed to create ClusterAuthAPI"))

			clusterAuthAPI.RegisterRoutes(cRouter, engine)

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

			{
				secretStore := googleadapter.NewSecretStore(secretStore)
				clientFactory := google.NewClientFactory(secretStore)

				service := googleproject.NewService(clientFactory)
				endpoints := googleprojectdriver.TraceEndpoints(googleprojectdriver.MakeEndpoints(
					service,
					kitxendpoint.Combine(endpointMiddleware...),
				))

				googleprojectdriver.RegisterHTTPHandlers(
					endpoints,
					orgRouter.PathPrefix("/cloud/google/projects").Subrouter(),
					kitxhttp.ServerOptions(httpServerOptions),
				)

				orgs.Any("/:orgid/cloud/google/projects", gin.WrapH(router))
			}

			orgs.GET("/:orgid", organizationAPI.GetOrganizations)
			orgs.DELETE("/:orgid", organizationAPI.DeleteOrganization)
		}
		v1.GET("/orgs", organizationAPI.GetOrganizations)
		v1.PUT("/orgs", organizationAPI.SyncOrganizations)

		{
			service := token.NewService(
				auth.UserExtractor{},
				tokenadapter.NewBankVaultsStore(tokenStore),
				tokenGenerator,
			)
			service = tokendriver.AuthorizationMiddleware(auth.NewAuthorizer(db, organizationStore))(service)

			endpoints := tokendriver.TraceEndpoints(tokendriver.MakeEndpoints(
				service,
				kitxendpoint.Combine(endpointMiddleware...),
			))

			tokendriver.RegisterHTTPHandlers(
				endpoints,
				apiRouter.PathPrefix("/tokens").Subrouter(),
				kitxhttp.ServerOptions(httpServerOptions),
			)

			v1.Any("/tokens", gin.WrapH(router))
			v1.Any("/tokens/*path", gin.WrapH(router))
		}

		{
			service := secrettype.NewTypeService()
			endpoints := secrettypedriver.TraceEndpoints(secrettypedriver.MakeEndpoints(
				service,
				kitxendpoint.Combine(endpointMiddleware...),
			))

			secrettypedriver.RegisterHTTPHandlers(
				endpoints,
				apiRouter.PathPrefix("/secret-types").Subrouter(),
				kitxhttp.ServerOptions(httpServerOptions),
			)

			v1.Any("/secret-types", gin.WrapH(router))
			v1.Any("/secret-types/*path", gin.WrapH(router))
		}

		backups.AddRoutes(orgs.Group("/:orgid/clusters/:id/backups"))
		backupservice.AddRoutes(orgs.Group("/:orgid/clusters/:id/backupservice"))
		restores.AddRoutes(orgs.Group("/:orgid/clusters/:id/restores"))
		schedules.AddRoutes(orgs.Group("/:orgid/clusters/:id/schedules"))
		buckets.AddRoutes(orgs.Group("/:orgid/backupbuckets"))
		backups.AddOrgRoutes(orgs.Group("/:orgid/backups"), clusterManager)
	}

	arkEvents.NewClusterEventHandler(arkEvents.NewClusterEvents(clusterEventBus), db, logrusLogger)
	if global.Config.Cluster.DisasterRecovery.Ark.SyncEnabled {
		go arkSync.RunSyncServices(
			context.Background(),
			db,
			arkClusterManager.New(clusterManager),
			logrusLogger.WithField("subsystem", "ark"),
			errorHandler,
			global.Config.Cluster.DisasterRecovery.Ark.BucketSyncInterval,
			global.Config.Cluster.DisasterRecovery.Ark.RestoreSyncInterval,
			global.Config.Cluster.DisasterRecovery.Ark.BackupSyncInterval,
		)
	}

	base.GET("api", api.MetaHandler(engine, basePath+"/api"))

	internalBindAddr := config.Pipeline.InternalAddr
	logger.Info("Pipeline internal API listening", map[string]interface{}{"address": "http://" + internalBindAddr})

	go createInternalAPIRouter(config, db, basePath, clusterAPI, cloudinfoClient, logger, logrusLogger).Run(internalBindAddr) // nolint: errcheck

	bindAddr := config.Pipeline.Addr
	certFile, keyFile := config.Pipeline.CertFile, config.Pipeline.KeyFile
	if certFile != "" && keyFile != "" {
		logger.Info("Pipeline API listening", map[string]interface{}{"address": "https://" + bindAddr})
		_ = engine.RunTLS(bindAddr, certFile, keyFile)
	} else {
		logger.Info("Pipeline API listening", map[string]interface{}{"address": "http://" + bindAddr})
		_ = engine.Run(bindAddr)
	}
}

func createInternalAPIRouter(
	conf configuration,
	db *gorm.DB,
	basePath string,
	clusterAPI *api.ClusterAPI,
	cloudinfoClient *cloudinfo.Client,
	logger logur.Logger,
	logrusLogger logrus.FieldLogger,
) *gin.Engine {
	// Initialise Gin router for Internal API
	internalRouter := gin.New()
	internalRouter.Use(correlationid.Middleware())
	internalRouter.Use(ginlog.Middleware(logrusLogger, conf.Audit.SkipPaths...))
	internalRouter.Use(gin.Recovery())
	if conf.Audit.Enabled {
		logger.Info("Audit enabled, installing Gin audit middleware to internal router")
		internalRouter.Use(audit.LogWriter(conf.Audit.SkipPaths, conf.Audit.Headers, db, logrusLogger))
	}
	internalGroup := internalRouter.Group(path.Join(basePath, "api", "v1/", "orgs"))
	internalGroup.Use(auth.InternalUserHandler)
	internalGroup.Use(api.OrganizationMiddleware)
	internalGroup.GET("/:orgid/clusters/:id/nodepools", api.NewInternalClusterAPI(cloudinfoClient).GetNodePools)
	internalGroup.PUT("/:orgid/clusters/:id/nodepools", clusterAPI.UpdateNodePools)
	return internalRouter
}
