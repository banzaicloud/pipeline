// Copyright © 2020 Banzai Cloud
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

package pkesecret

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"strings"

	"emperror.dev/errors"
	"github.com/banzaicloud/bank-vaults/pkg/sdk/vault"
	vaultapi "github.com/hashicorp/vault/api"

	"github.com/banzaicloud/pipeline/internal/common"
	"github.com/banzaicloud/pipeline/internal/secret/secrettype"
)

const (
	rsaKeySize             = 2048
	RSAPrivateKeyBlockType = "RSA PRIVATE KEY"
	PublicKeyBlockType     = "PUBLIC KEY"
)

func NewPkeSecreter(client *vault.Client, logger common.Logger) PkeSecreter {
	return PkeSecreter{
		client: client,
		logger: logger,
	}
}

type PkeSecreter struct {
	client *vault.Client

	logger common.Logger
}

func (s PkeSecreter) GeneratePkeSecret(organizationID uint, tags []string) (map[string]string, error) {
	clusterID := getClusterIDFromTags(tags)
	if clusterID == "" {
		return nil, errors.New("clusterID is missing from the tags")
	}

	mountInput := vaultapi.MountInput{
		Type:        "pki",
		Description: fmt.Sprintf("root PKI engine for cluster %s", clusterID),
		Config: vaultapi.MountConfigInput{
			MaxLeaseTTL:     "43801h",
			DefaultLeaseTTL: "43801h",
		},
	}

	// Mount a separate PKI engine for the cluster
	basePath := clusterPKIPath(organizationID, clusterID)
	path := fmt.Sprintf("%s/ca", basePath)

	err := s.client.RawClient().Sys().Mount(path, &mountInput)
	if err != nil {
		return nil, errors.Wrapf(err, "Error mounting pki engine for cluster %s", clusterID)
	}

	// Generate the root CA
	rootCAData := map[string]interface{}{
		"common_name": fmt.Sprintf("cluster-%s-ca", clusterID),
	}

	_, err = s.client.RawClient().Logical().Write(fmt.Sprintf("%s/root/generate/internal", path), rootCAData)
	if err != nil {
		// Unmount the pki engine first
		if err := s.client.RawClient().Sys().Unmount(path); err != nil {
			s.logger.Warn(fmt.Sprintf("failed to unmount secret path: %s", err.Error()), map[string]interface{}{
				"path": path,
			})
		}
		return nil, errors.Wrapf(err, "Error generating root CA for cluster %s", clusterID)
	}

	// Get root CA
	rootCA, err := s.client.RawClient().Logical().Read(fmt.Sprintf("%s/cert/ca", path))
	if err != nil {
		// Unmount the pki engine first
		if err := s.client.RawClient().Sys().Unmount(path); err != nil {
			s.logger.Warn(fmt.Sprintf("failed to unmount secret path: %s", err.Error()), map[string]interface{}{
				"path": path,
			})
		}

		return nil, errors.Wrapf(err, "Error reading root CA for cluster %s", clusterID)
	}

	ca := rootCA.Data["certificate"].(string)

	// Generate the intermediate CAs
	kubernetesCA, err := s.generateIntermediateCert(clusterID, basePath, secrettype.KubernetesCACommonName)
	if err != nil {
		// Unmount the pki backend first
		if err := s.client.RawClient().Sys().Unmount(path); err != nil {
			s.logger.Warn(fmt.Sprintf("failed to unmount secret path: %s", err.Error()), map[string]interface{}{
				"path": path,
			})
		}

		return nil, err
	}

	etcdCA, err := s.generateIntermediateCert(clusterID, basePath, secrettype.EtcdCACommonName)
	if err != nil {
		// Unmount the pki backend first
		if err := s.client.RawClient().Sys().Unmount(path); err != nil {
			s.logger.Warn(fmt.Sprintf("failed to unmount secret path: %s", err.Error()), map[string]interface{}{
				"path": path,
			})
		}

		return nil, err
	}

	frontProxyCA, err := s.generateIntermediateCert(clusterID, basePath, secrettype.KubernetesFrontProxyCACommonName)
	if err != nil {
		// Unmount the pki backend first
		if err := s.client.RawClient().Sys().Unmount(path); err != nil {
			s.logger.Warn(fmt.Sprintf("failed to unmount secret path: %s", err.Error()), map[string]interface{}{
				"path": path,
			})
		}

		return nil, err
	}

	// Service Account key-pair
	saPub, saPriv, err := generateSAKeyPair(clusterID)
	if err != nil {
		return nil, err
	}

	// Encryption Secret
	rnd := make([]byte, 32)
	_, err = rand.Read(rnd)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to encrypt secret")
	}
	encryptionSecret := base64.StdEncoding.EncodeToString(rnd)

	return map[string]string{
		secrettype.KubernetesCAKey:         kubernetesCA.Key,
		secrettype.KubernetesCACert:        kubernetesCA.Cert + "\n" + ca,
		secrettype.KubernetesCASigningCert: kubernetesCA.Cert,
		secrettype.EtcdCAKey:               etcdCA.Key,
		secrettype.EtcdCACert:              etcdCA.Cert + "\n" + ca,
		secrettype.FrontProxyCAKey:         frontProxyCA.Key,
		secrettype.FrontProxyCACert:        frontProxyCA.Cert + "\n" + ca,
		secrettype.SAPub:                   saPub,
		secrettype.SAKey:                   saPriv,
		secrettype.EncryptionSecret:        encryptionSecret,
	}, nil
}

func (s PkeSecreter) DeletePkeSecret(organizationID uint, tags []string) error {
	clusterID := getClusterIDFromTags(tags)
	basePath := clusterPKIPath(organizationID, clusterID)

	path := fmt.Sprintf("%s/ca", basePath)
	err := s.client.RawClient().Sys().Unmount(path)
	if err != nil {
		s.logger.Warn(fmt.Sprintf("failed to unmount secret path: %s", err.Error()), map[string]interface{}{
			"path": path,
		})
	}

	path = fmt.Sprintf("%s/%s", basePath, secrettype.KubernetesCACommonName)
	err = s.client.RawClient().Sys().Unmount(path)
	if err != nil {
		s.logger.Warn(fmt.Sprintf("failed to unmount secret path: %s", err.Error()), map[string]interface{}{
			"path": path,
		})
	}

	path = fmt.Sprintf("%s/%s", basePath, secrettype.EtcdCACommonName)
	err = s.client.RawClient().Sys().Unmount(path)
	if err != nil {
		s.logger.Warn(fmt.Sprintf("failed to unmount secret path: %s", err.Error()), map[string]interface{}{
			"path": path,
		})
	}

	path = fmt.Sprintf("%s/%s", basePath, secrettype.KubernetesFrontProxyCACommonName)
	err = s.client.RawClient().Sys().Unmount(path)
	if err != nil {
		s.logger.Warn(fmt.Sprintf("failed to unmount secret path: %s", err.Error()), map[string]interface{}{
			"path": path,
		})
	}

	return nil
}

func (s PkeSecreter) generateIntermediateCert(clusterID, basePath, commonName string) (*certificate, error) {
	mountInput := vaultapi.MountInput{
		Type:        "pki",
		Description: fmt.Sprintf("%s intermediate PKI engine for cluster %s", commonName, clusterID),
		Config: vaultapi.MountConfigInput{
			MaxLeaseTTL:     "43800h",
			DefaultLeaseTTL: "43800h",
		},
	}

	path := fmt.Sprintf("%s/%s", basePath, commonName)

	// Each intermediate and ca cert needs it's own pki mount, see:
	// https://github.com/hashicorp/vault/issues/1586#issuecomment-230300216
	err := s.client.RawClient().Sys().Mount(path, &mountInput)
	if err != nil {
		return nil, errors.Wrapf(err, "error mounting %s intermediate pki engine for cluster %s", commonName, clusterID)
	}

	caData := map[string]interface{}{
		"common_name": commonName,
	}

	caSecret, err := s.client.RawClient().Logical().Write(fmt.Sprintf("%s/intermediate/generate/exported", path), caData)
	if err != nil {
		// Unmount the pki backend first
		if err := s.client.RawClient().Sys().Unmount(path); err != nil {
			s.logger.Warn(fmt.Sprintf("failed to unmount secret path: %s", err.Error()), map[string]interface{}{
				"path": path,
			})
		}
		return nil, errors.Wrapf(err, "error generating %s intermediate cert for cluster %s", commonName, clusterID)
	}

	caSignData := map[string]interface{}{
		"csr":    caSecret.Data["csr"],
		"format": "pem_bundle",
	}

	caCertSecret, err := s.client.RawClient().Logical().Write(fmt.Sprintf("%s/ca/root/sign-intermediate", basePath), caSignData)
	if err != nil {
		// Unmount the pki backend first
		if err := s.client.RawClient().Sys().Unmount(path); err != nil {
			s.logger.Warn(fmt.Sprintf("failed to unmount secret path: %s", err.Error()), map[string]interface{}{
				"path": path,
			})
		}
		return nil, errors.Wrapf(err, "error signing %s intermediate cert for cluster %s", commonName, clusterID)
	}

	return &certificate{
		Key:  caSecret.Data["private_key"].(string),
		Cert: caCertSecret.Data["certificate"].(string),
	}, nil
}

type certificate struct {
	Cert string
	Key  string
}

const clusterIDTagName = "clusterID"

func getClusterIDFromTags(tags []string) string {
	for _, tag := range tags {
		if strings.HasPrefix(tag, clusterIDTagName+":") {
			return strings.TrimPrefix(tag, clusterIDTagName+":")
		}
	}

	// This should never happen
	return ""
}

func clusterPKIPath(organizationID uint, clusterID string) string {
	return fmt.Sprintf("clusters/%d/%s/pki", organizationID, clusterID)
}

func generateSAKeyPair(clusterID string) (pub, priv string, err error) {
	pk, err := rsa.GenerateKey(rand.Reader, rsaKeySize)
	if err != nil {
		return "", "", errors.Wrapf(err, "Error generating SA key pair for cluster %s, generate key failed", clusterID)
	}

	saPub, err := encodePublicKeyPEM(&pk.PublicKey)
	if err != nil {
		return "", "", errors.Wrapf(err, "Error generating SA key pair for cluster %s, encode public key failed", clusterID)
	}
	saPriv := encodePrivateKeyPEM(pk)

	return string(saPub), string(saPriv), nil
}

func encodePublicKeyPEM(key *rsa.PublicKey) ([]byte, error) {
	der, err := x509.MarshalPKIXPublicKey(key)
	if err != nil {
		return []byte{}, err
	}
	block := pem.Block{
		Type:  PublicKeyBlockType,
		Bytes: der,
	}
	return pem.EncodeToMemory(&block), nil
}

func encodePrivateKeyPEM(key *rsa.PrivateKey) []byte {
	block := pem.Block{
		Type:  RSAPrivateKeyBlockType,
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	}
	return pem.EncodeToMemory(&block)
}
