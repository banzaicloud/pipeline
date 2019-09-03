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
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/banzaicloud/pipeline/internal/app/pipelinectl/cli/commands"
)

// Provisioned by ldflags
// nolint: gochecknoglobals
var (
	version    string
	commitHash string
	buildDate  string
)

func main() {
	// rootCmd represents the base command when called without any subcommands
	rootCmd := &cobra.Command{
		Use:     appName,
		Short:   appName + " manages a Pipeline instance.",
		Version: version,
	}

	rootCmd.SetVersionTemplate(fmt.Sprintf("%s version %s (%s) built on %s\n", appName, version, commitHash, buildDate))

	flags := rootCmd.PersistentFlags()

	flags.StringP("url", "u", "http://127.0.0.1:9090", "Pipeline API URL")
	_ = viper.BindPFlag("api.url", flags.Lookup("url"))

	flags.Bool("verify", true, "Verify root CA")
	_ = viper.BindPFlag("api.verify", flags.Lookup("verify"))

	viper.SetEnvPrefix(envPrefix)
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	viper.AutomaticEnv()

	// Pipeline configuration
	viper.SetDefault("api.url", "http://127.0.0.1:9090")
	viper.SetDefault("api.verify", true)

	cobra.OnInitialize(func() {
		if !viper.GetBool("api.verify") {
			http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
		}
	})

	commands.AddCommands(rootCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)

		os.Exit(1)
	}
}
