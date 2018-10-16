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

package helm

import (
	"fmt"
	"time"

	"github.com/banzaicloud/pipeline/pkg/k8sclient"
	"github.com/goph/emperror"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	k8sapierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/helm/pkg/helm"
	"k8s.io/helm/pkg/helm/portforwarder"
	"k8s.io/helm/pkg/kube"
)

const tillerPortForwardRetryLimit = 2

func NewClient(kubeConfig []byte, logger logrus.FieldLogger) (*helm.Client, error) {
	var tillerTunnel *kube.Tunnel

	for i := 0; i < tillerPortForwardRetryLimit; i++ {
		config, err := k8sclient.NewClientConfig(kubeConfig)
		if err != nil {
			return nil, errors.WithMessage(err, "failed to create client config for helm client")
		}

		client, err := k8sclient.NewClientFromConfig(config)
		if err != nil {
			return nil, errors.WithMessage(err, "failed to create kubernetes client for helm client")
		}

		logger.Debug("create kubernetes tunnel")
		tillerTunnel, err = portforwarder.New("kube-system", client, config)
		if err != nil && k8sapierrors.IsUnauthorized(err) && i == 0 {
			logger.Debug("create tunnel attempt %d/%d failed: %s", i+1, 2, err.Error())
			time.Sleep(time.Millisecond * 20)

			continue
		} else if err != nil {
			return nil, emperror.Wrap(err, "failed to create kubernetes tunnel")
		}

		break
	}

	tillerTunnelAddress := fmt.Sprintf("localhost:%d", tillerTunnel.Local)
	logger.WithField("address", tillerTunnelAddress).Debug("created kubernetes tunnel on address")

	hclient := helm.NewClient(helm.Host(tillerTunnelAddress))

	return hclient, nil
}
