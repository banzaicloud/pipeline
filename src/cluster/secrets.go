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
	stderrors "errors"

	"emperror.dev/errors"
	v1 "k8s.io/api/core/v1"
	k8sapierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/banzaicloud/pipeline/internal/secret/kubesecret"
	"github.com/banzaicloud/pipeline/pkg/k8sclient"
	"github.com/banzaicloud/pipeline/pkg/k8sutil"
	"github.com/banzaicloud/pipeline/src/secret"
)

// InstallSecrets installs or updates secrets that matches the query under the name into namespace of a Kubernetes cluster.
// It returns the list of installed secret names and meta about how to mount them.
func InstallSecrets(cc CommonCluster, query *secret.ListSecretsQuery, namespace string) ([]string, error) {
	kubeConfig, err := cc.GetK8sConfig()
	if err != nil {
		log.Errorf("Error during getting config: %s", err.Error())
		return nil, err
	}

	return InstallSecretsByK8SConfig(kubeConfig, cc.GetOrganizationId(), query, namespace)
}

// InstallSecretsByK8SConfig is the same as InstallSecrets but use this if you already have a K8S config at hand.
func InstallSecretsByK8SConfig(kubeConfig []byte, orgID uint, query *secret.ListSecretsQuery, namespace string) ([]string, error) {
	// Values are always needed in this case
	query.Values = true

	clusterClient, err := k8sclient.NewClientFromKubeConfig(kubeConfig)
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

	var secretNames []string

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
		client, err := k8sclient.NewClientFromKubeConfig(kubeConfig)
		if err != nil {
			return nil, errors.WithMessage(err, "failed to create client for namespace creation")
		}

		err = k8sutil.EnsureNamespace(client, namespace)
		if err != nil {
			log.Errorf("Error checking namespace: %s", err.Error())
			return nil, err
		}

		kubeSecretRequest := kubesecret.KubeSecretRequest{
			Name:   s.Name,
			Type:   s.Type,
			Values: s.Values,
		}

		newK8sSecret, err := kubesecret.CreateKubeSecret(kubeSecretRequest)
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

		secretNames = append(secretNames, s.Name)
	}

	return secretNames, nil
}

type InstallSecretRequest struct {
	SourceSecretName string
	Namespace        string
	Spec             map[string]InstallSecretRequestSpecItem
	Update           bool
}

type InstallSecretRequestSpecItem struct {
	Source    string
	SourceMap map[string]string
	Value     string
}

var ErrSecretNotFound = stderrors.New("secret not found")
var ErrKubernetesSecretNotFound = stderrors.New("kubernetes secret not found")
var ErrKubernetesSecretAlreadyExists = stderrors.New("kubernetes secret already exists")

// InstallSecret installs a new secret under the name into namespace of a Kubernetes cluster.
// It returns the installed secret name and meta about how to mount it.
func InstallSecret(cc interface {
	GetK8sConfig() ([]byte, error)
	GetOrganizationId() uint
}, secretName string, req InstallSecretRequest) (string, error) {
	kubeConfig, err := cc.GetK8sConfig()
	if err != nil {
		return "", errors.Wrap(err, "failed to get k8s config")
	}

	return InstallSecretByK8SConfig(kubeConfig, cc.GetOrganizationId(), secretName, req)
}

// InstallSecretByK8SConfig is the same as InstallSecret but use this if you already have a K8S config at hand.
func InstallSecretByK8SConfig(kubeConfig []byte, orgID uint, secretName string, req InstallSecretRequest) (string, error) {
	clusterClient, err := k8sclient.NewClientFromKubeConfig(kubeConfig)
	if err != nil {
		return "", errors.Wrap(err, "failed to create kubernetes client")
	}

	kubeSecretRequest := kubesecret.KubeSecretRequest{
		Name:      secretName,
		Namespace: req.Namespace,
		Spec:      make(kubesecret.KubeSecretSpec, len(req.Spec)),
	}

	if req.SourceSecretName != "" {
		secretItem, err := secret.Store.GetByName(orgID, req.SourceSecretName)
		if err == secret.ErrSecretNotExists {
			return "", ErrSecretNotFound
		} else if err != nil {
			return "", errors.WithDetails(errors.Wrap(err, "failed to get secret"), "secret", req.SourceSecretName)
		}

		kubeSecretRequest.Type = secretItem.Type
		kubeSecretRequest.Values = secretItem.Values
	}

	for key, spec := range req.Spec {
		kubeSecretRequest.Spec[key] = kubesecret.KubeSecretSpecItem{
			Source:    spec.Source,
			SourceMap: spec.SourceMap,
			Value:     spec.Value,
		}
	}

	kubeSecret, err := kubesecret.CreateKubeSecret(kubeSecretRequest)
	if err != nil {
		return "", errors.WrapIf(err, "failed to create kubernetes secret")
	}

	if err := k8sutil.EnsureNamespace(clusterClient, req.Namespace); err != nil {
		return "", errors.WrapIf(err, "failed to ensure that namespace exists")
	}

	_, err = clusterClient.CoreV1().Secrets(req.Namespace).Create(&kubeSecret)
	if err != nil && k8sapierrors.IsAlreadyExists(err) {
		if req.Update {
			_, err = clusterClient.CoreV1().Secrets(req.Namespace).Update(&kubeSecret)
			return secretName, err
		}
		return "", ErrKubernetesSecretAlreadyExists
	} else if err != nil {
		return "", errors.WrapIf(err, "failed to create secret")
	}

	return secretName, nil
}

// MergeSecret merges a secret with an already existing one in a Kubernetes cluster.
// It returns the installed secret name and meta about how to mount it.
func MergeSecret(cc CommonCluster, secretName string, req InstallSecretRequest) (string, error) {
	kubeConfig, err := cc.GetK8sConfig()
	if err != nil {
		return "", errors.Wrap(err, "failed to get k8s config")
	}

	return MergeSecretByK8SConfig(kubeConfig, cc.GetOrganizationId(), secretName, req)
}

// MergeSecretByK8SConfig is the same as MergeSecret but use this if you already have a K8S config at hand.
func MergeSecretByK8SConfig(kubeConfig []byte, orgID uint, secretName string, req InstallSecretRequest) (string, error) {
	clusterClient, err := k8sclient.NewClientFromKubeConfig(kubeConfig)
	if err != nil {
		return "", errors.Wrap(err, "failed to create kubernetes client")
	}

	kubeSecretRequest := kubesecret.KubeSecretRequest{
		Name:      secretName,
		Namespace: req.Namespace,
		Spec:      make(kubesecret.KubeSecretSpec, len(req.Spec)),
	}

	if req.SourceSecretName != "" {
		secretItem, err := secret.Store.GetByName(orgID, req.SourceSecretName)
		if err == secret.ErrSecretNotExists {
			return "", ErrSecretNotFound
		} else if err != nil {
			return "", errors.WithDetails(errors.Wrap(err, "failed to get secret"), "secret", req.SourceSecretName)
		}

		kubeSecretRequest.Type = secretItem.Type
		kubeSecretRequest.Values = secretItem.Values
	}

	clusterSecret, err := clusterClient.CoreV1().Secrets(req.Namespace).Get(secretName, metav1.GetOptions{})
	if err != nil && k8sapierrors.IsNotFound(err) {
		return "", ErrKubernetesSecretNotFound
	} else if err != nil {
		return "", errors.WithDetails(errors.Wrap(err, "failed to get kubernetes secret"), "secret", secretName)
	}

	for key, spec := range req.Spec {
		kubeSecretRequest.Spec[key] = kubesecret.KubeSecretSpecItem{
			Source:    spec.Source,
			SourceMap: spec.SourceMap,
			Value:     spec.Value,
		}
	}

	kubeSecret, err := kubesecret.CreateKubeSecret(kubeSecretRequest)
	if err != nil {
		return "", errors.WrapIf(err, "failed to create kubernetes secret")
	}

	if clusterSecret.StringData == nil {
		clusterSecret.StringData = kubeSecret.StringData
	} else {
		for key, value := range kubeSecret.StringData {
			clusterSecret.StringData[key] = value
		}
	}

	_, err = clusterClient.CoreV1().Secrets(req.Namespace).Update(clusterSecret)
	if err != nil && k8sapierrors.IsNotFound(err) {
		return "", ErrKubernetesSecretNotFound
	} else if err != nil {
		return "", errors.WrapIf(err, "failed to update secret")
	}

	return secretName, nil
}
