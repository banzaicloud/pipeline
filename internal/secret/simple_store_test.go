package secret_test

import (
	"testing"

	. "github.com/banzaicloud/pipeline/internal/secret"
	"github.com/banzaicloud/pipeline/secret"
)

func TestSimpleStore_Get(t *testing.T) {
	s := &secret.SecretItemResponse{
		Values: map[string]string{
			"key": "value",
		},
	}
	store := NewSimpleStore(&storeMock{
		secret: s,
	})

	secretMap, err := store.Get(1, "secret")
	if err != nil {
		t.Error("unexpected error: ", err)
	}

	if secretMap["key"] != "value" {
		t.Errorf("unexpected secret map: %v", secretMap)
	}
}
