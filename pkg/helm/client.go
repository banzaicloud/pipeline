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
	"crypto/sha1"
	"encoding/hex"
	"fmt"

	"github.com/banzaicloud/pipeline/pkg/k8sclient"
	"github.com/goph/emperror"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"k8s.io/helm/pkg/helm"
	"k8s.io/helm/pkg/helm/portforwarder"
	"k8s.io/helm/pkg/kube"
	"k8s.io/helm/pkg/proto/hapi/chart"
	rls "k8s.io/helm/pkg/proto/hapi/services"
)

// Client encapsulates a Helm Client and a Tunnel for that client to interact with the Tiller pod
type Client struct {
	*kube.Tunnel
	*helm.Client

	clusterKey string
	cache      releaseCache
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

	h := sha1.New()
	h.Write(kubeConfig)
	clusterKey := hex.EncodeToString(h.Sum(nil))

	c := &Client{
		Tunnel: tillerTunnel,
		Client: hClient,

		clusterKey: clusterKey,
		cache:      newGoCacheReleaseCache(logger),
	}

	return c, nil
}

// ListReleases lists the current releases.
func (c *Client) ListReleases(opts ...helm.ReleaseListOption) (*rls.ListReleasesResponse, error) {
	releases, ok := c.cache.GetReleases(c.clusterKey)
	if ok {
		return releases, nil
	}

	releases, err := c.Client.ListReleases(opts...)
	if err != nil {
		return nil, err
	}

	c.cache.SaveReleases(c.clusterKey, releases)

	return releases, nil
}

func (c *Client) InstallRelease(chstr, ns string, opts ...helm.InstallOption) (*rls.InstallReleaseResponse, error) {
	c.cache.ClearReleases(c.clusterKey)

	return c.Client.InstallRelease(chstr, ns, opts...)
}

func (c *Client) InstallReleaseFromChart(chart *chart.Chart, ns string, opts ...helm.InstallOption) (*rls.InstallReleaseResponse, error) {
	c.cache.ClearReleases(c.clusterKey)

	return c.Client.InstallReleaseFromChart(chart, ns, opts...)
}

func (c *Client) DeleteRelease(rlsName string, opts ...helm.DeleteOption) (*rls.UninstallReleaseResponse, error) {
	c.cache.ClearReleases(c.clusterKey)

	return c.Client.DeleteRelease(rlsName, opts...)
}

func (c *Client) UpdateRelease(rlsName string, chstr string, opts ...helm.UpdateOption) (*rls.UpdateReleaseResponse, error) {
	c.cache.ClearReleases(c.clusterKey)

	return c.Client.UpdateRelease(rlsName, chstr, opts...)
}

func (c *Client) UpdateReleaseFromChart(rlsName string, chart *chart.Chart, opts ...helm.UpdateOption) (*rls.UpdateReleaseResponse, error) {
	c.cache.ClearReleases(c.clusterKey)

	return c.Client.UpdateReleaseFromChart(rlsName, chart, opts...)
}

func (c *Client) RollbackRelease(rlsName string, opts ...helm.RollbackOption) (*rls.RollbackReleaseResponse, error) {
	c.cache.ClearReleases(c.clusterKey)

	return c.Client.RollbackRelease(rlsName, opts...)
}
