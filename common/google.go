package common

import (
	"github.com/banzaicloud/pipeline/secret"
)

// ServiceAccount describes a GKE service account
type ServiceAccount struct {
	Type                   string `json:"type"`
	ProjectId              string `json:"project_id"`
	PrivateKeyId           string `json:"private_key_id"`
	PrivateKey             string `json:"private_key"`
	ClientEmail            string `json:"client_email"`
	ClientId               string `json:"client_id"`
	AuthUri                string `json:"auth_uri"`
	TokenUri               string `json:"token_uri"`
	AuthProviderX50CertUrl string `json:"auth_provider_x509_cert_url"`
	ClientX509CertUrl      string `json:"client_x509_cert_url"`
}

// NewGoogleServiceAccount creates a google service account object for authentication
func NewGoogleServiceAccount(s *secret.SecretsItemResponse) *ServiceAccount {
	return &ServiceAccount{
		Type:                   s.Values[secret.Type],
		ProjectId:              s.Values[secret.ProjectId],
		PrivateKeyId:           s.Values[secret.PrivateKeyId],
		PrivateKey:             s.Values[secret.PrivateKey],
		ClientEmail:            s.Values[secret.ClientEmail],
		ClientId:               s.Values[secret.ClientId],
		AuthUri:                s.Values[secret.AuthUri],
		TokenUri:               s.Values[secret.TokenUri],
		AuthProviderX50CertUrl: s.Values[secret.AuthX509Url],
		ClientX509CertUrl:      s.Values[secret.ClientX509Url],
	}
}
