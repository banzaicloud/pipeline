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
	"fmt"
	"os"
	"strings"

	"github.com/banzaicloud/pipeline/internal/pipelinectl/cli/commands"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func main() {
	// rootCmd represents the base command when called without any subcommands
	rootCmd := &cobra.Command{
		Use:     ServiceName,
		Short:   ServiceName + " manages a Pipeline instance.",
		Version: version,
	}

	rootCmd.SetVersionTemplate(fmt.Sprintf("pipelinectl version %s (%s) built on %s\n", version, commitHash, buildDate))

	flags := rootCmd.PersistentFlags()

	flags.StringP("url", "u", "http://127.0.0.1:9090", "Pipeline API URL")
	_ = viper.BindPFlag("api.url", flags.Lookup("url"))

	viper.SetEnvPrefix(EnvPrefix)
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	viper.AutomaticEnv()

	// Pipeline configuration
	viper.SetDefault("api.url", "http://127.0.0.1:9090")

	commands.AddCommands(rootCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)

		os.Exit(1)
	}
}
