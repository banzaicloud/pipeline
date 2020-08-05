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
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"emperror.dev/errors"
	"github.com/MakeNowJust/heredoc"
	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/prom2json"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type options struct {
	telemetryUrl string
	verify       bool
}

func NewTelemetryCommand() *cobra.Command {
	options := options{}
	cmd := &cobra.Command{
		Use:   "telemetry",
		Short: "Get telemetry information",
		Long: heredoc.Doc(`
			Get pipeline telemetry information
		`),
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			options.telemetryUrl = viper.GetString("telemetry.url")
			options.verify = viper.GetBool("telemetry.verify")

			cmd.SilenceErrors = true
			cmd.SilenceUsage = true

			status, err := getTelemetry(options)
			if err != nil {
				return err
			}

			metrics, err := json.Marshal(status)
			if err != nil {
				return errors.WrapIff(err, "unable to marshal metrics %+v", metrics)
			}

			fmt.Println(string(metrics))

			return nil
		},
	}

	return cmd
}

func getTelemetry(options options) ([]*prom2json.Family, error) {
	mfChan := make(chan *dto.MetricFamily, 1024)

	u, err := url.Parse(options.telemetryUrl)
	if err != nil {
		return nil, errors.WrapIf(err, "parsing url")
	}

	if u.Scheme == "file" {
		input, err := os.Open(filepath.Join(u.Host, u.Path))
		if err != nil {
			return nil, errors.WrapIf(err, "opening telemetry file")
		}
		go func() {
			if err := prom2json.ParseReader(input, mfChan); err != nil {
				log.Fatal("error reading metrics:", err)
			}
		}()
	} else {
		transport, err := makeTransport(options.verify)
		if err != nil {
			return nil, errors.WrapIf(err, "creating transport")
		}
		go func() {
			err := prom2json.FetchMetricFamilies(options.telemetryUrl, mfChan, transport)
			if err != nil {
				log.Fatalln(err)
			}
		}()
	}

	result := []*prom2json.Family{}
	for mf := range mfChan {
		result = append(result, prom2json.NewFamily(mf))
	}

	return result, nil
}

func makeTransport(skipServerCertCheck bool) (*http.Transport, error) {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.ResponseHeaderTimeout = time.Second
	tlsConfig := &tls.Config{
		InsecureSkipVerify: skipServerCertCheck,
	}
	transport.TLSClientConfig = tlsConfig
	return transport, nil
}
