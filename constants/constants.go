package constants

import "github.com/banzaicloud/banzai-types/constants"

const (
	// GenericSecret represents generic secret types, without schema
	GenericSecret = "generic"
	// AllSecrets represents generic secret types which selects all secrets
	AllSecrets = ""
	// SshSecretType marks secrets as of type "ssh"
	SshSecretType = "ssh"
)

// DefaultRules key matching for types
var DefaultRules = map[string][]string{
	constants.Amazon: {
		AwsAccessKeyId,
		AwsSecretAccessKey,
	},
	constants.Azure: {
		AzureClientId,
		AzureClientSecret,
		AzureTenantId,
		AzureSubscriptionId,
	},
	constants.Google: {
		Type,
		ProjectId,
		PrivateKeyId,
		PrivateKey,
		ClientEmail,
		ClientId,
		AuthUri,
		TokenUri,
		AuthX509Url,
		ClientX509Url,
	},
	constants.Kubernetes: {
		K8SConfig,
	},
	SshSecretType: {
		User,
		Identifier,
		PublicKeyData,
		PublicKeyFingerprint,
		PrivateKeyData,
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
)
