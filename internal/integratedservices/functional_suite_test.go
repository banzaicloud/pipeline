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
	"os"
	"regexp"

	"github.com/banzaicloud/bank-vaults/pkg/sdk/vault"
	"github.com/banzaicloud/pipeline/internal/cluster/clusteradapter/clustermodel"
	"github.com/banzaicloud/pipeline/internal/cmd"
	"github.com/banzaicloud/pipeline/internal/common/commonadapter"
	"github.com/banzaicloud/pipeline/internal/global"
	"github.com/banzaicloud/pipeline/internal/integratedservices"
	"github.com/banzaicloud/pipeline/internal/integratedservices/integratedserviceadapter"
	"github.com/banzaicloud/pipeline/internal/platform/cadence"
	"github.com/banzaicloud/pipeline/internal/platform/database"
	"github.com/banzaicloud/pipeline/internal/platform/log"
	"github.com/banzaicloud/pipeline/internal/providers/kubernetes/kubernetesadapter"
	"github.com/banzaicloud/pipeline/internal/secret/secretadapter"
	"github.com/banzaicloud/pipeline/internal/secret/types"
	"github.com/banzaicloud/pipeline/src/model"
	"github.com/banzaicloud/pipeline/src/secret"
	"github.com/stretchr/testify/suite"
	zaplog "logur.dev/integration/zap"
)

type Suite struct {
	suite.Suite

	projectDir     string
	kubeconfigPath string
	config         *cmd.Config

	integratedServiceServiceCreater func(...integratedservices.IntegratedServiceManager) (*integratedservices.IntegratedServiceService, error)
}

func (s *Suite) SetupSuite() {
	if m := flag.Lookup("test.run").Value.String(); m == "" || !regexp.MustCompile(m).MatchString(s.T().Name()) {
		s.T().Skip("skipping as execution was not requested explicitly using go test -run")
	}
	if os.Getenv("VAULT_ADDR") == "" {
		s.T().Fatal("VAULT_ADDR is not defined")
	}
	s.kubeconfigPath = os.Getenv("KUBECONFIG")
	if s.kubeconfigPath == "" {
		s.T().Fatal("KUBECONFIG is not defined")
	}
	s.projectDir = os.Getenv("PROJECT_DIR")
	if s.projectDir == "" {
		s.T().Fatal("PROJECT_DIR is not defined")
	}

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

	zaplog := zaplog.New(logurLogger)
	workflowClient, err := cadence.NewClient(s.config.Cadence, zaplog)
	s.Require().NoError(err)

	s.integratedServiceServiceCreater = func(managers ...integratedservices.IntegratedServiceManager) (*integratedservices.IntegratedServiceService, error) {
		featureRepository := integratedserviceadapter.NewGormIntegratedServiceRepository(db, commonLogger)
		registry := integratedservices.MakeIntegratedServiceManagerRegistry(managers)
		dispatcher := integratedserviceadapter.MakeCadenceIntegratedServiceOperationDispatcher(workflowClient, commonLogger)
		serviceFacade := integratedservices.MakeIntegratedServiceService(dispatcher, registry, featureRepository, commonLogger)
		return &serviceFacade, nil
	}
}
