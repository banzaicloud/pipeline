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

	"emperror.dev/emperror"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"k8s.io/helm/pkg/helm"
	"k8s.io/helm/pkg/helm/portforwarder"
	"k8s.io/helm/pkg/kube"

	"github.com/banzaicloud/pipeline/pkg/k8sclient"
)

// Client encapsulates a Helm Client and a Tunnel for that client to interact with the Tiller pod
type Client struct {
	*kube.Tunnel
	*helm.Client
}

func NewClient(kubeConfig []byte, logger logrus.FieldLogger) (*Client, error) {
	config, err := k8sclient.NewClientConfig(kubeConfig)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to create kubernetes client config for helm client")
	}

	client, err := k8sclient.NewClientFromConfig(config)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to create kubernetes client for helm client")
	}

	logger.Debug("create kubernetes tunnel")
	tillerTunnel, err := portforwarder.New("kube-system", client, config)
	if err != nil {
		return nil, emperror.Wrap(err, "failed to create kubernetes tunnel")
	}

	tillerTunnelAddress := fmt.Sprintf("localhost:%d", tillerTunnel.Local)
	logger.WithField("address", tillerTunnelAddress).Debug("created kubernetes tunnel on address")

	hClient := helm.NewClient(helm.Host(tillerTunnelAddress))

	return &Client{Tunnel: tillerTunnel, Client: hClient}, nil
}
