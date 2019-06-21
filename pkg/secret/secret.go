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

package secret

import (
	"github.com/banzaicloud/pipeline/pkg/cluster"
)

// FieldMeta describes how a secret field should be validated
type FieldMeta struct {
	Name        string `json:"name"`
	Required    bool   `json:"required"`
	Opaque      bool   `json:"opaque,omitempty"`
	Description string `json:"description,omitempty"`
}

// Meta describes how a secret is built up and how it should be sourced
type Meta struct {
	Fields   []FieldMeta    `json:"fields"`
	Sourcing SourcingMethod `json:"sourcing"`
}

// Alibaba keys
const (
	AlibabaRegion          = "ALIBABA_REGION_ID"
	AlibabaAccessKeyId     = "ALIBABA_ACCESS_KEY_ID"
	AlibabaSecretAccessKey = "ALIBABA_ACCESS_KEY_SECRET"
)

// Amazon keys
const (
	AwsRegion          = "AWS_REGION"
	AwsAccessKeyId     = "AWS_ACCESS_KEY_ID"
	AwsSecretAccessKey = "AWS_SECRET_ACCESS_KEY"
)

// Azure keys
const (
	AzureClientID       = "AZURE_CLIENT_ID"
	AzureClientSecret   = "AZURE_CLIENT_SECRET"
	AzureTenantID       = "AZURE_TENANT_ID"
	AzureSubscriptionID = "AZURE_SUBSCRIPTION_ID"
)

// Google keys
const (
	Type          = "type"
	ProjectId     = "project_id"
	PrivateKeyId  = "private_key_id"
	PrivateKey    = "private_key"
	ClientEmail   = "client_email"
	ClientId      = "client_id"
	AuthUri       = "auth_uri"
	TokenUri      = "token_uri"
	AuthX509Url   = "auth_provider_x509_cert_url"
	ClientX509Url = "client_x509_cert_url"
)

// Oracle keys
const (
	OracleUserOCID          = "user_ocid"
	OracleTenancyOCID       = "tenancy_ocid"
	OracleAPIKey            = "api_key"
	OracleAPIKeyFingerprint = "api_key_fingerprint"
	OracleRegion            = "region"
	OracleCompartmentOCID   = "compartment_ocid"
)

// Kubernetes keys
const (
	K8SConfig = "K8Sconfig"
)

// Ssh keys
const (
	User                 = "user"
	Identifier           = "identifier"
	PublicKeyData        = "public_key_data"
	PublicKeyFingerprint = "public_key_fingerprint"
	PrivateKeyData       = "private_key_data"
)

// TLS keys
const (
	TLSHosts    = "hosts"
	TLSValidity = "validity"
	CACert      = "caCert"
	CAKey       = "caKey"
	ServerKey   = "serverKey"
	ServerCert  = "serverCert"
	ClientKey   = "clientKey"
	ClientCert  = "clientCert"
)

// Distribution keys
const (
	KubernetesCACert        = "kubernetesCaCert"
	KubernetesCASigningCert = "kubernetesCaSigningCert"
	KubernetesCAKey         = "kubernetesCaKey"

	EtcdCACert = "etcdCaCert"
	EtcdCAKey  = "etcdCaKey"

	FrontProxyCACert = "frontProxyCaCert"
	FrontProxyCAKey  = "frontProxyCaKey"

	SAPub = "saPub"
	SAKey = "saKey"

	EncryptionSecret = "enc"

	// some useful helpers
	KubernetesCACommonName           = "kubernetes-ca"
	EtcdCACommonName                 = "etcd-ca"
	KubernetesFrontProxyCACommonName = "kubernetes-front-proxy-ca"
)

// Fn keys
const (
	MasterToken = "master_token"
)

// Password keys
const (
	Username = "username"
	Password = "password"
)

// Htpasswd extra keys (+Password keys)
const (
	HtpasswdFile = "htpasswd"
)

// Internal usage
const (
	TagKubeConfig     = "KubeConfig"
	TagBanzaiHidden   = "banzai:hidden"
	TagBanzaiReadonly = "banzai:readonly"
)

// ForbiddenTags are not supported in secret creation
// nolint: gochecknoglobals
var ForbiddenTags = []string{
	TagKubeConfig,
}

const (
	// GenericSecret represents generic secret types, without schema
	GenericSecret = "generic"
	// AllSecrets represents generic secret types which selects all secrets
	AllSecrets = ""
	// SSHSecretType marks secrets as of type "ssh"
	SSHSecretType = "ssh"
	// TLSSecretType marks secrets as of type "tls"
	TLSSecretType = "tls"
	// DistributionSecretType marks secrets as of type "distribution"
	PKESecretType = "pkecert"
	// FnSecretType marks secrets as of type "fn"
	FnSecretType = "fn"
	// PasswordSecretType marks secrets as of type "password"
	PasswordSecretType = "password"
	// HtpasswdSecretType marks secrets as of type "htpasswd"
	HtpasswdSecretType = "htpasswd"
)

// DefaultRules key matching for types
// nolint: gochecknoglobals
var DefaultRules = map[string]Meta{
	cluster.Alibaba: {
		Fields: []FieldMeta{
			{Name: AlibabaRegion, Required: false, Description: "Alibaba Cloud region"},
			{Name: AlibabaAccessKeyId, Required: true, Description: "Your Alibaba Cloud access key id"},
			{Name: AlibabaSecretAccessKey, Required: true, Description: "Your Alibaba Cloud secret access key id"},
		},
		Sourcing: EnvVar,
	},
	cluster.Amazon: {
		Fields: []FieldMeta{
			{Name: AwsRegion, Required: false, Description: "Amazon Cloud region"},
			{Name: AwsAccessKeyId, Required: true, Description: "Your Amazon Cloud access key id"},
			{Name: AwsSecretAccessKey, Required: true, Description: "Your Amazon Cloud secret access key id"},
		},
		Sourcing: EnvVar,
	},
	cluster.Azure: {
		Fields: []FieldMeta{
			{Name: AzureClientID, Required: true, Description: "Your application client id"},
			{Name: AzureClientSecret, Required: true, Description: "Your client secret id"},
			{Name: AzureTenantID, Required: true, Description: "Your tenant id"},
			{Name: AzureSubscriptionID, Required: true, Description: "Your subscription id"},
		},
		Sourcing: EnvVar,
	},
	cluster.Google: {
		Fields: []FieldMeta{
			{Name: Type, Required: true, Description: "service_account"},
			{Name: ProjectId, Required: true, Description: "Google Could Project Id. Find more about, Google Cloud secret fields here: https://beta.banzaicloud.io/docs/cloud-provider-credentials/google/gke_auth_credentials/#method-2-command-line"},
			{Name: PrivateKeyId, Required: true, Description: "Id of you private key"},
			{Name: PrivateKey, Required: true, Description: "Your private key "},
			{Name: ClientEmail, Required: true, Description: "Google service account client email"},
			{Name: ClientId, Required: true, Description: "Client Id"},
			{Name: AuthUri, Required: true, Description: "OAuth2 authentatication IRU"},
			{Name: TokenUri, Required: true, Description: "OAuth2 token URI"},
			{Name: AuthX509Url, Required: true, Description: "OAuth2 provider ceritficate URL"},
			{Name: ClientX509Url, Required: true, Description: "OAuth2 client ceritficate URL"},
		},
		Sourcing: EnvVar,
	},
	cluster.Kubernetes: {
		Fields: []FieldMeta{
			{Name: K8SConfig, Required: true},
		},
		Sourcing: Volume,
	},
	cluster.Oracle: {
		Fields: []FieldMeta{
			{Name: OracleUserOCID, Required: true, Description: "Your Oracle user OCID. Find more about, generating public key and fingerprint here: https://beta.banzaicloud.io/docs/cloud-provider-credentials/oracle/oke_auth_credentials/#generate-api-token"},
			{Name: OracleTenancyOCID, Required: true, Description: "Your tenancy OCID"},
			{Name: OracleAPIKey, Required: true, Description: "Your public key"},
			{Name: OracleAPIKeyFingerprint, Required: true, Description: "Fingerprint of you public key"},
			{Name: OracleRegion, Required: true, Description: "Oracle region"},
			{Name: OracleCompartmentOCID, Required: true, Description: "Your compartment OCID"},
		},
	},
	SSHSecretType: {
		Fields: []FieldMeta{
			{Name: User, Required: true},
			{Name: Identifier, Required: true},
			{Name: PublicKeyData, Required: true},
			{Name: PublicKeyFingerprint, Required: true},
			{Name: PrivateKeyData, Required: true},
		},
		Sourcing: Volume,
	},
	TLSSecretType: {
		Fields: []FieldMeta{
			{Name: TLSHosts, Required: true},
			{Name: TLSValidity, Required: false},
			{Name: CACert, Required: false},
			{Name: CAKey, Required: false},
			{Name: ServerKey, Required: false},
			{Name: ServerCert, Required: false},
			{Name: ClientKey, Required: false},
			{Name: ClientCert, Required: false},
		},
		Sourcing: Volume,
	},
	PKESecretType: {
		Fields: []FieldMeta{
			{Name: CACert, Required: false},
			{Name: CAKey, Required: false},

			{Name: KubernetesCACert, Required: false},
			{Name: KubernetesCAKey, Required: false},

			{Name: EtcdCACert, Required: false},
			{Name: EtcdCAKey, Required: false},

			{Name: FrontProxyCACert, Required: false},
			{Name: FrontProxyCAKey, Required: false},

			{Name: SAPub, Required: false},
			{Name: SAKey, Required: false},
		},
		Sourcing: Volume,
	},
	GenericSecret: {
		Fields:   []FieldMeta{},
		Sourcing: EnvVar,
	},
	FnSecretType: {
		Fields: []FieldMeta{
			{Name: MasterToken, Required: true},
		},
		Sourcing: EnvVar,
	},
	PasswordSecretType: {
		Fields: []FieldMeta{
			{Name: Username, Required: true, Description: "Your username"},
			{Name: Password, Required: false, Description: "Your password"},
		},
		Sourcing: EnvVar,
	},
	HtpasswdSecretType: {
		Fields: []FieldMeta{
			{Name: Username, Required: true, Opaque: true, Description: "Your username"},
			{Name: Password, Required: false, Opaque: true, Description: "Your password"},
			{Name: HtpasswdFile, Required: false},
		},
		Sourcing: Volume,
	},
}

// ListSecretsQuery represent a secret listing filter
type ListSecretsQuery struct {
	Type   string   `form:"type" json:"type"`
	IDs    []string `form:"ids" json:"ids"`
	Tags   []string `form:"tags" json:"tags"`
	Values bool     `form:"values" json:"values"`
}

// InstallSecretsToClusterRequest describes an InstallSecretToCluster request
type InstallSecretsToClusterRequest struct {
	Namespace string           `json:"namespace" binding:"required"`
	Query     ListSecretsQuery `json:"query" binding:"required"`
}

// SourcingMethod describes how an installed Secret should be sourced into a Pod in K8S
type SourcingMethod string

const (
	// EnvVar means the secret has to be sources an an env var
	EnvVar SourcingMethod = "env"
	// Volume means the secret has to be mounted an a volume
	Volume SourcingMethod = "volume"
)

// K8SSourceMeta describes which and how installed Secret should be sourced into a Pod in K8S
type K8SSourceMeta struct {
	Name     string         `json:"name"`
	Sourcing SourcingMethod `json:"sourcing"`
}
