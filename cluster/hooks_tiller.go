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
	"fmt"
	"time"

	"github.com/Masterminds/semver"
	"github.com/pkg/errors"
	"github.com/spf13/viper"

	"github.com/banzaicloud/pipeline/internal/backoff"
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

	var backoffConfig = backoff.ConstantBackoffConfig{
		Delay:      time.Duration(retrySleepSeconds) * time.Second,
		MaxRetries: retryAttempts,
	}
	var backoffPolicy = backoff.NewConstantBackoffPolicy(&backoffConfig)

	i := 0

	err = backoff.Retry(func() error {
		log.WithField("attempt", fmt.Sprintf("%d/%d", i, retryAttempts)).Info("waiting for tiller to come up")

		client, err := pkgHelm.NewClient(kubeConfig, log)
		if err != nil {
			log.Warnf("error during getting helm client: %s", err.Error())

			return err
		}
		defer client.Close()

		resp, err := client.GetVersion()
		if err != nil {
			return err
		}

		if semver.MustParse(resp.Version.SemVer).LessThan(requiredHelmVersion) {
			err := errors.New("tiller version is not up to date yet")
			log.Warn(err.Error())

			return err
		}

		return nil
	}, backoffPolicy)

	if err != nil {
		return errors.New("timeout during waiting for tiller to get ready")
	}

	return nil
}
