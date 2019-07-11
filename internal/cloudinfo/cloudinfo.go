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

package cloudinfo

import (
	"github.com/banzaicloud/pipeline/.gen/cloudinfo"
	"github.com/sirupsen/logrus"
)

type Client struct {
	apiClient *cloudinfo.APIClient
	logger    logrus.FieldLogger
}

func NewClient(cloudInfoEndpoint string, logger logrus.FieldLogger) *Client {

	cloudInfoClient := cloudinfo.NewAPIClient(&cloudinfo.Configuration{
		BasePath:      cloudInfoEndpoint,
		DefaultHeader: make(map[string]string),
		UserAgent:     "Pipeline/go",
	})

	return &Client{
		apiClient: cloudInfoClient,
		logger:    logger.WithFields(logrus.Fields{"cloudInfoEndpoint": cloudInfoEndpoint}),
	}
}
