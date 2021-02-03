// Copyright © 2020 Banzai Cloud
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

package integratedservices_test

import (
	"flag"
	"io/ioutil"
	"os"
	"regexp"

	"github.com/banzaicloud/bank-vaults/pkg/sdk/vault"
	"github.com/stretchr/testify/suite"
	zaplog "logur.dev/integration/zap"

	"github.com/banzaicloud/pipeline/internal/cluster/clusteradapter/clustermodel"
	"github.com/banzaicloud/pipeline/internal/cmd"
	"github.com/banzaicloud/pipeline/internal/common"
	"github.com/banzaicloud/pipeline/internal/common/commonadapter"
	"github.com/banzaicloud/pipeline/internal/global"
	"github.com/banzaicloud/pipeline/internal/helm/helmadapter"
	"github.com/banzaicloud/pipeline/internal/integratedservices"
	"github.com/banzaicloud/pipeline/internal/integratedservices/integratedserviceadapter"
	"github.com/banzaicloud/pipeline/internal/integratedservices/services"
	integratedServiceDNS "github.com/banzaicloud/pipeline/internal/integratedservices/services/dns"
	"github.com/banzaicloud/pipeline/internal/platform/cadence"
	"github.com/banzaicloud/pipeline/internal/platform/database"
	"github.com/banzaicloud/pipeline/internal/platform/log"
	"github.com/banzaicloud/pipeline/internal/providers/kubernetes/kubernetesadapter"
	"github.com/banzaicloud/pipeline/internal/secret/secretadapter"
	"github.com/banzaicloud/pipeline/internal/secret/types"
	"github.com/banzaicloud/pipeline/src/auth"
	"github.com/banzaicloud/pipeline/src/model"
	"github.com/banzaicloud/pipeline/src/secret"
)

type Suite struct {
	suite.Suite

	v2 bool

	kubeconfig   string
	config       *cmd.Config
	commonLogger common.Logger

	integratedServiceServiceCreater   func(...integratedservices.IntegratedServiceManager) (integratedservices.Service, error)
	integratedServiceServiceCreaterV2 func(integratedservices.ClusterKubeConfigFunc, ...integratedservices.IntegratedServiceManager) (integratedservices.Service, error)
}

func (s *Suite) SetupSuite() {
	if m := flag.Lookup("test.run").Value.String(); m == "" || !regexp.MustCompile(m).MatchString(s.T().Name()) {
		s.T().Skip("skipping as execution was not requested explicitly using go test -run")
	}
	if os.Getenv("VAULT_ADDR") == "" {
		s.T().Fatal("VAULT_ADDR is not defined")
	}
	kubeconfigPath := os.Getenv("KUBECONFIG")
	if kubeconfigPath == "" {
		s.T().Fatal("KUBECONFIG is not defined")
	}
	kubeconfig, err := ioutil.ReadFile(kubeconfigPath)
	if err != nil {
		s.T().Fatal("reading kubeconfig failed")
	}
	s.kubeconfig = string(kubeconfig)

	s.config = loadConfig()

	db, err := database.Connect(s.config.Database.Config)
	s.Require().NoError(err)

	global.SetDB(db)

	logger := log.NewLogrusLogger(s.config.Log)

	err = clustermodel.Migrate(db, logger)
	s.Require().NoError(err)

	err = kubernetesadapter.Migrate(db, logger)
	s.Require().NoError(err)

	err = model.Migrate(db, logger)
	s.Require().NoError(err)

	err = integratedserviceadapter.Migrate(db, logger)
	s.Require().NoError(err)

	err = helmadapter.Migrate(db, common.NoopLogger{})
	s.Require().NoError(err)

	vaultClient, err := vault.NewClientWithOptions()
	s.Require().NoError(err)

	global.SetVault(vaultClient)

	{
		secretStore := secretadapter.NewVaultStore(vaultClient, "secret")
		secretTypes := types.NewDefaultTypeList(types.DefaultTypeListConfig{})
		secret.InitSecretStore(secretStore, secretTypes)
	}

	logurLogger := log.NewLogger(s.config.Log)
	commonLogger := commonadapter.NewLogger(logurLogger)
	s.commonLogger = commonLogger

	zaplog := zaplog.New(logurLogger)
	workflowClient, err := cadence.NewClient(s.config.Cadence, zaplog)
	s.Require().NoError(err)

	s.integratedServiceServiceCreater = func(managers ...integratedservices.IntegratedServiceManager) (integratedservices.Service, error) {
		featureRepository := integratedserviceadapter.NewGormIntegratedServiceRepository(db, commonLogger)
		registry := integratedservices.MakeIntegratedServiceManagerRegistry(managers)
		dispatcher := integratedserviceadapter.MakeCadenceIntegratedServiceOperationDispatcher(workflowClient, commonLogger)
		serviceFacade := integratedservices.MakeIntegratedServiceService(dispatcher, registry, featureRepository, commonLogger)
		return &serviceFacade, nil
	}

	commonSecretStore := commonadapter.NewSecretStore(secret.Store, commonadapter.OrgIDContextExtractorFunc(auth.GetCurrentOrganizationID))

	s.integratedServiceServiceCreaterV2 = func(kubeConfigFunc integratedservices.ClusterKubeConfigFunc, managers ...integratedservices.IntegratedServiceManager) (integratedservices.Service, error) {
		registry := integratedservices.MakeIntegratedServiceManagerRegistry(managers)
		dispatcher := integratedserviceadapter.NewCadenceOperationDispatcher(workflowClient, commonLogger)
		specConversions := map[string]integratedservices.SpecConversion{
			integratedServiceDNS.IntegratedServiceName: integratedServiceDNS.NewSecretMapper(commonSecretStore),
		}
		outputResolvers := map[string]integratedserviceadapter.OutputResolver{
			integratedServiceDNS.IntegratedServiceName: &integratedServiceDNS.OutputResolver{},
		}
		serviceConversion := integratedserviceadapter.NewServiceConversion(services.NewServiceStatusMapper(), specConversions, outputResolvers)
		clusterRepository := integratedserviceadapter.NewCustomResourceRepository(kubeConfigFunc, commonLogger, serviceConversion, s.config.Cluster.Namespace)
		serviceFacade := integratedservices.NewISServiceV2(registry, dispatcher, clusterRepository, commonLogger)
		return serviceFacade, nil
	}
}
