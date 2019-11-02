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

package config

import (
	"fmt"
	"log"
	"strings"

	"github.com/spf13/viper"
)

const (
	// PipelineHeadNodePoolName name of our Head node pool for Pipeline Infra deployments
	PipelineHeadNodePoolName = "infra.headNodePoolName"

	// PipelineExternalURLInsecure specifies whether the external URL of the Pipeline is insecure
	// as uses self-signed CA cert
	PipelineExternalURLInsecure = "pipeline.externalURLInsecure"

	// PipelineUUID is an UUID that identifies the specific installation (deployment) of the platform
	PipelineUUID = "pipeline.uuid"
)

// Init initializes the configurations
func init() {

	viper.AddConfigPath("$HOME/config")
	viper.AddConfigPath("./")
	viper.AddConfigPath("./config")
	viper.AddConfigPath("$PIPELINE_CONFIG_DIR/")
	viper.SetConfigName("config")

	viper.SetDefault("pipeline.bindaddr", "127.0.0.1:9090")
	viper.SetDefault(PipelineExternalURLInsecure, false)
	viper.SetDefault("pipeline.certfile", "")
	viper.SetDefault("pipeline.keyfile", "")
	viper.SetDefault("pipeline.basepath", "")
	viper.SetDefault(PipelineUUID, "")

	// Find and read the config file
	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("Error reading config file, %s", err)
	}
	// Confirm which config file is used
	fmt.Printf("Using config: %s\n", viper.ConfigFileUsed())
	viper.SetEnvPrefix("pipeline")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()
	viper.AllowEmptyEnv(true)
}
