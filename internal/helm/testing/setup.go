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

package testing

import (
	"context"
	"io/ioutil"
	"testing"

	"emperror.dev/errors"
	"github.com/jinzhu/gorm"

	// for the in-memory db tests
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"

	"github.com/banzaicloud/pipeline/internal/common"
	"github.com/banzaicloud/pipeline/internal/common/commonadapter"
	"github.com/banzaicloud/pipeline/internal/helm"
	"github.com/banzaicloud/pipeline/internal/helm/helmadapter"
	"github.com/banzaicloud/pipeline/internal/platform/log"
	internaltesting "github.com/banzaicloud/pipeline/internal/testing"
	"github.com/banzaicloud/pipeline/src/secret"
)

type FakeOrg struct {
	OrgId   uint
	OrgName string
}

func (f FakeOrg) GetOrgNameByOrgID(ctx context.Context, orgID uint) (string, error) {
	if f.OrgId != orgID {
		return "", errors.Errorf("unknown org id: %d, expected: %d", orgID, f.OrgId)
	}
	return f.OrgName, nil
}

func SetupDatabase(t *testing.T) *gorm.DB {
	db, err := gorm.Open("sqlite3", "file::memory:")
	require.NoError(t, err)

	err = helmadapter.Migrate(db, common.NoopLogger{})
	require.NoError(t, err)

	return db
}

func SetupSecretStore() common.SecretStore {
	return commonadapter.NewSecretStore(secret.Store, commonadapter.OrgIDContextExtractorFunc(func(ctx context.Context) (uint, bool) {
		return 0, false
	}))
}

func ClusterKubeConfig(t *testing.T, clusterId uint) ([]byte, helm.ClusterService) {
	kubeConfigBytes := internaltesting.KubeConfigFromEnv(t)
	return kubeConfigBytes, helm.ClusterKubeConfigFunc(func(ctx context.Context, c uint) ([]byte, error) {
		if c != clusterId {
			return nil, errors.Errorf("invalid clusterid: %d expected: %d", c, clusterId)
		}
		return kubeConfigBytes, nil
	})
}

func HelmHome(t *testing.T) string {
	home, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatalf("%+v", err)
	}
	return home
}

func Loggers() (helm.Logger, logrus.FieldLogger) {
	logConfig := log.Config{
		Level:   "debug",
		NoColor: true,
	}
	return commonadapter.NewLogger(log.NewLogger(logConfig)), log.NewLogrusLogger(logConfig)
}
