package k8sutil

import (
	"time"

	"github.com/goph/emperror"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/bootstrap/token/util"
	"k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm"
)

// GetOrCreateBootstrapToken
// This function will ensure to have at least 1 token that expire at least 1 hour from now
// If this token not exists it will create one and returns ClusterBootstrapInfo
func GetOrCreateBootstrapToken(log logrus.FieldLogger, client kubernetes.Interface) (string, error) {
	namespace := "kube-system"
	options := metav1.ListOptions{
		FieldSelector: "type=bootstrap.kubernetes.io/token",
	}
	secrets, err := client.CoreV1().Secrets(namespace).List(options)
	if err != nil {
		return "", emperror.Wrapf(err, "couldn't get boot-tokens from %s", namespace)
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
		emperror.Wrap(err, "generate bootstrap token failed")
	}
	tokenString, err := kubeadm.NewBootstrapTokenString(tokenValue)
	if err != nil {
		emperror.Wrap(err, "generate bootstrap token failed")
	}
	token := kubeadm.BootstrapToken{
		Token:       tokenString,
		TTL:         &metav1.Duration{Duration: time.Hour * 1},
		Description: "Pipeline Node bootstrap token",
	}
	_, err = client.CoreV1().Secrets(namespace).Create(token.ToSecret())
	if err != nil {
		emperror.Wrap(err, "unable to create token")
	}
	return token.Token.String(), nil
}
