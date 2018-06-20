package constants

import (
	"github.com/banzaicloud/banzai-types/components"
	"github.com/banzaicloud/banzai-types/constants"
)

// SecretField describes how a secret field should be validated
type SecretField struct {
	Name     string `json:"name"`
	Required bool   `json:"required"`
}

// SecretMeta describes how a secret is built up and how it should be sourced
type SecretMeta struct {
	Fields   []SecretField                   `json:"fields"`
	Sourcing components.SecretSourcingMethod `json:"Sourcing"`
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
	// PasswordSecretType marks secrets as of type "password"
	PasswordSecretType = "password"
)

// DefaultRules key matching for types
var DefaultRules = map[string]SecretMeta{
	constants.Amazon: SecretMeta{
		Fields: []SecretField{
			{Name: AwsAccessKeyId, Required: true},
			{Name: AwsSecretAccessKey, Required: true},
		},
		Sourcing: components.EnvVar,
	},
	constants.Azure: SecretMeta{
		Fields: []SecretField{
			{Name: AzureClientId, Required: true},
			{Name: AzureClientSecret, Required: true},
			{Name: AzureTenantId, Required: true},
			{Name: AzureSubscriptionId, Required: true},
		},
		Sourcing: components.EnvVar,
	},
	constants.Google: SecretMeta{
		Fields: []SecretField{
			{Name: Type, Required: true},
			{Name: ProjectId, Required: true},
			{Name: PrivateKeyId, Required: true},
			{Name: PrivateKey, Required: true},
			{Name: ClientEmail, Required: true},
			{Name: ClientId, Required: true},
			{Name: AuthUri, Required: true},
			{Name: TokenUri, Required: true},
			{Name: AuthX509Url, Required: true},
			{Name: ClientX509Url, Required: true},
		},
		Sourcing: components.EnvVar,
	},
	constants.Kubernetes: SecretMeta{
		Fields: []SecretField{
			{Name: K8SConfig, Required: true},
		},
		Sourcing: components.Volume,
	},
	SSHSecretType: SecretMeta{
		Fields: []SecretField{
			{Name: User, Required: true},
			{Name: Identifier, Required: true},
			{Name: PublicKeyData, Required: true},
			{Name: PublicKeyFingerprint, Required: true},
			{Name: PrivateKeyData, Required: true},
		},
		Sourcing: components.Volume,
	},
	TLSSecretType: SecretMeta{
		Fields: []SecretField{
			{Name: TLSHosts, Required: true},
			{Name: TLSValidity, Required: false},
			{Name: CACert, Required: false},
			{Name: CAKey, Required: false},
			{Name: ServerKey, Required: false},
			{Name: ServerCert, Required: false},
			{Name: ClientKey, Required: false},
			{Name: ClientCert, Required: false},
		},
		Sourcing: components.Volume,
	},
	GenericSecret: SecretMeta{
		Fields:   []SecretField{},
		Sourcing: components.EnvVar,
	},
	FnSecretType: SecretMeta{
		Fields: []SecretField{
			{Name: MasterToken, Required: true},
		},
		Sourcing: components.EnvVar,
	},
	PasswordSecretType: SecretMeta{
		Fields: []SecretField{
			{Name: Username, Required: true},
			{Name: Password, Required: true},
		},
		Sourcing: components.EnvVar,
	},
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

// Password keys
const (
	Username = "username"
	Password = "password"
)

// Internal usage
const (
	TagKubeConfig = "KubeConfig"
)

// ForbiddenTags are not supported in secret creation
var ForbiddenTags = []string{
	TagKubeConfig,
}

// constants for posthooks
const (
	StoreKubeConfig                  = "StoreKubeConfig"
	PersistKubernetesKeys            = "PersistKubernetesKeys"
	UpdatePrometheusPostHook         = "UpdatePrometheusPostHook"
	InstallHelmPostHook              = "InstallHelmPostHook"
	InstallIngressControllerPostHook = "InstallIngressControllerPostHook"
	InstallClusterAutoscalerPostHook = "InstallClusterAutoscalerPostHook"
	InstallMonitoring                = "InstallMonitoring"
	InstallLogging                   = "InstallLogging"
	RegisterDomainPostHook           = "RegisterDomainPostHook"
)
