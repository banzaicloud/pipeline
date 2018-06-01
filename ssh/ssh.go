package ssh

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"github.com/banzaicloud/pipeline/config"
	"github.com/banzaicloud/pipeline/model"
	"github.com/banzaicloud/pipeline/secret"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
	"strings"
)

var logger *logrus.Logger
var log *logrus.Entry

// Simple init for logging
func init() {
	logger = config.Logger()
}

type Key struct {
	User                 string `json:"user,omitempty"`
	Identifier           string `json:"identifier,omitempty"`
	PublicKeyData        string `json:"publicKeyData,omitempty"`
	PublicKeyFingerprint string `json:"publicKeyFingerprint,omitempty"`
	PrivateKeyData       string `json:"PrivateKeyData,omitempty"`
}

// Get SSH Key from Bank Vaults
func KeyGet(organizationID string, ClusterID uint) (sshKey Key, err error) {
	log := logger.WithFields(logrus.Fields{"tag": "KeyGet"})
	log.Info("Get SSH key")
	if organizationID == "" || ClusterID == 0 {
		log.Debugf("KeyGet organizationID: %q ClusterID: %q", organizationID, ClusterID)
		return sshKey, fmt.Errorf("parameter missing")
	}
	db := model.GetDB()
	awsProperties := &model.AmazonClusterModel{ClusterModelId: ClusterID}
	if err := db.First(&awsProperties).Error; err != nil {
		log.Errorf("Get ssh key failed reason: %s", err.Error())
		return sshKey, err
	}

	vaultContent, err := secret.Store.Get(organizationID, awsProperties.SshSecretID)
	if err != nil {
		log.Debugf("organizationID: %q, SshSecretID: %q", organizationID, awsProperties.SshSecretID)
		log.Errorf("Get ssh key failed reason: %s", err.Error())
		return sshKey, err
	}
	sshKey.User = vaultContent.Values[secret.User]
	sshKey.Identifier = vaultContent.Values[secret.Identifier]
	sshKey.PublicKeyData = vaultContent.Values[secret.PublicKeyData]
	sshKey.PublicKeyFingerprint = vaultContent.Values[secret.PublicKeyFingerprint]
	sshKey.PrivateKeyData = vaultContent.Values[secret.PrivateKeyData]
	log.Debug("Get SSH Key Done.")
	return sshKey, nil
}

// KeyAdd for Generate and store SSH key
func KeyAdd(organizationID uint, clusterID uint) (secretID string, sshKey Key, err error) {
	log := logger.WithFields(logrus.Fields{"tag": "KeyAdd"})
	log.Info("Generate and store SSH key ")

	sshKey, err = KeyGenerator()
	if err != nil {
		log.Errorf("KeyGenerator failed reason: %s", err.Error())
		return secretID, sshKey, err
	}

	db := model.GetDB()
	var clusterName model.ClusterModel
	db.Model(&model.ClusterModel{ID: clusterID}).First(&clusterName)

	secretID, err = KeyStore(sshKey, organizationID, clusterName.Name)
	if err != nil {
		log.Errorf("KeyStore failed reason: %s", err.Error())
		return secretID, sshKey, err
	}

	//if err := db.Model(&model.AmazonClusterModel{ClusterModelId: clusterID}).Update(&model.AmazonClusterModel{SshSecretID: secretID}).Error; err != nil {
	//	log.Errorf("Set ssh key failed reason: %s", err.Error())
	//	return "", sshKey, err
	//}
	return secretID, sshKey, err
}

// KeyStore for store SSH Key to Bank Vaults
func KeyStore(key Key, organizationID uint, clusterName string) (secretID string, err error) {
	log := logger.WithFields(logrus.Fields{"tag": "KeyStore"})
	log.Info("Store SSH Key to Bank Vaults")
	secretID = secret.GenerateSecretID()
	var createSecretRequest secret.CreateSecretRequest
	createSecretRequest.Type = "ssh"
	createSecretRequest.Name = clusterName

	createSecretRequest.Values = map[string]string{
		secret.User:                 key.User,
		secret.Identifier:           key.Identifier,
		secret.PublicKeyData:        key.PublicKeyData,
		secret.PublicKeyFingerprint: key.PublicKeyFingerprint,
		secret.PrivateKeyData:       key.PrivateKeyData,
	}

	if err := secret.Store.Store(fmt.Sprint(organizationID), secretID, &createSecretRequest); err != nil {
		log.Errorf("Error during store: %s", err.Error())
		return "", err
	}

	log.Info("Store SSH Key stored.")
	return secretID, nil
}

// KeyGenerator for Generate new SSH Key
func KeyGenerator() (Key, error) {
	log := logger.WithFields(logrus.Fields{"tag": "KeyGenerator"})
	log.Info("Generate new ssh key")

	var key Key

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		log.Errorf("PrivateKey generator failed reason: %s", err.Error())
		return key, err
	}

	privateKeyPEM := &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)}
	keyBuff := new(bytes.Buffer)
	if err := pem.Encode(keyBuff, privateKeyPEM); err != nil {
		log.Errorf("PrivateKey generator failed reason: %s", err.Error())
		return key, err
	}
	key.PrivateKeyData = keyBuff.String()
	log.Debug("Private key generated.")

	pub, err := ssh.NewPublicKey(&privateKey.PublicKey)
	if err != nil {
		log.Errorf("PublicKey generator failed reason: %s", err.Error())
		return key, err
	}
	log.Debug("Public key generated.")

	key.PublicKeyData = fmt.Sprintf("%s %s \n", strings.TrimSuffix(string(ssh.MarshalAuthorizedKey(pub)), "\n"), "no-reply@banzaicloud.com")

	key.PublicKeyFingerprint = ssh.FingerprintSHA256(pub)
	log.Info("SSH key generated.")

	return key, nil
}
