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
	"github.com/jinzhu/gorm"
	"github.com/sirupsen/logrus"

	"github.com/banzaicloud/pipeline/internal/clusterfeature/clusterfeatureadapter"
	"github.com/banzaicloud/pipeline/internal/clustergroup/deployment"

	"github.com/banzaicloud/pipeline/auth"
	route53model "github.com/banzaicloud/pipeline/dns/route53/model"
	"github.com/banzaicloud/pipeline/internal/ark"
	"github.com/banzaicloud/pipeline/internal/audit"
	"github.com/banzaicloud/pipeline/internal/cluster"
	"github.com/banzaicloud/pipeline/internal/clustergroup"
	"github.com/banzaicloud/pipeline/internal/notification"
	"github.com/banzaicloud/pipeline/internal/providers"
	"github.com/banzaicloud/pipeline/model"
	"github.com/banzaicloud/pipeline/spotguide"
)

// Migrate runs migrations for the application.
func Migrate(db *gorm.DB, logger logrus.FieldLogger) error {
	if err := model.Migrate(db, logger); err != nil {
		return err
	}

	if err := auth.Migrate(db, logger); err != nil {
		return err
	}

	if err := route53model.Migrate(db, logger); err != nil {
		return err
	}

	if err := spotguide.Migrate(db, logger); err != nil {
		return err
	}

	if err := audit.Migrate(db, logger); err != nil {
		return err
	}

	if err := cluster.Migrate(db, logger); err != nil {
		return err
	}

	if err := clustergroup.Migrate(db, logger); err != nil {
		return err
	}

	if err := deployment.Migrate(db, logger); err != nil {
		return err
	}

	if err := providers.Migrate(db, logger); err != nil {
		return err
	}

	if err := ark.Migrate(db, logger); err != nil {
		return err
	}

	if err := notification.Migrate(db, logger); err != nil {
		return err
	}

	if err := clusterfeatureadapter.Migrate(db, logger); err != nil {
		return err
	}

	return nil
}
