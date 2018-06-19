package cluster

import (
	"github.com/banzaicloud/banzai-types/components"
	"github.com/banzaicloud/pipeline/helm"
	"github.com/banzaicloud/pipeline/secret"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// InstallSecrets installs all secrets thats matches the query under the name into namespace of a Kubernetes cluster.
// It returns the list of installed secret names and meta about how to mount them.
func InstallSecrets(cc CommonCluster, query *components.ListSecretsQuery, namespace string) ([]components.SecretK8SSourceMeta, error) {

	k8sConfig, err := cc.GetK8sConfig()
	if err != nil {
		log.Errorf("Error during getting config: %s", err.Error())
		return nil, err
	}

	return InstallSecretsByK8SConfig(k8sConfig, cc.GetOrganizationId(), query, namespace)
}

// InstallSecretsByK8SConfig is the same as InstallSecrets but use this if you already have a K8S config at hand.
func InstallSecretsByK8SConfig(k8sConfig []byte, organizationID uint, query *components.ListSecretsQuery, namespace string) ([]components.SecretK8SSourceMeta, error) {

	// Values are always needed in this case
	query.Values = true

	clusterClient, err := helm.GetK8sConnection(k8sConfig)
	if err != nil {
		log.Errorf("Error during building k8s client: %s", err.Error())
		return nil, err
	}

	secrets, err := secret.Store.List(organizationID, query)
	if err != nil {
		log.Errorf("Error during listing secrets: %s", err.Error())
		return nil, err
	}

	var secretSources []components.SecretK8SSourceMeta

	for _, secret := range secrets {
		k8sSecret := &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      secret.Name,
				Namespace: namespace,
			},
			StringData: map[string]string{},
		}
		for k, v := range secret.Values {
			k8sSecret.StringData[k] = v
		}

		_, err = clusterClient.CoreV1().Secrets(namespace).Create(k8sSecret)
		if err != nil {
			log.Errorf("Error during creating k8s secret: %s", err.Error())
			return nil, err
		}

		secretSources = append(secretSources, secret.K8SSourceMeta())
	}

	return secretSources, nil
}
