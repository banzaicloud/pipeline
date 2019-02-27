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

package commands

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type drainOptions struct {
	apiUrl string
}

// NewDrainCommand creates a new cobra.Command for `pipelinectl drain`.
func NewDrainCommand() *cobra.Command {
	options := drainOptions{}

	cmd := &cobra.Command{
		Use:   "drain",
		Short: "Turn on drain mode",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			options.apiUrl = viper.GetString("api.apiUrl")

			cmd.SilenceErrors = true
			cmd.SilenceUsage = true

			return runDrain(options)
		},
	}

	return cmd
}

func newDrainRequest(apiUrl string) (*http.Request, error) {
	u, err := url.Parse(apiUrl)
	if err != nil {
		return nil, errors.Errorf("invalid api url: %s", apiUrl)
	}

	u.Path = "/-/drain"

	req, err := http.NewRequest("", u.String(), nil)
	return req, errors.Wrap(err, "failed  to create HTTP request")
}

func runDrain(options drainOptions) error {
	req, err := newDrainRequest(options.apiUrl)
	if err != nil {
		return err
	}

	req.Method = http.MethodPost

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return errors.Wrap(err, "enabling drain mode failed")
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		fmt.Println("Drain mode is enabled.")

		return nil
	}

	return errors.New("enabling drain mode failed")
}

// NewUndrainCommand creates a new cobra.Command for `pipelinectl undrain`.
func NewUndrainCommand() *cobra.Command {
	options := drainOptions{}

	cmd := &cobra.Command{
		Use:   "undrain",
		Short: "Turn off drain mode",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			options.apiUrl = viper.GetString("api.apiUrl")

			cmd.SilenceErrors = true
			cmd.SilenceUsage = true

			return runUndrain(options)
		},
	}

	return cmd
}

func runUndrain(options drainOptions) error {
	req, err := newDrainRequest(options.apiUrl)
	if err != nil {
		return err
	}

	req.Method = http.MethodDelete

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return errors.Wrap(err, "disabling drain mode failed")
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		fmt.Println("Drain mode is disabled.")

		return nil
	}

	return errors.New("disabling drain mode failed")
}
