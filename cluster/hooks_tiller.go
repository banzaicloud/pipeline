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

package cluster

import (
	"time"

	"github.com/Masterminds/semver"
	"github.com/pkg/errors"
	"github.com/spf13/viper"

	pkgHelm "github.com/banzaicloud/pipeline/pkg/helm"
)

// WaitingForTillerComeUp waits until till to come up
func WaitingForTillerComeUp(kubeConfig []byte) error {
	requiredHelmVersion, err := semver.NewVersion(viper.GetString("helm.tillerVersion"))
	if err != nil {
		return err
	}

	retryAttempts := viper.GetInt(pkgHelm.HELM_RETRY_ATTEMPT_CONFIG)
	retrySleepSeconds := viper.GetInt(pkgHelm.HELM_RETRY_SLEEP_SECONDS)

	for i := 0; i <= retryAttempts; i++ {
		log.Infof("Waiting for tiller to come up %d/%d", i, retryAttempts)

		client, err := pkgHelm.NewClient(kubeConfig, log)
		if err == nil {
			defer client.Close()

			resp, err := client.GetVersion()
			if err != nil {
				return err
			}

			if !semver.MustParse(resp.Version.SemVer).LessThan(requiredHelmVersion) {
				return nil
			}

			log.Warn("Tiller version is not up to date yet")
		} else {
			log.Warnf("Error during getting helm client: %s", err.Error())
		}

		time.Sleep(time.Duration(retrySleepSeconds) * time.Second)
	}

	return errors.New("Timeout during waiting for tiller to get ready")
}
