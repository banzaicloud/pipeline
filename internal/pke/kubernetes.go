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

package pke

import (
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
)

const (
	DefaultServiceCIDR = "10.32.0.0/24"
	DefaultPodCIDR     = "10.200.0.0/16"
	DefaultNetwork     = "weave"
)

type CRI struct {
	Runtime       string
	RuntimeConfig map[string]interface{}
}

type Kubernetes struct {
	Version string
	RBAC    bool
	Network Network
	CRI     CRI
	OIDC    OIDC
}

type OIDC struct {
	Enabled bool
}

// Network represents a K8s network
type Network struct {
	ServiceCIDR    string
	PodCIDR        string
	Provider       string
	ProviderConfig map[string]interface{}
}

// KubernetesPreparer implements Kubernetes preparation
type KubernetesPreparer struct {
	criPreparer     CRIPreparer
	logger          logrus.FieldLogger
	namespace       string
	networkPreparer NetworkPreparer
}

// MakeKubernetesPreparer returns an instance of KubernetesPreparer
func MakeKubernetesPreparer(logger logrus.FieldLogger, namespace string) KubernetesPreparer {
	namespace = strings.TrimSuffix(namespace, ".")

	return KubernetesPreparer{
		criPreparer:     MakeCRIPreparer(logger, namespace+".CRI"),
		logger:          logger,
		namespace:       namespace,
		networkPreparer: MakeNetworkPreparer(logger, namespace+".Network"),
	}
}

// Prepare validates and provides defaults for Kubernetes fields
func (p KubernetesPreparer) Prepare(k *Kubernetes) error {
	if k.Version == "" {
		// TODO check if we can provide good default
		p.logger.Errorf("%s.Version must be specified", p.namespace)
		return validationErrorf("K8s version must be specified")
	}
	if err := p.criPreparer.Prepare(&k.CRI); err != nil {
		return err
	}
	if err := p.networkPreparer.Prepare(&k.Network); err != nil {
		return err
	}
	return nil
}

// CRIPreparer implements CRI preparation
type CRIPreparer struct {
	logger    logrus.FieldLogger
	namespace string
}

// MakeCRIPreparer returns an instance of CRIPreparer
func MakeCRIPreparer(logger logrus.FieldLogger, namespace string) CRIPreparer {
	namespace = strings.TrimSuffix(namespace, ".")

	return CRIPreparer{
		logger:    logger,
		namespace: namespace,
	}
}

// Prepare validates and provides defaults for CRI fields
func (p CRIPreparer) Prepare(c *CRI) error {
	// TODO: implement CRI preparation
	return nil
}

// NetworkPreparer implements Network preparation
type NetworkPreparer struct {
	logger    logrus.FieldLogger
	namespace string
}

// MakeNetworkPreparer returns an instance of NetworkPreparer
func MakeNetworkPreparer(logger logrus.FieldLogger, namespace string) NetworkPreparer {
	namespace = strings.TrimSuffix(namespace, ".")

	return NetworkPreparer{
		logger:    logger,
		namespace: namespace,
	}
}

// Prepare validates and provides defaults for Network fields
func (p NetworkPreparer) Prepare(n *Network) error {
	if n.PodCIDR == "" {
		n.PodCIDR = DefaultPodCIDR
		p.logger.Debugf("%s.PodCIDR not specified, defaulting to [%s]", p.namespace, n.PodCIDR)
	}
	if n.ServiceCIDR == "" {
		n.ServiceCIDR = DefaultServiceCIDR
		p.logger.Debugf("%s.ServiceCIDR not specified, defaulting to [%s]", p.namespace, n.ServiceCIDR)
	}
	if n.Provider == "" {
		n.Provider = DefaultNetwork
		p.logger.Debugf("%s.Provider not specified, defaulting to [%s]", p.namespace, n.Provider)
	}
	// TODO: ProviderConfig defaults
	return nil
}

type validationError struct {
	msg string
}

func validationErrorf(msg string, args ...interface{}) validationError {
	if len(args) > 0 {
		msg = fmt.Sprintf(msg, args...)
	}
	return validationError{
		msg: msg,
	}
}

func (e validationError) Error() string {
	return e.msg
}

func (e validationError) InputValidationError() bool {
	return true
}
