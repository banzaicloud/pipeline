package cluster

import (
	"github.com/banzaicloud/pipeline/helm"
	"github.com/banzaicloud/pipeline/secret"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// InstallSecrets installs all secrets thats matches the query under the name into namespace of a Kubernetes cluster.
func InstallSecrets(cc CommonCluster, query *secret.ListSecretsQuery, name, namespace string) error {

	config, err := cc.GetK8sConfig()
	if err != nil {
		log.Errorf("Error during getting config: %s", err.Error())
		return err
	}

	clusterClient, err := helm.GetK8sConnection(config)
	if err != nil {
		log.Errorf("Error during building k8s client: %s", err.Error())
		return err
	}

	items, err := secret.Store.List(cc.GetOrganizationId(), query)
	if err != nil {
		log.Errorf("Error during listing secrets: %s", err.Error())
		return err
	}

	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		StringData: map[string]string{},
	}
	for _, item := range items {
		for k, v := range item.Values {
			secret.StringData[k] = v
		}
	}
	_, err = clusterClient.CoreV1().Secrets(namespace).Create(secret)
	if err != nil {
		log.Errorf("Error during creating k8s secret: %s", err.Error())
		return err
	}

	return nil
}
