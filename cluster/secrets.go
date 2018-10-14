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
	"github.com/banzaicloud/pipeline/helm"
	intSecret "github.com/banzaicloud/pipeline/internal/secret"
	secretTypes "github.com/banzaicloud/pipeline/pkg/secret"
	"github.com/banzaicloud/pipeline/secret"
	"github.com/goph/emperror"
	"github.com/pkg/errors"
	"k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// InstallSecrets installs or updates secrets that matches the query under the name into namespace of a Kubernetes cluster.
// It returns the list of installed secret names and meta about how to mount them.
func InstallSecrets(cc CommonCluster, query *secretTypes.ListSecretsQuery, namespace string) ([]secretTypes.K8SSourceMeta, error) {

	k8sConfig, err := cc.GetK8sConfig()
	if err != nil {
		log.Errorf("Error during getting config: %s", err.Error())
		return nil, err
	}

	return InstallSecretsByK8SConfig(k8sConfig, cc.GetOrganizationId(), query, namespace)
}

// InstallSecretsByK8SConfig is the same as InstallSecrets but use this if you already have a K8S config at hand.
func InstallSecretsByK8SConfig(k8sConfig []byte, orgID uint, query *secretTypes.ListSecretsQuery, namespace string) ([]secretTypes.K8SSourceMeta, error) {

	// Values are always needed in this case
	query.Values = true

	clusterClient, err := helm.GetK8sConnection(k8sConfig)
	if err != nil {
		log.Errorf("Error during building k8s client: %s", err.Error())
		return nil, err
	}

	secrets, err := secret.Store.List(orgID, query)
	if err != nil {
		log.Errorf("Error during listing secrets: %s", err.Error())
		return nil, err
	}

	clusterSecretList, err := clusterClient.CoreV1().Secrets(namespace).List(metav1.ListOptions{})
	if err != nil {
		log.Errorf("Error during getting k8s secrets of the cluster: %s", err.Error())
		return nil, err
	}

	var secretSources []secretTypes.K8SSourceMeta

	for _, s := range secrets {
		k8sSecret := v1.Secret{
			StringData: make(map[string]string),
		}
		create := true

		for i := 0; i < len(clusterSecretList.Items); i++ {
			if clusterSecretList.Items[i].Name == s.Name {
				k8sSecret = clusterSecretList.Items[i]

				if k8sSecret.StringData == nil {
					k8sSecret.StringData = make(map[string]string)
				}

				create = false // update existing k8s secret

				break
			}
		}
		err := helm.CreateNamespaceIfNotExist(k8sConfig, namespace)
		if err != nil {
			log.Errorf("Error checking namespace: %s", err.Error())
			return nil, err
		}

		kubeSecretRequest := intSecret.KubeSecretRequest{
			Name:   s.Name,
			Type:   s.Type,
			Values: s.Values,
		}

		newK8sSecret, err := intSecret.CreateKubeSecret(kubeSecretRequest)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create k8s secret")
		}

		if create {
			newK8sSecret.ObjectMeta.Namespace = namespace

			_, err = clusterClient.CoreV1().Secrets(namespace).Create(&newK8sSecret)
		} else {
			k8sSecret.Data = nil // Clear data so that it is created from string data again
			k8sSecret.StringData = newK8sSecret.StringData

			_, err = clusterClient.CoreV1().Secrets(namespace).Update(&k8sSecret)
		}

		if err != nil {
			log.Errorf("Error during creating k8s secret: %s", err.Error())
			return nil, err
		}

		secretSources = append(secretSources, s.K8SSourceMeta())
	}

	return secretSources, nil
}

type InstallSecretRequest struct {
	SecretName string                                  `json:"name" yaml:"name"`
	Namespace  string                                  `json:"namespace" yaml:"namespace"`
	Spec       map[string]InstallSecretRequestSpecItem `json:"spec,omitempty" yaml:"spec,omitempty"`
}

type InstallSecretRequestSpecItem struct {
	Source    string            `json:"source,omitempty" yaml:"source,omitempty"`
	SourceMap map[string]string `json:"sourceMap,omitempty" yaml:"sourceMap,omitempty"`
}

// InstallSecret installs or updates a secret under the name into namespace of a Kubernetes cluster.
// It returns the installed secret name and meta about how to mount it.
func InstallSecret(cc CommonCluster, secretName string, req InstallSecretRequest) (*secretTypes.K8SSourceMeta, error) {
	k8sConfig, err := cc.GetK8sConfig()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get k8s config")
	}

	return InstallSecretByK8SConfig(k8sConfig, cc.GetOrganizationId(), secretName, req)
}

// InstallSecretByK8SConfig is the same as InstallSecrets but use this if you already have a K8S config at hand.
func InstallSecretByK8SConfig(k8sConfig []byte, orgID uint, secretName string, req InstallSecretRequest) (*secretTypes.K8SSourceMeta, error) {
	clusterClient, err := helm.GetK8sConnection(k8sConfig)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create k8s client")
	}

	secretItem, err := secret.Store.GetByName(orgID, req.SecretName)
	if err != nil {
		return nil, emperror.With(errors.Wrap(err, "failed to get secret"), "secret", secretName)
	}

	// Whether a new secret should be created
	var create bool

	kubeSecret, err := clusterClient.CoreV1().Secrets(req.Namespace).Get(secretName, metav1.GetOptions{})
	if err != nil && k8serrors.IsNotFound(err) {
		create = true
	} else if err != nil {
		return nil, errors.Wrap(err, "failed to get existing secret from cluster")
	}

	if err := helm.CreateNamespaceIfNotExist(k8sConfig, req.Namespace); err != nil {
		return nil, errors.Wrap(err, "failed to check existing namespace")
	}

	kubeSecretRequest := intSecret.KubeSecretRequest{
		Name:   secretItem.Name,
		Type:   secretItem.Type,
		Values: secretItem.Values,
		Spec:   intSecret.KubeSecretSpec{},
	}

	for key, spec := range req.Spec {
		kubeSecretRequest.Spec[key] = intSecret.KubeSecretSpecItem{
			Source:    spec.Source,
			SourceMap: spec.SourceMap,
		}
	}

	newKubeSecret, err := intSecret.CreateKubeSecret(kubeSecretRequest)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create k8s secret")
	}

	if create {
		newKubeSecret.ObjectMeta.Namespace = req.Namespace

		_, err = clusterClient.CoreV1().Secrets(req.Namespace).Create(&newKubeSecret)
	} else {
		kubeSecret.Data = nil // Clear data so that it is created from string data again
		kubeSecret.StringData = newKubeSecret.StringData

		_, err = clusterClient.CoreV1().Secrets(req.Namespace).Update(kubeSecret)
	}

	if err != nil {
		return nil, errors.Wrap(err, "failed to create or update secret")
	}

	sourceMeta := secretItem.K8SSourceMeta()

	return &sourceMeta, nil
}
