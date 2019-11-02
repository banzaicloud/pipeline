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
	"regexp"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
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

	ControlPlaneNamespace = "infra.control-plane-namespace" // Namespace where the pipeline and prometheus runs
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
	viper.SetDefault("pipeline.uipath", "/ui")
	viper.SetDefault("pipeline.basepath", "")
	viper.SetDefault("pipeline.signupRedirectPath", "/ui")
	viper.SetDefault(PipelineUUID, "")
	viper.SetDefault("audit.enabled", true)
	viper.SetDefault("audit.headers", []string{"secretId"})
	viper.SetDefault("audit.skippaths", []string{"/auth/dex/callback", "/pipeline/api"})
	viper.SetDefault("tls.validity", "8760h") // 1 year

	_ = viper.BindEnv(ControlPlaneNamespace, "KUBERNETES_NAMESPACE")
	viper.SetDefault(ControlPlaneNamespace, "default")

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

// GetCORS gets CORS related config
func GetCORS() cors.Config {
	viper.SetDefault("cors.AllowAllOrigins", true)
	viper.SetDefault("cors.AllowOrigins", []string{})
	viper.SetDefault("cors.AllowOriginsRegexp", "")
	viper.SetDefault("cors.AllowMethods", []string{"PUT", "DELETE", "GET", "POST", "OPTIONS", "PATCH"})
	viper.SetDefault("cors.AllowHeaders", []string{"Origin", "Authorization", "Content-Type", "secretId", "Banzai-Cloud-Pipeline-UUID"})
	viper.SetDefault("cors.ExposeHeaders", []string{"Content-Length"})
	viper.SetDefault("cors.AllowCredentials", true)
	viper.SetDefault("cors.MaxAge", 12)

	config := cors.DefaultConfig()
	config.AllowAllOrigins = viper.GetBool("cors.AllowAllOrigins")
	if !config.AllowAllOrigins {
		allowOriginsRegexp := viper.GetString("cors.AllowOriginsRegexp")
		if allowOriginsRegexp != "" {
			originsRegexp, err := regexp.Compile(fmt.Sprintf("^(%s)$", allowOriginsRegexp))
			if err == nil {
				config.AllowOriginFunc = func(origin string) bool {
					return originsRegexp.Match([]byte(origin))
				}
			}
		} else if allowOrigins := viper.GetStringSlice("cors.AllowOrigins"); len(allowOrigins) > 0 {
			config.AllowOrigins = allowOrigins
		}
	}

	config.AllowMethods = viper.GetStringSlice("cors.AllowMethods")
	config.AllowHeaders = viper.GetStringSlice("cors.AllowHeaders")
	config.ExposeHeaders = viper.GetStringSlice("cors.ExposeHeaders")
	config.AllowCredentials = viper.GetBool("cors.AllowCredentials")
	maxAge := viper.GetInt("cors.MaxAge")
	config.MaxAge = time.Duration(maxAge) * time.Hour
	return config
}
