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

package k8sclient

import (
	"context"
	"net"
	"time"

	"emperror.dev/emperror"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// NewClient creates a new Kubernetes client from config.
func NewClientFromConfig(config *rest.Config) (*kubernetes.Clientset, error) {
	if viper.GetBool("pipeline.forceGlobal") {
		config.Dial = blockerDial(config.Dial)
	}
	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, emperror.Wrap(err, "failed to create client for config")
	}

	return client, nil
}

// NewClientFromKubeConfig creates a new Kubernetes client from raw kube config.
func NewClientFromKubeConfig(kubeConfig []byte) (*kubernetes.Clientset, error) {
	config, err := NewClientConfig(kubeConfig)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to create client config")
	}
	return NewClientFromConfig(config)
}

// NewClientFromKubeConfig creates a new Kubernetes client from raw kube config.
func NewClientFromKubeConfigWithTimeout(kubeConfig []byte, timeout time.Duration) (*kubernetes.Clientset, error) {
	config, err := NewClientConfig(kubeConfig)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to create client config")
	}
	config.Timeout = timeout
	return NewClientFromConfig(config)
}

// NewInClusterClient returns a Kubernetes client based on in-cluster configuration.
func NewInClusterClient() (*kubernetes.Clientset, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, errors.Wrap(err, "failed to fetch in-cluster configuration")
	}

	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create client for config")
	}

	return client, nil
}

// globallyRoutable decides if an address is a globally routable IPv4 address
func globallyRoutable(ip net.IP) bool {
	if ip == nil || !ip.IsGlobalUnicast() {
		return false
	}
	for _, cidr := range []string{"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"} {
		_, net, _ := net.ParseCIDR(cidr)
		if net.Contains(ip) {
			return false
		}
	}
	return true
}

type dialer func(context.Context, string, string) (net.Conn, error)

// blockerDial wraps the default dialer but blocks connections to non-globally-routable addresses
func blockerDial(original dialer) dialer {
	return func(ctx context.Context, network, address string) (net.Conn, error) {
		if original == nil {
			dialer := &net.Dialer{}
			original = dialer.DialContext
		}
		if address == "tcp" {
			address = "tcp4" // TODO implement filtering for ipv6
		}
		conn, err := original(ctx, network, address)
		if err == nil {
			host, _, err := net.SplitHostPort(conn.RemoteAddr().String())
			if err != nil {
				conn.Close()
				return nil, emperror.Wrap(err, "failed to parse remote address")
			}
			ip := net.ParseIP(host)
			if ip == nil || !globallyRoutable(ip) {
				conn.Close()
				return nil, errors.New("remote address is not a global unicast address")
			}
		}

		return conn, err
	}
}
