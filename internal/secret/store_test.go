package secret_test

import "github.com/banzaicloud/pipeline/secret"

type storeMock struct {
	secret *secret.SecretItemResponse
	err    error
}

func (m *storeMock) Get(organizationID uint, secretID string) (*secret.SecretItemResponse, error) {
	return m.secret, m.err
}
