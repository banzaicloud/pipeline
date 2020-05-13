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
	"crypto/tls"
	"encoding/base32"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"regexp"
	"syscall"
	"time"

	"emperror.dev/emperror"
	"emperror.dev/errors"
	"emperror.dev/errors/match"
	"github.com/ThreeDotsLabs/watermill/components/cqrs"
	"github.com/ThreeDotsLabs/watermill/message"
	watermillMiddleware "github.com/ThreeDotsLabs/watermill/message/router/middleware"
	evbus "github.com/asaskevich/EventBus"
	bauth "github.com/banzaicloud/bank-vaults/pkg/sdk/auth"
	"github.com/banzaicloud/bank-vaults/pkg/sdk/vault"
	ginprometheus "github.com/banzaicloud/go-gin-prometheus"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/tracing/opencensus"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/gorilla/mux"
	"github.com/jinzhu/gorm"
	"github.com/mitchellh/mapstructure"
	"github.com/oklog/run"
	"github.com/prometheus/client_golang/prometheus"
	auth2 "github.com/qor/auth"
	appkitendpoint "github.com/sagikazarmark/appkit/endpoint"
	appkiterrors "github.com/sagikazarmark/appkit/errors"
	appkitrun "github.com/sagikazarmark/appkit/run"
	"github.com/sagikazarmark/kitx/correlation"
	kitxendpoint "github.com/sagikazarmark/kitx/endpoint"
	kitxtransport "github.com/sagikazarmark/kitx/transport"
	kitxhttp "github.com/sagikazarmark/kitx/transport/http"
	"github.com/sagikazarmark/ocmux"
	"github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"go.uber.org/cadence/.gen/go/shared"
	zaplog "logur.dev/integration/zap"
	"logur.dev/logur"

	"github.com/banzaicloud/pipeline/internal/helm/helmadapter"

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
	process "github.com/banzaicloud/pipeline/internal/app/pipeline/process/app"
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
	"github.com/banzaicloud/pipeline/internal/cluster/distribution/eks"
	"github.com/banzaicloud/pipeline/internal/cluster/distribution/eks/eksadapter"
	eksDriver "github.com/banzaicloud/pipeline/internal/cluster/distribution/eks/eksprovider/driver"
	"github.com/banzaicloud/pipeline/internal/cluster/endpoints"
	prometheusMetrics "github.com/banzaicloud/pipeline/internal/cluster/metrics/adapters/prometheus"
	"github.com/banzaicloud/pipeline/internal/clustergroup"
	cgroupAdapter "github.com/banzaicloud/pipeline/internal/clustergroup/adapter"
	"github.com/banzaicloud/pipeline/internal/clustergroup/deployment"
	"github.com/banzaicloud/pipeline/internal/cmd"
	"github.com/banzaicloud/pipeline/internal/common/commonadapter"
	"github.com/banzaicloud/pipeline/internal/dashboard"
	"github.com/banzaicloud/pipeline/internal/federation"
	"github.com/banzaicloud/pipeline/internal/global"
	"github.com/banzaicloud/pipeline/internal/global/globalcluster"
	"github.com/banzaicloud/pipeline/internal/global/nplabels"
	"github.com/banzaicloud/pipeline/internal/helm"
	"github.com/banzaicloud/pipeline/internal/helm/helmdriver"
	"github.com/banzaicloud/pipeline/internal/integratedservices"
	"github.com/banzaicloud/pipeline/internal/integratedservices/integratedserviceadapter"
	"github.com/banzaicloud/pipeline/internal/integratedservices/integratedservicesdriver"
	"github.com/banzaicloud/pipeline/internal/integratedservices/services"
	integratedServiceDNS "github.com/banzaicloud/pipeline/internal/integratedservices/services/dns"
	"github.com/banzaicloud/pipeline/internal/integratedservices/services/dns/dnsadapter"
	"github.com/banzaicloud/pipeline/internal/integratedservices/services/expiry"
	"github.com/banzaicloud/pipeline/internal/integratedservices/services/ingress"
	integratedServiceLogging "github.com/banzaicloud/pipeline/internal/integratedservices/services/logging"
	featureMonitoring "github.com/banzaicloud/pipeline/internal/integratedservices/services/monitoring"
	"github.com/banzaicloud/pipeline/internal/integratedservices/services/securityscan"
	"github.com/banzaicloud/pipeline/internal/integratedservices/services/securityscan/securityscanadapter"
	integratedServiceVault "github.com/banzaicloud/pipeline/internal/integratedservices/services/vault"
	cgFeatureIstio "github.com/banzaicloud/pipeline/internal/istio/istiofeature"
	"github.com/banzaicloud/pipeline/internal/kubernetes"
	"github.com/banzaicloud/pipeline/internal/monitor"
	intPKE "github.com/banzaicloud/pipeline/internal/pke"
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
	azurePKEAdapter "github.com/banzaicloud/pipeline/internal/providers/azure/pke/adapter"
	azurePKEDriver "github.com/banzaicloud/pipeline/internal/providers/azure/pke/driver"
	"github.com/banzaicloud/pipeline/internal/providers/google"
	"github.com/banzaicloud/pipeline/internal/providers/google/googleadapter"
	vspherePKEAdapter "github.com/banzaicloud/pipeline/internal/providers/vsphere/pke/adapter"
	vspherePKEDriver "github.com/banzaicloud/pipeline/internal/providers/vsphere/pke/driver"
	"github.com/banzaicloud/pipeline/internal/secret/pkesecret"
	"github.com/banzaicloud/pipeline/internal/secret/restricted"
	"github.com/banzaicloud/pipeline/internal/secret/secretadapter"
	"github.com/banzaicloud/pipeline/internal/secret/types"
	pkgAuth "github.com/banzaicloud/pipeline/pkg/auth"
	"github.com/banzaicloud/pipeline/pkg/cloudinfo"
	"github.com/banzaicloud/pipeline/pkg/ctxutil"
	"github.com/banzaicloud/pipeline/pkg/hook"
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
	"github.com/banzaicloud/pipeline/src/cluster"
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
	commonErrorHandler := emperror.WithFilter(
		emperror.WithContextExtractor(errorHandler, appkit.ContextExtractor),
		appkiterrors.IsServiceError, // filter out client errors
	)

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

	// Connect to database
	db, err := database.Connect(config.Database.Config)
	emperror.Panic(errors.WithMessage(err, "failed to initialize db"))
	global.SetDB(db)

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

	commonSecretStore := commonadapter.NewSecretStore(secret.Store, commonadapter.OrgIDContextExtractorFunc(auth.GetCurrentOrganizationID))

	organizationStore := authadapter.NewGormOrganizationStore(db)

	const organizationTopic = "organization"
	var organizationSyncer auth.OIDCOrganizationSyncer
	{
		eventBus, _ := cqrs.NewEventBus(
			publisher,
			func(eventName string) string { return organizationTopic },
			eventMarshaler,
		)
		eventDispatcher := auth.NewOrganizationEventDispatcher(eventBus)

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
	serviceAccountService := auth.NewServiceAccountService()
	auth.Init(db, config.Auth, tokenStore, tokenManager, organizationSyncer, serviceAccountService)

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

	externalBaseURL := config.Pipeline.External.URL
	if externalBaseURL == "" {
		externalBaseURL = "http://" + config.Pipeline.Addr
		logger.Warn("no pipeline.external_url set, falling back to bind address", map[string]interface{}{
			"fallback": externalBaseURL,
		})
	}

	externalURLInsecure := config.Pipeline.External.Insecure

	workflowClient, err := cadence.NewClient(config.Cadence, zaplog.New(logur.WithFields(logger, map[string]interface{}{"component": "cadence-client"})))
	if err != nil {
		errorHandler.Handle(errors.WrapIf(err, "Failed to configure Cadence client"))
	}

	releaseDeleter := cmd.CreateReleaseDeleter(config.Helm, db, commonSecretStore, commonLogger)

	clusterManager := cluster.NewManager(clusters, secretValidator, clusterEvents, statusChangeDurationMetric, clusterTotalMetric, workflowClient, logrusLogger, errorHandler, clusteradapter.NewStore(db, clusters), releaseDeleter)
	commonClusterGetter := common.NewClusterGetter(clusterManager, logrusLogger, errorHandler)

	var group run.Group

	if config.SpotMetrics.Enabled {
		ctx, cancel := context.WithCancel(context.Background())
		exporter := monitor.NewSpotMetricsExporter(
			ctx,
			clusterManager,
			logrusLogger.WithField("subsystem", "spot-metrics-exporter"),
		)

		group.Add(
			func() error {
				exporter.Run(config.SpotMetrics.CollectionInterval)

				return nil
			},
			func(err error) {
				cancel()
			},
		)
	}

	cloudinfoClient := cloudinfo.NewClient(cloudinfoapi.NewAPIClient(&cloudinfoapi.Configuration{
		BasePath:      config.Cloudinfo.Endpoint,
		DefaultHeader: make(map[string]string),
		UserAgent:     fmt.Sprintf("Pipeline/%s", version),
	}))

	azurePKEClusterStore := azurePKEAdapter.NewClusterStore(db, commonLogger)
	gormVspherePKEClusterStore := vspherePKEAdapter.NewClusterStore(db)
	k8sPreparer := intPKE.MakeKubernetesPreparer(logrusLogger, "Kubernetes")
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
		PKEOnVsphere: vspherePKEDriver.MakeVspherePKEClusterCreator(
			commonLogger,
			vspherePKEDriver.ClusterConfig{
				OIDCIssuerURL:               config.Auth.OIDC.Issuer,
				PipelineExternalURL:         externalBaseURL,
				PipelineExternalURLInsecure: externalURLInsecure,
			},
			k8sPreparer,
			authdriver.NewOrganizationGetter(db),
			secret.Store,
			gormVspherePKEClusterStore,
			workflowClient,
		),
	}

	orgService := helmadapter.NewOrgService(commonLogger)

	unifiedHelmReleaser, helmFacade := cmd.CreateUnifiedHelmReleaser(
		config.Helm,
		db,
		commonSecretStore,
		helm.ClusterKubeConfigFunc(clusterManager.KubeConfigFunc()),
		orgService,
		commonLogger,
	)

	cgroupAdapter := cgroupAdapter.NewClusterGetter(clusterManager)
	clusterGroupManager := clustergroup.NewManager(cgroupAdapter, clustergroup.NewClusterGroupRepository(db, logrusLogger), logrusLogger, errorHandler)
	federationHandler := federation.NewFederationHandler(cgroupAdapter, config.Cluster.Namespace, logrusLogger, errorHandler, config.Cluster.Federation, config.Cluster.DNS.Config, unifiedHelmReleaser)
	deploymentManager := deployment.NewCGDeploymentManager(db, cgroupAdapter, logrusLogger, errorHandler, deployment.NewHelmService(helmFacade, unifiedHelmReleaser))

	serviceMeshFeatureHandler := cgFeatureIstio.NewServiceMeshFeatureHandler(cgroupAdapter, logrusLogger, errorHandler, config.Cluster.Backyards, unifiedHelmReleaser)
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
		PKEOnVsphere: vspherePKEDriver.MakeClusterUpdater(
			commonLogger,
			vspherePKEDriver.ClusterConfig{
				OIDCIssuerURL:               config.Auth.OIDC.Issuer,
				PipelineExternalURL:         externalBaseURL,
				PipelineExternalURLInsecure: externalURLInsecure,
			},
			authdriver.NewOrganizationGetter(db),
			secret.Store,
			gormVspherePKEClusterStore,
			workflowClient,
		),
	}

	configFactory := kubernetes.NewConfigFactory(commonSecretStore)
	clientFactory := kubernetes.NewClientFactory(configFactory)
	dynamicClientFactory := kubernetes.NewDynamicClientFactory(configFactory)

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
			commonLogger,
			commonErrorHandler,
		)
		emperror.Panic(err)

		// TODO: refactor authentication middleware
		base.Any("frontend/notifications", gin.WrapH(router))
	}

	base.GET("version", gin.WrapH(buildinfo.Handler(buildInfo)))

	auth.Install(engine)
	auth.StartTokenStoreGC(tokenStore)

	enforcer := auth.NewRbacEnforcer(organizationStore, serviceAccountService, commonLogger)
	authorizationMiddleware := ginauth.NewMiddleware(enforcer, basePath, errorHandler)

	clusterSecretStore := clustersecret.NewStore(
		clustersecretadapter.NewClusterManagerAdapter(clusterManager),
		clustersecretadapter.NewSecretStore(secret.Store),
	)

	clusterAuthService, err := intClusterAuth.NewDexClusterAuthService(clusterSecretStore)
	emperror.Panic(errors.WrapIf(err, "failed to create DexClusterAuthService"))

	dashboardAPI := dashboard.NewDashboardAPI(clusterManager, clusterGroupManager, logrusLogger, errorHandler, config.Auth, clusterAuthService)
	dgroup := base.Group(path.Join("dashboard", "orgs"))
	dgroup.Use(auth.InternalHandler)
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

	organizationAPI := api.NewOrganizationAPI(organizationSyncer, auth.NewRefreshTokenStore(tokenStore), config.Helm)
	userAPI := api.NewUserAPI(db, logrusLogger, errorHandler)

	networkAPI := api.NewNetworkAPI(logrusLogger)

	{
		// cancel cancel shared spotguides sync workflow
		err = workflowClient.CancelWorkflow(context.Background(), "scrape-shared-spotguides", "")
		if _, ok := err.(*shared.EntityNotExistsError); err != nil && !ok {
			errorHandler.Handle(errors.WrapIf(err, "failed to cancel shared spotguides sync workflow"))
		}
	}

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
		unifiedHelmReleaser,
		config.Auth,
		clusterAuthService,
	)

	v1 := base.Group("api/v1")
	apiRouter := router.PathPrefix("/api/v1").Subrouter()
	{
		apiRouter.NotFoundHandler = problems.StatusProblemHandler(problems.NewStatusProblem(http.StatusNotFound))
		apiRouter.MethodNotAllowedHandler = problems.StatusProblemHandler(problems.NewStatusProblem(http.StatusMethodNotAllowed))

		v1.Use(auth.InternalHandler)
		v1.Use(auth.Handler)
		capdriver.RegisterHTTPHandler(mapCapabilities(config), commonErrorHandler, v1)
		v1.GET("/me", userAPI.GetCurrentUser)

		endpointMiddleware := []endpoint.Middleware{
			correlation.Middleware(),
			opencensus.TraceEndpoint("", opencensus.WithSpanName(func(ctx context.Context, _ string) string {
				name, _ := kitxendpoint.OperationName(ctx)

				return name
			})),
			appkitendpoint.LoggingMiddleware(logger),
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
				cRouter.GET("/nodes", api.GetClusterNodes)

				cRouter.GET("/secrets", api.ListClusterSecrets)
				cs := helm.ClusterKubeConfigFunc(clusterManager.KubeConfigFunc())

				{
					if config.Helm.V3 {
						endpoints := helmdriver.MakeEndpoints(
							helmFacade,
							kitxendpoint.Combine(endpointMiddleware...),
						)

						helmdriver.RegisterReleaserHTTPHandlers(endpoints,
							clusterRouter.PathPrefix("/deployments").Subrouter(),
							kitxhttp.ServerOptions(httpServerOptions),
						)

						cRouter.POST("/deployments", gin.WrapH(router))
						cRouter.GET("/deployments", gin.WrapH(router))
						cRouter.GET("/deployments/:name", gin.WrapH(router))
						cRouter.PUT("/deployments/:name", gin.WrapH(router))
						cRouter.HEAD("/deployments/:name", gin.WrapH(router))
						cRouter.DELETE("/deployments/:name", gin.WrapH(router))
						cRouter.GET("/deployments/:name/resources", gin.WrapH(router))

						// other version dependant operations
						cRouter.GET("/endpoints", api.MakeEndpointLister(cs, helmFacade, logger).ListEndpoints)
					} else {
						cRouter.POST("/deployments", api.CreateDeployment)
						cRouter.GET("/deployments", api.ListDeployments)
						cRouter.GET("/deployments/:name", api.GetDeployment)
						cRouter.PUT("/deployments/:name", api.UpgradeDeployment)
						cRouter.HEAD("/deployments/:name", api.HelmDeploymentStatus)
						cRouter.HEAD("/deployments", api.GetTillerStatus)
						cRouter.DELETE("/deployments/:name", api.DeleteDeployment)
						cRouter.GET("/deployments/:name/resources", api.GetDeploymentResources)

						// other version dependant operations
						releaseChecker := api.NewReleaseChecker(cs)
						cRouter.GET("/endpoints", api.MakeEndpointLister(cs, releaseChecker, logger).ListEndpoints)
					}
				}

				cRouter.GET("/images", api.ListImages)

				if config.Helm.V3 {
					imageDeploymentHandler := api.NewImageDeploymentsHandler(helmFacade, cs, logger)
					cRouter.GET("/images/:imageDigest/deployments", imageDeploymentHandler.GetImageDeployments)
				} else {
					imageDeploymentHandler := api.NewImageDeploymentsHandler(api.NewHelm2ReleaseLister(cs), cs, logger)
					cRouter.GET("/images/:imageDigest/deployments", imageDeploymentHandler.GetImageDeployments)
				}

				cRouter.GET("/deployments/:name/images", api.GetDeploymentImages)
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

					service := intCluster.NewService(
						clusterStore,
						clusteradapter.NewCadenceClusterManager(workflowClient),
						clusterGroupManager,
						map[string]intCluster.Service{
							"eks": clusteradapter.NewEKSService(eks.NewService(
								clusterStore,
								eksadapter.NewNodePoolStore(db),
								eksadapter.NewNodePoolManager(workflowClient, config.Pipeline.Enterprise),
							)),
						},
						clusteradapter.NewNodePoolStore(db, clusterStore),
						intCluster.NodePoolValidators{
							intCluster.NewCommonNodePoolValidator(labelValidator),
							intCluster.NewDistributionNodePoolValidator(map[string]intCluster.NodePoolValidator{
								"eks": eksadapter.NewNodePoolValidator(db),
							}),
						},
						intCluster.NodePoolProcessors{
							intCluster.NewCommonNodePoolProcessor(labelSource),
							intCluster.NewDistributionNodePoolProcessor(map[string]intCluster.NodePoolProcessor{
								"eks": eksadapter.NewNodePoolProcessor(db),
							}),
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

					endpoints := clusterdriver.MakeEndpoints(
						service,
						kitxendpoint.Combine(endpointMiddleware...),
					)

					clusterdriver.RegisterHTTPHandlers(
						endpoints,
						clusterRouter,
						kitxhttp.ServerOptions(httpServerOptions),
					)

					cRouter.DELETE("", gin.WrapH(router))
					cRouter.Any("/nodepools", gin.WrapH(router))
					cRouter.Any("/nodepools/:nodePoolName", gin.WrapH(router))
					cRouter.Any("/nodepools/:nodePoolName/update", gin.WrapH(router))
				}
			}

			// Cluster IntegratedService API
			var integratedServicesService integratedservices.Service
			{
				featureRepository := integratedserviceadapter.NewGormIntegratedServiceRepository(db, commonLogger)
				clusterGetter := integratedserviceadapter.MakeClusterGetter(clusterManager)
				clusterPropertyGetter := dnsadapter.NewClusterPropertyGetter(clusterManager)
				endpointManager := endpoints.NewEndpointManager(commonLogger)
				integratedServiceManagers := []integratedservices.IntegratedServiceManager{
					securityscan.MakeIntegratedServiceManager(commonLogger, config.Cluster.SecurityScan.Config),
				}

				if config.Cluster.DNS.Enabled {
					integratedServiceManagers = append(integratedServiceManagers, integratedServiceDNS.NewIntegratedServicesManager(clusterPropertyGetter, clusterPropertyGetter, config.Cluster.DNS.Config))
				}

				if config.Cluster.Vault.Enabled {
					integratedServiceManagers = append(integratedServiceManagers, integratedServiceVault.MakeIntegratedServiceManager(clusterGetter, commonSecretStore, config.Cluster.Vault.Config, commonLogger))
				}

				if config.Cluster.Monitoring.Enabled {
					integratedServiceManagers = append(integratedServiceManagers, featureMonitoring.MakeIntegratedServiceManager(
						clusterGetter,
						commonSecretStore,
						endpointManager,
						unifiedHelmReleaser,
						config.Cluster.Monitoring.Config,
						commonLogger,
					))
				}

				if config.Cluster.Logging.Enabled {
					integratedServiceManagers = append(integratedServiceManagers, integratedServiceLogging.MakeIntegratedServiceManager(
						clusterGetter,
						commonSecretStore,
						endpointManager,
						config.Cluster.Logging.Config,
						commonLogger,
					))
				}

				if config.Cluster.SecurityScan.Enabled {
					customAnchoreConfigProvider := securityscan.NewCustomAnchoreConfigProvider(
						featureRepository,
						commonSecretStore,
						commonLogger,
					)

					configProvider := anchore2.ConfigProviderChain{customAnchoreConfigProvider}

					if config.Cluster.SecurityScan.Anchore.Enabled {
						configProvider = append(configProvider, securityscan.NewClusterAnchoreConfigProvider(
							config.Cluster.SecurityScan.Anchore.Endpoint,
							securityscanadapter.NewUserNameGenerator(securityscanadapter.NewClusterService(clusterManager)),
							securityscanadapter.NewUserSecretStore(commonSecretStore),
							config.Cluster.SecurityScan.Anchore.Insecure,
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

				if config.Cluster.Ingress.Enabled {
					integratedServiceManagers = append(integratedServiceManagers, ingress.NewManager(
						config.Cluster.Ingress.Config,
						unifiedHelmReleaser,
						commonLogger,
					))
				}

				integratedServiceManagerRegistry := integratedservices.MakeIntegratedServiceManagerRegistry(integratedServiceManagers)
				integratedServiceOperationDispatcher := integratedserviceadapter.MakeCadenceIntegratedServiceOperationDispatcher(workflowClient, commonLogger)
				integratedServicesService = integratedservices.MakeIntegratedServiceService(integratedServiceOperationDispatcher, integratedServiceManagerRegistry, featureRepository, commonLogger)
				endpoints := integratedservicesdriver.MakeEndpoints(
					integratedServicesService,
					kitxendpoint.Combine(endpointMiddleware...),
				)

				{
					integratedservicesdriver.RegisterHTTPHandlers(
						endpoints,
						clusterRouter.PathPrefix("/services").Subrouter(),
						kitxhttp.ServerOptions(httpServerOptions),
					)

					cRouter.Any("/services", gin.WrapH(router))
					cRouter.Any("/services/:serviceName", gin.WrapH(router))
				}

				// set up legacy endpoint
				{
					integratedservicesdriver.RegisterHTTPHandlers(
						endpoints,
						clusterRouter.PathPrefix("/features").Subrouter(),
						kitxhttp.ServerOptions(httpServerOptions),
					)

					cRouter.Any("/features", gin.WrapH(router))
					cRouter.Any("/features/:featureName", gin.WrapH(router))
				}
			}

			hpaApi := api.NewHPAAPI(integratedServicesService, clientFactory, configFactory, commonClusterGetter, errorHandler)
			cRouter.GET("/hpa", hpaApi.GetHpaResource)
			cRouter.PUT("/hpa", hpaApi.PutHpaResource)
			cRouter.DELETE("/hpa", hpaApi.DeleteHpaResource)

			// ClusterGroupAPI
			cgroupsAPI := cgroupAPI.NewAPI(clusterGroupManager, deploymentManager, logrusLogger, errorHandler)
			cgroupsAPI.AddRoutes(orgs.Group("/:orgid/clustergroups"))

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
			{
				endpoints := helmdriver.MakeEndpoints(
					helmFacade,
					kitxendpoint.Combine(endpointMiddleware...),
				)
				helmdriver.RegisterHTTPHandlers(endpoints,
					orgRouter.PathPrefix("/helm").Subrouter(),
					kitxhttp.ServerOptions(httpServerOptions),
				)

				orgs.POST("/:orgid/helm/repos", gin.WrapH(router))
				orgs.GET("/:orgid/helm/repos", gin.WrapH(router))
				orgs.PATCH("/:orgid/helm/repos/:name", gin.WrapH(router))
				orgs.PUT("/:orgid/helm/repos/:name", gin.WrapH(router))
				orgs.DELETE("/:orgid/helm/repos/:name", gin.WrapH(router))
				orgs.PUT("/:orgid/helm/repos/:name/update", gin.WrapH(router))

				orgs.GET("/:orgid/helm/charts", gin.WrapH(router))

				// TODO using "chart" instead of  "charts" for backwards compatibility
				orgs.GET("/:orgid/helm/chart/:reponame/:name", gin.WrapH(router))
			}
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
				secretStore := googleadapter.NewSecretStore(commonSecretStore)
				clientFactory := google.NewClientFactory(secretStore)

				service := googleproject.NewService(clientFactory)
				endpoints := googleprojectdriver.MakeEndpoints(
					service,
					kitxendpoint.Combine(endpointMiddleware...),
				)

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

			endpoints := tokendriver.MakeEndpoints(
				service,
				kitxendpoint.Combine(endpointMiddleware...),
			)

			tokendriver.RegisterHTTPHandlers(
				endpoints,
				apiRouter.PathPrefix("/tokens").Subrouter(),
				kitxhttp.ServerOptions(httpServerOptions),
			)

			v1.Any("/tokens", gin.WrapH(router))
			v1.Any("/tokens/*path", gin.WrapH(router))
		}

		{
			service := secrettype.NewService(secretTypes)
			endpoints := secrettypedriver.MakeEndpoints(
				service,
				kitxendpoint.Combine(endpointMiddleware...),
			)

			secrettypedriver.RegisterHTTPHandlers(
				endpoints,
				apiRouter.PathPrefix("/secret-types").Subrouter(),
				kitxhttp.ServerOptions(httpServerOptions),
			)

			v1.Any("/secret-types", gin.WrapH(router))
			v1.Any("/secret-types/*path", gin.WrapH(router))
		}

		{
			err := process.RegisterApp(
				orgRouter,
				db,
				workflowClient,
				commonLogger,
				commonErrorHandler,
			)
			emperror.Panic(err)

			orgs.Any("/:orgid/processes", gin.WrapH(router))
			orgs.Any("/:orgid/processes/*path", gin.WrapH(router))
		}

		backups.AddRoutes(orgs.Group("/:orgid/clusters/:id/backups"))
		backupservice.AddRoutes(orgs.Group("/:orgid/clusters/:id/backupservice"), unifiedHelmReleaser)
		restores.AddRoutes(orgs.Group("/:orgid/clusters/:id/restores"))
		schedules.AddRoutes(orgs.Group("/:orgid/clusters/:id/schedules"))
		buckets.AddRoutes(orgs.Group("/:orgid/backupbuckets"))
		backups.AddOrgRoutes(orgs.Group("/:orgid/backups"), clusterManager)
	}

	arkEvents.NewClusterEventHandler(arkEvents.NewClusterEvents(clusterEventBus), db, logrusLogger)
	if config.Cluster.DisasterRecovery.Ark.SyncEnabled {
		ctx, cancel := context.WithCancel(context.Background())

		group.Add(
			func() error {
				arkSync.RunSyncServices(
					ctx,
					db,
					arkClusterManager.New(clusterManager),
					logrusLogger.WithField("subsystem", "ark"),
					errorHandler,
					config.Cluster.DisasterRecovery.Ark.BucketSyncInterval,
					config.Cluster.DisasterRecovery.Ark.RestoreSyncInterval,
					config.Cluster.DisasterRecovery.Ark.BackupSyncInterval,
				)

				return nil
			},
			func(err error) {
				cancel()
			},
		)
	}

	base.GET("api", api.MetaHandler(engine, basePath+"/api"))

	{
		logger := logur.WithField(logger, "server", "internal")

		server := &http.Server{
			Handler:  createInternalAPIRouter(config, db, basePath, clusterAPI, cloudinfoClient, logger, logrusLogger),
			ErrorLog: log.NewErrorStandardLogger(logger),
		}
		defer server.Close()

		logger.Info("listening on address", map[string]interface{}{"address": config.Pipeline.InternalAddr})

		ln, err := net.Listen("tcp", config.Pipeline.InternalAddr)
		emperror.Panic(err)

		group.Add(appkitrun.LogServe(logger)(appkitrun.HTTPServe(server, ln, 5*time.Second)))
	}

	{
		logger := logur.WithField(logger, "server", "app")

		server := &http.Server{
			Handler:  engine,
			ErrorLog: log.NewErrorStandardLogger(logger),
		}
		defer server.Close()

		ln, err := net.Listen("tcp", config.Pipeline.Addr)
		emperror.Panic(err)

		scheme := "http"

		caCertFile, certFile, keyFile := config.Pipeline.CACertFile, config.Pipeline.CertFile, config.Pipeline.KeyFile
		if certFile != "" && keyFile != "" {
			tlsConfig := &tls.Config{}

			if caCertFile != "" {
				tlsConfig, err = auth.TLSConfigForClientAuth(caCertFile)
				emperror.Panic(err)
			}

			serverCertificate, err := tls.LoadX509KeyPair(certFile, keyFile)
			emperror.Panic(err)

			tlsConfig.Certificates = []tls.Certificate{serverCertificate}

			ln = tls.NewListener(ln, tlsConfig)
			scheme = "https"
		}

		logger.Info("listening on address", map[string]interface{}{"address": config.Pipeline.Addr, "scheme": scheme})

		group.Add(appkitrun.LogServe(logger)(appkitrun.HTTPServe(server, ln, 5*time.Second)))
	}

	// Setup signal handler
	group.Add(run.SignalHandler(context.Background(), syscall.SIGINT, syscall.SIGTERM))

	err = group.Run()
	emperror.WithFilter(errorHandler, match.As(&run.SignalError{}).MatchError).Handle(err)
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
