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

package eksadapter

import (
	"context"
	"fmt"
	"io/ioutil"
	"strings"
	"testing"
	"time"

	"github.com/jinzhu/gorm"

	//  SQLite driver used for integration test
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"

	"github.com/banzaicloud/pipeline/internal/cluster/distribution/eks/eksmodel"
)

func setUpDatabase(t *testing.T) *gorm.DB {
	db, err := gorm.Open("sqlite3", "file::memory:")
	require.NoError(t, err)

	logger := logrus.New()
	logger.SetOutput(ioutil.Discard)

	tables := []interface{}{
		&eksmodel.AmazonNodePoolsModel{},
		&eksmodel.EKSClusterModel{},
	}

	var tableNames string
	for _, table := range tables {
		tableNames += fmt.Sprintf(" %s", db.NewScope(table).TableName())
	}

	logger.WithFields(logrus.Fields{
		"table_names": strings.TrimSpace(tableNames),
	}).Info("migrating auth tables")

	err = db.AutoMigrate(tables...).Error
	require.NoError(t, err)

	return db
}

func TestListNodePoolNames(t *testing.T) {
	db := setUpDatabase(t)
	store := NewNodePoolStore(db)

	clusterID1 := uint(5)
	clusterID2 := uint(10)
	clusterID3 := uint(666)
	eksClusterID1 := uint(1)
	eksClusterID2 := uint(6)
	now := time.Time{}
	routeTableID := ""
	vpcCIDR := "192.168.0.0/16"
	vpcID := "vpc-0cb87f9bcff31a60f"
	eksClusters := []eksmodel.EKSClusterModel{
		{
			ID:        eksClusterID1,
			Version:   "2.3.4",
			ClusterID: clusterID1,
			NodePools: []*eksmodel.AmazonNodePoolsModel{
				{
					ID:               11,
					CreatedAt:        now,
					CreatedBy:        12,
					ClusterID:        eksClusterID1,
					Name:             fmt.Sprintf("%d-pool-1", eksClusterID1),
					NodeSpotPrice:    "0.13",
					Autoscaling:      false,
					NodeMinCount:     14,
					NodeMaxCount:     15,
					Count:            16,
					NodeVolumeSize:   0,
					NodeImage:        "node-image",
					NodeInstanceType: "node-instance-type",
					Labels:           map[string]string{},
					Delete:           false,
				},
				{
					ID:               23,
					CreatedAt:        now,
					CreatedBy:        24,
					ClusterID:        eksClusterID1,
					Name:             fmt.Sprintf("%d-pool-2", eksClusterID1),
					NodeSpotPrice:    "0.25",
					Autoscaling:      false,
					NodeMinCount:     26,
					NodeMaxCount:     27,
					Count:            28,
					NodeVolumeSize:   0,
					NodeImage:        "node-image",
					NodeInstanceType: "node-instance-type",
					Labels:           map[string]string{},
					Delete:           false,
				},
			},
			VpcId:                 &vpcID,
			VpcCidr:               &vpcCIDR,
			RouteTableId:          &routeTableID,
			DefaultUser:           false,
			ClusterRoleId:         "",
			NodeInstanceRoleId:    "node-instance-role-id",
			LogTypes:              []string{"log-type-1", "log-type-2"},
			APIServerAccessPoints: eksmodel.JSONStringArray{"public"},
			CurrentWorkflowID:     "b63db127-3242-4544-8f62-7306d435977a",
			SSHGenerated:          true,
		},
		{
			ID:        eksClusterID2,
			Version:   "7.8.9",
			ClusterID: clusterID2,
			NodePools: []*eksmodel.AmazonNodePoolsModel{
				{
					ID:               17,
					CreatedAt:        now,
					CreatedBy:        18,
					ClusterID:        eksClusterID2,
					Name:             fmt.Sprintf("%d-pool-1", eksClusterID2),
					NodeSpotPrice:    "0.19",
					Autoscaling:      false,
					NodeMinCount:     20,
					NodeMaxCount:     21,
					Count:            22,
					NodeVolumeSize:   0,
					NodeImage:        "node-image",
					NodeInstanceType: "node-instance-type",
					Labels:           map[string]string{},
					Delete:           false,
				},
			},
			VpcId:                 &vpcID,
			VpcCidr:               &vpcCIDR,
			RouteTableId:          &routeTableID,
			DefaultUser:           false,
			ClusterRoleId:         "",
			NodeInstanceRoleId:    "node-instance-role-id",
			APIServerAccessPoints: eksmodel.JSONStringArray{"public"},
			CurrentWorkflowID:     "eb3b128f-c934-4215-85fa-b17f7f446387",
			SSHGenerated:          true,
		},
	}
	for _, eksCluster := range eksClusters {
		err := db.Save(&eksCluster).Error
		require.NoError(t, err)
	}

	actualCluster1NodePoolNames, err := store.ListNodePoolNames(context.Background(), clusterID1)
	require.NoError(t, err)
	require.Equal(t, []string{fmt.Sprintf("%d-pool-1", eksClusterID1), fmt.Sprintf("%d-pool-2", eksClusterID1)}, actualCluster1NodePoolNames)

	actualCluster2NodePoolNames, err := store.ListNodePoolNames(context.Background(), clusterID2)
	require.NoError(t, err)
	require.Equal(t, []string{fmt.Sprintf("%d-pool-1", eksClusterID2)}, actualCluster2NodePoolNames)

	actualNonExistentClusterNodePoolNames, err := store.ListNodePoolNames(context.Background(), clusterID3)
	require.Error(t, err)
	require.Nil(t, actualNonExistentClusterNodePoolNames)
}
