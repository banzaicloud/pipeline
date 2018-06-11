package ssh

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"strings"

	"github.com/banzaicloud/pipeline/config"
	"github.com/banzaicloud/pipeline/constants"
	"github.com/banzaicloud/pipeline/model"
	"github.com/banzaicloud/pipeline/secret"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
)

var logger *logrus.Logger

// Simple init for logging
func init() {
	logger = config.Logger()
}

//Key struct for store ssh key data
type Key struct {
	User                 string `json:"user,omitempty"`
	Identifier           string `json:"identifier,omitempty"`
	PublicKeyData        string `json:"publicKeyData,omitempty"`
	PublicKeyFingerprint string `json:"publicKeyFingerprint,omitempty"`
	PrivateKeyData       string `json:"PrivateKeyData,omitempty"`
}

// NewKey constructs a Ssh Key from the values stored
// in the given secret
func NewKey(s *secret.SecretsItemResponse) *Key {
	return &Key{
		User:                 s.Values[constants.User],
		Identifier:           s.Values[constants.Identifier],
		PublicKeyData:        s.Values[constants.PublicKeyData],
		PublicKeyFingerprint: s.Values[constants.PublicKeyFingerprint],
		PrivateKeyData:       s.Values[constants.PrivateKeyData],
	}
}

// KeyAdd for Generate and store SSH key
func KeyAdd(organizationId uint, clusterId uint) (string, error) {
	log := logger.WithFields(logrus.Fields{"tag": "KeyAdd"})
	log.Info("Generate and store SSH key ")

	sshKey, err := KeyGenerator()
	if err != nil {
		log.Errorf("KeyGenerator failed reason: %s", err.Error())
		return "", err
	}

	db := model.GetDB()
	cluster := model.ClusterModel{ID: clusterId}
	if err = db.First(&cluster).Error; err != nil {
		log.Errorf("Cluster with id=% not found: %s", cluster.ID, err.Error())
		return "", err
	}
	secretId, err := KeyStore(sshKey, organizationId, cluster.Name)
	if err != nil {
		log.Errorf("KeyStore failed reason: %s", err.Error())
		return "", err
	}
	return secretId, nil
}

// KeyStore for store SSH Key to Bank Vaults
func KeyStore(key *Key, organizationID uint, clusterName string) (secretID string, err error) {
	log := logger.WithFields(logrus.Fields{"tag": "KeyStore"})
	log.Info("Store SSH Key to Bank Vaults")
	var createSecretRequest secret.CreateSecretRequest
	createSecretRequest.Type = constants.SshSecretType
	createSecretRequest.Name = clusterName

	createSecretRequest.Values = map[string]string{
		constants.User:                 key.User,
		constants.Identifier:           key.Identifier,
		constants.PublicKeyData:        key.PublicKeyData,
		constants.PublicKeyFingerprint: key.PublicKeyFingerprint,
		constants.PrivateKeyData:       key.PrivateKeyData,
	}

	secretID, err = secret.Store.Store(fmt.Sprint(organizationID), &createSecretRequest)

	if err != nil {
		log.Errorf("Error during store: %s", err.Error())
		return "", err
	}

	log.Info("SSH Key stored.")
	return
}

// KeyGenerator for Generate new SSH Key
func KeyGenerator() (*Key, error) {
	log := logger.WithFields(logrus.Fields{"tag": "KeyGenerator"})
	log.Info("Generate new ssh key")

	key := new(Key)

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
