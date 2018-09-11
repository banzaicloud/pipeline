package cluster

import (
	"fmt"

	"github.com/banzaicloud/pipeline/helm"
	secretTypes "github.com/banzaicloud/pipeline/pkg/secret"
	"github.com/banzaicloud/pipeline/secret"
	"k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
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
		var k8sSecret *v1.Secret
		create := true

		for i := 0; i < len(clusterSecretList.Items); i++ {
			if clusterSecretList.Items[i].Name == s.Name {
				k8sSecret = &clusterSecretList.Items[i]

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

		if k8sSecret == nil {
			k8sSecret = &v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      s.Name,
					Namespace: namespace,
				},
				StringData: map[string]string{},
			}
		}

		for k, v := range s.Values {
			k8sSecret.StringData[k] = v
		}

		if create {
			_, err = clusterClient.CoreV1().Secrets(namespace).Create(k8sSecret)
		} else {
			_, err = clusterClient.CoreV1().Secrets(namespace).Update(k8sSecret)
		}

		if err != nil {
			log.Errorf("Error during creating k8s secret: %s", err.Error())
			return nil, err
		}

		secretSources = append(secretSources, s.K8SSourceMeta())
	}

	return secretSources, nil
}

// InstallSecretWithVaultID installs a secret which determined by the vaultID to the given namespace
func InstallSecretWithVaultID(cc CommonCluster, secretID, namespace string) (*secretTypes.K8SSourceMeta, error) {
	k8sConfig, err := cc.GetK8sConfig()
	if err != nil {
		return nil, fmt.Errorf("error during getting config: %s", err.Error())
	}
	return InstallSecretWithVaultIDByK8SConfig(k8sConfig, cc.GetOrganizationId(), secretID, namespace)
}

// InstallSecretWithVaultIDByK8SConfig is the same as InstallSecretWithVaultID but use this if you already have a K8S config at hand.
func InstallSecretWithVaultIDByK8SConfig(k8sConfig []byte, orgID uint, secretID, namespace string) (*secretTypes.K8SSourceMeta, error) {

	clusterClient, err := helm.GetK8sConnection(k8sConfig)
	if err != nil {
		return nil, fmt.Errorf("error during getting config: %s", err.Error())
	}

	resolvedSecret, err := secret.Store.Get(orgID, secretID)
	if err != nil {
		return nil, fmt.Errorf("error during getting secrets with ID %s: %s", secretID, err.Error())
	}

	var secretSources secretTypes.K8SSourceMeta

	k8sSecret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      resolvedSecret.Name,
			Namespace: namespace,
		},
		StringData: map[string]string{},
	}
	for k, v := range resolvedSecret.Values {
		k8sSecret.StringData[k] = v
	}

	err = helm.CreateNamespaceIfNotExist(k8sConfig, namespace)
	if err != nil {
		log.Errorf("Error checking namespace: %s", err.Error())
		return nil, err
	}

	_, err = clusterClient.CoreV1().Secrets(namespace).Get(k8sSecret.Name, metav1.GetOptions{})
	create := false
	if apierrors.IsNotFound(err) {
		create = true
	} else if err != nil {
		log.Errorf("Error checking k8s secret: %s", err.Error())
		return nil, err
	}

	if create {
		_, err = clusterClient.CoreV1().Secrets(namespace).Create(k8sSecret)
	} else {
		_, err = clusterClient.CoreV1().Secrets(namespace).Update(k8sSecret)
	}
	if err != nil {
		return nil, fmt.Errorf("error during creating k8s secret: %s", err.Error())
	}

	secretSources = resolvedSecret.K8SSourceMeta()

	return &secretSources, nil
}
