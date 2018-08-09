package secret_test

import (
	"testing"

	"github.com/banzaicloud/pipeline/secret"
)

func TestRandomString(t *testing.T) {

	cases := []struct {
		name    string
		genType string
		length  int
		isError bool
	}{
		{name: "randAlpha", genType: "randAlpha", length: 12, isError: false},
		{name: "randAlphaNum", genType: "randAlphaNum", length: 13, isError: false},
		{name: "randNumeric", genType: "randNumeric", length: 14, isError: false},
		{name: "randAscii", genType: "randAscii", length: 99, isError: false},
		{name: "Wrong Type", genType: "randAlha", length: 0, isError: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := secret.RandomString(tc.name, tc.length)
			if err != nil {
				if !tc.isError {
					t.Errorf("Error occours: %s", err.Error())
				}
			} else if tc.isError {
				t.Errorf("Not occours error")
			}
			if len(result) != tc.length {
				t.Errorf("result length mismatch")
			}

		})
	}

}
