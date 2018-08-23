package secret_test

import (
	"reflect"
	"testing"

	pkgSecret "github.com/banzaicloud/pipeline/pkg/secret"
	"github.com/banzaicloud/pipeline/secret"
)

const (
	orgID = 19
)

var version = 1

func TestBlockingTags(t *testing.T) {

	cases := []struct {
		name    string
		request *secret.CreateSecretRequest
	}{
		{name: "readonly", request: &requestReadOnly},
		{name: "forbidden", request: &requestForbidden},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			secretID, err := secret.RestrictedStore.Store(orgID, tc.request)

			if err != nil {
				t.Errorf("error during storing readonly secret: %s", err.Error())
				t.FailNow()
			}

			err = secret.RestrictedStore.Delete(orgID, secretID)
			if err == nil {
				t.Error("readonly secret deleted..")
				t.FailNow()

				tc.request.Tags = append(tc.request.Tags, "newtag")

				err = secret.RestrictedStore.Update(orgID, secretID, tc.request)
				if err == nil {
					t.Error("readonly secret updated..")
					t.FailNow()
				}

				expErr := secret.ReadOnlyError{SecretID: secretID}
				if !reflect.DeepEqual(err, expErr) {
					t.Errorf("expected error: %s, got: %s", expErr, err.Error())
					t.FailNow()
				}
			}
		})
	}

}

var (
	requestReadOnly = secret.CreateSecretRequest{
		Name: "readonly",
		Type: pkgSecret.Password,
		Values: map[string]string{
			"key": "value",
		},
		Tags: []string{
			pkgSecret.TagBanzaiReadonly,
		},
		Version:   &version,
		UpdatedBy: "banzaiuser",
	}

	requestForbidden = secret.CreateSecretRequest{
		Name: "forbidden",
		Type: pkgSecret.Password,
		Values: map[string]string{
			"key": "value",
		},
		Tags:      pkgSecret.ForbiddenTags,
		Version:   &version,
		UpdatedBy: "banzaiuser",
	}
)
