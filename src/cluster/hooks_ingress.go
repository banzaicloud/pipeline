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
	"crypto/x509/pkix"
	"encoding/base64"
	"fmt"
	"strings"

	"emperror.dev/emperror"
	"github.com/banzaicloud/bank-vaults/pkg/sdk/tls"
	"github.com/ghodss/yaml"
	"github.com/pkg/errors"

	"github.com/banzaicloud/pipeline/internal/global"
	"github.com/banzaicloud/pipeline/internal/global/ingresscert"
	"github.com/banzaicloud/pipeline/internal/secret/secrettype"
	pkgHelm "github.com/banzaicloud/pipeline/pkg/helm"
	"github.com/banzaicloud/pipeline/secret"
	"github.com/banzaicloud/pipeline/src/auth"
	"github.com/banzaicloud/pipeline/src/dns"
)

type ingressControllerValues struct {
	Traefik traefikValues `json:"traefik"`
}

type traefikValues struct {
	SSL sslTraefikValues `json:"ssl"`
}

type sslTraefikValues struct {
	Enabled        bool     `json:"enabled"`
	GenerateTLS    bool     `json:"generateTLS"`
	DefaultCN      string   `json:"defaultCN"`
	DefaultSANList []string `json:"defaultSANList"`
	DefaultCert    string   `json:"defaultCert"`
	DefaultKey     string   `json:"defaultKey"`
}

const DefaultCertSecretName = "default-ingress-cert"

// InstallIngressControllerPostHook post hooks can't return value, they can log error and/or update state?
func InstallIngressControllerPostHook(cluster CommonCluster) error {
	defaultCertSecret, err := secret.Store.GetByName(cluster.GetOrganizationId(), DefaultCertSecretName)
	if err == secret.ErrSecretNotExists {
		certGenerator := ingresscert.GetCertGenerator()

		orgID := cluster.GetOrganizationId()
		organization, err := auth.GetOrganizationById(orgID)
		if err != nil {
			return emperror.WrapWith(err, "failed to get organization", "organizationId", orgID)
		}

		baseDomain := strings.ToLower(global.Config.Cluster.DNS.BaseDomain)

		certRequest := tls.ServerCertificateRequest{
			Subject: pkix.Name{
				CommonName: "banzaicloud.io",
			},
		}

		if baseDomain != "" {
			orgDomainName := strings.ToLower(fmt.Sprintf("%s.%s", organization.Name, baseDomain))
			err = dns.ValidateSubdomain(orgDomainName)
			if err != nil {
				return emperror.Wrap(err, "invalid domain for TLS cert")
			}

			wildcardOrgDomainName := fmt.Sprintf("*.%s", orgDomainName)
			err = dns.ValidateWildcardSubdomain(wildcardOrgDomainName)
			if err != nil {
				return emperror.Wrap(err, "invalid wildcard domain for TLS cert")
			}

			certRequest = tls.ServerCertificateRequest{
				Subject: pkix.Name{
					CommonName: wildcardOrgDomainName,
				},
				DNSNames: []string{orgDomainName, wildcardOrgDomainName},
			}
		}

		rootCA, cert, key, err := certGenerator.GenerateServerCertificate(certRequest)
		if err != nil {
			return errors.Wrap(err, "failed to generate certificate")
		}

		defaultCertSecretRequest := &secret.CreateSecretRequest{
			Name: DefaultCertSecretName,
			Type: secrettype.TLSSecretType,
			Values: map[string]string{
				secrettype.CACert:     string(rootCA),
				secrettype.ServerCert: string(cert),
				secrettype.ServerKey:  string(key),
			},
			Tags: []string{
				secret.TagBanzaiReadonly,
			},
		}

		secretId, err := secret.Store.Store(cluster.GetOrganizationId(), defaultCertSecretRequest)
		if err != nil {
			return errors.Wrap(err, "failed to save generated certificate")
		}

		defaultCertSecret, err = secret.Store.Get(cluster.GetOrganizationId(), secretId)
		if err != nil {
			return errors.Wrap(err, "failed to load generated certificate")
		}
	} else if err != nil {
		return errors.Wrap(err, "failed to check default ingress cert existence")
	}

	ingressValues := ingressControllerValues{
		Traefik: traefikValues{
			SSL: sslTraefikValues{
				Enabled:     true,
				DefaultCert: base64.StdEncoding.EncodeToString([]byte(defaultCertSecret.Values[secrettype.ServerCert])),
				DefaultKey:  base64.StdEncoding.EncodeToString([]byte(defaultCertSecret.Values[secrettype.ServerKey])),
			},
		},
	}

	ingressValuesJson, err := yaml.Marshal(ingressValues)
	if err != nil {
		return emperror.Wrap(err, "converting ingress config to json failed")
	}

	namespace := global.Config.Cluster.Namespace

	return installDeployment(cluster, namespace, pkgHelm.BanzaiRepository+"/pipeline-cluster-ingress", "ingress", ingressValuesJson, "", false)
}
