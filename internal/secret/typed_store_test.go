package secret_test

import (
	"testing"

	. "github.com/banzaicloud/pipeline/internal/secret"
	"github.com/banzaicloud/pipeline/secret"
)

func TestTypedStore_Get(t *testing.T) {
	s := &secret.SecretItemResponse{
		Values: map[string]string{
			"key": "value",
		},
		Type: "type",
	}
	store := NewTypedStore(
		&storeMock{
			secret: s,
		},
		"type",
	)

	// TODO: check secret
	_, err := store.Get(1, "secret")
	if err != nil {
		t.Error("unexpected error: ", err)
	}
}
