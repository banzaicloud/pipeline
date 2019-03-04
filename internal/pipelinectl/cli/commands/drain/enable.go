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

// NewEnableCommand creates a new cobra.Command for `pipelinectl drain enable`.
func NewEnableCommand() *cobra.Command {
	options := drainOptions{}

	cmd := &cobra.Command{
		Use:   "enable",
		Short: "Enable drain",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			options.apiUrl = viper.GetString("api.url")

			cmd.SilenceErrors = true
			cmd.SilenceUsage = true

			return runEnable(options)
		},
	}

	return cmd
}

func runEnable(options drainOptions) error {
	req, err := newDrainRequest(options.apiUrl)
	if err != nil {
		return err
	}

	req.Method = http.MethodPost

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return errors.Wrap(err, "enabling drain failed")
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		fmt.Println("Drain is enabled.")

		return nil
	}

	return errors.New("enabling drain failed")
}
