package ssh

// TODO this file has to be moved to the `secret` package

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"strings"

	"github.com/banzaicloud/pipeline/config"
	"github.com/banzaicloud/pipeline/model"
	secretTypes "github.com/banzaicloud/pipeline/pkg/secret"
	"github.com/banzaicloud/pipeline/secret"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
)

var log *logrus.Logger

// Simple init for logging
func init() {
	log = config.Logger()
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
func NewKey(s *secret.SecretItemResponse) *Key {
	return &Key{
		User:                 s.Values[secretTypes.User],
		Identifier:           s.Values[secretTypes.Identifier],
		PublicKeyData:        s.Values[secretTypes.PublicKeyData],
		PublicKeyFingerprint: s.Values[secretTypes.PublicKeyFingerprint],
		PrivateKeyData:       s.Values[secretTypes.PrivateKeyData],
	}
}

// KeyAdd for Generate and store SSH key
func KeyAdd(organizationId uint, clusterId uint) (string, error) {
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
	log.Info("Store SSH Key to Bank Vaults")
	var createSecretRequest secret.CreateSecretRequest
	createSecretRequest.Type = secretTypes.SSHSecretType
	createSecretRequest.Name = clusterName

	createSecretRequest.Values = map[string]string{
		secretTypes.User:                 key.User,
		secretTypes.Identifier:           key.Identifier,
		secretTypes.PublicKeyData:        key.PublicKeyData,
		secretTypes.PublicKeyFingerprint: key.PublicKeyFingerprint,
		secretTypes.PrivateKeyData:       key.PrivateKeyData,
	}

	secretID, err = secret.Store.Store(organizationID, &createSecretRequest)

	if err != nil {
		log.Errorf("Error during store: %s", err.Error())
		return "", err
	}

	log.Info("SSH Key stored.")
	return
}

// KeyGenerator for Generate new SSH Key
func KeyGenerator() (*Key, error) {
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
