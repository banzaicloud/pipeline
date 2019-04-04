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

package k8sutil

import (
	"time"

	"github.com/goph/emperror"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/cluster-bootstrap/token/util"
	"k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm"
)

const KubeSystemNamespace = "kube-system"
const TokenSecretTypeFieldSelector = "type=bootstrap.kubernetes.io/token"

// GetOrCreateBootstrapToken
// This function will ensure to have at least 1 token that expire at least 1 hour from now
// GetOrCreateBootstrapToken returns a token for joining the cluster, creating a new one if there isn't any with enough time until expiration
func GetOrCreateBootstrapToken(log logrus.FieldLogger, client kubernetes.Interface) (string, error) {
	namespace := KubeSystemNamespace
	options := metav1.ListOptions{
		FieldSelector: TokenSecretTypeFieldSelector,
	}
	secrets, err := client.CoreV1().Secrets(namespace).List(options)
	if err != nil {
		return "", emperror.WrapWith(err, "namespace", namespace)
	}
	for _, s := range secrets.Items {
		token, err := kubeadm.BootstrapTokenFromSecret(&s)
		if err != nil {
			return "", emperror.Wrap(err, "unable to parse token")
		}
		log.Debugf("Token found %s with expiration %s", token.Token, token.Expires)
		// Check expiration for token to be at least 10Minute available
		expiration := metav1.NewTime(token.Expires.Add(time.Minute * 10))
		now := metav1.Now()
		if now.Before(&expiration) {
			return token.Token.String(), nil
		}

	}
	tokenValue, err := util.GenerateBootstrapToken()
	if err != nil {
		emperror.Wrap(err, "bootstrap token generation failed")
	}
	tokenString, err := kubeadm.NewBootstrapTokenString(tokenValue)
	if err != nil {
		emperror.Wrap(err, "bootstrap token generation failed")
	}
	token := kubeadm.BootstrapToken{
		Token:       tokenString,
		TTL:         &metav1.Duration{Duration: time.Hour * 1},
		Description: "Pipeline Node bootstrap token",
		Usages:      []string{"authentication", "signing"},
		Groups:      []string{"system:bootstrappers:kubeadm:default-node-token"},
	}
	_, err = client.CoreV1().Secrets(namespace).Create(token.ToSecret())
	if err != nil {
		emperror.Wrap(err, "unable to create token")
	}
	return tokenString.String(), nil
}
