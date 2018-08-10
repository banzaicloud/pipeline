package secret

import (
	"github.com/banzaicloud/pipeline/pkg/cluster"
	oracle "github.com/banzaicloud/pipeline/pkg/providers/oracle/secret"
)

// FieldMeta describes how a secret field should be validated
type FieldMeta struct {
	Name        string `json:"name"`
	Required    bool   `json:"required"`
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
	TagKubeConfig   = "KubeConfig"
	TagBanzaiHidden = "banzai:hidden"
)

// ForbiddenTags are not supported in secret creation
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
	// FnSecretType marks secrets as of type "fn"
	FnSecretType = "fn"
	// PasswordSecretType marks secrets as of type "password"
	PasswordSecretType = "password"
)

// DefaultRules key matching for types
var DefaultRules = map[string]Meta{
	cluster.Alibaba: {
		Fields: []FieldMeta{
			{Name: AlibabaRegion, Required: false},
			{Name: AlibabaAccessKeyId, Required: true},
			{Name: AlibabaSecretAccessKey, Required: true},
		},
		Sourcing: EnvVar,
	},
	cluster.Amazon: {
		Fields: []FieldMeta{
			{Name: AwsRegion, Required: false},
			{Name: AwsAccessKeyId, Required: true},
			{Name: AwsSecretAccessKey, Required: true},
		},
		Sourcing: EnvVar,
	},
	cluster.Azure: {
		Fields: []FieldMeta{
			{Name: AzureClientId, Required: true},
			{Name: AzureClientSecret, Required: true},
			{Name: AzureTenantId, Required: true},
			{Name: AzureSubscriptionId, Required: true},
		},
		Sourcing: EnvVar,
	},
	cluster.Google: {
		Fields: []FieldMeta{
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
			{Name: oracle.UserOCID, Required: true},
			{Name: oracle.TenancyOCID, Required: true},
			{Name: oracle.APIKey, Required: true},
			{Name: oracle.APIKeyFingerprint, Required: true},
			{Name: oracle.Region, Required: true},
			{Name: oracle.CompartmentOCID, Required: true},
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
			{Name: Username, Required: true},
			{Name: Password, Required: true},
		},
		Sourcing: EnvVar,
	},
}

// ListSecretsQuery represent a secret listing filter
type ListSecretsQuery struct {
	Type   string `form:"type" json:"type"`
	Tag    string `form:"tag" json:"tag"`
	Values bool   `form:"values" json:"values"`
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
