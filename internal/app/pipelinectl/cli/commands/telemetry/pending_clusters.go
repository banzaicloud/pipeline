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

package telemetry

import (
	"fmt"

	"github.com/MakeNowJust/heredoc"
	"github.com/prometheus/prom2json"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// NewTelemetryCommand creates a new cobra.Command for `pipelinectl status`.
func NewPendingClustersCommand() *cobra.Command {
	options := options{}
	cmd := &cobra.Command{
		Use:   "pending-clusters",
		Short: "Get the count of clusters in pending (creating, deleting, or updating) status",
		Long: heredoc.Doc(`
			Get the list of clusters in pending status
		`),
		Args: cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			options.telemetryUrl = viper.GetString("telemetry.url")
			options.verify = viper.GetBool("telemetry.verify")

			cmd.SilenceErrors = true
			cmd.SilenceUsage = true

			status, err := getTelemetry(options)
			if err != nil {
				return err
			}

			pending := sumPendingClusters(status)

			fmt.Println(pending)

			return nil
		},
	}

	return cmd
}

func sumPendingClusters(status []*prom2json.Family) int {
	pending := 0
	for _, family := range status {
		if family.Name == "pipeline_cluster_active_total" {
			for _, m := range family.Metrics {
				if m, ok := m.(prom2json.Metric); ok {
					if s, ok := m.Labels["status"]; ok {
						if s == "CREATING" || s == "DELETING" || s == "UPDATING" {
							pending++
						}
					}
				}
			}
		}
	}
	return pending
}
