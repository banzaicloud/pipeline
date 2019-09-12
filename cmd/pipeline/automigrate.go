// +build automigrate
// Copyright © 2019 Banzai Cloud
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
	"io/ioutil"

	"github.com/banzaicloud/pipeline/config"
	"github.com/banzaicloud/pipeline/internal/common/commonadapter"

	"github.com/sirupsen/logrus"
)

func main() {
	db := config.DB()

	logger := logrus.New()
	logger.SetOutput(ioutil.Discard)

	err := Migrate(db, logger, commonadapter.NewNoopLogger())
	if err != nil {
		panic(err)
	}
}
