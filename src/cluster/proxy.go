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

package cluster

import (
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	utilnet "k8s.io/apimachinery/pkg/util/net"
	"k8s.io/apimachinery/pkg/util/proxy"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/transport"

	"github.com/banzaicloud/pipeline/pkg/k8sclient"
)

const defaultProxyExpirationMinutes = 10

type KubeAPIProxy struct {
	Handler gin.HandlerFunc
}

type responder struct{}

func (r *responder) Error(w http.ResponseWriter, req *http.Request, err error) {
	log.Errorf("Error while proxying request: %v", err)
	http.Error(w, err.Error(), http.StatusInternalServerError)
}

// makeUpgradeTransport creates a transport that explicitly bypasses HTTP2 support
// for proxy connections that must upgrade.
func makeUpgradeTransport(config *rest.Config, keepalive time.Duration) (proxy.UpgradeRequestRoundTripper, error) {
	transportConfig, err := config.TransportConfig()
	if err != nil {
		return nil, err
	}
	tlsConfig, err := transport.TLSConfigFor(transportConfig)
	if err != nil {
		return nil, err
	}
	rt := utilnet.SetOldTransportDefaults(&http.Transport{
		TLSClientConfig: tlsConfig,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: keepalive,
		}).DialContext,
	})

	upgrader, err := transport.HTTPWrappersForConfig(transportConfig, proxy.MirrorRequest)
	if err != nil {
		return nil, err
	}
	return proxy.NewUpgradeRequestRoundTripper(rt, upgrader), nil
}

func defaultProxyTransport(requestSchema string, requestHost string, apiProxyPrefix string, internalTransport http.RoundTripper) *proxy.Transport {
	rewritingTransport := &proxy.Transport{
		Scheme:       requestSchema,
		Host:         requestHost,
		PathPrepend:  apiProxyPrefix,
		RoundTripper: internalTransport,
	}
	return rewritingTransport
}

// NewKubeAPIProxy creates a new Kubernetes API Server Proxy to the given cluster with a well-defined keep-alive timeout.
func NewKubeAPIProxy(requestSchema string, requestHost string, apiProxyPrefix string, cluster CommonCluster, keepalive time.Duration) (*KubeAPIProxy, error) {

	kubeConfig, err := cluster.GetK8sConfig()
	if err != nil {
		return nil, err
	}

	cfg, err := k8sclient.NewClientConfig(kubeConfig)
	if err != nil {
		return nil, err
	}

	host := cfg.Host
	if !strings.HasSuffix(host, "/") {
		host = host + "/"
	}
	target, err := url.Parse(host)
	if err != nil {
		return nil, err
	}

	responder := &responder{}
	transport, err := rest.TransportFor(cfg)
	if err != nil {
		return nil, err
	}
	upgradeTransport, err := makeUpgradeTransport(cfg, keepalive)
	if err != nil {
		return nil, err
	}
	proxyTransport := defaultProxyTransport(requestSchema, requestHost, apiProxyPrefix, transport)

	proxy := proxy.NewUpgradeAwareHandler(target, proxyTransport, false, false, responder)
	proxy.UpgradeTransport = upgradeTransport
	proxy.UseRequestLocation = true

	proxyServer := http.Handler(proxy)
	proxyServer = stripLeaveSlash(apiProxyPrefix, proxyServer)

	return &KubeAPIProxy{Handler: gin.WrapH(proxyServer)}, nil
}

// like http.StripPrefix, but always leaves an initial slash. (so that our
// regexps will work.)
func stripLeaveSlash(prefix string, h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		p := strings.TrimPrefix(req.URL.Path, prefix)
		if len(p) >= len(req.URL.Path) {
			http.NotFound(w, req)
			return
		}
		if len(p) > 0 && p[:1] != "/" {
			p = "/" + p
		}
		req.URL.Path = p
		h.ServeHTTP(w, req)
	})
}
