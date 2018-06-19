package constants

import (
	"github.com/banzaicloud/banzai-types/constants"
)

// SecretField describes how a secret field should be validated
type SecretField struct {
	Name     string `json:"name"`
	Required bool   `json:"required"`
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
	// FnSecretType marks secrets as of type "fn"
	FnSecretType = "fn"
)

// DefaultRules key matching for types
var DefaultRules = map[string][]SecretField{
	constants.Amazon: {
		SecretField{Name: AwsAccessKeyId, Required: true},
		SecretField{Name: AwsSecretAccessKey, Required: true},
	},
	constants.Azure: {
		SecretField{Name: AzureClientId, Required: true},
		SecretField{Name: AzureClientSecret, Required: true},
		SecretField{Name: AzureTenantId, Required: true},
		SecretField{Name: AzureSubscriptionId, Required: true},
	},
	constants.Google: {
		SecretField{Name: Type, Required: true},
		SecretField{Name: ProjectId, Required: true},
		SecretField{Name: PrivateKeyId, Required: true},
		SecretField{Name: PrivateKey, Required: true},
		SecretField{Name: ClientEmail, Required: true},
		SecretField{Name: ClientId, Required: true},
		SecretField{Name: AuthUri, Required: true},
		SecretField{Name: TokenUri, Required: true},
		SecretField{Name: AuthX509Url, Required: true},
		SecretField{Name: ClientX509Url, Required: true},
	},
	constants.Kubernetes: {
		SecretField{Name: K8SConfig, Required: true},
	},
	SSHSecretType: {
		SecretField{Name: User, Required: true},
		SecretField{Name: Identifier, Required: true},
		SecretField{Name: PublicKeyData, Required: true},
		SecretField{Name: PublicKeyFingerprint, Required: true},
		SecretField{Name: PrivateKeyData, Required: true},
	},
	TLSSecretType: {
		SecretField{Name: TLSHosts, Required: true},
		SecretField{Name: TLSValidity, Required: false},
		SecretField{Name: CACert, Required: false},
		SecretField{Name: CAKey, Required: false},
		SecretField{Name: ServerKey, Required: false},
		SecretField{Name: ServerCert, Required: false},
		SecretField{Name: ClientKey, Required: false},
		SecretField{Name: ClientCert, Required: false},
	},
	FnSecretType: {
		SecretField{Name: MasterToken, Required: true},
	},
	GenericSecret: {},
}

// Amazon keys
const (
	AwsAccessKeyId     = "AWS_ACCESS_KEY_ID"
	AwsSecretAccessKey = "AWS_SECRET_ACCESS_KEY"
)

// Azure keys
const (
	AzureClientId       = "AZURE_CLIENT_ID"
	AzureClientSecret   = "AZURE_CLIENT_SECRET"
	AzureTenantId       = "AZURE_TENANT_ID"
	AzureSubscriptionId = "AZURE_SUBSCRIPTION_ID"
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

// Fn keys
const (
	MasterToken = "master_token"
)

// Internal usage
const (
	TagKubeConfig = "KubeConfig"
)

// ForbiddenTags are not supported in secret creation
var ForbiddenTags = []string{
	TagKubeConfig,
}
