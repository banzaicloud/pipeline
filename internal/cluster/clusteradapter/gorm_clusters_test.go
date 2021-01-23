// Copyright © 2021 Banzai Cloud
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

package clusteradapter

import (
	"fmt"
	"testing"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/banzaicloud/pipeline/internal/cluster"
	"github.com/banzaicloud/pipeline/internal/cluster/clusteradapter/clustermodel"
)

type GormClustersTestSuite struct {
	suite.Suite
	db       *gorm.DB
	clusters Clusters
}

func (g *GormClustersTestSuite) SetupTest() {
	g.clusters = Clusters{
		db: g.db,
	}
}

func (g *GormClustersTestSuite) SetupSuite() {
	db, err := gorm.Open("sqlite3", "file::memory:")
	require.NoError(g.T(), err)

	g.db = db
	tables := []interface{}{
		clustermodel.ClusterModel{},
	}
	if e := g.db.AutoMigrate(tables...).Error; e != nil {
		g.Fail("faie=lsed to migrate")
	}

	// insert 10 records to the database
	for i := 1; i < 11; i++ {
		models :=
			&clustermodel.ClusterModel{
				ID:             uint(i),
				UID:            fmt.Sprintf("testuuid_%d", i),
				Name:           "testing",
				OrganizationID: 1,
			}
		err := g.db.Create(models).Error
		require.NoError(g.T(), err)
	}
}

func (g *GormClustersTestSuite) Test_FindNextWithGreaterID_RecordFound() {
	// GIVEN
	lastClusterID := uint(3)

	// WHNE
	orgID, clusterID, err := g.clusters.FindNextWithGreaterID(lastClusterID)

	// THEN
	require.NoError(g.T(), err)
	require.Equal(g.T(), uint(1), orgID)
	require.Equal(g.T(), uint(4), clusterID)
}

func (g *GormClustersTestSuite) Test_FindNextWithGreaterID_RecordNotFound() {
	// GIVEN
	// the last cluster id is greater than the greatest id in the database
	greatClusterID := uint(15)

	// WHEN
	_, _, err := g.clusters.FindNextWithGreaterID(greatClusterID)

	// THEN
	require.Error(g.T(), err)
	assert.True(g.T(), cluster.IsNotFoundError(err))
}

// TestGormClustersTestSuite triggers the suite
func TestGormClustersTestSuite(t *testing.T) {
	suite.Run(t, new(GormClustersTestSuite))
}
