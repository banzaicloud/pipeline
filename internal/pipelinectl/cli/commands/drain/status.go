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

package drain

import (
	"fmt"
	"net/http"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// NewStatusCommand creates a new cobra.Command for `pipelinectl drain status`.
func NewStatusCommand() *cobra.Command {
	options := drainOptions{}

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Get drain status",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			options.apiUrl = viper.GetString("api.url")

			cmd.SilenceErrors = true
			cmd.SilenceUsage = true

			return runStatus(options)
		},
	}

	return cmd
}

func runStatus(options drainOptions) error {
	req, err := newDrainRequest(options.apiUrl)
	if err != nil {
		return err
	}

	req.Method = http.MethodHead

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return errors.Wrap(err, "getting drain status failed")
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		fmt.Println("Drain is enabled.")

	case http.StatusNotFound:
		fmt.Println("Drain is disabled.")

	default:
		fmt.Println("Drain status is unknown.")
	}

	return nil
}
